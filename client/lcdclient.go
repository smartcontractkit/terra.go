package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/msg"
	"github.com/smartcontractkit/terra.go/tx"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	tmclient "github.com/tendermint/tendermint/rpc/client/http"
	terraapp "github.com/terra-money/core/app"
	terraappparams "github.com/terra-money/core/app/params"
)

// LCDClient outer interface for building & signing & broadcasting tx
type LCDClient struct {
	HttpUrl       string
	ChainID       string
	GasPrice      msg.DecCoin
	GasAdjustment msg.Dec
	codec         *codec.LegacyAmino

	PrivKey        key.PrivKey
	EncodingConfig terraappparams.EncodingConfig

	httpc *http.Client
	tmc   *tmclient.HTTP
}

// NewLCDClient create new LCDClient
func NewLCDClient(WsUrl, HttpUrl, chainID string, gasPrice msg.DecCoin, gasAdjustment msg.Dec, tmKey key.PrivKey, httpTimeout time.Duration) (*LCDClient, error) {
	u, err := url.Parse(WsUrl)
	if err != nil {
		return nil, err
	}
	client, err := tmclient.New(fmt.Sprintf("tcp://%s", u.Host), u.Path)
	if err != nil {
		return nil, err
	}
	return &LCDClient{
		HttpUrl:        HttpUrl,
		ChainID:        chainID,
		GasPrice:       gasPrice,
		codec:          codec.NewLegacyAmino(),
		GasAdjustment:  gasAdjustment,
		PrivKey:        tmKey,
		EncodingConfig: terraapp.MakeEncodingConfig(),
		httpc:          &http.Client{Timeout: httpTimeout},
		tmc:            client,
	}, nil
}

// CreateTxOptions tx creation options
type CreateTxOptions struct {
	Msgs []msg.Msg
	Memo string

	// Optional parameters
	AccountNumber uint64
	Sequence      uint64
	GasLimit      uint64
	FeeAmount     msg.Coins

	SignMode      tx.SignMode
	FeeGranter    msg.AccAddress
	TimeoutHeight uint64
}

// CreateAndSignTx build and sign tx
func (lcd *LCDClient) CreateAndSignTx(ctx context.Context, options CreateTxOptions) (*tx.Builder, error) {
	txbuilder := tx.NewTxBuilder(lcd.EncodingConfig.TxConfig)
	txbuilder.SetFeeAmount(options.FeeAmount)
	txbuilder.SetFeeGranter(options.FeeGranter)
	txbuilder.SetGasLimit(options.GasLimit)
	txbuilder.SetMemo(options.Memo)
	txbuilder.SetMsgs(options.Msgs...)
	txbuilder.SetTimeoutHeight(options.TimeoutHeight)

	// use direct sign mode as default
	if tx.SignModeUnspecified == options.SignMode {
		options.SignMode = tx.SignModeDirect
	}

	if options.AccountNumber == 0 || options.Sequence == 0 {
		account, err := lcd.LoadAccount(ctx, msg.AccAddress(lcd.PrivKey.PubKey().Address()))
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to load account")
		}

		options.AccountNumber = account.GetAccountNumber()
		options.Sequence = account.GetSequence()
	}

	gasLimit := int64(options.GasLimit)
	if options.GasLimit == 0 {
		simulateRes, err := lcd.Simulate(ctx, txbuilder, options)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to simulate")
		}

		gasLimit = lcd.GasAdjustment.MulInt64(int64(simulateRes.GasInfo.GasUsed)).Ceil().RoundInt64()
		txbuilder.SetGasLimit(uint64(gasLimit))
	}

	if options.FeeAmount.IsZero() {
		computeTaxRes, err := lcd.ComputeTax(ctx, txbuilder)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to compute tax")
		}

		gasFee := msg.NewCoin(lcd.GasPrice.Denom, lcd.GasPrice.Amount.MulInt64(gasLimit).Ceil().RoundInt())
		txbuilder.SetFeeAmount(computeTaxRes.TaxAmount.Add(gasFee))
	}

	err := txbuilder.Sign(options.SignMode, tx.SignerData{
		AccountNumber: options.AccountNumber,
		ChainID:       lcd.ChainID,
		Sequence:      options.Sequence,
	}, lcd.PrivKey, true)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to sign tx")
	}

	return &txbuilder, nil
}

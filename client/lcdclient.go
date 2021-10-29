package client

import (
	"context"
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/terra-project/terra.go/msg"
	"github.com/terra-project/terra.go/tx"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	terraapp "github.com/terra-money/core/app"
	terraappparams "github.com/terra-money/core/app/params"
)

// LCDClient outer interface for building & signing & broadcasting tx
type LCDClient struct {
	URL           string
	ChainID       string
	GasPrice      msg.DecCoin
	GasAdjustment msg.Dec

	Keystore       keyring.Keyring
	EncodingConfig terraappparams.EncodingConfig

	c *http.Client
}

// NewLCDClient create new LCDClient
func NewLCDClient(URL, chainID string, gasPrice msg.DecCoin, gasAdjustment msg.Dec, keystore keyring.Keyring, httpTimeout time.Duration) *LCDClient {
	return &LCDClient{
		URL:            URL,
		ChainID:        chainID,
		GasPrice:       gasPrice,
		GasAdjustment:  gasAdjustment,
		Keystore:       keystore,
		EncodingConfig: terraapp.MakeEncodingConfig(),
		c:              &http.Client{Timeout: httpTimeout},
	}
}

// CreateTxOptions tx creation options
type CreateTxOptions struct {
	Msgs    []msg.Msg
	Memo    string
	Keyname string
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
		pubkey, err := lcd.Keystore.Key(options.Keyname)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to load key")
		}

		account, err := lcd.LoadAccount(ctx, msg.AccAddress(pubkey.GetPubKey().Address()))
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

		gasLimit = lcd.GasAdjustment.MulInt64(int64(simulateRes.GasInfo.GasUsed)).TruncateInt64()
		txbuilder.SetGasLimit(uint64(gasLimit))
	}

	if options.FeeAmount.IsZero() {
		computeTaxRes, err := lcd.ComputeTax(ctx, txbuilder)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to compute tax")
		}

		gasFee := msg.NewCoin(lcd.GasPrice.Denom, lcd.GasPrice.Amount.MulInt64(gasLimit).TruncateInt())
		txbuilder.SetFeeAmount(computeTaxRes.TaxAmount.Add(gasFee))
	}

	err := txbuilder.Sign(options.SignMode, tx.SignerData{
		AccountNumber: options.AccountNumber,
		ChainID:       lcd.ChainID,
		Sequence:      options.Sequence,
	}, lcd.Keystore, options.Keyname, true)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to sign tx")
	}

	return &txbuilder, nil
}

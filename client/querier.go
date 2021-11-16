package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"golang.org/x/net/context/ctxhttp"

	"github.com/smartcontractkit/terra.go/msg"
	"github.com/smartcontractkit/terra.go/tx"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	customauthtx "github.com/terra-money/core/custom/auth/tx"
)

// QueryAccountResData response
type QueryAccountResData struct {
	Address       msg.AccAddress `json:"address"`
	AccountNumber msg.Int        `json:"account_number"`
	Sequence      msg.Int        `json:"sequence"`
}

// QueryAccountRes response
type QueryAccountRes struct {
	Account QueryAccountResData `json:"account"`
}

func (lcd LCDClient) LoadAccount(ctx context.Context, address msg.AccAddress) (res authtypes.AccountI, err error) {
	resp, err := ctxhttp.Get(ctx, lcd.httpc, lcd.HttpUrl+fmt.Sprintf("/cosmos/auth/v1beta1/accounts/%s", address))
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to estimate")
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response code %d: %s", resp.StatusCode, string(out))
	}

	var response authtypes.QueryAccountResponse
	err = lcd.EncodingConfig.Marshaler.UnmarshalJSON(out, &response)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to unmarshal response")
	}

	return response.Account.GetCachedValue().(authtypes.AccountI), nil
}

// Simulate tx and get response
func (lcd LCDClient) Simulate(ctx context.Context, txbuilder tx.Builder, options CreateTxOptions) (*sdktx.SimulateResponse, error) {
	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: &secp256k1.PubKey{},
		Data: &signing.SingleSignatureData{
			SignMode: options.SignMode,
		},
		Sequence: options.Sequence,
	}
	if err := txbuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	bz, err := txbuilder.GetTxBytes()
	if err != nil {
		return nil, err
	}

	reqBytes, err := lcd.EncodingConfig.Marshaler.MarshalJSON(&sdktx.SimulateRequest{
		TxBytes: bz,
	})
	if err != nil {
		return nil, err
	}

	resp, err := ctxhttp.Post(ctx, lcd.httpc, lcd.HttpUrl+"/cosmos/tx/v1beta1/simulate", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to estimate")
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response code %d: %s", resp.StatusCode, string(out))
	}

	var response sdktx.SimulateResponse
	err = lcd.EncodingConfig.Marshaler.UnmarshalJSON(out, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
// Deprecated: It's only used for testing the deprecated Simulate gRPC endpoint
// using a proto Tx field.
type protoTxProvider interface {
	GetProtoTx() *sdktx.Tx
}

// ComputeTax compute tax
func (lcd LCDClient) ComputeTax(ctx context.Context, txbuilder tx.Builder) (*customauthtx.ComputeTaxResponse, error) {
	protoProvider := txbuilder.TxBuilder.(protoTxProvider)
	protoTx := protoProvider.GetProtoTx()
	reqBytes, err := lcd.EncodingConfig.Marshaler.MarshalJSON(&customauthtx.ComputeTaxRequest{
		Tx: protoTx,
	})
	if err != nil {
		return nil, err
	}

	resp, err := ctxhttp.Post(ctx, lcd.httpc, lcd.HttpUrl+"/terra/tx/v1beta1/compute_tax", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to estimate")
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response code %d: %s", resp.StatusCode, string(out))
	}

	var response customauthtx.ComputeTaxResponse
	err = lcd.EncodingConfig.Marshaler.UnmarshalJSON(out, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (lcd LCDClient) TxSearch(ctx context.Context, query string, prove bool, orderBy string) (*ctypes.ResultTxSearch, error) {
	return lcd.tmc.TxSearch(ctx, query, prove, nil, nil, orderBy)
}

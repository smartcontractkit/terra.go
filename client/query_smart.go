package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/terra.go/msg"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

type ABCIQueryParams struct {
	ContractAddress string
	Msg             []byte
}

func NewAbciQueryParams(contractAddress string, msg []byte) ABCIQueryParams {
	return ABCIQueryParams{contractAddress, msg}
}

func (lcd LCDClient) Query(ctx context.Context, addr msg.AccAddress, qMsg ABCIQueryParams, qResponse interface{}) error {
	bz, err := lcd.codec.MarshalJSON(qMsg)
	if err != nil {
		return err
	}
	resp, err := lcd.tmc.ABCIQuery(ctx, "custom/wasm/contractStore", tmbytes.HexBytes(hex.EncodeToString(bz)))

	if err != nil {
		return err
	}

	if resp.Response.Code != 0 {
		return fmt.Errorf(resp.Response.Log)
	}

	if err := json.Unmarshal(resp.Response.Value, &qResponse); err != nil {
		return err
	}

	return nil
}

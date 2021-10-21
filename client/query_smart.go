package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/smartcontractkit/terra.go/msg"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
)

// QuerySmart query smart contract store with qMsg marshalling it into JSON and encoding as a Base64 query param
func (lcd LCDClient) QuerySmart(ctx context.Context, addr msg.AccAddress, qMsg interface{}, qResponse interface{}) error {
	url := fmt.Sprintf("%s/terra/wasm/v1beta1/contracts/%s/store", lcd.URL, addr.String())
	reqBytes, err := json.Marshal(qMsg)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to marshal")
	}
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create query request")
	}
	sEnc := base64.StdEncoding.EncodeToString(reqBytes)
	q := r.URL.Query()
	q.Add("query_msg", sEnc)
	r.URL.RawQuery = q.Encode()
	resp, err := ctxhttp.Do(ctx, lcd.c, r)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to query")
	}
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to read response")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 response code %d: %s", resp.StatusCode, string(out))
	}
	if err := json.Unmarshal(out, &qResponse); err != nil {
		return err
	}
	return nil
}

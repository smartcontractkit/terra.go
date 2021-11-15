package client

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/terra.go/tx"

	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type BroadcastTxMethod string

const (
	// Returns with response from CheckTx, does not wait for DeliverTx
	BroadcastSync BroadcastTxMethod = "broadcast_tx_sync"
	// Returns right away with no response
	BroadcastAsync BroadcastTxMethod = "broadcast_tx_async"
	// Returns with response from CheckTx and DeliverTx
	BroadcastBlock BroadcastTxMethod = "broadcast_tx_commit"
)

// Broadcast - no-lint
func (lcd LCDClient) Broadcast(ctx context.Context, txbuilder *tx.Builder, bcMode BroadcastTxMethod) (*ctypes.ResultBroadcastTx, error) {
	txBytes, err := txbuilder.GetTxBytes()
	if err != nil {
		return nil, err
	}
	var resp *ctypes.ResultBroadcastTx

	switch mode := bcMode; mode {
	case BroadcastAsync:
		resp, err = lcd.broadcastAsync(ctx, txBytes)
	case BroadcastSync:
		resp, err = lcd.broadcastSync(ctx, txBytes)
	case BroadcastBlock:
		resp, err = lcd.broadcastBlock(ctx, txBytes)
	}

	return resp, err
}

func (lcd LCDClient) broadcastSync(ctx context.Context, tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	resp, err := lcd.tmc.BroadcastTxSync(ctx, tx)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("Broadcast Error: %s", resp.Log)
	}

	return resp, nil
}

func (lcd LCDClient) broadcastAsync(ctx context.Context, tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	resp, err := lcd.tmc.BroadcastTxAsync(ctx, tx)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("Broadcast Error: %s", resp.Log)
	}

	return resp, nil
}
func (lcd LCDClient) broadcastBlock(ctx context.Context, tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	resp, err := lcd.tmc.BroadcastTxCommit(ctx, tx)
	if err != nil {
		return nil, err
	}

	if resp.DeliverTx.Code != 0 {
		return nil, fmt.Errorf("Broadcast Error: %s", resp.DeliverTx.Log)
	}

	return &ctypes.ResultBroadcastTx{
		Code:      resp.DeliverTx.Code,
		Data:      resp.DeliverTx.Data,
		Log:       resp.DeliverTx.Log,
		Codespace: resp.DeliverTx.Codespace,
		Hash:      resp.Hash,
	}, nil
}

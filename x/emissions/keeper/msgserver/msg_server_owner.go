package msgserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// UpdateOwner updates the payout owner for a registered node.
func (ms msgServer) UpdateOwner(ctx context.Context, msg *types.UpdateOwnerRequest) (_ *types.UpdateOwnerResponse, err error) {
	defer metrics.RecordMetrics("UpdateOwner", time.Now(), &err)

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	var oldOwner string
	if msg.IsReputer {
		oldOwner, err = ms.k.UpdateReputerOwner(ctx, msg.Sender, msg.NewOwner)
	} else {
		oldOwner, err = ms.k.UpdateWorkerOwner(ctx, msg.Sender, msg.NewOwner)
	}
	if err != nil {
		return nil, err
	}

	types.EmitNodeOwnerUpdatedEvent(ctx, msg.Sender, oldOwner, msg.NewOwner, msg.IsReputer)

	return &types.UpdateOwnerResponse{
		Success: true,
		Message: "Node owner updated",
	}, nil
}

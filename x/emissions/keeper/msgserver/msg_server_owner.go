package msgserver

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// TransferActorOwnership updates the payout owner for a registered node.
func (ms msgServer) TransferActorOwnership(ctx context.Context, msg *types.TransferActorOwnershipRequest) (_ *types.TransferActorOwnershipResponse, err error) {
	defer metrics.RecordMetrics("TransferActorOwnership", time.Now(), &err)

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var oldOwner string
	if msg.IsReputer {
		nodeInfo, getErr := ms.k.GetReputerInfo(sdkCtx, msg.Sender)

		if errors.Is(getErr, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer %s", msg.Sender)
		} else if getErr != nil {
			return nil, errorsmod.Wrap(getErr, "error getting reputer info")
		}
		if nodeInfo.NodeAddress != msg.Sender {
			return nil, errorsmod.Wrapf(
				types.ErrInvariantFailure,
				"stored reputer node address %s does not match key %s",
				nodeInfo.NodeAddress,
				msg.Sender,
			)
		}
		oldOwner, err = ms.k.UpdateReputerOwner(ctx, msg.Sender, msg.NewOwner)
	} else {
		nodeInfo, getErr := ms.k.GetWorkerInfo(sdkCtx, msg.Sender)

		if errors.Is(getErr, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "worker %s", msg.Sender)
		} else if getErr != nil {
			return nil, errorsmod.Wrap(getErr, "error getting worker info")
		}
		if nodeInfo.NodeAddress != msg.Sender {
			return nil, errorsmod.Wrapf(
				types.ErrInvariantFailure,
				"stored worker node address %s does not match key %s",
				nodeInfo.NodeAddress,
				msg.Sender,
			)
		}
		oldOwner, err = ms.k.UpdateWorkerOwner(ctx, msg.Sender, msg.NewOwner)
	}
	if err != nil {
		return nil, err
	}

	types.EmitNodeOwnerUpdatedEvent(ctx, msg.Sender, oldOwner, msg.NewOwner, msg.IsReputer)

	return &types.TransferActorOwnershipResponse{}, nil
}

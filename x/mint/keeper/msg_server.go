package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.MsgServiceServer = msgServiceServer{}

// msgServiceServer is a wrapper of Keeper.
type msgServiceServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the x/mint MsgServer interface.
func NewMsgServerImpl(k Keeper) types.MsgServiceServer {
	return &msgServiceServer{
		Keeper: k,
	}
}

// UpdateParams updates the params.
func (ms msgServiceServer) UpdateParams(ctx context.Context, msg *types.UpdateParamsRequest) (*types.UpdateParamsResponse, error) {
	isAdmin, err := ms.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint update params", msg.Sender)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.UpdateParamsResponse{}, nil
}

func (ms msgServiceServer) RecalculateTargetEmission(ctx context.Context, msg *types.RecalculateTargetEmissionRequest) (*types.RecalculateTargetEmissionResponse, error) {
	isAdmin, err := ms.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint recalculate target emission", msg.Sender)
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	moduleParams, err := ms.Keeper.GetParams(sdkCtx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting module params")
	}

	blocksPerMonth, err := ms.Keeper.GetParamsBlocksPerMonth(sdkCtx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting blocks per month")
	}

	vPercentADec, err := ms.Keeper.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting validators vs allora percent reward")
	}
	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return nil, errors.Wrap(err, "error getting validators vs allora percent reward from dec")
	}
	ecosystemMintSupplyRemaining, err := ms.Keeper.GetEcosystemMintSupplyRemaining(sdkCtx, moduleParams)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ecosystem mint supply remaining")
	}
	ecosystemBalance, err := ms.Keeper.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ecosystem balance")
	}

	_, _, err = RecalculateTargetEmission(
		sdkCtx,
		ms.Keeper,
		uint64(sdkCtx.BlockHeight()),
		blocksPerMonth,
		moduleParams,
		ecosystemBalance,
		ecosystemMintSupplyRemaining,
		vPercent,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error recalculating target emission")
	}

	return &types.RecalculateTargetEmissionResponse{}, nil
}

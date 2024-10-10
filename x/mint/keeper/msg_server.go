package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.MsgServiceServer = msgServiceServer{} //nolint: exhaustruct

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
// if the sender is a whitelist admin
// it optionally also allows recalculating the target emission
func (ms msgServiceServer) UpdateParams(ctx context.Context, msg *types.UpdateParamsRequest) (*types.UpdateParamsResponse, error) {
	isAdmin, err := ms.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "error checking if admin")
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint update params", msg.Sender)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating params")
	}

	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, errors.Wrap(err, "error setting params")
	}
	err = emissionstypes.ValidateBlocksPerMonth(msg.BlocksPerMonth)
	if err != nil {
		return nil, errors.Wrap(err, "error validating blocks per month")
	}
	err = ms.Keeper.SetEmissionsParamsBlocksPerMonth(ctx, msg.BlocksPerMonth)
	if err != nil {
		return nil, errors.Wrap(err, "error setting blocks per month")
	}

	if msg.RecalculateTargetEmission {
		sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			err := getArgsForRecalculate(ctx, ms)
		if err != nil {
			return nil, errors.Wrap(err, "error getting args for recalculate")
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
	}

	return &types.UpdateParamsResponse{}, nil
}

// RecalculateTargetEmission recalculates the target emission
// if the sender is a whitelist admin
func (ms msgServiceServer) RecalculateTargetEmission(ctx context.Context, msg *types.RecalculateTargetEmissionRequest) (*types.RecalculateTargetEmissionResponse, error) {
	isAdmin, err := ms.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "error checking if admin")
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint recalculate target emission", msg.Sender)
	}

	sdkCtx, moduleParams, blocksPerMonth, vPercent, ecosystemMintSupplyRemaining, ecosystemBalance, err := getArgsForRecalculate(ctx, ms)
	if err != nil {
		return nil, errors.Wrap(err, "error getting args for recalculate")
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

// helper function to reduce repetitive code
// getting parameters for the recalculate target emission arguments
func getArgsForRecalculate(ctx context.Context, ms msgServiceServer) (
	sdkCtx sdk.Context,
	moduleParams types.Params,
	blocksPerMonth uint64,
	vPercent math.LegacyDec,
	ecosystemMintSupplyRemaining math.Int,
	ecosystemBalance math.Int,
	err error,
) {
	sdkCtx = sdk.UnwrapSDKContext(ctx)

	moduleParams, err = ms.Keeper.GetParams(sdkCtx)
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting module params")
	}

	blocksPerMonth, err = ms.Keeper.GetParamsBlocksPerMonth(sdkCtx)
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting blocks per month")
	}

	vPercentADec, err := ms.Keeper.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting validators vs allora percent reward")
	}
	vPercent, err = vPercentADec.SdkLegacyDec()
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting validators vs allora percent reward from dec")
	}
	ecosystemMintSupplyRemaining, err = ms.Keeper.GetEcosystemMintSupplyRemaining(sdkCtx, moduleParams)
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting ecosystem mint supply remaining")
	}
	ecosystemBalance, err = ms.Keeper.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return sdkCtx, moduleParams, blocksPerMonth, vPercent,
			ecosystemMintSupplyRemaining, ecosystemBalance,
			errors.Wrap(err, "error getting ecosystem balance")
	}

	return sdkCtx, moduleParams, blocksPerMonth, vPercent,
		ecosystemMintSupplyRemaining, ecosystemBalance, nil
}

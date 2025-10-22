package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/mint/types"
)

var _ types.MsgServiceServer = msgServiceServer{} // nolint: exhaustruct

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
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	isAdmin, err := ms.IsWhitelistAdmin(sdkCtx, msg.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "error checking if admin")
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint update params", msg.Sender)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating params")
	}

	moduleParams, err := ms.Keeper.GetParams(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting module params")
	}

	startingEmissionsBlockHeight, err := ms.Keeper.GetStartingEmissionsBlockHeight(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting starting emissions block height")
	}

	// TODO Add here: if emissions are to be enabled in the request, and it was disabled before,
	// we need to set the starting block height,
	// and schedule a job to recalculate the target emission at the end of the month and every month thereafter.
	if msg.Params.EmissionEnabled {
		if !moduleParams.EmissionEnabled && startingEmissionsBlockHeight == 0 {
			// only if emissions move from disabled to enabled, we need to set the starting block height
			// and schedule a job to recalculate the target emission at the end of the month and every month thereafter.
			err = ms.Keeper.SetStartingEmissionsBlockHeight(ctx, sdkCtx.BlockHeight())
			if err != nil {
				return nil, errors.Wrap(err, "error setting starting emissions block height")
			}
			// force the months already unlocked to 0
			err = ms.Keeper.SetMonthsAlreadyUnlocked(ctx, math.NewInt(0))
			if err != nil {
				return nil, errors.Wrap(err, "error setting months already unlocked")
			}
		}
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
		blockCountSinceTGE := uint64(sdkCtx.BlockHeight() - startingEmissionsBlockHeight)
		_, _, err = RecalculateTargetEmission(
			sdkCtx,
			ms.Keeper,
			blockCountSinceTGE,
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

	types.EmitNewParamsSetEvent(ctx, msg.Params, msg.BlocksPerMonth)
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

	startingEmissionsBlockHeight, err := ms.Keeper.GetStartingEmissionsBlockHeight(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting starting emissions block height")
	}
	blockCountSinceTGE := uint64(sdkCtx.BlockHeight() - startingEmissionsBlockHeight)
	_, _, err = RecalculateTargetEmission(
		sdkCtx,
		ms.Keeper,
		blockCountSinceTGE,
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

package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/mint/metrics"
)

func EmitNewTokenomicsSetEvent(ctx sdk.Context, stakedTokenAmount, circulatingAmount, emissionsAmount math.Int) {
	metrics.IncrProducerEventCount(metrics.TOKENOMICS_SET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTokenomicsSetEventBase(stakedTokenAmount, circulatingAmount, emissionsAmount))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewTokenomicsSetEvent", "error", err)
	}
}

func NewTokenomicsSetEventBase(stakedTokenAmount, circulatingAmount, emissionsAmount math.Int) proto.Message {
	return &EventTokenomicsSet{
		StakedTokenAmount: stakedTokenAmount,
		CirculatingSupply: circulatingAmount,
		EmissionsAmount:   emissionsAmount,
	}
}

func EmitNewEcosystemTokenMintSetEvent(ctx sdk.Context, blockHeight uint64, amount math.Int) {
	metrics.IncrProducerEventCount(metrics.ECOSYSTEM_TOKEN_MINT_SET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(EcosystemTokenMintSetEventBase(blockHeight, amount))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewEcosystemTokenMintSetEvent", "error", err)
	}
}

func EcosystemTokenMintSetEventBase(blockHeight uint64, tokenAmount math.Int) proto.Message {
	return &EventEcosystemTokenMintSet{
		BlockHeight: blockHeight,
		TokenAmount: tokenAmount,
	}
}

func EmitNewRewardCurrentBlockEmissionEvent(ctx sdk.Context, blockHeight uint64, amount math.Int) {
	metrics.IncrProducerEventCount(metrics.REWARD_CURRENT_BLOCK_EMISSION_EVENT)
	err := ctx.EventManager().EmitTypedEvent(RewardCurrentBlockEmissionEventBase(blockHeight, amount))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewRewardCurrentBlockEmissionEvent", "error", err)
	}
}

func RewardCurrentBlockEmissionEventBase(blockHeight uint64, tokenAmount math.Int) proto.Message {
	return &EventRewardCurrentBlockEmission{
		BlockHeight: blockHeight,
		TokenAmount: tokenAmount,
	}
}

func EmitNewRecalculateTargetEmissionEvent(ctx context.Context, emissionPerUnitStakedToken math.LegacyDec, emissionPerMonth, blockEmission math.Int) {
	metrics.IncrProducerEventCount(metrics.RECALCULATE_TARGET_EMISSION_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	emissionPerUnitStakedTokenDec, err := alloraMath.NewDecFromSdkLegacyDec(emissionPerUnitStakedToken)
	if err != nil {
		sdkCtx.Logger().Warn("Error converting emission per unit staked token to allora dec", "error", err)
		return
	}
	err = sdkCtx.EventManager().EmitTypedEvent(NewRecalculateTargetEmissionEventBase(emissionPerUnitStakedTokenDec, emissionPerMonth, blockEmission))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewRecalculateTargetEmissionEvent", "error", err)
	}
}

func NewRecalculateTargetEmissionEventBase(emissionPerUnitStakedToken alloraMath.Dec, emissionPerMonth, blockEmission math.Int) proto.Message {
	return &EventRecalculateTargetEmission{
		EmissionPerUnitStakedToken: emissionPerUnitStakedToken,
		EmissionPerMonth:           emissionPerMonth,
		BlockEmission:              blockEmission,
	}
}

func EmitNewParamsSetEvent(ctx context.Context, params Params, blocksPerMonth uint64) {
	metrics.IncrProducerEventCount(metrics.PARAMS_SET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewParamsSetEventBase(params, blocksPerMonth))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewParamsSetEvent", "error", err)
	}
}

func NewParamsSetEventBase(params Params, blocksPerMonth uint64) proto.Message {
	return &EventParamsSet{
		Params:         &params,
		BlocksPerMonth: blocksPerMonth,
	}
}

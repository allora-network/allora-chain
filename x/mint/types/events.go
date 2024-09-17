package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func EmitNewTokenomicsSetEvent(ctx sdk.Context, stakedTokenAmount, circulatingAmount, emissionsAmount math.Int) {
	err := ctx.EventManager().EmitTypedEvent(NewTokenomicsSetEventBase(stakedTokenAmount, circulatingAmount, emissionsAmount))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewTokenomicsSetEvent: ", err.Error())
	}
}

func NewTokenomicsSetEventBase(stakedTokenAmount, circulatingAmount, emissionsAmount math.Int) proto.Message {
	return &EventTokenomicsSet{
		StakedTokenAmount: stakedTokenAmount,
		CirculatingSupply: circulatingAmount,
		EmissionsAmount:   emissionsAmount,
	}
}

func EmitNewEcosystemTokenMintSetEvent(ctx sdk.Context, blockHeight int64, amount math.Int) {
	err := ctx.EventManager().EmitTypedEvent(EcosystemTokenMintSetEventBase(blockHeight, amount))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewEcosystemTokenMintSetEvent: ", err.Error())
	}
}

func EcosystemTokenMintSetEventBase(blockHeight int64, tokenAmount math.Int) proto.Message {
	return &EventEcosystemTokenMintSet{
		BlockHeight: blockHeight,
		TokenAmount: tokenAmount,
	}
}

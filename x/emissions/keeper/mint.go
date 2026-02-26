package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k Keeper) GetTotalStake(ctx context.Context) (cosmosMath.Int, error) {
	return k.stakingKeeper.GetTotalStake(ctx)
}

func (k Keeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (alloraMath.Dec, error) {
	return k.scoresKeeper.GetPreviousPercentageRewardToStakedReputers(ctx)
}

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.paramsKeeper.GetParams(ctx)
}

func (k Keeper) IsWhitelistAdmin(ctx context.Context, admin string) (bool, error) {
	return k.whitelistsKeeper.IsWhitelistAdmin(ctx, admin)
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.paramsKeeper.SetParams(ctx, params)
}

func (k Keeper) HandleMonthlyRewardsReset(ctx context.Context) error {
	return k.weightsKeeper.HandleMonthlyRewardsReset(ctx)
}

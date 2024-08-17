package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/mint/types"
)

// InitGenesis new mint genesis
func (k Keeper) InitGenesis(ctx context.Context, ak types.AccountKeeper, data *types.GenesisState) {
	if err := k.Params.Set(ctx, data.Params); err != nil {
		panic(err)
	}

	err := k.PreviousRewardEmissionPerUnitStakedToken.Set(
		ctx,
		data.PreviousRewardEmissionPerUnitStakedToken,
	)
	if err != nil {
		panic(err)
	}

	err = k.PreviousBlockEmission.Set(ctx, data.PreviousBlockEmission)
	if err != nil {
		panic(err)
	}

	if err := k.EcosystemTokensMinted.Set(ctx, data.EcosystemTokensMinted); err != nil {
		panic(err)
	}

	ak.GetModuleAccount(ctx, types.ModuleName)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	previousRewardEmissionPerUnitStakedToken, err := k.PreviousRewardEmissionPerUnitStakedToken.Get(ctx)
	if err != nil {
		panic(err)
	}

	previousBlockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		panic(err)
	}

	ecosystemTokensMinted, err := k.EcosystemTokensMinted.Get(ctx)
	if err != nil {
		panic(err)
	}

	return types.NewGenesisState(
		params,
		previousRewardEmissionPerUnitStakedToken,
		previousBlockEmission,
		ecosystemTokensMinted,
	)
}

package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/mint/types"
)

// InitGenesis new mint genesis
func (keeper Keeper) InitGenesis(ctx context.Context, ak types.AccountKeeper, data *types.GenesisState) {
	if err := keeper.Params.Set(ctx, data.Params); err != nil {
		panic(err)
	}

	if err := keeper.PreviousRewardEmissionsPerUnitStakedToken.Set(ctx, data.PreviousRewardEmissionsPerUnitStakedToken); err != nil {
		panic(err)
	}

	if err := keeper.EcosystemTokensMinted.Set(ctx, data.EcosystemTokensMinted); err != nil {
		panic(err)
	}

	ak.GetModuleAccount(ctx, types.ModuleName)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (keeper Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := keeper.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	previousRewardEmissionPerUnitStakedToken, err := keeper.PreviousRewardEmissionsPerUnitStakedToken.Get(ctx)
	if err != nil {
		panic(err)
	}

	ecosystemTokensMinted, err := keeper.EcosystemTokensMinted.Get(ctx)
	if err != nil {
		panic(err)
	}

	return types.NewGenesisState(params, previousRewardEmissionPerUnitStakedToken, ecosystemTokensMinted)
}

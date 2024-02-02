package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	state "github.com/upshot-tech/protocol-state-machine-module"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *state.GenesisState) error {
	if err := k.params.Set(ctx, data.Params); err != nil {
		return err
	}

	// ensure the module account exists
	moduleAccount := k.authKeeper.GetModuleAccount(ctx, state.ModuleName)
	k.authKeeper.SetModuleAccount(ctx, moduleAccount)
	k.SetLastRewardsUpdate(ctx, 0)
	k.SetTotalStake(ctx, cosmosMath.NewUint(0))
	k.IncrementTopicId(ctx) // reserve topic ID 0 for future use

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*state.GenesisState, error) {
	params, err := k.params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &state.GenesisState{
		Params: params,
	}, nil
}

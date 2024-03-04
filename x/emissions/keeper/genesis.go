package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *state.GenesisState) error {

	// ensure the module account exists
	stakingModuleAccount := k.authKeeper.GetModuleAccount(ctx, state.AlloraStakingModuleName)
	k.authKeeper.SetModuleAccount(ctx, stakingModuleAccount)
	requestsModuleAccount := k.authKeeper.GetModuleAccount(ctx, state.AlloraRequestsModuleName)
	k.authKeeper.SetModuleAccount(ctx, requestsModuleAccount)
	if err := k.SetLastRewardsUpdate(ctx, 0); err != nil {
		return err
	}
	if err := k.SetTotalStake(ctx, cosmosMath.NewUint(0)); err != nil {
		return err
	}
	// reserve topic ID 0 for future use
	if _, err := k.IncrementTopicId(ctx); err != nil {
		return err
	}

	// add core team to the whitelists
	if err := k.addCoreTeamToWhitelists(ctx, data.CoreTeamAddresses); err != nil {
		return err
	}

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*state.GenesisState, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &state.GenesisState{
		Params: params,
	}, nil
}

func (k *Keeper) addCoreTeamToWhitelists(ctx context.Context, coreTeamAddresses []string) error {
	for _, addr := range coreTeamAddresses {
		accAddress, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
		k.AddWhitelistAdmin(ctx, accAddress)
		k.AddToTopicCreationWhitelist(ctx, accAddress)
		k.AddToWeightSettingWhitelist(ctx, accAddress)
	}

	return nil
}

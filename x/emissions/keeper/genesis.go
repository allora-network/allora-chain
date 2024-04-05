package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {

	// ensure the module account exists
	stakingModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraStakingAccountName)
	k.authKeeper.SetModuleAccount(ctx, stakingModuleAccount)
	requestsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraRequestsAccountName)
	k.authKeeper.SetModuleAccount(ctx, requestsModuleAccount)
	alloraRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraRewardsModuleAccount)
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
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GenesisState{
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
		k.AddToReputerWhitelist(ctx, accAddress)
	}

	return nil
}

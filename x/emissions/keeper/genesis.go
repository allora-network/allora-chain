package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
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
	alloraPendingRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraPendingRewardsModuleAccount)
	if err := k.SetTotalStake(ctx, cosmosMath.ZeroInt()); err != nil {
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

	// For mint module inflation rate calculation set the initial
	// "previous percentage of rewards that went to staked reputers" to 30%
	if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.3")); err != nil {
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
		k.AddWhitelistAdmin(ctx, addr)
	}
	return nil
}

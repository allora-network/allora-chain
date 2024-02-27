package keeper

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *state.GenesisState) error {
	if err := k.SetParams(ctx, data.Params); err != nil {
		return err
	}

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
	if err := k.addCoreTeamToWhitelists(ctx); err != nil {
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

func (k *Keeper) addCoreTeamToWhitelists(ctx context.Context) error {
	coreTeamAddresses := []string{
		"allo1srjcynn6jw5l709upwwx70gs3eyjdmkgufcqdt",
		"allo1uu2fk9gmjkmy8qme46w8hr8j2exfjnsqnweznz",
		"allo1fk0pt0fj03fxdyyq0mhj5cfq5kv9ncwmtqvy7u",
		"allo1d6hegr59ftc43xxd2n2vn5ndley699jefwdgq8",
		"allo1xw354nrfpsw6x3sf7aqnrek0wluyx58zv2uc75",
		"allo1d54qdljsc3srsy8fz6zzrx90hyzuevnccg88md",
		"allo1fumaxxyjwdv7y5uh6wxslhxmqf32jtvnr8edlf",
		"allo1memyg7exjzjdpdv98cfm6u0y0lsz4ev8mk57hc",
		"allo1shzv768qrxaextjwz0aj6nzhm3cyy4pdug8jy6",
		"allo12heywwqc75mgk6qg3n0mryw58jn6ujtp7tfvs9",
	}

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

package v2

import (
	"fmt"

	"cosmossdk.io/collections"
	emissionsTypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// for upgrade v0.2.8, delete the contents of
// the stakeRemoval and delegateStakeRemoval
// key value stores
func MigrateStore(
	ctx sdk.Context,
	stakeRemoval *collections.Map[collections.Pair[uint64, string], emissionsTypes.StakePlacement],
	delegateStakeRemoval *collections.Map[collections.Triple[uint64, string, string], emissionsTypes.DelegateStakePlacement],
) error {
	ctx.Logger().Warn(fmt.Sprintf(
		"###################################################" +
			"### MIGRATING STORE FROM VERSION 1 TO VERSION 2 ###" +
			"###################################################",
	))
	err := stakeRemoval.Clear(ctx, nil)
	if err != nil {
		return err
	}
	return delegateStakeRemoval.Clear(ctx, nil)
}

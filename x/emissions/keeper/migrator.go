package keeper

import (
	v2 "github.com/allora-network/allora-chain/x/emissions/migrations/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns Migrator instance for the state migration.
func NewMigrator(k Keeper) Migrator {
	return Migrator{
		keeper: k,
	}
}

// Migrate1to2 migrates the x/mint module state from the consensus version 1 to
// version 2. For now this is a no-op since we don't actually have any state to
// upgrade, but in the future this function or a 2to3 would be used to handle
// state migrations between versions of the emissions module.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx)
}

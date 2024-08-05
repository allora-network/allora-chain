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

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc)
}

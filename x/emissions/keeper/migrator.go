package keeper

import (
	"github.com/allora-network/allora-chain/x/emissions/migrations/v0_3_0"
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

func (m Migrator) Migrate0_2_14to0_3_0(ctx sdk.Context) error {
	return v0_3_0.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc)
}

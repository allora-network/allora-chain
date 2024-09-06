package v0_4_1 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	UpgradeName = "v0.4.1"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return moduleManager.RunMigrations(ctx, configurator, vm)
	}
}

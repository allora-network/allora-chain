package vintegration

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	UpgradeName = "vintegration"
)

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		fmt.Println(">>>>>>>>>>>>>>>>>>> vintegration upgrade handler")
		return moduleManager.RunMigrations(ctx, configurator, vm)
	}
}

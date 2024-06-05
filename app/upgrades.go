package app

import (
	"fmt"

	"github.com/allora-network/allora-chain/app/upgrades/vIntegration"
)

func (app *AlloraApp) setupUpgradeHandlers() {
	// set up the vIntegration upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		vIntegration.UpgradeName,
		vIntegration.CreateUpgradeHandler(
			app.ModuleManager, app.Configurator(),
		),
	)

	// check we will be able to do the upgrade
	_, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}
}

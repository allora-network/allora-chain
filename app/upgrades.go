package app

import (
	v0_2_8 "github.com/allora-network/allora-chain/app/upgrades/v0.2.8"
)

func (app *AlloraApp) setupUpgradeHandlers() {
	// set up the v0.2.8 upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		v0_2_8.UpgradeName,
		v0_2_8.CreateUpgradeHandler(
			app.ModuleManager, app.Configurator(),
		),
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		app.Logger().Error("failed to read upgrade info from disk", "error", err)
		return
	}
	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

}

package app

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
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
	if err == nil {
		if upgradeInfo.Name == v0_2_8.UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
			storeUpgrades := storetypes.StoreUpgrades{
				Deleted: []string{"capability"},
			}

			// configure store loader that checks if version == upgradeHeight and applies store upgrades
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

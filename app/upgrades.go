package app

import (
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/allora-network/allora-chain/app/upgrades/v0_3_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_4_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_5_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_6_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_7_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_8_0"
)

var upgradeHandlers = []upgrades.Upgrade{
	v0_3_0.Upgrade,
	v0_4_0.Upgrade,
	v0_5_0.Upgrade,
	v0_6_0.Upgrade,
	v0_7_0.Upgrade,
	v0_8_0.Upgrade,
	// Add more upgrade handlers here
	// ...
}

func (app *AlloraApp) setupUpgradeHandlers(appKeepers *keepers.AppKeepers) {
	for _, handler := range upgradeHandlers {
		app.UpgradeKeeper.SetUpgradeHandler(handler.UpgradeName,
			handler.CreateUpgradeHandler(app.ModuleManager, app.Configurator(), appKeepers))

	}
}

func (app *AlloraApp) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range upgradeHandlers {
		if upgradeInfo.Name == upgrade.UpgradeName {
			storeUpgrades := upgrade.StoreUpgrades
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

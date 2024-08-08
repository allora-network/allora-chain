package app

import (
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/allora-network/allora-chain/app/upgrades/v0_3_0"
)

var upgradeHandlers = []upgrades.Upgrade{
	v0_3_0.Upgrade,
	// Add more upgrade handlers here
	// ...
}

func (app *AlloraApp) setupUpgradeHandlers() {
	for _, handler := range upgradeHandlers {
		app.UpgradeKeeper.SetUpgradeHandler(handler.UpgradeName, handler.CreateUpgradeHandler(app.ModuleManager, app.Configurator()))
	}
}

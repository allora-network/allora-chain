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

// #1 this is not how you do an upgrade plan, you should be
// submiting a /x/gov proposal to the chain through a transaction
// #2 this code does not appear to actually be called anywhere
// when we do an upgrade we would not do it throught this function
// this code should be deleted at a future date as soon
// as it is not needed as a reference
// func (app *AlloraApp) scheduleUpgrades(ctx sdk.Context) {
// 	upgradePlanV0_3_0 := upgradetypes.Plan{
// 		Name:   v0_3_0.UpgradeName,
// 		Height: 10, // TODO update this to the correct height
// 		Info:   "https://link.to/v0.3.0-info",
// 	}
// 	app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlanV0_3_0)
// }

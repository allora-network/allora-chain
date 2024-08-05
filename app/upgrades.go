package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/allora-network/allora-chain/app/upgrades/v0_3_0"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (app *AlloraApp) scheduleUpgrades(ctx sdk.Context) {
	upgradePlanV0_3_0 := upgradetypes.Plan{
		Name:   v0_3_0.UpgradeName,
		Height: 10, // TODO update this to the correct height
		Info:   "https://link.to/v0.3.0-info",
	}
	app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlanV0_3_0)
}

package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/allora-network/allora-chain/app/upgrades/v1_0_0"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var upgradeHandlers = []upgrades.Upgrade{
	v1_0_0.Upgrade,
	// Add more upgrade handlers here
	// ...
}

func (app *AlloraApp) setupUpgradeHandlers() {
	for _, handler := range upgradeHandlers {
		app.UpgradeKeeper.SetUpgradeHandler(handler.UpgradeName, handler.CreateUpgradeHandler(app.ModuleManager, app.Configurator()))
	}
}

func (app *AlloraApp) scheduleUpgrades(ctx sdk.Context) {
	upgradePlanV1_0_0 := upgradetypes.Plan{
		Name:   v1_0_0.UpgradeName,
		Height: 10, // TODO update this to the correct height
		Info:   "https://link.to/v1.0.0-info",
	}
	app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlanV1_0_0)
}

package upgrades

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string
	// Function that creates an upgrade handler
	CreateUpgradeHandler func(mm *module.Manager, configurator module.Configurator) upgradetypes.UpgradeHandler
}

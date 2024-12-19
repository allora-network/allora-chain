package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// Upgrade defines an upgrade that is to be acted upon by state migrations from the SDK `x/upgrade` module, it defines
// the necessary fields that a SoftwareUpgradeProposal must have written in order for the state migration to go smoothly.
// An upgrade must implement this struct, and then set it in the app.go.
type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string

	// CreateUpgradeHandler defines the function that creates an upgrade handler
	CreateUpgradeHandler func(mm *module.Manager, configurator module.Configurator, keepers *keepers.AppKeepers) upgradetypes.UpgradeHandler

	// StoreUpgrades should be used for any new modules introduced, new modules deleted, or store names renamed.
	StoreUpgrades storetypes.StoreUpgrades
}

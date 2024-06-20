package v0_2_8

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	// UpgradeName plan name used by /x/gov
	UpgradeName = "v0.2.8"
	// UpgradeInfo cosmovisor binaries used for the upgrade
	UpgradeInfo = `'{"binaries":{"darwin/amd64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.8/allorad_darwin_amd64","darwin/arm64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.7/allorad_darwin_arm64","linux/arm64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.7/allorad_linux_arm64","linux/amd64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.7/allorad_linux_amd64","windows/amd64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.7/allorad_windows_amd64.exe","windows/arm64":"https://github.com/allora-network/allora-chain/releases/download/v0.2.7/allorad_windows_arm64.exe"}}'`
)

// upgrade handler for v0.2.8 for the /x/upgrade module
func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return moduleManager.RunMigrations(ctx, configurator, vm)
	}
}

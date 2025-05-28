package v0_13_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"fmt"
	"path/filepath"

	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	dbm "github.com/cometbft/cometbft-db"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	UpgradeName = "v0.13.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: nil, Renamed: nil, Deleted: nil},
	PreStartupUpgrade:    PreStartupUpgradeHandler,
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("RUN MIGRATIONS")

		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}

func PreStartupUpgradeHandler(appOpts servertypes.AppOptions) error {
	stateStore, err := openStateStore(appOpts)
	if err != nil {
		return err
	}
	defer stateStore.Close()

	return resetValidatorsProposerPriorities(stateStore)
}

// resetValidatorsProposerPriorities resets the proposer priorities of all validators as if the validator set was just initialized.
// This is necessary to ensure that the proposer priorities are perfectly consistent across the network.
//
// NOTE: This is idempotent, if run multiple times the same proposer priorities will be set again.
func resetValidatorsProposerPriorities(stateStore sm.Store) error {
	state, err := stateStore.Load()
	if err != nil {
		return err
	}

	vp := state.NextValidators.TotalVotingPower()

	// Reset the proposer priorities of all validators as a brand new set with same priorities
	for _, val := range state.NextValidators.Validators {
		val.ProposerPriority = -(vp + (vp >> 3))
	}

	// Increment the proposer priorities to set an appropriate proposer
	state.NextValidators.IncrementProposerPriority(1)

	return stateStore.Save(state)
}

func openStateStore(appOpts servertypes.AppOptions) (sm.Store, error) {
	rootDir, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok {
		return nil, fmt.Errorf("home option not found or not a string in app options")
	}
	dbBackend, ok := appOpts.Get("db_backend").(string)
	if !ok {
		return nil, fmt.Errorf("db_backend option not found or not a string in app options")
	}
	dbPath, ok := appOpts.Get("db_dir").(string)
	if !ok {
		return nil, fmt.Errorf("db_dir option not found or not a string in app options")
	}
	discardABCIResponses, ok := appOpts.Get("storage.discard_abci_responses").(bool)
	if !ok {
		return nil, fmt.Errorf("storage.discard_abci_responses option not found or not a string in app options")
	}

	dbType := dbm.BackendType(dbBackend)
	dbDir := rootify(dbPath, rootDir)

	stateDB, err := dbm.NewDB("state", dbType, dbDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not open state db")
	}

	return sm.NewStore(stateDB, sm.StoreOptions{
		// Should not have any impact: there's no ABCI responses to manage for the usage we have.
		DiscardABCIResponses: discardABCIResponses,
	}), nil
}

func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

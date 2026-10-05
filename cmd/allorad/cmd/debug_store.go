package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/store/rootmulti"

	"github.com/allora-network/allora-chain/app"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
)

const (
	flagRollbackTo = "to"
	flagYes        = "yes"
)

// NewDebugStoreCmd returns the `debug-store` command group: read-only diagnostics and a
// gated recovery path for a multistore left inconsistent by an interrupted migration
// (the stock `rollback` command panics on such a store because it loads the app with
// loadLatest=true before it can act).
func NewDebugStoreCmd() *cobra.Command {
	cmd := &cobra.Command{ //nolint:exhaustruct // dependency code don't want to change the way it works
		Use:   "debug-store",
		Short: "Inspect and recover an application multistore left inconsistent by an interrupted migration",
	}

	cmd.PersistentFlags().String(flags.FlagHome, app.DefaultNodeHome, "The application home directory")
	cmd.AddCommand(
		newDebugStoreInspectCmd(),
		newDebugStoreCheckTipCmd(),
		newDebugStoreForceRollbackCmd(),
	)

	return cmd
}

func homeFromCmd(cmd *cobra.Command) (string, error) {
	home, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("application home not set")
	}
	return home, nil
}

func openApplicationDB(home string) (dbm.DB, error) {
	vp := viper.New()
	backend := server.GetAppDBBackend(vp)
	return dbm.NewDB("application", backend, filepath.Join(home, "data"))
}

func commitInfoAt(db dbm.DB, version int64) (*storetypes.CommitInfo, error) {
	bz, err := db.Get([]byte(fmt.Sprintf("s/%d", version)))
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, fmt.Errorf("no commitInfo at version %d", version)
	}
	var ci storetypes.CommitInfo
	if err := ci.Unmarshal(bz); err != nil {
		return nil, err
	}
	return &ci, nil
}

func newDebugStoreInspectCmd() *cobra.Command {
	return &cobra.Command{ //nolint:exhaustruct // dependency code don't want to change the way it works
		Use:   "inspect",
		Short: "Print the store versions recorded at the latest committed height and one height before it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := homeFromCmd(cmd)
			if err != nil {
				return err
			}

			db, err := openApplicationDB(home)
			if err != nil {
				return err
			}
			defer db.Close()

			latest := rootmulti.GetLatestVersion(db)
			cmd.Printf("s/latest = %d\n", latest)

			for _, v := range []int64{latest, latest - 1} {
				ci, err := commitInfoAt(db, v)
				if err != nil {
					cmd.Printf("commitInfo s/%d: %v\n", v, err)
					continue
				}
				cmd.Printf("commitInfo s/%d (version=%d, %d stores):\n", v, ci.Version, len(ci.StoreInfos))
				for _, si := range ci.StoreInfos {
					cmd.Printf("    %-24s v%d\n", si.Name, si.CommitId.Version)
				}
			}
			return nil
		},
	}
}

// buildUnloadedApp constructs the app with loadLatest=false so store construction
// (and hence store-key registration) succeeds even when the tip is inconsistent;
// the caller decides which version to attempt loading via the returned CommitMultiStore.
func buildUnloadedApp(logger log.Logger, db dbm.DB) (*rootmulti.Store, error) {
	vp := viper.New()
	alloraApp, err := app.NewAlloraApp(logger, db, nil, false, vp)
	if err != nil {
		return nil, err
	}

	rms, ok := alloraApp.CommitMultiStore().(*rootmulti.Store)
	if !ok {
		return nil, errors.New("app's CommitMultiStore is not a *rootmulti.Store")
	}
	return rms, nil
}

func newDebugStoreCheckTipCmd() *cobra.Command {
	return &cobra.Command{ //nolint:exhaustruct // dependency code don't want to change the way it works
		Use:   "check-tip",
		Short: "Attempt to load every store at the latest committed height, without writing anything",
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := homeFromCmd(cmd)
			if err != nil {
				return err
			}

			db, err := openApplicationDB(home)
			if err != nil {
				return err
			}
			defer db.Close()

			latest := rootmulti.GetLatestVersion(db)
			logger := log.NewLogger(cmd.OutOrStdout())

			rms, err := buildUnloadedApp(logger, db)
			if err != nil {
				return err
			}

			cmd.Printf("attempting to load all stores at latest=%d ...\n", latest)
			if err := rms.LoadVersion(latest); err != nil {
				return fmt.Errorf("load failed at %d: %w", latest, err)
			}
			cmd.Printf("load OK — tip %d is consistent and fully loadable\n", latest)
			return nil
		},
	}
}

func newDebugStoreForceRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{ //nolint:exhaustruct // dependency code don't want to change the way it works
		Use:   "force-rollback",
		Short: "Roll back the multistore to a target version left by an interrupted migration",
		Long: `Roll back the application multistore to a target version.

This bypasses the stock 'rollback' command, which constructs the app with
loadLatest=true and panics before it can act if the tip is inconsistent
(e.g. after an interrupted store migration). By default this only verifies
that the target version loads cleanly; pass --yes to actually delete
versions above the target and rewrite s/latest.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := homeFromCmd(cmd)
			if err != nil {
				return err
			}
			target, err := cmd.Flags().GetInt64(flagRollbackTo)
			if err != nil {
				return err
			}
			doWrite, err := cmd.Flags().GetBool(flagYes)
			if err != nil {
				return err
			}

			db, err := openApplicationDB(home)
			if err != nil {
				return err
			}
			defer db.Close()

			latest := rootmulti.GetLatestVersion(db)
			if target <= 0 || target >= latest {
				return fmt.Errorf("refusing: target %d must satisfy 0 < target < latest(%d)", target, latest)
			}

			logger := log.NewLogger(cmd.OutOrStdout())
			rms, err := buildUnloadedApp(logger, db)
			if err != nil {
				return err
			}

			cmd.Printf("loading all stores at target version %d ...\n", target)
			if err := rms.LoadVersion(target); err != nil {
				return fmt.Errorf("LoadVersion(%d) failed: %w", target, err)
			}
			cmd.Printf("load OK — every store can be loaded at %d\n", target)

			if !doWrite {
				cmd.Printf("\ndry run. To roll back (delete versions > %d and set s/latest=%d), re-run with --%s\n",
					target, target, flagYes)
				return nil
			}

			cmd.Printf("rolling back to %d (deleting any versions > %d) ...\n", target, target)
			if err := rms.RollbackToVersion(target); err != nil {
				return fmt.Errorf("RollbackToVersion(%d) failed: %w", target, err)
			}
			cmd.Printf("done. s/latest is now %d\n", rootmulti.GetLatestVersion(db))
			return nil
		},
	}

	cmd.Flags().Int64(flagRollbackTo, 0, "target version to roll back to (required)")
	cmd.Flags().Bool(flagYes, false, "perform the rollback instead of only validating the target version loads")
	_ = cmd.MarkFlagRequired(flagRollbackTo)

	return cmd
}

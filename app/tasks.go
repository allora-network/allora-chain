package app

import (
	"strings"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/pkg/errors"

	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

// InitializeRecurringTasks schedules recurring tasks after the application state is loaded.
func (app *AlloraApp) InitializeRecurringTasks() {
	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{ //nolint:exhaustruct
		Height: app.LastBlockHeight(),
	})

	mintTaskHandlers := app.MintKeeper.TaskHandlers()
	if err := app.SchedulerKeeper.RegisterTaskHandlers(mintTaskHandlers); err != nil {
		if !errors.Is(err, schedulertypes.ErrInvalidTaskHandler) || !strings.Contains(err.Error(), "duplicate task handler") {
			app.Logger().Error("failed to register mint task handlers", "err", err)
			return
		}
	}

	if err := app.MintKeeper.ScheduleEmissionRecalculationTask(ctx, app.SchedulerKeeper, 0); err != nil {
		app.Logger().Error("failed to schedule emission recalculation task", "err", err)
		return
	}

	app.Logger().Info("scheduled emission recalculation task")
}

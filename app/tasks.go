package app

import (
	"fmt"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	cosmoserrors "cosmossdk.io/errors"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

// InitializeRecurringTasks schedules recurring tasks after the application state is loaded.
func (app *AlloraApp) InitializeRecurringTasks() {
	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{ //nolint:exhaustruct
		Height: app.LastBlockHeight(),
	})

	mintTaskHandlers := app.MintKeeper.TaskHandlers()
	if err := app.SchedulerKeeper.RegisterTaskHandlers(mintTaskHandlers); err != nil {
		if !cosmoserrors.IsOf(err, schedulertypes.ErrTaskHandlerAlreadyExists) {
			panic(fmt.Sprintf("failed to register mint task handlers: %v", err))
		}
	}

	if err := app.MintKeeper.ScheduleEmissionRecalculationTask(ctx, app.SchedulerKeeper, 0); err != nil {
		if cosmoserrors.IsOf(err, schedulertypes.ErrTaskAlreadyExists) {
			app.Logger().Info("emission recalculation task already scheduled")
			return
		}
		panic(fmt.Sprintf("failed to schedule emission recalculation task: %v", err))
	}

	app.Logger().Info("scheduled emission recalculation task")
}

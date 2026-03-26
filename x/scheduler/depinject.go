package scheduler

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	modulev1 "github.com/allora-network/allora-chain/x/scheduler/api/scheduler/module/v1"
	"github.com/allora-network/allora-chain/x/scheduler/keeper"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

var _ depinject.OnePerModuleType = AppModule{} //nolint:exhaustruct

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.RegisterModule(&modulev1.Module{},
		appconfig.Provide(ProvideModule),
		appconfig.Invoke(InvokeRegisterTaskHandlers),
	)
}

type ModuleInputs struct {
	depinject.In

	Config       *modulev1.Module
	Cdc          codec.Codec
	StoreService store.KVStoreService
}

type ModuleOutputs struct {
	depinject.Out

	Module          appmodule.AppModule
	SchedulerKeeper *keeper.Keeper
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := keeper.NewKeeper(in.StoreService, in.Cdc)
	m := NewAppModule(&k)
	return ModuleOutputs{SchedulerKeeper: &k, Module: m, Out: depinject.Out{}}
}

func InvokeRegisterTaskHandlers(keeper *keeper.Keeper, perModTaskHandlers map[string]types.TaskHandlers) error {
	if keeper == nil || perModTaskHandlers == nil {
		return nil
	}

	allTaskHandlers := make([]types.TaskHandler, 0)
	for _, taskHandlers := range perModTaskHandlers { //nolint:maprange // it is later sorted on RegisterTaskHandlers
		for _, taskHandler := range taskHandlers {
			allTaskHandlers = append(allTaskHandlers, taskHandler)
		}
	}

	return keeper.RegisterTaskHandlers(allTaskHandlers)
}

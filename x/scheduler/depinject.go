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

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.RegisterModule(&modulev1.Module{},
		appconfig.Provide(ProvideModule),
		appconfig.Invoke(InvokeRegisterTaskSpec),
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

	SchedulerKeeper keeper.Keeper
	Module          appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := keeper.NewKeeper(in.StoreService, in.Cdc)
	m := NewAppModule(k)
	return ModuleOutputs{SchedulerKeeper: k, Module: m}
}

func InvokeRegisterTaskSpec(keeper *keeper.Keeper, perModTaskTypes map[string]types.TaskTypes) error {
	if keeper == nil || perModTaskTypes == nil {
		return nil
	}

	allTaskTypes := make([]types.TaskType, 0)
	for _, taskTypes := range perModTaskTypes {
		for _, taskType := range taskTypes {
			allTaskTypes = append(allTaskTypes, taskType)
		}
	}

	return keeper.RegisterTaskTypes(allTaskTypes)
}

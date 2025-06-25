package module

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/codec"

	modulev1 "github.com/allora-network/allora-chain/x/emissions/api/emissions/module/v1"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var _ appmodule.AppModule = AppModule{} // nolint: exhaustruct

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

func init() {
	appmodule.Register(
		&modulev1.Module{
			FeeCollectorName: authtypes.FeeCollectorName,
		},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Cdc             codec.Codec
	StoreService    store.KVStoreService
	AddressCodec    address.Codec
	AccountKeeper   keeper.AccountKeeper
	BankKeeper      keeper.BankKeeper
	SchedulerKeeper keeper.SchedulerKeeper

	Config *modulev1.Module
}

type ModuleOutputs struct {
	depinject.Out

	Module    appmodule.AppModule
	Keeper    keeper.Keeper
	TaskSpecs schedulertypes.TaskSpecs
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	feeCollectorName := in.Config.FeeCollectorName
	if feeCollectorName == "" {
		feeCollectorName = authtypes.FeeCollectorName
	}

	k := keeper.NewKeeper(
		in.Cdc,
		in.AddressCodec,
		in.StoreService,
		in.AccountKeeper,
		in.BankKeeper,
		in.SchedulerKeeper,
		feeCollectorName,
	)
	m := NewAppModule(in.Cdc, k)

	return ModuleOutputs{Module: m, Keeper: k, TaskSpecs: k.TaskSpecs(), Out: depinject.Out{}}
}

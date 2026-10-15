package module

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"

	modulev1 "github.com/allora-network/allora-chain/x/emissions/api/emissions/module/v1"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var _ appmodule.AppModule = AppModule{} //nolint:exhaustruct

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

	Module       appmodule.AppModule
	Keeper       keeper.Keeper
	TaskHandlers schedulertypes.TaskHandlers
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
	// Keep lifecycle hooks and close handlers bound to this keeper instance.
	k.GetTopicKeeper().SetLifecycleHooks(&k)
	k.SetEpochCloseHandlers(
		func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
			return actorutils.CloseWorkerNonce(&k, ctx, topic, nonce)
		},
		func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
			return actorutils.CloseReputerNonce(&k, ctx, topic, nonce)
		},
	)
	m := NewAppModule(in.Cdc, k)

	return ModuleOutputs{Module: m, Keeper: k, TaskHandlers: k.TaskHandlers(), Out: depinject.Out{}}
}

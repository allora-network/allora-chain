package scheduler

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/allora-network/allora-chain/x/scheduler/keeper"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	_ module.HasGenesis         = AppModule{}
	_ appmodule.AppModule       = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
)

const ConsensusVersion = 1

// AppModule implements the AppModule interface for the scheduler module.
type AppModule struct {
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object.
func NewAppModule(keeper *keeper.Keeper) AppModule {
	return AppModule{
		keeper: keeper,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// GetKeeper returns the module's keeper instance - needed to avoid keeper instance mismatch
func (am AppModule) GetKeeper() *keeper.Keeper {
	return am.keeper
}

// Name returns the scheduler module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the scheduler module's types for the given codec.
func (AppModule) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModule) RegisterInterfaces(_ codectypes.InterfaceRegistry) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the scheduler module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	return nil
}

// DefaultGenesis returns the scheduler module's default genesis state.
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis performs genesis state validation for the scheduler module.
func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

// InitGenesis performs the scheduler module's genesis initialization
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, bz json.RawMessage) {}

// ExportGenesis returns the scheduler module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ConsensusVersion implements HasConsensusVersion
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// BeginBlock executes all ABCI BeginBlock logic respective to the scheduler module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)

	return am.keeper.BeginBlock(ctx)
}

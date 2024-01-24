package module

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cosmossdk.io/core/appmodule"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	state "github.com/upshot-tech/protocol-state-machine-module"
	keeper "github.com/upshot-tech/protocol-state-machine-module/keeper"
)

var (
	_ module.AppModuleBasic   = AppModule{}
	_ module.HasGenesis       = AppModule{}
	_ appmodule.AppModule     = AppModule{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 1

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

func NewAppModuleBasic(m AppModule) module.AppModuleBasic {
	return module.CoreAppModuleBasicAdaptor(m.Name(), m)
}

// Name returns the state module's name.
func (AppModule) Name() string { return state.ModuleName }

// RegisterLegacyAminoCodec registers the state module's types on the LegacyAmino codec.
// New modules do not need to support Amino.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the state module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := state.RegisterQueryHandlerClient(context.Background(), mux, state.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the state module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	state.RegisterInterfaces(registry)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	state.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	state.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	// Register in place module state migration migrations
	// m := keeper.NewMigrator(am.keeper)
	// if err := cfg.RegisterMigration(state.ModuleName, 1, m.Migrate1to2); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", state.ModuleName, err))
	// }
}

// DefaultGenesis returns default genesis state as raw bytes for the module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(state.NewGenesisState())
}

// ValidateGenesis performs genesis state validation for the circuit module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data state.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", state.ModuleName, err)
	}

	return data.Validate()
}

// InitGenesis performs genesis initialization for the state module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState state.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if err := am.keeper.InitGenesis(ctx, &genesisState); err != nil {
		panic(fmt.Sprintf("failed to initialize %s genesis state: %v", state.ModuleName, err))
	}
}

// ExportGenesis returns the exported genesis state as raw bytes for the circuit
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to export %s genesis state: %v", state.ModuleName, err))
	}

	return cdc.MustMarshalJSON(gs)
}

// BeginBlock returns the begin blocker for the upshot module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	c := sdk.UnwrapSDKContext(ctx)
	return am.keeper.BeginBlocker(c)
}

// EndBlock returns the end blocker for the upshot module.
func (am AppModule) EndBlock(ctx context.Context) error {
	fmt.Printf("\n ---------------- EndBlock called ------------------- \n")

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Ensure that enough blocks have passed to hit an epoch.
	// If not, skip rewards calculation
	blockNumber := sdkCtx.BlockHeight()
	lastRewardsUpdate, err := am.keeper.GetLastRewardsUpdate(sdkCtx)
	if err != nil {
		return err
	}
	blocksSinceLastUpdate := blockNumber - lastRewardsUpdate
	if blocksSinceLastUpdate < 0 {
		panic("Block number is less than last rewards update block number")
	}
	if blocksSinceLastUpdate < EPOCH_LENGTH {
		return nil
	} else {
		err := globalEmissionPerTopic(sdkCtx, am, uint64(blocksSinceLastUpdate))
		if err != nil {
			fmt.Println("Error calculating global emission per topic: ", err)
		}
	}

	go makeAPICall()

	return nil

}

func makeAPICall() {
	url := os.Getenv("BLOCKLESS_API_URL")
	functionId := os.Getenv("BLOCKLESS_FUNCTION_ID")

	method := "POST"

	payload := strings.NewReader(`{
		"function_id": "` + functionId + `",
		"method": "upshot-function-example.wasm",
		"config": {
			"env_vars": [
				{
					"name": "BLS_REQUEST_PATH",
					"value": "/api"
				},
				{
					"name": "UPSHOT_ARG_PARAMS",
					"value": "ETH"
				}
			],
			"number_of_nodes": 1
		}
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
}

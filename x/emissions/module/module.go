package module

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
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

// EndBlock returns the end blocker for the emissions module.
func (am AppModule) EndBlock(ctx context.Context) error {
	fmt.Printf("\n ---------------- EndBlock ------------------- \n")

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Ensure that enough blocks have passed to hit an epoch.
	// If not, skip rewards calculation
	blockNumber := sdkCtx.BlockHeight()
	currentTime := uint64(sdkCtx.BlockTime().Unix())
	lastRewardsUpdate, err := am.keeper.GetLastRewardsUpdate(sdkCtx)
	if err != nil {
		return err
	}

	topTopicsActiveWithDemand, metDemand, err := ChurnAndDrawFromRequestsToGetTopActiveTopicsAndMetDemand(sdkCtx, am.keeper, currentTime)
	if err != nil {
		fmt.Println("Error getting active topics and met demand: ", err)
		return err
	}

	err = am.keeper.SendCoinsFromModuleToModule(ctx, state.AlloraRequestsModuleName, state.AlloraStakingModuleName, sdk.NewCoins(sdk.NewCoin("stake", cosmosMath.NewInt(metDemand.BigInt().Int64()))))
	if err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return err
	}

	blocksSinceLastUpdate := blockNumber - lastRewardsUpdate
	if blocksSinceLastUpdate < 0 {
		panic("Block number is less than last rewards update block number")
	}
	if blocksSinceLastUpdate >= am.keeper.EpochLength() {
		err = emitRewards(sdkCtx, am)
		// the following code does NOT halt the chain in case of an error in rewards payments
		// if an error occurs and rewards payments are not made, globally they will still accumulate
		// and we can retroactively pay them out
		if err != nil {
			fmt.Println("Error calculating global emission per topic: ", err)
			panic(err)
		}
	}

	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	// Within each loop, execute the inference and weight cadence checks
	for _, topic := range *topTopicsActiveWithDemand {
		// Parallelize the inference and weight cadence checks
		go func(topic *state.Topic) {
			// Check the cadence of inferences
			if currentTime-topic.InferenceLastRan >= topic.InferenceCadence {
				fmt.Printf("Inference cadence met for topic: %v metadata: %s default arg: %s. \n", topic.Id, topic.Metadata, topic.DefaultArg)

				go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id)

				// Update the last inference ran
				am.keeper.UpdateTopicInferenceLastRan(sdkCtx, topic.Id, currentTime)
			}

			// Check the cadence of weight calculations
			if currentTime-topic.WeightLastRan >= topic.WeightCadence {
				fmt.Printf("Weight cadence met for topic: %v metadata: %s default arg: %s \n", topic.Id, topic.Metadata, topic.DefaultArg)

				// Get Latest Weights
				weights, err := am.keeper.GetWeightsFromTopic(sdkCtx, topic.Id)
				if err != nil {
					fmt.Println("Error getting latest weights: ", err)
					return
				}

				// Get Lastest Inference
				inferences, err := am.keeper.GetLatestInferencesFromTopic(sdkCtx, topic.Id)
				if err != nil {
					fmt.Println("Error getting latest inferences: ", err)
					return
				}

				go generateWeights(weights, inferences, topic.WeightLogic, topic.WeightMethod, topic.Id)

				// Update the last weight ran
				am.keeper.UpdateTopicWeightLastRan(sdkCtx, topic.Id, currentTime)
			}
		}(&topic)
	}

	return nil
}

package module

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"cosmossdk.io/core/appmodule"
	cosmosMath "cosmossdk.io/math"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	state "github.com/allora-network/allora-chain/x/emissions"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
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
	lastRewardsUpdate, err := am.keeper.GetLastRewardsUpdate(sdkCtx)
	if err != nil {
		return err
	}
	blocksSinceLastUpdate := blockNumber - lastRewardsUpdate
	if blocksSinceLastUpdate < 0 {
		panic("Block number is less than last rewards update block number")
	}
	if blocksSinceLastUpdate < am.keeper.EpochLength() {
		return nil
	}
	err = emitRewards(sdkCtx, am)
	// the following code does NOT halt the chain in case of an error in rewards payments
	// if an error occurs and rewards payments are not made, globally they will still accumulate
	// and we can retroactively pay them out
	if err != nil {
		fmt.Println("Error calculating global emission per topic: ", err)
		panic(err)
	}

	// Execute the inference and weight cadence checks
	topics, err := am.keeper.GetActiveTopics(sdkCtx)
	if err != nil {
		fmt.Println("Error getting active topics: ", err)
		return err
	}

	fmt.Println("Active topics: ", len(topics))

	// START OF demand-driven pricing model
	// Big Principle:
	//		The price of inference is determined by the demand for inference vs the scarcity of capacity to run topic-level shit
	// Find the demand per epoch of the topic by counting inference requests per topic that have not yet been filtered out
	// Order topics by "demand" (num unfiltered requests)
	// Take top keeper.MAX_TOPICS_PER_BLOCK number of topics with the highest demand (sort desc by num requests)
	// If number of topics > keeper.TARGET_CAPACITY_PER_BLOCK, price := price * (1 + keeper.PRICE_CHANGE_PERCENT)
	// If number of topics < keeper.TARGET_CAPACITY_PER_BLOCK, price := max(keeper.MIN_PRICE_PER_EPOCH, price * (1 - keeper.PRICE_CHANGE_PERCENT))
	// Deduct from all unfiltered requests `price` amount among those top topics
	// Delete requests with negligible demand leftover

	currentTime := uint64(sdkCtx.BlockTime().Unix())
	numValidRequestsPerTopic := make(map[uint64]uint64) // using price of previous block as heuristic to calc total current demand
	validRequestsPerTopic := make(map[uint64][]state.InferenceRequest)
	minNextPrice, price, maxNextPrice, err := am.keeper.GetCurrentAndNextPossiblePricePerEpoch(sdkCtx)
	activeTopics := make([]state.Topic, 0)
	if err != nil {
		fmt.Println("Error getting current and next possible price per epoch: ", err)
		return err
	}
	for _, topic := range topics {
		inferenceRequests, err := am.keeper.GetMempoolInferenceRequestsForTopic(sdkCtx, topic.Id)
		if err != nil {
			fmt.Println("Error getting mempool inference requests: ", err)
			return err
		}

		// Loop through inference requests and check if cadence has been met
		// AND if their remaining bid amount is greater than the lowest possible next price (optimization),
		// AND if their max price per inference is less than or equal to the max possible next price (optimization),
		// AND request is not yet expired
		// then add to list of requests to be served
		for _, req := range inferenceRequests {
			reqId, err := req.GetRequestId()
			if err != nil {
				fmt.Println("Error getting request id: ", err)
				return err
			}
			reqDemand, err := am.keeper.GetRequestDemand(sdkCtx, reqId)
			if err != nil {
				fmt.Println("Error getting request demand: ", err)
				return err
			}
			if req.LastChecked+req.Cadence <= currentTime && reqDemand.GTE(minNextPrice) && req.MaxPricePerInference.LTE(maxNextPrice) && req.TimestampValidUntil > currentTime {
				validRequestsPerTopic[topic.Id] = append(validRequestsPerTopic[topic.Id], req)
				numValidRequestsPerTopic[topic.Id]++
			}
		}

		if numValidRequestsPerTopic[topic.Id] > 0 {
			activeTopics = append(activeTopics, *topic)
		}
		if price.Mul(cosmosMath.NewUint(numValidRequestsPerTopic[topic.Id])).LT(cosmosMath.NewUint(keeper.MIN_DEMAND)) {
			// TODO: Add more intelligent inactivation logic; should be easier to be inactive than it currently is
			fmt.Printf("Inactivating topic due to no demand: %v metadata: %s", topic.Id, topic.Metadata)
			am.keeper.InactivateTopic(sdkCtx, topic.Id)
		}
	}

	// Sort topics by numValidRequestsPerTopic
	sortedTopics := SortDescWithRandomTiebreaker[state.Topic](activeTopics, numValidRequestsPerTopic, currentTime)
	// Take top am.keeper.MAX_TOPICS_PER_BLOCK number of topics with the highest demand
	sortedTopics = sortedTopics[:uint(math.Min(float64(len(sortedTopics)), keeper.MAX_TOPICS_PER_BLOCK))]
	if len(sortedTopics) > keeper.TARGET_CAPACITY_PER_BLOCK {
		price = maxNextPrice // increase price because demand is high
		am.keeper.SetPricePerEpoch(sdkCtx, price)
	} else if len(sortedTopics) < keeper.TARGET_CAPACITY_PER_BLOCK {
		price = minNextPrice // decrease price because demand is low
		am.keeper.SetPricePerEpoch(sdkCtx, price)
	}

	// Determine how many funds to draw from demand
	totalFundsToDrawFromDemand := cosmosMath.NewInt(0)
	for _, topic := range activeTopics {
		for _, req := range validRequestsPerTopic[topic.Id] {
			reqId, err := req.GetRequestId()
			if err != nil {
				fmt.Println("Error getting request demand: ", err)
				return err
			}
			reqDemand, err := am.keeper.GetRequestDemand(sdkCtx, reqId)
			if err != nil {
				fmt.Println("Error getting request demand: ", err)
				return err
			}
			// all the previous conditionals were already applied to the requests in the previous loop
			if reqDemand.GTE(price) && req.MaxPricePerInference.LTE(price) {
				// should never be negative due to immediately preceding control flow
				newReqDemand := reqDemand.Sub(price)
				am.keeper.SetRequestDemand(sdkCtx, reqId, newReqDemand)
				if newReqDemand.LT(cosmosMath.NewUint(keeper.MIN_UNMET_DEMAND)) {
					// should never be negative due to wrapping control flow
					am.keeper.RemoveFromMempool(sdkCtx, req)
					// TODO: Do something with the leftover dust? Maybe leave as is?
					// This is just a 1-time "cost" the consumer incurs when they create "subscriptions" they don't ever refill nor fill enough
				}
				validRequestsPerTopic[topic.Id] = append(validRequestsPerTopic[topic.Id], req)
			}
		}
		totalDemandPerTopic := price.Mul(cosmosMath.NewUint(numValidRequestsPerTopic[topic.Id]))
		totalFundsToDrawFromDemand = totalFundsToDrawFromDemand.Add(cosmosMath.NewInt(totalDemandPerTopic.BigInt().Int64()))
	}

	err = am.keeper.SendCoinsFromModuleToModule(sdkCtx, state.ModuleName, state.AlloraStakingModuleName, sdk.NewCoins(sdk.NewCoin("stake", totalFundsToDrawFromDemand)))
	if err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return err
	}

	// END OF demand-driven pricing model

	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	for _, topic := range activeTopics {
		// Parallelize the inference and weight cadence checks
		go func(topic *state.Topic) {
			// Check the cadence of inferences
			if currentTime-topic.InferenceLastRan >= topic.InferenceCadence {
				fmt.Printf("Inference cadence met for topic: %v metadata: %s", topic.Id, topic.Metadata)

				go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.Metadata, topic.Id)

				// Update the last inference ran
				am.keeper.UpdateTopicInferenceLastRan(sdkCtx, topic.Id, currentTime)
			}

			// Check the cadence of weight calculations
			if currentTime-topic.WeightLastRan >= topic.WeightCadence {
				fmt.Printf("Weight cadence met for topic: %v metadata: %s", topic.Id, topic.Metadata)

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

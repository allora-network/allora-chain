package inference_synthesis

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Worker = string
type BlockHeight = int64
type TopicId = uint64
type Regret = alloraMath.Dec
type Loss = alloraMath.Dec
type Weight = alloraMath.Dec
type InferenceValue = alloraMath.Dec
type Stake = cosmosMath.Int

// Need to differentiate between the two types of regrets because workers may complete tasks
// for both roles and may have different regrets for those different roles
type RegretInformedWeights struct {
	inferers    map[Worker]Weight
	forecasters map[Worker]Weight
}

type SynthRequest struct {
	Ctx                 sdk.Context
	K                   keeper.Keeper
	TopicId             TopicId
	Inferences          *emissions.Inferences
	Forecasts           *emissions.Forecasts
	NetworkCombinedLoss Loss
	EpsilonTopic        alloraMath.Dec
	EpsilonSafeDiv      alloraMath.Dec
	PNorm               alloraMath.Dec
	CNorm               alloraMath.Dec
}

type SynthPaletteFactory struct{}

type SynthPalette struct {
	Ctx               sdk.Context
	K                 keeper.Keeper
	Logger            log.Logger
	TopicId           TopicId
	AllInferersAreNew bool
	// Should use this as a source of truth regarding for which inferers to have data calculated
	// i.e. if an inferer is not present here, calculate a network inference without their data
	// Must be unique values
	Inferers          []Worker
	InferenceByWorker map[Worker]*emissions.Inference
	// Must respect the order of sister `inferers` property
	InfererRegrets map[Worker]*alloraMath.Dec
	// Should use this as a source of truth regarding for which forecasters to have data calculated
	// i.e. if an forecaster is not present here, calculate a network inference without their data
	// Must be unique values
	Forecasters                      []Worker
	ForecastByWorker                 map[Worker]*emissions.Forecast
	ForecastImpliedInferenceByWorker map[Worker]*emissions.Inference
	// Must respect the order of sister `forecasters` property
	ForecasterRegrets   map[Worker]*alloraMath.Dec
	NetworkCombinedLoss Loss
	EpsilonTopic        alloraMath.Dec
	EpsilonSafeDiv      alloraMath.Dec
	PNorm               alloraMath.Dec
	CNorm               alloraMath.Dec
}

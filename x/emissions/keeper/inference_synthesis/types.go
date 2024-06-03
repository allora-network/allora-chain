package inference_synthesis

import (
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

type StatefulRegret struct {
	regret        Regret
	noPriorRegret bool
}

type StdDevRegrets struct {
	stdDevInferenceRegrets Regret
	stdDevCombinedRegrets  Regret
	// StdDevOneInForecastRegret map[Worker]Regret // max regret for each one-in forecaster
}

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
	Epsilon             alloraMath.Dec
	PNorm               alloraMath.Dec
	CNorm               alloraMath.Dec
}

type SynthPaletteFactory struct{}

type SynthPalette struct {
	ctx     sdk.Context
	k       keeper.Keeper
	topicId TopicId
	// Should use this as a source of truth regarding for which inferers to have data calculated
	// i.e. if an inferer is not present here, calculate a network inference without their data
	// Must be unique values
	inferers          []Worker
	inferenceByWorker map[Worker]*emissions.Inference
	// Must respect the order of sister `inferers` property
	infererRegrets map[Worker]StatefulRegret
	// Should use this as a source of truth regarding for which forecasters to have data calculated
	// i.e. if an forecaster is not present here, calculate a network inference without their data
	// Must be unique values
	forecasters                      []Worker
	forecastByWorker                 map[Worker]*emissions.Forecast
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference
	// Must respect the order of sister `forecasters` property
	forecasterRegrets    map[Worker]StatefulRegret
	allInferersAreNew    bool
	allForecastersAreNew bool
	allWorkersAreNew     bool // Simple conjunction of the two above
	networkCombinedLoss  Loss
	epsilon              alloraMath.Dec
	pNorm                alloraMath.Dec
	cNorm                alloraMath.Dec
}

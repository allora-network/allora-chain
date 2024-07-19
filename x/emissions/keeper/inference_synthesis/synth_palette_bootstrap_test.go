package inference_synthesis_test

import (
	"fmt"
	"reflect"
	"testing"

	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestSynthPalette_Clone(t *testing.T) {
	inferencesByWorker := map[synth.Worker]*emissionstypes.Inference{
		"inferer1": {
			TopicId: uint64(1),
			Value:   alloraMath.MustNewDecFromString("0.1"),
		},
		"inferer2": {
			TopicId: uint64(1),
			Value:   alloraMath.MustNewDecFromString("0.2"),
		},
	}

	palette := synth.SynthPalette{
		Ctx:                              sdk.Context{},
		K:                                keeper.Keeper{},
		Logger:                           log.NewNopLogger(),
		TopicId:                          uint64(1),
		Inferers:                         []synth.Worker{"inferer1", "inferer2"},
		InferenceByWorker:                inferencesByWorker,
		InfererRegrets:                   make(map[synth.Worker]*alloraMath.Dec),
		Forecasters:                      []synth.Worker{"forecaster1", "forecaster2"},
		ForecastByWorker:                 make(map[synth.Worker]*emissionstypes.Forecast),
		ForecastImpliedInferenceByWorker: make(map[synth.Worker]*emissionstypes.Inference),
		ForecasterRegrets:                make(map[synth.Worker]*alloraMath.Dec),
		NetworkCombinedLoss:              alloraMath.MustNewDecFromString("0.0"),
		Epsilon:                          alloraMath.MustNewDecFromString("0.01"),
		PNorm:                            alloraMath.MustNewDecFromString("2"),
		CNorm:                            alloraMath.MustNewDecFromString("1"),
	}

	clone := palette.Clone()

	// Assert that the cloned palette is equal to the original palette
	if !reflect.DeepEqual(clone, palette) {
		t.Errorf("Clone() failed: cloned palette is not equal to the original palette")
	}

	// Assert that the cloned palette is a deep copy
	cloneK := fmt.Sprintf("%v", clone.K)
	paletteK := fmt.Sprintf("%v", palette.K)
	cloneInferers := fmt.Sprintf("%v", clone.Inferers)
	paletteInferers := fmt.Sprintf("%v", palette.Inferers)
	cloneForecasters := fmt.Sprintf("%v", clone.Forecasters)
	paletteForecasters := fmt.Sprintf("%v", palette.Forecasters)
	cloneInferenceByWorker := fmt.Sprintf("%v", clone.InferenceByWorker)
	paletteInferenceByWorker := fmt.Sprintf("%v", palette.InferenceByWorker)
	cloneCNorm := fmt.Sprintf("%v", clone.CNorm)
	paletteCNorm := fmt.Sprintf("%v", palette.CNorm)
	clonePNorm := fmt.Sprintf("%v", clone.PNorm)
	palettePNorm := fmt.Sprintf("%v", palette.PNorm)
	cloneEpsilon := fmt.Sprintf("%v", clone.Epsilon)
	paletteEpsilon := fmt.Sprintf("%v", palette.Epsilon)
	paletteForecasterRegrets := fmt.Sprintf("%v", palette.ForecasterRegrets)
	cloneForecasterRegrets := fmt.Sprintf("%v", clone.ForecasterRegrets)
	if &clone.Ctx == &palette.Ctx || cloneK != paletteK || &clone.Logger == &palette.Logger ||
		&clone.K == &palette.K || cloneInferers != paletteInferers || cloneForecasters != paletteForecasters ||
		cloneInferenceByWorker != paletteInferenceByWorker ||
		&clone.Inferers == &palette.Inferers || &clone.InferenceByWorker == &palette.InferenceByWorker ||
		&clone.InfererRegrets == &palette.InfererRegrets || &clone.Forecasters == &palette.Forecasters ||
		&clone.ForecastByWorker == &palette.ForecastByWorker || &clone.ForecastImpliedInferenceByWorker == &palette.ForecastImpliedInferenceByWorker ||
		&clone.ForecasterRegrets == &palette.ForecasterRegrets || cloneForecasterRegrets != paletteForecasterRegrets ||
		&clone.CNorm == &palette.CNorm || &clone.PNorm == &palette.PNorm || &clone.Epsilon == &palette.Epsilon ||
		cloneCNorm != paletteCNorm || clonePNorm != palettePNorm || cloneEpsilon != paletteEpsilon {
		t.Errorf("Clone() failed: cloned palette is not a deep copy")
	}
}

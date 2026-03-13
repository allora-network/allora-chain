// Run all benchmarks in this package:
//
//	go test ./x/emissions/keeper/inference_synthesis -run '^$' -bench . -benchmem
//
// Run only network inference benchmarks:
//
//	go test ./x/emissions/keeper/inference_synthesis -run '^$' -bench 'CalcNetworkInferences' -benchmem
package inferencesynthesis_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionsmodule "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

type benchFixture struct {
	args           inferencesynthesis.CalcNetworkInferencesArgs
	numLabels      int
	numInferer     int
	registryLabels int
}

type labelUsagePattern string

const (
	labelUsageDenseAll                  labelUsagePattern = "dense_all_labels"
	labelUsageSparseSameHot4FullVec     labelUsagePattern = "sparse_same_hot4_fullvec"
	labelUsageSparseDisjointHot4FullVec labelUsagePattern = "sparse_disjoint_hot4_fullvec"
	labelUsageMixed25Dense75Sparse      labelUsagePattern = "mixed_25_dense_75_sparse"
)

func BenchmarkCalcNetworkInferencesWorstCase(b *testing.B) {
	b.ReportAllocs()

	cases := []struct {
		name          string
		numInferer    int
		numForecaster int
		numLabels     int
	}{
		{name: "i32_f6_l8_chain_limits", numInferer: 32, numForecaster: 6, numLabels: 8},
		{name: "i32_f6_l16_stress_labels", numInferer: 32, numForecaster: 6, numLabels: 16},
		{name: "i32_f6_l32_stress_labels", numInferer: 32, numForecaster: 6, numLabels: 32},
		{name: "i32_f0_l8_no_forecasters", numInferer: 32, numForecaster: 0, numLabels: 8},
		{name: "i32_f0_l32_no_forecasters", numInferer: 32, numForecaster: 0, numLabels: 32},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			fixture := mustBuildBenchmarkFixtureWithPattern(
				b,
				tc.numInferer,
				tc.numForecaster,
				tc.numLabels,
				labelUsageDenseAll,
			)
			b.ReportMetric(float64(fixture.numLabels), "labels")
			b.ReportMetric(float64(fixture.registryLabels), "registry_labels")
			b.ReportMetric(float64(fixture.numInferer), "inferers")
			b.ReportMetric(float64(tc.numForecaster), "forecasters")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := inferencesynthesis.CalcNetworkInferences(fixture.args)
				if err != nil {
					b.Fatalf("CalcNetworkInferences failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkCalcNetworkInferencesLabelUsagePatterns(b *testing.B) {
	b.ReportAllocs()

	const numInferer = 32
	const numForecaster = 6
	const numLabels = 32

	patterns := []labelUsagePattern{
		labelUsageDenseAll,
		labelUsageSparseSameHot4FullVec,
		labelUsageSparseDisjointHot4FullVec,
		labelUsageMixed25Dense75Sparse,
	}

	for _, p := range patterns {
		b.Run(string(p), func(b *testing.B) {
			fixture := mustBuildBenchmarkFixtureWithPattern(b, numInferer, numForecaster, numLabels, p)
			b.ReportMetric(float64(fixture.numLabels), "labels")
			b.ReportMetric(float64(fixture.registryLabels), "registry_labels")
			b.ReportMetric(float64(fixture.numInferer), "inferers")
			b.ReportMetric(float64(numForecaster), "forecasters")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := inferencesynthesis.CalcNetworkInferences(fixture.args)
				if err != nil {
					b.Fatalf("CalcNetworkInferences failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkCalcNetworkInferencesRegistryExpansion(b *testing.B) {
	b.ReportAllocs()

	const numInferer = 32
	const numForecaster = 6
	const labelsPerInferer = 8
	const maxRegistryLabels = numInferer * labelsPerInferer // worst case when all inferer label-sets are disjoint

	cases := []struct {
		name           string
		registryPct    int
		registryLabels int
	}{
		{name: "registry_25pct", registryPct: 25, registryLabels: maxRegistryLabels * 25 / 100},
		{name: "registry_50pct", registryPct: 50, registryLabels: maxRegistryLabels * 50 / 100},
		{name: "registry_75pct", registryPct: 75, registryLabels: maxRegistryLabels * 75 / 100},
		{name: "registry_100pct_worst_case", registryPct: 100, registryLabels: maxRegistryLabels},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			fixture := mustBuildRegistryExpansionFixture(
				b,
				numInferer,
				numForecaster,
				labelsPerInferer,
				tc.registryLabels,
			)
			b.ReportMetric(float64(labelsPerInferer), "labels_per_inferer")
			b.ReportMetric(float64(maxRegistryLabels), "max_registry_labels")
			b.ReportMetric(float64(tc.registryPct), "registry_pct")
			b.ReportMetric(float64(fixture.registryLabels), "registry_labels")
			b.ReportMetric(float64(fixture.numInferer), "inferers")
			b.ReportMetric(float64(numForecaster), "forecasters")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := inferencesynthesis.CalcNetworkInferences(fixture.args)
				if err != nil {
					b.Fatalf("CalcNetworkInferences failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkCalcNetworkInferencesRegistryExpansionMaxLabels32(b *testing.B) {
	b.ReportAllocs()

	const numInferer = 32
	const numForecaster = 6
	const labelsPerInferer = 32
	const maxRegistryLabels = numInferer * labelsPerInferer // worst case when all inferer label-sets are disjoint

	cases := []struct {
		name           string
		registryPct    int
		registryLabels int
	}{
		{name: "registry_25pct", registryPct: 25, registryLabels: maxRegistryLabels * 25 / 100},
		{name: "registry_50pct", registryPct: 50, registryLabels: maxRegistryLabels * 50 / 100},
		{name: "registry_75pct", registryPct: 75, registryLabels: maxRegistryLabels * 75 / 100},
		{name: "registry_100pct_worst_case", registryPct: 100, registryLabels: maxRegistryLabels},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			fixture := mustBuildRegistryExpansionFixture(
				b,
				numInferer,
				numForecaster,
				labelsPerInferer,
				tc.registryLabels,
			)
			b.ReportMetric(float64(labelsPerInferer), "labels_per_inferer")
			b.ReportMetric(float64(maxRegistryLabels), "max_registry_labels")
			b.ReportMetric(float64(tc.registryPct), "registry_pct")
			b.ReportMetric(float64(fixture.registryLabels), "registry_labels")
			b.ReportMetric(float64(fixture.numInferer), "inferers")
			b.ReportMetric(float64(numForecaster), "forecasters")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := inferencesynthesis.CalcNetworkInferences(fixture.args)
				if err != nil {
					b.Fatalf("CalcNetworkInferences failed: %v", err)
				}
			}
		})
	}
}

func mustBuildBenchmarkFixtureWithPattern(
	b *testing.B,
	numInferer int,
	numForecaster int,
	numLabels int,
	pattern labelUsagePattern,
) benchFixture {
	b.Helper()

	ctx, k := mustInitKeeperContext(b)
	topicID := uint64(1)
	blockHeight := int64(10)

	totalWorkers := numInferer + numForecaster + 1
	_, _, _, addrsStr := alloratestutil.GenerateTestAccounts(totalWorkers)
	creator := addrsStr[0]

	topic := emissionstypes.Topic{
		Id:                       topicID,
		Creator:                  creator,
		Metadata:                 "benchmark-topic",
		LossMethod:               "mse",
		EpochLength:              100,
		EpochLastEnded:           0,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.MustNewDecFromString("3"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                emissionstypes.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.1"),
	}
	if err := k.SetTopic(ctx, topicID, topic); err != nil {
		b.Fatalf("SetTopic failed: %v", err)
	}

	for i := 0; i < numLabels; i++ {
		label := fmt.Sprintf("label_%d", i)
		if _, err := k.RegisterEpochLabel(ctx, topicID, blockHeight, label); err != nil {
			b.Fatalf("RegisterEpochLabel(%s) failed: %v", label, err)
		}
	}
	registry, err := k.GetEpochLabelRegistry(ctx, topicID, blockHeight)
	if err != nil {
		b.Fatalf("GetEpochLabelRegistry failed: %v", err)
	}
	registryLabels := len(registry.GetLabels())
	if registryLabels != numLabels {
		b.Fatalf("unexpected EpochLabelRegistry size: got=%d want=%d", registryLabels, numLabels)
	}

	inferers := make([]string, numInferer)
	for i := 0; i < numInferer; i++ {
		inferers[i] = addrsStr[i+1]
	}
	forecasters := make([]string, numForecaster)
	for i := 0; i < numForecaster; i++ {
		forecasters[i] = addrsStr[1+numInferer+i]
	}

	infererToInference := make(map[string]*emissionstypes.Inference, numInferer)
	infererToRegret := make(map[string]*alloraMath.Dec, numInferer)
	for i, inferer := range inferers {
		values := buildInferenceValues(i, numInferer, numLabels, pattern)
		infererToInference[inferer] = &emissionstypes.Inference{
			TopicId:     topicID,
			BlockHeight: blockHeight,
			Inferer:     inferer,
			Values:      values,
			ExtraData:   nil,
			Proof:       "",
		}
		regret := alloraMath.MustNewDecFromString("0.02")
		infererToRegret[inferer] = &regret

		ts := emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret}
		if err := k.SetInfererNetworkRegret(ctx, topicID, inferer, ts); err != nil {
			b.Fatalf("SetInfererNetworkRegret failed: %v", err)
		}
		if err := k.SetNaiveInfererNetworkRegret(ctx, topicID, inferer, ts); err != nil {
			b.Fatalf("SetNaiveInfererNetworkRegret failed: %v", err)
		}
	}

	forecasterToForecast := make(map[string]*emissionstypes.Forecast, numForecaster)
	forecasterToRegret := make(map[string]*alloraMath.Dec, numForecaster)
	for fi, forecaster := range forecasters {
		elements := make([]*emissionstypes.ForecastElement, 0, numInferer)
		for ii, inferer := range inferers {
			elements = append(elements, &emissionstypes.ForecastElement{
				Inferer: inferer,
				Value:   alloraMath.NewDecFromInt64(int64(2000 + fi + ii)),
			})
		}
		forecasterToForecast[forecaster] = &emissionstypes.Forecast{
			TopicId:          topicID,
			BlockHeight:      blockHeight,
			Forecaster:       forecaster,
			ForecastElements: elements,
			ExtraData:        nil,
		}
		regret := alloraMath.MustNewDecFromString("0.03")
		forecasterToRegret[forecaster] = &regret
		if err := k.SetForecasterNetworkRegret(
			ctx,
			topicID,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		); err != nil {
			b.Fatalf("SetForecasterNetworkRegret failed: %v", err)
		}
	}

	// Populate all regret tables used by one-out/one-in paths in CalcNetworkInferences.
	for _, withheldInferer := range inferers {
		for _, inferer := range inferers {
			if err := k.SetOneOutInfererInfererNetworkRegret(
				ctx,
				topicID,
				withheldInferer,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.04"),
				},
			); err != nil {
				b.Fatalf("SetOneOutInfererInfererNetworkRegret failed: %v", err)
			}
		}
		for _, forecaster := range forecasters {
			if err := k.SetOneOutInfererForecasterNetworkRegret(
				ctx,
				topicID,
				withheldInferer,
				forecaster,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.05"),
				},
			); err != nil {
				b.Fatalf("SetOneOutInfererForecasterNetworkRegret failed: %v", err)
			}
		}
	}

	for _, withheldForecaster := range forecasters {
		for _, inferer := range inferers {
			if err := k.SetOneOutForecasterInfererNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.06"),
				},
			); err != nil {
				b.Fatalf("SetOneOutForecasterInfererNetworkRegret failed: %v", err)
			}

			if err := k.SetOneInForecasterNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.08"),
				},
			); err != nil {
				b.Fatalf("SetOneInForecasterNetworkRegret failed: %v", err)
			}
		}
		for _, forecaster := range forecasters {
			if err := k.SetOneOutForecasterForecasterNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				forecaster,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.07"),
				},
			); err != nil {
				b.Fatalf("SetOneOutForecasterForecasterNetworkRegret failed: %v", err)
			}
		}
		if err := k.SetOneInForecasterNetworkRegret(
			ctx,
			topicID,
			withheldForecaster,
			withheldForecaster,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString("0.09"),
			},
		); err != nil {
			b.Fatalf("SetOneInForecasterNetworkRegret(self) failed: %v", err)
		}
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	forecastImplied, infererRegretsOut, forecasterRegretsOut, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 log.NewNopLogger(),
			Ctx:                    ctx,
			K:                      k,
			TopicId:                topicID,
			AllInferersAreNew:      false,
			Inferers:               inferers,
			InfererToInference:     infererToInference,
			InfererToRegret:        infererToRegret,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecasterToForecast,
			ForecasterToRegret:     forecasterToRegret,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           topic.Epsilon,
			PNorm:                  topic.PNorm,
			CNorm:                  topic.CNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
		},
	)
	if err != nil {
		b.Fatalf("CalcForecastImpliedInferences setup failed: %v", err)
	}

	args := inferencesynthesis.CalcNetworkInferencesArgs{
		Ctx:                                  ctx,
		K:                                    k,
		Logger:                               log.NewNopLogger(),
		TopicId:                              topicID,
		Inferers:                             inferers,
		InfererToInference:                   infererToInference,
		InfererToRegret:                      infererRegretsOut,
		AllInferersAreNew:                    false,
		Forecasters:                          forecasters,
		ForecasterToForecast:                 forecasterToForecast,
		ForecasterToRegret:                   forecasterRegretsOut,
		ForecasterToForecastImpliedInference: forecastImplied,
		NetworkCombinedLoss:                  &networkCombinedLoss,
		EpsilonTopic:                         topic.Epsilon,
		EpsilonSafeDiv:                       alloraMath.MustNewDecFromString("0.0000001"),
		PNorm:                                topic.PNorm,
		CNorm:                                topic.CNorm,
		RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
		InferenceBlockHeight:                 blockHeight,
		LabelRegistry:                        &registry,
	}

	// Sanity check before benchmark timing starts.
	if _, _, err := inferencesynthesis.CalcNetworkInferences(args); err != nil {
		b.Fatalf("warmup CalcNetworkInferences failed: %v", err)
	}

	return benchFixture{
		args:           args,
		numLabels:      numLabels,
		numInferer:     numInferer,
		registryLabels: registryLabels,
	}
}

func buildInferenceValues(
	infererIdx int,
	numInferer int,
	numLabels int,
	pattern labelUsagePattern,
) []alloraMath.Dec {
	hot := 4
	if hot > numLabels {
		hot = numLabels
	}

	buildDense := func() []alloraMath.Dec {
		values := make([]alloraMath.Dec, numLabels)
		for l := 0; l < numLabels; l++ {
			values[l] = alloraMath.NewDecFromInt64(int64(1000 + infererIdx + l))
		}
		return values
	}

	switch pattern {
	case labelUsageDenseAll:
		return buildDense()
	case labelUsageSparseSameHot4FullVec:
		values := make([]alloraMath.Dec, numLabels)
		for l := 0; l < hot; l++ {
			values[l] = alloraMath.NewDecFromInt64(int64(1000 + infererIdx + l))
		}
		return values
	case labelUsageSparseDisjointHot4FullVec:
		values := make([]alloraMath.Dec, numLabels)
		start := (infererIdx * hot) % numLabels
		for off := 0; off < hot; off++ {
			idx := (start + off) % numLabels
			values[idx] = alloraMath.NewDecFromInt64(int64(1000 + infererIdx + idx))
		}
		return values
	case labelUsageMixed25Dense75Sparse:
		denseCutoff := numInferer / 4
		if infererIdx < denseCutoff {
			return buildDense()
		}
		values := make([]alloraMath.Dec, numLabels)
		for l := 0; l < hot; l++ {
			values[l] = alloraMath.NewDecFromInt64(int64(1000 + infererIdx + l))
		}
		return values
	default:
		return buildDense()
	}
}

func mustBuildRegistryExpansionFixture(
	b *testing.B,
	numInferer int,
	numForecaster int,
	labelsPerInferer int,
	registryLabels int,
) benchFixture {
	b.Helper()

	ctx, k := mustInitKeeperContext(b)
	topicID := uint64(1)
	blockHeight := int64(10)

	totalWorkers := numInferer + numForecaster + 1
	_, _, _, addrsStr := alloratestutil.GenerateTestAccounts(totalWorkers)
	creator := addrsStr[0]

	topic := emissionstypes.Topic{
		Id:                       topicID,
		Creator:                  creator,
		Metadata:                 "benchmark-topic",
		LossMethod:               "mse",
		EpochLength:              100,
		EpochLastEnded:           0,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.MustNewDecFromString("3"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                emissionstypes.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.1"),
	}
	if err := k.SetTopic(ctx, topicID, topic); err != nil {
		b.Fatalf("SetTopic failed: %v", err)
	}

	for i := 0; i < registryLabels; i++ {
		label := fmt.Sprintf("label_%d", i)
		if _, err := k.RegisterEpochLabel(ctx, topicID, blockHeight, label); err != nil {
			b.Fatalf("RegisterEpochLabel(%s) failed: %v", label, err)
		}
	}
	registry, err := k.GetEpochLabelRegistry(ctx, topicID, blockHeight)
	if err != nil {
		b.Fatalf("GetEpochLabelRegistry failed: %v", err)
	}
	actualRegistryLabels := len(registry.GetLabels())
	if actualRegistryLabels != registryLabels {
		b.Fatalf("unexpected EpochLabelRegistry size: got=%d want=%d", actualRegistryLabels, registryLabels)
	}

	inferers := make([]string, numInferer)
	for i := 0; i < numInferer; i++ {
		inferers[i] = addrsStr[i+1]
	}
	forecasters := make([]string, numForecaster)
	for i := 0; i < numForecaster; i++ {
		forecasters[i] = addrsStr[1+numInferer+i]
	}

	assignments := buildInfererLabelAssignments(numInferer, labelsPerInferer, registryLabels)

	infererToInference := make(map[string]*emissionstypes.Inference, numInferer)
	infererToRegret := make(map[string]*alloraMath.Dec, numInferer)
	for i, inferer := range inferers {
		values := buildInferenceValuesFromAssignments(i, assignments, registryLabels)
		infererToInference[inferer] = &emissionstypes.Inference{
			TopicId:     topicID,
			BlockHeight: blockHeight,
			Inferer:     inferer,
			Values:      values,
			ExtraData:   nil,
			Proof:       "",
		}
		regret := alloraMath.MustNewDecFromString("0.02")
		infererToRegret[inferer] = &regret

		ts := emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret}
		if err := k.SetInfererNetworkRegret(ctx, topicID, inferer, ts); err != nil {
			b.Fatalf("SetInfererNetworkRegret failed: %v", err)
		}
		if err := k.SetNaiveInfererNetworkRegret(ctx, topicID, inferer, ts); err != nil {
			b.Fatalf("SetNaiveInfererNetworkRegret failed: %v", err)
		}
	}

	forecasterToForecast := make(map[string]*emissionstypes.Forecast, numForecaster)
	forecasterToRegret := make(map[string]*alloraMath.Dec, numForecaster)
	for fi, forecaster := range forecasters {
		elements := make([]*emissionstypes.ForecastElement, 0, numInferer)
		for ii, inferer := range inferers {
			elements = append(elements, &emissionstypes.ForecastElement{
				Inferer: inferer,
				Value:   alloraMath.NewDecFromInt64(int64(2000 + fi + ii)),
			})
		}
		forecasterToForecast[forecaster] = &emissionstypes.Forecast{
			TopicId:          topicID,
			BlockHeight:      blockHeight,
			Forecaster:       forecaster,
			ForecastElements: elements,
			ExtraData:        nil,
		}
		regret := alloraMath.MustNewDecFromString("0.03")
		forecasterToRegret[forecaster] = &regret
		if err := k.SetForecasterNetworkRegret(
			ctx,
			topicID,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		); err != nil {
			b.Fatalf("SetForecasterNetworkRegret failed: %v", err)
		}
	}

	for _, withheldInferer := range inferers {
		for _, inferer := range inferers {
			if err := k.SetOneOutInfererInfererNetworkRegret(
				ctx,
				topicID,
				withheldInferer,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.04"),
				},
			); err != nil {
				b.Fatalf("SetOneOutInfererInfererNetworkRegret failed: %v", err)
			}
		}
		for _, forecaster := range forecasters {
			if err := k.SetOneOutInfererForecasterNetworkRegret(
				ctx,
				topicID,
				withheldInferer,
				forecaster,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.05"),
				},
			); err != nil {
				b.Fatalf("SetOneOutInfererForecasterNetworkRegret failed: %v", err)
			}
		}
	}
	for _, withheldForecaster := range forecasters {
		for _, inferer := range inferers {
			if err := k.SetOneOutForecasterInfererNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.06"),
				},
			); err != nil {
				b.Fatalf("SetOneOutForecasterInfererNetworkRegret failed: %v", err)
			}
			if err := k.SetOneInForecasterNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.08"),
				},
			); err != nil {
				b.Fatalf("SetOneInForecasterNetworkRegret failed: %v", err)
			}
		}
		for _, forecaster := range forecasters {
			if err := k.SetOneOutForecasterForecasterNetworkRegret(
				ctx,
				topicID,
				withheldForecaster,
				forecaster,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       alloraMath.MustNewDecFromString("0.07"),
				},
			); err != nil {
				b.Fatalf("SetOneOutForecasterForecasterNetworkRegret failed: %v", err)
			}
		}
		if err := k.SetOneInForecasterNetworkRegret(
			ctx,
			topicID,
			withheldForecaster,
			withheldForecaster,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString("0.09"),
			},
		); err != nil {
			b.Fatalf("SetOneInForecasterNetworkRegret(self) failed: %v", err)
		}
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	forecastImplied, infererRegretsOut, forecasterRegretsOut, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 log.NewNopLogger(),
			Ctx:                    ctx,
			K:                      k,
			TopicId:                topicID,
			AllInferersAreNew:      false,
			Inferers:               inferers,
			InfererToInference:     infererToInference,
			InfererToRegret:        infererToRegret,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecasterToForecast,
			ForecasterToRegret:     forecasterToRegret,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           topic.Epsilon,
			PNorm:                  topic.PNorm,
			CNorm:                  topic.CNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
		},
	)
	if err != nil {
		b.Fatalf("CalcForecastImpliedInferences setup failed: %v", err)
	}

	args := inferencesynthesis.CalcNetworkInferencesArgs{
		Ctx:                                  ctx,
		K:                                    k,
		Logger:                               log.NewNopLogger(),
		TopicId:                              topicID,
		Inferers:                             inferers,
		InfererToInference:                   infererToInference,
		InfererToRegret:                      infererRegretsOut,
		AllInferersAreNew:                    false,
		Forecasters:                          forecasters,
		ForecasterToForecast:                 forecasterToForecast,
		ForecasterToRegret:                   forecasterRegretsOut,
		ForecasterToForecastImpliedInference: forecastImplied,
		NetworkCombinedLoss:                  &networkCombinedLoss,
		EpsilonTopic:                         topic.Epsilon,
		EpsilonSafeDiv:                       alloraMath.MustNewDecFromString("0.0000001"),
		PNorm:                                topic.PNorm,
		CNorm:                                topic.CNorm,
		RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
		InferenceBlockHeight:                 blockHeight,
		LabelRegistry:                        &registry,
	}

	if _, _, err := inferencesynthesis.CalcNetworkInferences(args); err != nil {
		b.Fatalf("warmup CalcNetworkInferences failed: %v", err)
	}

	return benchFixture{
		args:           args,
		numLabels:      registryLabels,
		numInferer:     numInferer,
		registryLabels: actualRegistryLabels,
	}
}

func buildInfererLabelAssignments(
	numInferer int,
	labelsPerInferer int,
	registryLabels int,
) [][]int {
	assignments := make([][]int, numInferer)
	for i := range assignments {
		assignments[i] = make([]int, 0, labelsPerInferer)
	}

	// First, ensure every registry label is used at least once.
	for label := 0; label < registryLabels; label++ {
		inferer := label % numInferer
		if len(assignments[inferer]) < labelsPerInferer && !containsInt(assignments[inferer], label) {
			assignments[inferer] = append(assignments[inferer], label)
		}
	}

	// Then, fill each inferer up to labelsPerInferer with deterministic label picks.
	for inferer := 0; inferer < numInferer; inferer++ {
		candidate := inferer % registryLabels
		for len(assignments[inferer]) < labelsPerInferer {
			if !containsInt(assignments[inferer], candidate) {
				assignments[inferer] = append(assignments[inferer], candidate)
			}
			candidate = (candidate + 1) % registryLabels
		}
	}
	return assignments
}

func buildInferenceValuesFromAssignments(
	infererIdx int,
	assignments [][]int,
	registryLabels int,
) []alloraMath.Dec {
	values := make([]alloraMath.Dec, registryLabels)
	for _, labelIdx := range assignments[infererIdx] {
		values[labelIdx] = alloraMath.NewDecFromInt64(int64(1000 + infererIdx + labelIdx))
	}
	return values
}

func containsInt(xs []int, needle int) bool {
	for _, x := range xs {
		if x == needle {
			return true
		}
	}
	return false
}

func mustInitKeeperContext(b *testing.B) (sdk.Context, keeper.Keeper) {
	b.Helper()

	keyEmissions := storetypes.NewKVStoreKey("emissions")
	keyAccount := storetypes.NewKVStoreKey("account")
	keyBank := storetypes.NewKVStoreKey("bank")
	storeServiceAccount := runtime.NewKVStoreService(keyAccount)
	storeServiceEmissions := runtime.NewKVStoreService(keyEmissions)
	storeServiceBank := runtime.NewKVStoreService(keyBank)

	ctx := sdktestutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{
			"emissions": keyEmissions,
			"account":   keyAccount,
			"bank":      keyBank,
		},
		nil,
		nil,
	).WithHeaderInfo(header.Info{
		Height:  1,
		Hash:    []byte("1"),
		AppHash: []byte("1"),
		ChainID: "benchmarknet",
		Time:    time.Now(),
	}).WithLogger(log.NewNopLogger())

	encCfg := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		emissionsmodule.AppModule{},
	)

	maccPerms := map[string][]string{
		"fee_collector":                         {"minter"},
		"mint":                                  {"minter"},
		emissionstypes.AlloraStakingAccountName: {"burner", "minter", "staking"},
		emissionstypes.AlloraRewardsAccountName: {"minter"},
		emissionstypes.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		"ecosystem":              {"minter"},
		"bonded_tokens_pool":     {"burner", "staking"},
		"not_bonded_tokens_pool": {"burner", "staking"},
	}
	accountKeeper := authkeeper.NewAccountKeeper(
		encCfg.Codec,
		storeServiceAccount,
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress("gov").String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeServiceBank,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	emissionsKeeper := keeper.NewKeeper(
		encCfg.Codec,
		address.NewBech32Codec(params.Bech32PrefixAccAddr),
		storeServiceEmissions,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)

	// Seed module defaults (params, etc.) expected by topic validation.
	appModule := emissionsmodule.NewAppModule(encCfg.Codec, emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)

	return ctx, emissionsKeeper
}

package inferencesynthesis_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type InferenceSynthesisTestSuite struct {
	testutil.TestSuite
}

func TestInferenceSynthesisTestSuite(t *testing.T) {
	suite.Run(t, &InferenceSynthesisTestSuite{
		testutil.NewTestSuite("inference_synthesis"),
	})
}

// ConvertInferencesToWorkerAttributedValues converts an Inferences type to a slice of WorkerAttributedValue
func (s *InferenceSynthesisTestSuite) ConvertInferencesToWorkerAttributedValues(inferences emissionstypes.Inferences) []*emissionstypes.WorkerAttributedValue {
	result := make([]*emissionstypes.WorkerAttributedValue, 0, len(inferences.Inferences))

	for _, inf := range inferences.Inferences {
		var v alloraMath.Dec
		if len(inf.Values) > 0 {
			v = inf.Values[0]
		} else {
			v = alloraMath.ZeroDec()
		}

		result = append(result, &emissionstypes.WorkerAttributedValue{
			Worker: inf.Inferer,
			Value:  v,
		})
	}

	return result
}

func (s *InferenceSynthesisTestSuite) mockEmptyLossBundle(
	combinedAndNaiveValue alloraMath.Dec,
	infererValues []*emissionstypes.WorkerAttributedValue,
) emissionstypes.LossBundle {
	return emissionstypes.LossBundle{
		TopicId: uint64(1),
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &emissionstypes.Nonce{BlockHeight: 200},
		},
		Reputer:                       s.AddrsStr(9),
		ExtraData:                     nil,
		CombinedValue:                 combinedAndNaiveValue,
		InfererValues:                 infererValues,
		ForecasterValues:              nil,
		NaiveValue:                    combinedAndNaiveValue,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
}

func (s *InferenceSynthesisTestSuite) getEpochValueBundleByEpoch(epochNumber int) (
	ctx sdk.Context,
	k keeper.Keeper,
	logger log.Logger,
	topicId uint64,
	allInferersAreNew bool,
	inferers []string,
	infererToInference map[string]*emissionstypes.Inference,
	infererToRegret map[string]*alloraMath.Dec,
	forecasters []string,
	forecasterToForecast map[string]*emissionstypes.Forecast,
	forecasterToForecastImpliedInference map[string]*emissionstypes.Inference,
	forecasterToRegret map[string]*alloraMath.Dec,
	networkCombinedLoss alloraMath.Dec,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	epochGetters map[int]func(header string) alloraMath.Dec,
	registry *emissionstypes.EpochLabelRegistry,
) {
	k = *s.EmissionsKeeper()
	rk := s.RegretsKeeper()
	ctx = s.Ctx()
	topicId = uint64(1)
	blockHeight := int64(1)

	epochGetters = alloratestutil.GetSimulatedValuesGetterForEpochs()
	epochGet := epochGetters[epochNumber]

	networkLossPrevious := alloraMath.ZeroDec()

	if epochNumber > 0 {
		epochPrevGet := epochGetters[epochNumber-1]
		networkLossPrevious = epochPrevGet("network_loss")
		worker0 := s.AddrsStr(0)
		worker1 := s.AddrsStr(1)
		worker2 := s.AddrsStr(2)
		worker3 := s.AddrsStr(3)
		worker4 := s.AddrsStr(4)
		workerList := []string{worker0, worker1, worker2, worker3, worker4}

		forecaster5 := s.AddrsStr(8)
		forecaster6 := s.AddrsStr(9)
		forecaster7 := s.AddrsStr(10)
		forecasterList := make([]string, 8)
		forecasterList[5] = forecaster5
		forecasterList[6] = forecaster6
		forecasterList[7] = forecaster7

		// SET EPOCH 2 VALUES IN THE SYSTEM
		// Set inferer network regrets
		infererNetworkRegrets :=
			map[string]inferencesynthesis.Regret{
				worker0: epochPrevGet("inference_regret_worker_0"),
				worker1: epochPrevGet("inference_regret_worker_1"),
				worker2: epochPrevGet("inference_regret_worker_2"),
				worker3: epochPrevGet("inference_regret_worker_3"),
				worker4: epochPrevGet("inference_regret_worker_4"),
			}
		for inferer, regret := range infererNetworkRegrets {
			err := rk.SetInfererNetworkRegret(
				s.Ctx(),
				topicId,
				inferer,
				emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
			)
			s.Require().NoError(err)
		}

		// Set forecaster network regrets
		forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
			forecaster5: epochPrevGet("inference_regret_worker_5"),
			forecaster6: epochPrevGet("inference_regret_worker_6"),
			forecaster7: epochPrevGet("inference_regret_worker_7"),
		}
		for forecaster, regret := range forecasterNetworkRegrets {
			err := rk.SetForecasterNetworkRegret(
				s.Ctx(),
				topicId,
				forecaster,
				emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
			)
			s.Require().NoError(err)
		}

		// Set naive inferer network regrets
		infererNaiveNetworkRegrets :=
			map[string]inferencesynthesis.Regret{
				worker0: epochPrevGet("naive_inference_regret_worker_0"),
				worker1: epochPrevGet("naive_inference_regret_worker_1"),
				worker2: epochPrevGet("naive_inference_regret_worker_2"),
				worker3: epochPrevGet("naive_inference_regret_worker_3"),
				worker4: epochPrevGet("naive_inference_regret_worker_4"),
			}
		for inferer, regret := range infererNaiveNetworkRegrets {
			err := rk.SetNaiveInfererNetworkRegret(
				s.Ctx(),
				topicId,
				inferer,
				emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
			)
			s.Require().NoError(err)
		}

		// Set one-out inferer inferer network regrets
		setOneOutInfererInfererNetworkRegret := func(infererIndex int, infererIndex2 int, epochGetter func(string) alloraMath.Dec) {
			infererName := workerList[infererIndex]
			infererName2 := workerList[infererIndex2]
			headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_oneout_" + strconv.Itoa(infererIndex2)
			err := rk.SetOneOutInfererInfererNetworkRegret(
				s.Ctx(),
				topicId,
				infererName2,
				infererName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGetter(headerName),
				},
			)
			s.Require().NoError(err)
		}
		for inferer := 0; inferer < 5; inferer++ {
			for inferer2 := 0; inferer2 < 5; inferer2++ {
				setOneOutInfererInfererNetworkRegret(inferer, inferer2, epochPrevGet)
			}
		}

		// Set one-out inferer forecaster network regrets
		setOneOutInfererForecasterNetworkRegret := func(infererIndex int, forecasterIndex int, epochGetter func(string) alloraMath.Dec) {
			infererName := workerList[infererIndex]
			forecasterName := forecasterList[forecasterIndex]
			headerName := "inference_regret_worker_" + strconv.Itoa(forecasterIndex) + "_oneout_" + strconv.Itoa(infererIndex)
			err := rk.SetOneOutInfererForecasterNetworkRegret(
				s.Ctx(),
				topicId,
				infererName,
				forecasterName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGetter(headerName),
				},
			)
			s.Require().NoError(err)
		}
		for forecaster := 5; forecaster < 8; forecaster++ {
			for inferer := 0; inferer < 5; inferer++ {
				setOneOutInfererForecasterNetworkRegret(inferer, forecaster, epochPrevGet)
			}
		}

		// Set one-out forecaster inferer network regrets
		setOneOutForecasterInfererNetworkRegret := func(infererIndex int, forecasterIndex int, epochGetter func(string) alloraMath.Dec) {
			infererName := workerList[infererIndex]
			forecasterName := forecasterList[forecasterIndex]
			headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_oneout_" + strconv.Itoa(forecasterIndex)
			err := rk.SetOneOutForecasterInfererNetworkRegret(
				s.Ctx(),
				topicId,
				forecasterName,
				infererName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGetter(headerName),
				},
			)
			s.Require().NoError(err)
		}
		for inferer := 0; inferer < 5; inferer++ {
			for forecaster := 5; forecaster < 8; forecaster++ {
				setOneOutForecasterInfererNetworkRegret(inferer, forecaster, epochPrevGet)
			}
		}

		// Set one-out forecaster forecaster network regrets
		setOneOutForecasterForecasterNetworkRegret := func(forecasterIndex int, forecasterIndex2 int, epochGetter func(string) alloraMath.Dec) {
			forecasterName := forecasterList[forecasterIndex]
			forecasterName2 := forecasterList[forecasterIndex2]
			headerName := "inference_regret_worker_" + strconv.Itoa(forecasterIndex) + "_oneout_" + strconv.Itoa(forecasterIndex2)
			err := rk.SetOneOutForecasterForecasterNetworkRegret(
				s.Ctx(),
				topicId,
				forecasterName2,
				forecasterName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGetter(headerName),
				},
			)
			s.Require().NoError(err)
		}
		for forecaster := 5; forecaster < 8; forecaster++ {
			for forecaster2 := 5; forecaster2 < 8; forecaster2++ {
				setOneOutForecasterForecasterNetworkRegret(forecaster, forecaster2, epochPrevGet)
			}
		}

		// Set one-in network regrets
		setOneInForecasterNetworkRegret := func(forecasterIndex int, infererIndex int, epochGetter func(string) alloraMath.Dec) {
			forecasterName := forecasterList[forecasterIndex+5]
			infererName := workerList[infererIndex]
			headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_onein_" + strconv.Itoa(forecasterIndex)
			err := rk.SetOneInForecasterNetworkRegret(
				s.Ctx(),
				topicId,
				forecasterName,
				infererName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGetter(headerName),
				},
			)
			s.Require().NoError(err)
		}
		setOneInForecasterSelfRegret := func(forecaster int, epochGet func(string) alloraMath.Dec) {
			forecasterName := forecasterList[forecaster+5]
			headerName := "inference_regret_worker_5_onein_" + strconv.Itoa(forecaster)
			err := rk.SetOneInForecasterNetworkRegret(
				s.Ctx(),
				topicId,
				forecasterName,
				forecasterName,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight,
					Value:       epochGet(headerName),
				},
			)
			s.Require().NoError(err)
		}

		for forecaster := 0; forecaster < 3; forecaster++ {
			// Set self one-in network regrets
			setOneInForecasterSelfRegret(forecaster, epochPrevGet)
			for inferer := 0; inferer < 5; inferer++ {
				setOneInForecasterNetworkRegret(forecaster, inferer, epochPrevGet)
			}
		}
	}

	for workerIndex := 0; workerIndex < 5; workerIndex++ {
		stakeValue := epochGet("reputer_stake_" + strconv.Itoa(workerIndex))

		stakeValueScaled, err := stakeValue.Mul(alloraMath.MustNewDecFromString("1e18"))
		s.Require().NoError(err)

		stakeValueFloored, err := stakeValueScaled.Floor()
		s.Require().NoError(err)

		stakeInt, ok := cosmosMath.NewIntFromString(stakeValueFloored.String())
		s.Require().True(ok)

		workerString := s.AddrsStr(workerIndex)
		err = k.GetStakingKeeper().AddReputerStake(s.Ctx(), topicId, workerString, stakeInt)
		s.Require().NoError(err)
	}

	// SET EPOCH 3 VALUES IN VALUE BUNDLE
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{},
	}

	for infererIndex := 0; infererIndex < 5; infererIndex++ {
		inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
			Inferer:     s.AddrsStr(infererIndex),
			Values:      []alloraMath.Dec{epochGet("inference_" + strconv.Itoa(infererIndex))},
			ExtraData:   nil,
			Proof:       "",
			BlockHeight: blockHeight,
			TopicId:     topicId,
		})
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{},
	}

	for forecasterIndex := 0; forecasterIndex < 3; forecasterIndex++ {
		forecastElements := make([]*emissionstypes.ForecastElement, 0)
		for infererIndex := 0; infererIndex < 5; infererIndex++ {
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: s.AddrsStr(infererIndex),
				Value:   epochGet("forecasted_loss_" + strconv.Itoa(forecasterIndex) + "_for_" + strconv.Itoa(infererIndex)),
			})
		}
		forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
			Forecaster:       s.AddrsStr(forecasterIndex + 8),
			ForecastElements: forecastElements,
			TopicId:          topicId,
			BlockHeight:      blockHeight,
			ExtraData:        nil,
		})
	}

	epsilonTopic = alloraMath.MustNewDecFromString("0.01")
	epsilonSafeDiv = alloraMath.MustNewDecFromString("0.0000001")
	pNorm = alloraMath.MustNewDecFromString("3.0")
	cNorm = alloraMath.MustNewDecFromString("0.75")
	networkCombinedLoss = networkLossPrevious

	moduleParams := emissionstypes.DefaultParams()
	moduleParams.EpsilonSafeDiv = epsilonSafeDiv

	topic := s.MockTopic()
	topic.Id = topicId
	topic.PNorm = pNorm
	topic.Epsilon = epsilonTopic
	topic.CNorm = cNorm

	_, err := k.GetTopicKeeper().RegisterEpochLabel(ctx, topic.Id, blockHeight, "y")
	s.Require().NoError(err)

	registryVal, err := k.GetTopicKeeper().GetEpochLabelRegistry(ctx, topicId, blockHeight)
	s.Require().NoError(err)

	var calcArgs inferencesynthesis.CalcNetworkInferencesArgs
	calcArgs, err = inferencesynthesis.GetCalcNetworkInferenceArgs(
		ctx,
		k,
		topicId,
		&inferences,
		forecasts,
		topic,
		&networkCombinedLoss,
		moduleParams,
		blockHeight,
		&registryVal,
	)
	s.Require().NoError(err)
	return ctx, k, ctx.Logger(), topicId, calcArgs.AllInferersAreNew,
		calcArgs.Inferers, calcArgs.InfererToInference, calcArgs.InfererToRegret,
		calcArgs.Forecasters, calcArgs.ForecasterToForecast,
		calcArgs.ForecasterToForecastImpliedInference, calcArgs.ForecasterToRegret,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGetters, &registryVal
}

func (s *InferenceSynthesisTestSuite) getNetworkCalcArgs(
	blockHeight int64,
	inferences *emissionstypes.Inferences,
	forecasts *emissionstypes.Forecasts,
	networkCombinedLoss alloraMath.Dec,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) inferencesynthesis.CalcNetworkInferencesArgs {
	topicId := uint64(1)
	topic := s.MockTopic()
	topic.Id = topicId
	topic.Epsilon = epsilonTopic
	topic.PNorm = pNorm
	topic.CNorm = cNorm

	moduleParams := emissionstypes.DefaultParams()
	moduleParams.EpsilonSafeDiv = epsilonSafeDiv

	_, err := s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, blockHeight, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)

	var calcArgs inferencesynthesis.CalcNetworkInferencesArgs
	calcArgs, err = inferencesynthesis.GetCalcNetworkInferenceArgs(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		inferences,
		forecasts,
		topic,
		&networkCombinedLoss,
		moduleParams,
		blockHeight,
		&registry,
	)
	s.Require().NoError(err)
	return calcArgs
}

func (s *InferenceSynthesisTestSuite) incrementRegretsInTopic(
	topicId uint64,
	actor string,
	times int,
	actorType emissionstypes.ActorType,
) {
	for index := 0; index < times; index++ {
		if actorType == emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED {
			err := s.EmissionsKeeper().IncrementCountInfererInclusionsInTopic(s.Ctx(), topicId, actor)
			s.Require().NoError(err)
		} else if actorType == emissionstypes.ActorType_ACTOR_TYPE_FORECASTER {
			err := s.EmissionsKeeper().IncrementCountForecasterInclusionsInTopic(s.Ctx(), topicId, actor)
			s.Require().NoError(err)
		}
	}
}

func (s *InferenceSynthesisTestSuite) testCorrectCombinedInitialValueForEpoch(epoch int) {
	_, _, logger, topicId, allInferersAreNew,
		inferers, inferenceByWorker, infererRegrets,
		forecasters, _, forecastImpliedInferenceByWorker, forecasterRegrets,
		_, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGet, registry := s.getEpochValueBundleByEpoch(epoch)

	_, combinedValue, err := inferencesynthesis.GetCombinedInference(
		inferencesynthesis.GetCombinedInferenceArgs{
			Logger:                               logger,
			TopicId:                              topicId,
			Inferers:                             inferers,
			InfererToInference:                   inferenceByWorker,
			InfererToRegret:                      infererRegrets,
			AllInferersAreNew:                    allInferersAreNew,
			Forecasters:                          forecasters,
			ForecasterToRegret:                   forecasterRegrets,
			ForecasterToForecastImpliedInference: forecastImpliedInferenceByWorker,
			EpsilonTopic:                         epsilonTopic,
			EpsilonSafeDiv:                       epsilonSafeDiv,
			PNorm:                                pNorm,
			CNorm:                                cNorm,
			RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
			NumLabels:                            len(registry.GetLabels()),
		})
	s.Require().NoError(err)
	alloratestutil.InEpsilon5(s.T(), combinedValue[0], epochGet[epoch]("network_inference").String())
}

func (s *InferenceSynthesisTestSuite) TestCorrectCombinedValueEpoch2() {
	s.testCorrectCombinedInitialValueForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectCombnedValueEpoch3() {
	s.testCorrectCombinedInitialValueForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) TestCorrectCombinedValueEpoch4() {
	s.testCorrectCombinedInitialValueForEpoch(304)
}

func (s *InferenceSynthesisTestSuite) testCorrectNaiveValueForEpoch(epoch int) {
	ctx, k, logger, topicId, allInferersAreNew,
		inferers, inferenceByWorker, _,
		forecasters, _, forecastImpliedInferenceByWorker, forecasterRegrets,
		_, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGet, registry := s.getEpochValueBundleByEpoch(epoch)
	naiveValue, err := inferencesynthesis.GetNaiveInference(
		inferencesynthesis.GetNaiveInferenceArgs{
			Ctx:                                  ctx,
			Logger:                               logger,
			TopicId:                              topicId,
			K:                                    k,
			Inferers:                             inferers,
			InfererToInference:                   inferenceByWorker,
			AllInferersAreNew:                    allInferersAreNew,
			Forecasters:                          forecasters,
			ForecasterToRegret:                   forecasterRegrets,
			ForecasterToForecastImpliedInference: forecastImpliedInferenceByWorker,
			EpsilonTopic:                         epsilonTopic,
			EpsilonSafeDiv:                       epsilonSafeDiv,
			PNorm:                                pNorm,
			CNorm:                                cNorm,
			RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
			LabelRegistry:                        registry,
			NumLabels:                            len(registry.GetLabels()),
		})
	s.Require().NoError(err)
	alloratestutil.InEpsilon5(s.T(), naiveValue[0].Value, epochGet[epoch]("network_naive_inference").String())
}

func (s *InferenceSynthesisTestSuite) TestCorrectNaiveValueEpoch2() {
	s.testCorrectNaiveValueForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectNaiveValueEpoch3() {
	s.testCorrectNaiveValueForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) testCorrectOneOutInfererValuesForEpoch(epoch int) {
	ctx, k, logger, topicId, allInferersAreNew,
		inferers, inferenceByWorker, infererRegrets,
		forecasters, forecastByWorker, _, forecasterRegrets,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGet, registry := s.getEpochValueBundleByEpoch(epoch)

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)

	expectedValues := map[string]alloraMath.Dec{
		worker0: epochGet[epoch]("network_inference_oneout_0"),
		worker1: epochGet[epoch]("network_inference_oneout_1"),
		worker2: epochGet[epoch]("network_inference_oneout_2"),
		worker3: epochGet[epoch]("network_inference_oneout_3"),
		worker4: epochGet[epoch]("network_inference_oneout_4"),
	}

	oneOutInfererValues, err := inferencesynthesis.GetOneOutInfererInferences(
		inferencesynthesis.GetOneOutInfererInferencesArgs{
			Ctx:                    ctx,
			K:                      k,
			Logger:                 logger,
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			AllInferersAreNew:      allInferersAreNew,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilonTopic,
			EpsilonSafeDiv:         epsilonSafeDiv,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          registry,
			NumLabels:              len(registry.GetLabels()),
		})
	s.Require().NoError(err)

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneOutInfererValues {
			if workerAttributedValue.WithheldInferer == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.GetCombinedInference()[0].Value.String()) // Reduced to Epsilon4 after removal of regret boundaries
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutInfererValuesEpoch2() {
	s.testCorrectOneOutInfererValuesForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutInfererValuesEpoch3() {
	s.testCorrectOneOutInfererValuesForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) testCorrectOneOutForecasterValuesForEpoch(epoch int) {
	ctx, k, logger, topicId, allInferersAreNew,
		inferers, inferenceByWorker, infererRegrets,
		forecasters, forecastByWorker, forecastImpliedInferenceByWorker, forecasterRegrets,
		_, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGet, registry := s.getEpochValueBundleByEpoch(epoch)
	oneOutForecasterValues, err := inferencesynthesis.GetOneOutForecasterInferences(
		inferencesynthesis.GetOneOutForecasterInferencesArgs{
			Ctx:                                  ctx,
			K:                                    k,
			Logger:                               logger,
			TopicId:                              topicId,
			Inferers:                             inferers,
			InfererToInference:                   inferenceByWorker,
			InfererToRegret:                      infererRegrets,
			AllInferersAreNew:                    allInferersAreNew,
			Forecasters:                          forecasters,
			ForecasterToForecast:                 forecastByWorker,
			ForecasterToRegret:                   forecasterRegrets,
			ForecasterToForecastImpliedInference: forecastImpliedInferenceByWorker,
			EpsilonTopic:                         epsilonTopic,
			EpsilonSafeDiv:                       epsilonSafeDiv,
			PNorm:                                pNorm,
			CNorm:                                cNorm,
			RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
			LabelRegistry:                        registry,
			NumLabels:                            len(registry.GetLabels()),
		})
	s.Require().NoError(err)

	forecaster0 := s.AddrsStr(8)
	forecaster1 := s.AddrsStr(9)
	forecaster2 := s.AddrsStr(10)

	expectedValues := map[string]alloraMath.Dec{
		forecaster0: epochGet[epoch]("network_inference_oneout_5"),
		forecaster1: epochGet[epoch]("network_inference_oneout_6"),
		forecaster2: epochGet[epoch]("network_inference_oneout_7"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneOutForecasterValues {
			if workerAttributedValue.WithheldForecaster == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.GetCombinedInference()[0].Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutForecasterValuesEpoch2() {
	s.testCorrectOneOutForecasterValuesForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutForecasterValuesEpoch3() {
	s.testCorrectOneOutForecasterValuesForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutForecasterValuesEpoch4() {
	s.testCorrectOneOutForecasterValuesForEpoch(304)
}

func (s *InferenceSynthesisTestSuite) testCorrectOneInForecasterValuesForEpoch(epoch int) {
	ctx, k, logger, topicId, allInferersAreNew,
		inferers, inferenceByWorker, _,
		forecasters, _, forecastImpliedInferenceByWorker, _,
		_, epsilonTopic, epsilonSafeDiv, pNorm, cNorm, epochGet, registry := s.getEpochValueBundleByEpoch(epoch)
	oneInForecasterValues, err := inferencesynthesis.GetOneInForecasterInferences(
		inferencesynthesis.GetOneInForecasterInferencesArgs{
			Ctx:                                  ctx,
			K:                                    k,
			Logger:                               logger,
			TopicId:                              topicId,
			Inferers:                             inferers,
			InfererToInference:                   inferenceByWorker,
			AllInferersAreNew:                    allInferersAreNew,
			Forecasters:                          forecasters,
			ForecasterToForecastImpliedInference: forecastImpliedInferenceByWorker,
			EpsilonTopic:                         epsilonTopic,
			EpsilonSafeDiv:                       epsilonSafeDiv,
			PNorm:                                pNorm,
			CNorm:                                cNorm,
			RegretScalePlusEpsilon:               alloraMath.ZeroDec(),
			LabelRegistry:                        registry,
			NumLabels:                            len(registry.GetLabels()),
		})
	s.Require().NoError(err)

	forecaster0 := s.AddrsStr(8)
	forecaster1 := s.AddrsStr(9)
	forecaster2 := s.AddrsStr(10)

	expectedValues := map[string]alloraMath.Dec{
		forecaster0: epochGet[epoch]("network_naive_inference_onein_0"),
		forecaster1: epochGet[epoch]("network_naive_inference_onein_1"),
		forecaster2: epochGet[epoch]("network_naive_inference_onein_2"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneInForecasterValues {
			if workerAttributedValue.Forecaster == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.GetCombinedInference()[0].Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneInForecasterValuesEpoch2() {
	s.testCorrectOneInForecasterValuesForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneInForecasterValuesEpoch3() {
	s.testCorrectOneInForecasterValuesForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneInForecasterValuesEpoch4() {
	s.testCorrectOneInForecasterValuesForEpoch(304)
}

func (s *InferenceSynthesisTestSuite) TestBuildNetworkInferencesIncompleteData() {
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")}},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.71")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilonTopic := alloraMath.MustNewDecFromString("0.0001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	calcArgs := s.getNetworkCalcArgs(
		blockHeightInferences, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(calcArgs)
	s.Require().NoError(err)

	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// OneInForecastValues come empty because regrets are epsilon
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesTwoWorkerTwoForecasters() {
	k := s.RegretsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(300)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.5")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.7")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
				ExtraData: nil,
			},
		},
	}

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilonTopic := alloraMath.MustNewDecFromString("0.0001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	calcArgs := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	s.Require().NoError(err)

	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(calcArgs)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 2)
	s.Require().Len(valueBundle.OneOutForecasterValues, 2)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesThreeWorkerThreeForecasters() {
	k := s.RegretsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(300)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)

	forecaster1 := s.AddrsStr(4)
	forecaster2 := s.AddrsStr(5)
	forecaster3 := s.AddrsStr(6)

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("200")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("300")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("400")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("500")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("600")},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("700")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("800")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("900")},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("100")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("200")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("300")},
				},
				ExtraData: nil,
			},
		},
	}

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.007"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.008"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.009"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	networkCombinedLoss := alloraMath.MustNewDecFromString("10000")
	epsilonTopic := alloraMath.MustNewDecFromString("0.001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	calcArgs := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(
		calcArgs,
	)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesThreeWorkerTwoForecastersValidOneForecasterInvalid() {
	k := s.RegretsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(300)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)

	forecaster1 := s.AddrsStr(4)
	forecaster2 := s.AddrsStr(5)
	forecaster3 := s.AddrsStr(6)

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("200")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("300")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("400")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("500")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("600")},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("700")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("800")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("900")},
				},
				ExtraData: nil,
			},
			// Invalid forecaster info
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "unexistent_worker1", Value: alloraMath.MustNewDecFromString("100")},
					{Inferer: "unexistent_worker2", Value: alloraMath.MustNewDecFromString("200")},
					{Inferer: "unexistent_worker3", Value: alloraMath.MustNewDecFromString("300")},
				},
				ExtraData: nil,
			},
		},
	}

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.007"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.008"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.009"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	networkCombinedLoss := alloraMath.MustNewDecFromString("10000")
	epsilonTopic := alloraMath.MustNewDecFromString("0.001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	calcArgs := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	networkInferences, weights, err := inferencesynthesis.CalcNetworkInferences(
		calcArgs,
	)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(networkInferences)
	s.Require().NotNil(networkInferences.CombinedValue)
	s.Require().NotNil(networkInferences.NaiveValue)
	s.Require().Len(networkInferences.InfererValues, 3)
	s.Require().Len(networkInferences.ForecasterValues, 2)

	s.Require().Len(networkInferences.OneOutInfererValues, 3)
	s.Require().Len(networkInferences.OneOutForecasterValues, 2)
	s.Require().Len(networkInferences.OneInForecasterValues, 2)

	s.Require().NotNil(weights)
	s.Require().Len(weights.Inferers, 3)
	s.Require().Len(weights.Forecasters, 2)
}

func (s *InferenceSynthesisTestSuite) TestCalcOneInInferencesTwoForecastersOldTwoInferersNewOneOldOneNew() {
	k := s.RegretsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)

	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100")}},
			{Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("200")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("100")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("500")},
				},
			},
			{
				Forecaster: worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("700")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("200")},
				},
			},
		},
	}

	blockHeight := int64(300)
	// Set inferer network regrets - Just for worker1
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001"), BlockHeight: blockHeight})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.008"), BlockHeight: blockHeight})
	s.Require().NoError(err)

	networkCombinedLoss := alloraMath.MustNewDecFromString("10000")
	epsilonTopic := alloraMath.MustNewDecFromString("0.001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	calcArgs := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(calcArgs)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		if oneInForecasterValue.Forecaster == worker1 {
			s.Require().True(oneInForecasterValue.GetCombinedInference()[0].Value.Gt(alloraMath.ZeroDec()))
		} else if oneInForecasterValue.Forecaster == worker2 {
			s.Require().True(oneInForecasterValue.GetCombinedInference()[0].Value.Gt(alloraMath.ZeroDec()))
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetOneOutInfererImpliedInferences() {
	k := *s.EmissionsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(300)

	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	forecaster1 := s.AddrsStr(4)
	forecaster2 := s.AddrsStr(5)

	// Set up input data with 3 inferers and 2 forecasters
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("200")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("300")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("150")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("250")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("350")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("120")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("220")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("320")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilonTopic := alloraMath.MustNewDecFromString("0.0001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	args := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm)

	oneOutInfererForecastImpliedValues, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(
		inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
			Ctx:                    ctx,
			K:                      k,
			Logger:                 ctx.Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			Inferers:               args.Inferers,
			InfererToInference:     args.InfererToInference,
			InfererToRegret:        args.InfererToRegret,
			AllInferersAreNew:      args.AllInferersAreNew,
			Forecasters:            args.Forecasters,
			ForecasterToForecast:   args.ForecasterToForecast,
			ForecasterToRegret:     args.ForecasterToRegret,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilonTopic,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          args.LabelRegistry,
			NumLabels:              args.NumLabels,
		})
	s.Require().NoError(err)

	// Test specific assertions for one-out inferer implied inferences
	s.Require().NotNil(oneOutInfererForecastImpliedValues)
	s.Require().Len(oneOutInfererForecastImpliedValues, 6)

	// For each forecaster, verify their implied inferences when each inferer is removed
	for _, forecasterValues := range oneOutInfererForecastImpliedValues {
		s.Require().Contains([]string{forecaster1, forecaster2}, forecasterValues.Forecaster)
		s.Require().Contains([]string{worker1, worker2, worker3}, forecasterValues.WithheldInferer)
		s.Require().Len(forecasterValues.GetCombinedInference(), 1)

		oneOutValue := forecasterValues.GetCombinedInference()[0]
		s.Require().True(oneOutValue.Value.IsPositive())
		s.Require().True(oneOutValue.Value.Lt(alloraMath.MustNewDecFromString("400")))
		s.Require().True(oneOutValue.Value.Gt(alloraMath.MustNewDecFromString("50")))
	}

	// Test edge case: no forecasters
	emptyResult, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(
		inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
			Ctx:                    ctx,
			K:                      k,
			Logger:                 ctx.Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			Inferers:               args.Inferers,
			InfererToInference:     args.InfererToInference,
			InfererToRegret:        args.InfererToRegret,
			AllInferersAreNew:      args.AllInferersAreNew,
			Forecasters:            []string{}, // Empty forecasters
			ForecasterToForecast:   make(map[string]*emissionstypes.Forecast),
			ForecasterToRegret:     make(map[string]*alloraMath.Dec),
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilonTopic,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          args.LabelRegistry,
			NumLabels:              args.NumLabels,
		})
	s.Require().NoError(err)
	s.Require().Empty(emptyResult)

	// Test edge case: single inferer (should return empty result as one-out requires at least 2 inferers)
	singleInfererResult, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(
		inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
			Ctx:                    ctx,
			K:                      k,
			Logger:                 ctx.Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			Inferers:               []string{worker1},
			InfererToInference:     map[string]*emissionstypes.Inference{worker1: inferences.Inferences[0]},
			InfererToRegret:        map[string]*alloraMath.Dec{},
			AllInferersAreNew:      true,
			Forecasters:            args.Forecasters,
			ForecasterToForecast:   args.ForecasterToForecast,
			ForecasterToRegret:     args.ForecasterToRegret,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilonTopic,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry: &emissionstypes.EpochLabelRegistry{
				TopicId: topicId,
				EpochId: 1,
				Labels: []*emissionstypes.TopicLabel{{
					Id:   1,
					Name: "y",
				}},
			},
			NumLabels: 1,
		})
	s.Require().NoError(err)
	s.Require().Empty(singleInfererResult)
}

func (s *InferenceSynthesisTestSuite) TestGetOneOutInfererImpliedInferences2inferers2forecasters() {
	k := *s.EmissionsKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)
	blockHeight := int64(300)

	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	forecaster1 := s.AddrsStr(3)
	forecaster2 := s.AddrsStr(4)

	// Set up input data with 2 inferers and 2 forecasters
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100")}},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("200")}},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("150")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("250")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("120")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("220")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilonTopic := alloraMath.MustNewDecFromString("0.0001")
	epsilonSafeDiv := alloraMath.MustNewDecFromString("0.0000001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	args := s.getNetworkCalcArgs(
		blockHeight, inferences, forecasts,
		networkCombinedLoss, epsilonTopic, epsilonSafeDiv, pNorm, cNorm,
	)

	oneOutInfererForecastImpliedValues, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(
		inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
			Ctx:                    ctx,
			K:                      k,
			Logger:                 ctx.Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			Inferers:               args.Inferers,
			InfererToInference:     args.InfererToInference,
			InfererToRegret:        args.InfererToRegret,
			AllInferersAreNew:      args.AllInferersAreNew,
			Forecasters:            args.Forecasters,
			ForecasterToForecast:   args.ForecasterToForecast,
			ForecasterToRegret:     args.ForecasterToRegret,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilonTopic,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          args.LabelRegistry,
			NumLabels:              args.NumLabels,
		},
	)
	s.Require().NoError(err)

	s.Require().NotNil(oneOutInfererForecastImpliedValues)
	s.Require().Len(oneOutInfererForecastImpliedValues, 4) // 2 forecasters * 2 withheld inferers

	seen := make(map[string]map[string]bool)

	for _, v := range oneOutInfererForecastImpliedValues {
		s.Require().Contains([]string{forecaster1, forecaster2}, v.Forecaster)
		s.Require().Contains([]string{worker1, worker2}, v.WithheldInferer)
		s.Require().Len(v.GetCombinedInference(), 1)

		if _, ok := seen[v.Forecaster]; !ok {
			seen[v.Forecaster] = make(map[string]bool)
		}
		s.Require().False(seen[v.Forecaster][v.WithheldInferer], "duplicate forecaster/withheldInferer pair")
		seen[v.Forecaster][v.WithheldInferer] = true

		oneOutValue := v.GetCombinedInference()[0]
		s.Require().True(oneOutValue.Value.IsPositive())
		s.Require().True(oneOutValue.Value.Lt(alloraMath.MustNewDecFromString("300")))
		s.Require().True(oneOutValue.Value.Gt(alloraMath.MustNewDecFromString("50")))
	}

	s.Require().Len(seen, 2)
	s.Require().Len(seen[forecaster1], 2)
	s.Require().Len(seen[forecaster2], 2)
}

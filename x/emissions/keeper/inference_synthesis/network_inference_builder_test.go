package inferencesynthesis_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"cosmossdk.io/core/header"
	clog "cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"

	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type InferenceSynthesisTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	key             *storetypes.KVStoreKey
	privKeys        []secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
	pubKeyHexStr    []string
}

func (s *InferenceSynthesisTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                         {"minter"},
		"mint":                                  {"minter"},
		emissionstypes.AlloraStakingAccountName: {"burner", "minter", "staking"},
		emissionstypes.AlloraRewardsAccountName: {"minter"},
		emissionstypes.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		"bonded_tokens_pool":     {"burner", "staking"},
		"not_bonded_tokens_pool": {"burner", "staking"},
		multiPerm:                {"burner", "minter", "staking"},
		randomPerm:               {"random"},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		encCfg.Codec,
		storeService,
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress("gov").String(),
	)

	s.privKeys, s.pubKeyHexStr, s.addrs, s.addrsStr = alloratestutil.GenerateTestAccounts(15)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		clog.NewNopLogger(),
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = keeper.NewKeeper(
		encCfg.Codec,
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range s.addrsStr {
		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.Require().NoError(err)
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, 1, emissionstypes.Topic{
		Id:                       1,
		Creator:                  s.addrsStr[0],
		Metadata:                 "metadata",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.01"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.01"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.01"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.01"),
	})
	s.Require().NoError(err)
}

func TestInferenceSynthesisTestSuite(t *testing.T) {
	suite.Run(t, new(InferenceSynthesisTestSuite))
}

func (s *InferenceSynthesisTestSuite) signValueBundle(
	reputerValueBundle *emissionstypes.ValueBundle,
	privateKey secp256k1.PrivKey,
) []byte {
	require := s.Require()
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")
	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	return valueBundleSignature
}

func (s *InferenceSynthesisTestSuite) mockTopic() emissionstypes.Topic {
	return emissionstypes.Topic{
		Id:                     1,
		Creator:                s.addrsStr[0],
		Metadata:               "metadata",
		LossMethod:             "mse",
		EpochLastEnded:         0,
		EpochLength:            100,
		GroundTruthLag:         100,
		WorkerSubmissionWindow: 100,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
		// Set Initial Regret
		InitialRegret:            alloraMath.OneDec(),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.01"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.01"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.01"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.01"),
	}
}

func (s *InferenceSynthesisTestSuite) mockEmptyValueBundle(
	combinedAndNaiveValue alloraMath.Dec,
) emissionstypes.ValueBundle {
	return emissionstypes.ValueBundle{
		TopicId: uint64(1),
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &emissionstypes.Nonce{BlockHeight: 200},
		},
		Reputer:                       s.addrsStr[9],
		ExtraData:                     nil,
		CombinedValue:                 combinedAndNaiveValue,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    combinedAndNaiveValue,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
}

func (s *InferenceSynthesisTestSuite) getEpochValueBundleByEpoch(epochNumber int) (
	inferencesynthesis.SynthPalette,
	map[int]func(header string) alloraMath.Dec,
) {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeight := int64(1)

	epochGetters := alloratestutil.GetSimulatedValuesGetterForEpochs()
	epochGet := epochGetters[epochNumber]

	networkLossPrevious := alloraMath.ZeroDec()

	if epochNumber > 0 {
		epochPrevGet := epochGetters[epochNumber-1]
		networkLossPrevious = epochPrevGet("network_loss")
		worker0 := s.addrsStr[0]
		worker1 := s.addrsStr[1]
		worker2 := s.addrsStr[2]
		worker3 := s.addrsStr[3]
		worker4 := s.addrsStr[4]
		workerList := []string{worker0, worker1, worker2, worker3, worker4}

		forecaster5 := s.addrsStr[8]
		forecaster6 := s.addrsStr[9]
		forecaster7 := s.addrsStr[10]
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
			err := s.emissionsKeeper.SetInfererNetworkRegret(
				s.ctx,
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
			err := s.emissionsKeeper.SetForecasterNetworkRegret(
				s.ctx,
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
			err := s.emissionsKeeper.SetNaiveInfererNetworkRegret(
				s.ctx,
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
			err := k.SetOneOutInfererInfererNetworkRegret(
				s.ctx,
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
			err := k.SetOneOutInfererForecasterNetworkRegret(
				s.ctx,
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
			err := k.SetOneOutForecasterInfererNetworkRegret(
				s.ctx,
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
			err := k.SetOneOutForecasterForecasterNetworkRegret(
				s.ctx,
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
			err := k.SetOneInForecasterNetworkRegret(
				s.ctx,
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
			err := k.SetOneInForecasterNetworkRegret(
				s.ctx,
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

		workerString := s.addrsStr[workerIndex]
		err = k.AddReputerStake(s.ctx, topicId, workerString, stakeInt)
		s.Require().NoError(err)
	}

	// SET EPOCH 3 VALUES IN VALUE BUNDLE
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{},
	}

	for infererIndex := 0; infererIndex < 5; infererIndex++ {
		inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
			Inferer: s.addrsStr[infererIndex],
			Value:   epochGet("inference_" + strconv.Itoa(infererIndex)),
		})
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{},
	}

	for forecasterIndex := 0; forecasterIndex < 3; forecasterIndex++ {
		forecastElements := make([]*emissionstypes.ForecastElement, 0)
		for infererIndex := 0; infererIndex < 5; infererIndex++ {
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: s.addrsStr[infererIndex],
				Value:   epochGet("forecasted_loss_" + strconv.Itoa(forecasterIndex) + "_for_" + strconv.Itoa(infererIndex)),
			})
		}
		forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
			Forecaster:       s.addrsStr[forecasterIndex+8],
			ForecastElements: forecastElements,
		})
	}

	paletteFactory := inferencesynthesis.SynthPaletteFactory{}
	synthPalette, err := paletteFactory.BuildPaletteFromRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkLossPrevious,
			EpsilonTopic:        alloraMath.MustNewDecFromString("0.01"),
			EpsilonSafeDiv:      alloraMath.MustNewDecFromString("0.0000001"),
			PNorm:               alloraMath.MustNewDecFromString("3.0"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		})
	s.Require().NoError(err)

	return synthPalette, epochGetters
}

func (s *InferenceSynthesisTestSuite) testCorrectCombinedInitialValueForEpoch(epoch int) {
	synthPalette, epochGet := s.getEpochValueBundleByEpoch(epoch)
	_, combinedValue, err := inferencesynthesis.GetCombinedInference(synthPalette)
	s.Require().NoError(err)
	alloratestutil.InEpsilon5(s.T(), combinedValue, epochGet[epoch]("network_inference").String())
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
	synthPalette, epochGet := s.getEpochValueBundleByEpoch(epoch)
	naiveValue, err := inferencesynthesis.GetNaiveInference(synthPalette)
	s.Require().NoError(err)
	alloratestutil.InEpsilon5(s.T(), naiveValue, epochGet[epoch]("network_naive_inference").String())
}

func (s *InferenceSynthesisTestSuite) TestCorrectNaiveValueEpoch2() {
	s.testCorrectNaiveValueForEpoch(302)
}

func (s *InferenceSynthesisTestSuite) TestCorrectNaiveValueEpoch3() {
	s.testCorrectNaiveValueForEpoch(303)
}

func (s *InferenceSynthesisTestSuite) testCorrectOneOutInfererValuesForEpoch(epoch int) {
	synthPalette, epochGet := s.getEpochValueBundleByEpoch(epoch)

	worker0 := s.addrsStr[0]
	worker1 := s.addrsStr[1]
	worker2 := s.addrsStr[2]
	worker3 := s.addrsStr[3]
	worker4 := s.addrsStr[4]

	expectedValues := map[string]alloraMath.Dec{
		worker0: epochGet[epoch]("network_inference_oneout_0"),
		worker1: epochGet[epoch]("network_inference_oneout_1"),
		worker2: epochGet[epoch]("network_inference_oneout_2"),
		worker3: epochGet[epoch]("network_inference_oneout_3"),
		worker4: epochGet[epoch]("network_inference_oneout_4"),
	}

	oneOutInfererValues, err := inferencesynthesis.GetOneOutInfererInferences(synthPalette)
	s.Require().NoError(err)

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneOutInfererValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.Value.String())
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
	synthPalette, epochGet := s.getEpochValueBundleByEpoch(epoch)
	oneOutForecasterValues, err := inferencesynthesis.GetOneOutForecasterInferences(synthPalette)
	s.Require().NoError(err)

	forecaster0 := s.addrsStr[8]
	forecaster1 := s.addrsStr[9]
	forecaster2 := s.addrsStr[10]

	expectedValues := map[string]alloraMath.Dec{
		forecaster0: epochGet[epoch]("network_inference_oneout_5"),
		forecaster1: epochGet[epoch]("network_inference_oneout_6"),
		forecaster2: epochGet[epoch]("network_inference_oneout_7"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneOutForecasterValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.Value.String())
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
	synthPalette, epochGet := s.getEpochValueBundleByEpoch(epoch)
	oneInForecasterValues, err := inferencesynthesis.GetOneInForecasterInferences(synthPalette)
	s.Require().NoError(err)

	forecaster0 := s.addrsStr[8]
	forecaster1 := s.addrsStr[9]
	forecaster2 := s.addrsStr[10]

	expectedValues := map[string]alloraMath.Dec{
		forecaster0: epochGet[epoch]("network_naive_inference_onein_0"),
		forecaster1: epochGet[epoch]("network_naive_inference_onein_1"),
		forecaster2: epochGet[epoch]("network_naive_inference_onein_2"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range oneInForecasterValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon5(s.T(), expectedValue, workerAttributedValue.Value.String())
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
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := s.addrsStr[1]
	worker2 := s.addrsStr[2]

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
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

	// Call the function without setting regrets
	paletteFactory := inferencesynthesis.SynthPaletteFactory{}
	synthPalette, err := paletteFactory.BuildPaletteFromRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: alloraMath.MustNewDecFromString("1"),
			EpsilonTopic:        alloraMath.MustNewDecFromString("0.0001"),
			EpsilonSafeDiv:      alloraMath.MustNewDecFromString("0.0000001"),
			PNorm:               alloraMath.MustNewDecFromString("2"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		},
	)
	s.Require().NoError(err)
	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(synthPalette)
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
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeight := int64(300)
	worker1 := s.addrsStr[1]
	worker2 := s.addrsStr[2]
	worker3 := s.addrsStr[3]
	worker4 := s.addrsStr[4]

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
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
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	paletteFactory := inferencesynthesis.SynthPaletteFactory{}
	synthPalette, err := paletteFactory.BuildPaletteFromRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: alloraMath.MustNewDecFromString("0.2"),
			EpsilonTopic:        alloraMath.MustNewDecFromString("0.0001"),
			EpsilonSafeDiv:      alloraMath.MustNewDecFromString("0.0000001"),
			PNorm:               alloraMath.MustNewDecFromString("2"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		},
	)
	s.Require().NoError(err)
	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(synthPalette)
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
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeight := int64(300)
	worker1 := s.addrsStr[1]
	worker2 := s.addrsStr[2]
	worker3 := s.addrsStr[3]

	forecaster1 := s.addrsStr[4]
	forecaster2 := s.addrsStr[5]
	forecaster3 := s.addrsStr[6]

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker1, Value: alloraMath.MustNewDecFromString("100")},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{TopicId: topicId, BlockHeight: blockHeight, Inferer: worker3, Value: alloraMath.MustNewDecFromString("300")},
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
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.007")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.008")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.009")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.002")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.003")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.006")})
	s.Require().NoError(err)

	paletteFactory := inferencesynthesis.SynthPaletteFactory{}
	synthPalette, err := paletteFactory.BuildPaletteFromRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: alloraMath.MustNewDecFromString("10000"),
			EpsilonTopic:        alloraMath.MustNewDecFromString("0.001"),
			EpsilonSafeDiv:      alloraMath.MustNewDecFromString("0.0000001"),
			PNorm:               alloraMath.MustNewDecFromString("2"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		},
	)
	s.Require().NoError(err)
	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(synthPalette)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

func (s *InferenceSynthesisTestSuite) TestCalc0neInInferencesTwoForecastersOldTwoInferersNewOneOldOneNew() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := s.addrsStr[1]
	worker2 := s.addrsStr[2]

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("100")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("200")},
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

	// Set inferer network regrets - Just for worker1
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.004")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.005")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.001")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.008")})
	s.Require().NoError(err)

	paletteFactory := inferencesynthesis.SynthPaletteFactory{}
	synthPalette, err := paletteFactory.BuildPaletteFromRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: alloraMath.MustNewDecFromString("10000"),
			EpsilonTopic:        alloraMath.MustNewDecFromString("0.001"),
			EpsilonSafeDiv:      alloraMath.MustNewDecFromString("0.0000001"),
			PNorm:               alloraMath.MustNewDecFromString("2"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		},
	)
	s.Require().NoError(err)
	valueBundle, _, err := inferencesynthesis.CalcNetworkInferences(synthPalette)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		if oneInForecasterValue.Worker == worker1 {
			s.Require().True(oneInForecasterValue.Value.Gt(alloraMath.ZeroDec()))
		} else if oneInForecasterValue.Worker == worker2 {
			s.Require().True(oneInForecasterValue.Value.Gt(alloraMath.ZeroDec()))
		}
	}
}

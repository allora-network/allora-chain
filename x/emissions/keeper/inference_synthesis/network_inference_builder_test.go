package inference_synthesis_test

import (
	"strconv"
	"testing"
	"time"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"

	// cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
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
	"github.com/stretchr/testify/require"
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
	msgServer       types.MsgServer
	key             *storetypes.KVStoreKey
	addrs           []sdk.AccAddress
	addrsStr        []string
}

func (s *InferenceSynthesisTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		"bonded_tokens_pool":                             {"burner", "staking"},
		"not_bonded_tokens_pool":                         {"burner", "staking"},
		multiPerm:                                        {"burner", "minter", "staking"},
		randomPerm:                                       {"random"},
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

	var addrs []sdk.AccAddress = make([]sdk.AccAddress, 0)
	var addrsStr []string = make([]string, 0)
	pubkeys := simtestutil.CreateTestPubKeys(5)
	for i := 0; i < 5; i++ {
		addrs = append(addrs, sdk.AccAddress(pubkeys[i].Address()))
		addrsStr = append(addrsStr, addrs[i].String())
	}
	s.addrs = addrs
	s.addrsStr = addrsStr

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
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
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrsStr {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(InferenceSynthesisTestSuite))
}

func (s *InferenceSynthesisTestSuite) getEpoch3ValueBundle() (
	*inferencesynthesis.NetworkInferenceBuilder,
	map[int]func(header string) alloraMath.Dec,
) {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeight := int64(1)

	epochGet := GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[2]
	epoch3Get := epochGet[3]

	// SET EPOCH 2 VALUES IN THE SYSTEM
	infererNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			"worker0": epoch2Get("inference_regret_worker_0"),
			"worker1": epoch2Get("inference_regret_worker_1"),
			"worker2": epoch2Get("inference_regret_worker_2"),
			"worker3": epoch2Get("inference_regret_worker_3"),
			"worker4": epoch2Get("inference_regret_worker_4"),
		}

	// Set inferer network regrets
	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"forecaster0": epoch2Get("inference_regret_worker_5"),
		"forecaster1": epoch2Get("inference_regret_worker_6"),
		"forecaster2": epoch2Get("inference_regret_worker_7"),
	}

	// Set forecaster network regrets
	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set one-in network regrets
	setOneInForecasterNetworkRegret := func(forecasterIndex int, infererIndex int, epochGetter func(string) alloraMath.Dec) {
		forecasterName := "forecaster" + strconv.Itoa(forecasterIndex)
		infererName := "worker" + strconv.Itoa(infererIndex)
		headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_onein_" + strconv.Itoa(forecasterIndex)

		k.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecasterName,
			infererName,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       epochGetter(headerName),
			},
		)
	}

	// Set one-in self network regrets
	setOneInForecasterSelfRegret := func(forecaster int, epochGetter func(string) alloraMath.Dec) {
		forecasterName := "forecaster" + strconv.Itoa(forecaster)
		headerName := "inference_regret_worker_5_onein_" + strconv.Itoa(forecaster)

		k.SetOneInForecasterSelfNetworkRegret(
			s.ctx,
			topicId,
			forecasterName,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       epochGetter(headerName),
			},
		)
	}

	for forecaster := 0; forecaster < 3; forecaster++ {
		setOneInForecasterSelfRegret(forecaster, epoch2Get)
		for inferer := 0; inferer < 5; inferer++ {
			setOneInForecasterNetworkRegret(forecaster, inferer, epoch2Get)
		}
	}

	for workerIndex := 0; workerIndex < 5; workerIndex++ {
		stakeValue := epoch3Get("reputer_stake_" + strconv.Itoa(workerIndex))

		stakeValueScaled, err := stakeValue.Mul(alloraMath.MustNewDecFromString("1e18"))
		require.NoError(s.T(), err)

		stakeValueFloored, err := stakeValueScaled.Floor()
		require.NoError(s.T(), err)

		stakeInt, ok := cosmosMath.NewIntFromString(stakeValueFloored.String())
		s.Require().True(ok)

		workerString := "worker" + strconv.Itoa(workerIndex)
		err = k.AddStake(s.ctx, topicId, workerString, stakeInt)
		require.NoError(s.T(), err)
	}

	// SET EPOCH 3 VALUES IN VALUE BUNDLE
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{},
	}

	for infererIndex := 0; infererIndex < 5; infererIndex++ {
		inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
			Inferer: "worker" + strconv.Itoa(infererIndex),
			Value:   epoch3Get("inference_" + strconv.Itoa(infererIndex)),
		})
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{},
	}

	for forecasterIndex := 0; forecasterIndex < 3; forecasterIndex++ {
		forecastElements := make([]*emissionstypes.ForecastElement, 0)
		for infererIndex := 0; infererIndex < 5; infererIndex++ {
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: "worker" + strconv.Itoa(infererIndex),
				Value:   epoch3Get("forecasted_loss_" + strconv.Itoa(forecasterIndex) + "_for_" + strconv.Itoa(infererIndex)),
			})
		}
		forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
			Forecaster:       "forecaster" + strconv.Itoa(forecasterIndex),
			ForecastElements: forecastElements,
		})
	}

	networkInferenceBuilder, err := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: epoch2Get("network_loss"),
			Epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			FTolerance:          alloraMath.MustNewDecFromString("0.01"),
			PNorm:               alloraMath.MustNewDecFromString("3.0"),
			CNorm:               alloraMath.MustNewDecFromString("0.75"),
		},
	)
	require.NoError(s.T(), err)

	return networkInferenceBuilder, epochGet
}

func (s *InferenceSynthesisTestSuite) TestCorrectCombinedValue() {

	networkInferenceBuilder, epochGet := s.getEpoch3ValueBundle()
	valueBundle := networkInferenceBuilder.SetCombinedValue().Build()
	s.Require().NotNil(valueBundle.CombinedValue)
	expected := epochGet[3]("network_inference")
	actual := valueBundle.CombinedValue
	alloratestutil.InEpsilon2(s.T(), actual, expected.String())
}

func (s *InferenceSynthesisTestSuite) TestCorrectNaiveValue() {

	networkInferenceBuilder, epochGet := s.getEpoch3ValueBundle()
	valueBundle := networkInferenceBuilder.SetNaiveValue().Build()
	alloratestutil.InEpsilon2(s.T(), valueBundle.NaiveValue, epochGet[3]("network_naive_inference").String())
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutInfererValues() {

	networkInferenceBuilder, epochGet := s.getEpoch3ValueBundle()

	expectedValues := map[string]alloraMath.Dec{
		"worker0": epochGet[3]("network_inference_oneout_0"),
		"worker1": epochGet[3]("network_inference_oneout_1"),
		"worker2": epochGet[3]("network_inference_oneout_2"),
		"worker3": epochGet[3]("network_inference_oneout_3"),
		"worker4": epochGet[3]("network_inference_oneout_4"),
	}

	valueBundle := networkInferenceBuilder.SetOneOutInfererValues().Build()

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range valueBundle.OneOutInfererValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon2(s.T(), expectedValue, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneOutForecasterValues() {
	networkInferenceBuilder, epochGet := s.getEpoch3ValueBundle()
	valueBundle := networkInferenceBuilder.SetOneOutForecasterValues().Build()

	expectedValues := map[string]alloraMath.Dec{
		"forecaster0": epochGet[3]("network_inference_oneout_5"),
		"forecaster1": epochGet[3]("network_inference_oneout_6"),
		"forecaster2": epochGet[3]("network_inference_oneout_7"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range valueBundle.OneOutForecasterValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon2(s.T(), expectedValue, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCorrectOneInForecasterValues() {
	networkInferenceBuilder, epochGet := s.getEpoch3ValueBundle()
	valueBundle := networkInferenceBuilder.SetOneInValues().Build()

	expectedValues := map[string]alloraMath.Dec{
		"forecaster0": epochGet[3]("network_naive_inference_onein_0"),
		"forecaster1": epochGet[3]("network_naive_inference_onein_1"),
		"forecaster2": epochGet[3]("network_naive_inference_onein_2"),
	}

	for worker, expectedValue := range expectedValues {
		found := false
		for _, workerAttributedValue := range valueBundle.OneInForecasterValues {
			if workerAttributedValue.Worker == worker {
				found = true
				alloratestutil.InEpsilon2(s.T(), expectedValue, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestBuildNetworkInferencesIncompleteData() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := "worker1"
	worker2 := "worker2"

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

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function without setting regrets
	networkInferenceBuilder, err := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             epsilon,
			FTolerance:          fTolerance,
			PNorm:               pNorm,
			CNorm:               cNorm,
		},
	)
	s.Require().NoError(err)
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

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

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	// fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

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

	networkInferenceBuilder, err := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             epsilon,
			PNorm:               pNorm,
			CNorm:               cNorm,
		},
	)
	s.Require().NoError(err)
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

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

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	forecaster1 := "forecaster1"
	forecaster2 := "forecaster2"
	forecaster3 := "forecaster3"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
			{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.6")},
				},
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.7")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
			{
				Forecaster: forecaster3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.3")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.7")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.8")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.9")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	networkInferenceBuilder, err := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             epsilon,
			FTolerance:          fTolerance,
			PNorm:               pNorm,
			CNorm:               cNorm,
		},
	)
	s.Require().NoError(err)
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

package inference_synthesis_test

import (
	slog "log"
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
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inferencesynthesis"
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
	*emissionstypes.ValueBundle,
	map[int]func(header string) alloraMath.Dec,
) {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeight := int64(1)

	epochGet := GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[2]
	epoch3Get := epochGet[2]

	// SET EPOCH 2 VALUES IN THE SYSTEM
	infererNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			"worker0": epoch2Get("inference_regret_worker_0"),
			"worker1": epoch2Get("inference_regret_worker_1"),
			"worker2": epoch2Get("inference_regret_worker_2"),
			"worker3": epoch2Get("inference_regret_worker_3"),
			"worker4": epoch2Get("inference_regret_worker_4"),
		}

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

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

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
		stakeValue := epoch2Get("reputer_stake_" + strconv.Itoa(workerIndex))
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

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: epoch3Get("network_loss"),
			Epsilon:             alloraMath.MustNewDecFromString("0.001"),
			PNorm:               alloraMath.MustNewDecFromString("0.75"),
			CNorm:               alloraMath.MustNewDecFromString("3.0"),
		},
	)
	return networkInferenceBuilder.CalcAndSetNetworkInferences().Build(), epochGet
}

func (s *InferenceSynthesisTestSuite) TestCorrectCombinedValue() {

	valueBundle, epochGet := s.getEpoch3ValueBundle()
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	alloratestutil.InEpsilon2(s.T(), epochGet[3]("network_inference"), valueBundle.CombinedValue.String())
}

func (s *InferenceSynthesisTestSuite) TestBuildNetworkInferences() {
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
	epsilon := alloraMath.MustNewDecFromString("0.001")
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

	// Call the function

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
}

func (s *InferenceSynthesisTestSuite) TestBuildNetworkInferencesSameInfererForecasters() {
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
	epsilon := alloraMath.MustNewDecFromString("0.001")
	// fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// s.Require().NotEmpty(valueBundle.OneInForecasterValues)

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	networkInferenceBuilder = inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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
	valueBundle = networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
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
	// fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function without setting regrets
	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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
	// fTolerance := alloraMath.MustNewDecFromString("0.01")
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

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
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
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

/*
func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation1() {
	topicId := inferencesynthesis.TopicId(1)
	ctx := s.ctx
	k := s.emissionsKeeper

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
			{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
			{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
		},
	}


	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
	}
	maxRegret := alloraMath.MustNewDecFromString("0.9871536722074480")

	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
		"worker1": alloraMath.MustNewDecFromString("0.910174442412618"),
		"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
		"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
		"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.06470631905627390")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkInferecesBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 ctx,
			K:                   k,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             moduleParams.Epsilon,
			PNorm:               topic.PNorm,
			CNorm:               moduleParams.CNorm,
		},
	)

	networkCombinedInferenceValue, err := inferencesynthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		NewWorkersAreNew(false),
		maxRegret,
		epsilon,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.00001"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation2() {
	topicId := inferencesynthesis.TopicId(1)

	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.14361768314408600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.23422685055675900")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.18201270373970600")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.19840891048468800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.19696044261177800")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.20289734770434400")},
	}
	maxRegret := alloraMath.MustNewDecFromString("0.9737035757621540")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.NewDecFromInt64(2)
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.5576393860961080"),
		"worker1": alloraMath.MustNewDecFromString("0.8588215562008240"),
		"worker2": alloraMath.MustNewDecFromString("0.9737035757621540"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.7535724745797420"),
		"worker4": alloraMath.MustNewDecFromString("0.7658774622830770"),
		"worker5": alloraMath.MustNewDecFromString("0.7185104293863190"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.19486643996868")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkCombinedInferenceValue, err := inferencesynthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		NewWorkersAreNew(false),
		maxRegret,
		epsilon,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.00001"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}
*/

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferencesMultipleWorkers() {
	topicId := inferencesynthesis.TopicId(1)

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
			{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
			{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
		},
	}

	/*
		forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
			"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
			"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
			"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
		}
	*/
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker3",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011708024633613200")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.013382222402411400")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3.82471429104471e-05")},
				},
			},
			{
				Forecaster: "worker4",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011486217283808300")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0060528036329761000")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0005337255825785730")},
				},
			},
			{
				Forecaster: "worker5",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.001810780808278390")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0018544539679880700")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.001251454152216520")},
				},
			},
		},
	}
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
		"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
		"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
	}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
		"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
		"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
	}
	// maxRegret := alloraMath.MustNewDecFromString("0.987153672207448")
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.0156937658327922")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	// fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 s.ctx,
			K:                   s.emissionsKeeper,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             epsilon,
			PNorm:               pNorm,
			CNorm:               cNorm,
		},
	)
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	slog.Printf("valueBundle: %+v", valueBundle)

	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0711130346780"},
		{Worker: "worker1", Value: "-0.077954217717"},
		{Worker: "worker2", Value: "-0.0423024599518"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker3", Value: "-0.06351714496"},
		{Worker: "worker4", Value: "-0.06471822091"},
		{Worker: "worker5", Value: "-0.06495348528"},
	}

	oneOutInfererValues := valueBundle.OneOutInfererValues
	oneOutForecasterValues := valueBundle.OneOutForecasterValues

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
		found := false
		for _, oneOutInference := range oneOutInfererValues {
			if expected.Worker == oneOutInference.Worker {
				found = true
				alloratestutil.InEpsilon2(s.T(), oneOutInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}

	for _, expected := range expectedOneOutImpliedInferences {
		found := false
		for _, oneOutImpliedInference := range oneOutForecasterValues {
			if expected.Worker == oneOutImpliedInference.Worker {
				found = true
				alloratestutil.InEpsilon3(s.T(), oneOutImpliedInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferences5Workers3Forecasters() {
	topicId := inferencesynthesis.TopicId(1)

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
			{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
			{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
			{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
			{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
		},
	}

	/*
		forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
			"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.08944493117005920")},
			"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.07333218290300560")},
			"forecaster2": {Value: alloraMath.MustNewDecFromString("-0.07756206109376570")},
		}
	*/

	// epoch 3
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.003305466418410120")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002788248228566030")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(".0000240536828602367")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0008240378476798250")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.0000186192181193532")},
				},
			},
			{
				Forecaster: "forecaster1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.002308441286328890")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0000214380788596749")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.012560171044167200")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.017998563880697900")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00020024906252089700")},
				},
			},
			{
				Forecaster: "forecaster2",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.005369218152594270")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002578158768320300")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0076008583603885900")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0076269073955871000")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00035670236460009500")},
				},
			},
		},
	}
	// epoch 2
	infererNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			"worker0": alloraMath.MustNewDecFromString("0.29240710390153500"),
			"worker1": alloraMath.MustNewDecFromString("0.4182220944854450"),
			"worker2": alloraMath.MustNewDecFromString("0.17663501719135000"),
			"worker3": alloraMath.MustNewDecFromString("0.49617463489106400"),
			"worker4": alloraMath.MustNewDecFromString("0.27996060999688600"),
		}
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		"forecaster0": alloraMath.MustNewDecFromString("0.816066375505268"),
		"forecaster1": alloraMath.MustNewDecFromString("0.8234558901838660"),
		"forecaster2": alloraMath.MustNewDecFromString("0.8196673550408280"),
	}
	// maxRegret := alloraMath.MustNewDecFromString("0.8234558901838660")

	// epoch 2
	networkCombinedLoss := alloraMath.MustNewDecFromString(".0000127791308799785")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
		inferencesynthesis.SynthRequest{
			Ctx:                 s.ctx,
			K:                   s.emissionsKeeper,
			TopicId:             topicId,
			Inferences:          &inferences,
			Forecasts:           forecasts,
			NetworkCombinedLoss: networkCombinedLoss,
			Epsilon:             epsilon,
			PNorm:               pNorm,
			CNorm:               cNorm,
		},
	)
	valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0878179883784"},
		{Worker: "worker1", Value: "-0.0834415833800"},
		{Worker: "worker2", Value: "-0.0760530852479"},
		{Worker: "worker3", Value: "-0.0769113408092"},
		{Worker: "worker4", Value: "-0.0977096283034"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "forecaster0", Value: "-0.0847805342051"},
		{Worker: "forecaster1", Value: "-0.0882088249132"},
		{Worker: "forecaster2", Value: "-0.0872998460256"},
	}

	oneOutInfererValues := valueBundle.OneOutInfererValues
	oneOutForecasterValues := valueBundle.OneOutForecasterValues

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
		found := false
		for _, oneOutInference := range oneOutInfererValues {
			if expected.Worker == oneOutInference.Worker {
				found = true
				alloratestutil.InEpsilon2(s.T(), oneOutInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}

	for _, expected := range expectedOneOutImpliedInferences {
		found := false
		for _, oneOutImpliedInference := range oneOutForecasterValues {
			if expected.Worker == oneOutImpliedInference.Worker {
				found = true
				alloratestutil.InEpsilon3(s.T(), oneOutImpliedInference.Value, expected.Value)
			}
		}
		if !found {
			s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
		}
	}
}

/*
func (s *InferenceSynthesisTestSuite) TestCalcOneInInferences() {
	topicId := inferencesynthesis.TopicId(1)

	tests := []struct {
		name                        string
		inferences                  emissionstypes.Inferences
		forecastImpliedInferences   map[string]*emissionstypes.Inference
		maxRegretsByOneInForecaster map[string]inferencesynthesis.Regret
		epsilon                     alloraMath.Dec
		pNorm                       alloraMath.Dec
		cNorm                       alloraMath.Dec
		infererNetworkRegrets       map[string]inferencesynthesis.Regret
		forecasterNetworkRegrets    map[string]inferencesynthesis.Regret
		expectedOneInInferences     []*emissionstypes.WorkerAttributedValue
		expectedErr                 error
	}{
		{ // EPOCH 3
			name: "basic functionality",
			inferences: emissionstypes.Inferences{
				Inferences: []*emissionstypes.Inference{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
				},
			},
			forecastImpliedInferences: map[string]*emissionstypes.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
			},
			maxRegretsByOneInForecaster: map[string]inferencesynthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker4": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker5": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			epsilon: alloraMath.MustNewDecFromString("0.0001"),
			pNorm:   alloraMath.MustNewDecFromString("2.0"),
			cNorm:   alloraMath.MustNewDecFromString("0.75"),
			infererNetworkRegrets: map[string]inferencesynthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
				"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
				"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			forecasterNetworkRegrets: map[string]inferencesynthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
				"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
				"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
			},
			expectedOneInInferences: []*emissionstypes.WorkerAttributedValue{
				{Worker: "worker3", Value: alloraMath.MustNewDecFromString("-0.06502630286365970")},
				{Worker: "worker4", Value: alloraMath.MustNewDecFromString("-0.06356081320547800")},
				{Worker: "worker5", Value: alloraMath.MustNewDecFromString("-0.06325114823960220")},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			for inferer, regret := range tc.infererNetworkRegrets {
				s.emissionsKeeper.SetInfererNetworkRegret(
					s.ctx,
					topicId,
					inferer,
					emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			for forecaster, regret := range tc.forecasterNetworkRegrets {
				s.emissionsKeeper.SetForecasterNetworkRegret(
					s.ctx,
					topicId,
					forecaster,
					emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			networkInferenceBuilder := inferencesynthesis.NewNetworkInferenceBuilderFromSynthRequest(
				inferencesynthesis.SynthRequest{
					Ctx:                 s.ctx,
					K:                   s.emissionsKeeper,
					TopicId:             topicId,
					Inferences:          &tc.inferences,
					Forecasts:           tc.forecasts,
					NetworkCombinedLoss: tc.networkCombinedLoss,
					Epsilon:             tc.epsilon,
					PNorm:               tc.pNorm,
					CNorm:               tc.cNorm,
				},
			)
			valueBundle := networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

			oneInInferences, err := inferencesynthesis.CalcOneInInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				alloraMath.GetSortedKeys(tc.inferenceByWorker),
				tc.forecastImpliedInferences,
				alloraMath.GetSortedKeys(tc.forecastImpliedInferences),
				NewWorkersAreNew(false),
				tc.maxRegretsByOneInForecaster,
				tc.epsilon,
				tc.pNorm,
				tc.cNorm,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().Len(oneInInferences, len(tc.expectedOneInInferences), "Unexpected number of one-in inferences")

				for _, expected := range tc.expectedOneInInferences {
					found := false
					for _, actual := range oneInInferences {
						if expected.Worker == actual.Worker {
							s.Require().True(
								alloraMath.InDelta(
									expected.Value,
									actual.Value,
									alloraMath.MustNewDecFromString("0.0001"),
								),
								"Mismatch in value for one-in inference of worker %s, expected %v, actual %v",
								expected.Worker,
								expected.Value,
								actual.Value,
							)
							found = true
							break
						}
					}
					if !found {
						s.FailNow("Matching worker not found", "Worker %s not found in actual inferences", expected.Worker)
					}
				}
			}
		})
	}
}

*/

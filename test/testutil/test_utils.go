package testutil

import (
	"errors"
	"math/rand"
	"slices"
	"strconv"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/suite"

	"github.com/allora-network/allora-chain/app/params"
	alloralog "github.com/allora-network/allora-chain/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/fn"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/keeper/queryserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

type TestSuite struct {
	suite.Suite
	ModuleName string

	ctx                   sdk.Context
	codec                 codec.Codec
	storeServiceBank      store.KVStoreService
	storeServiceEmissions store.KVStoreService
	accountKeeper         authkeeper.AccountKeeper
	bankKeeper            bankkeeper.BaseKeeper
	emissionsKeeper       *keeper.Keeper
	mintKeeper            minttypes.MintKeeper
	stakingKeeper         minttypes.StakingKeeper
	emissionsAppModule    module.AppModule
	emissionsQueryServer  types.QueryServiceServer
	mintAppModule         mint.AppModule
	emissionsMsgServer    types.MsgServiceServer
	key                   *storetypes.KVStoreKey
	privKeys              []secp256k1.PrivKey
	addrs                 []sdk.AccAddress
	addrsStr              []string
	pubKeyHexStr          []string
}

func (s *TestSuite) Ctx() sdk.Context {
	return s.ctx
}

func (s *TestSuite) Addrs() []sdk.AccAddress {
	return s.addrs
}

func (s *TestSuite) AddrsStr() []string {
	return s.addrsStr
}

func (s *TestSuite) PubKeyHexStr() []string {
	return s.pubKeyHexStr
}

func (s *TestSuite) PrivKeys() []secp256k1.PrivKey {
	return s.privKeys
}

func (s *TestSuite) EmissionsKeeper() *keeper.Keeper {
	return s.emissionsKeeper
}

func (s *TestSuite) AccountKeeper() authkeeper.AccountKeeper {
	return s.accountKeeper
}

func (s *TestSuite) BankKeeper() keeper.BankKeeper {
	return s.bankKeeper
}

func (s *TestSuite) MintKeeper() minttypes.MintKeeper {
	return s.mintKeeper
}

func (s *TestSuite) StakingKeeper() minttypes.StakingKeeper {
	return s.stakingKeeper
}

func (s *TestSuite) EmissionsAppModule() module.AppModule {
	return s.emissionsAppModule
}

func (s *TestSuite) EmissionsMsgServer() types.MsgServiceServer {
	return s.emissionsMsgServer
}

func (s *TestSuite) EmissionsQueryServer() types.QueryServiceServer {
	return s.emissionsQueryServer
}

func (s *TestSuite) Codec() codec.Codec {
	return s.codec
}

func (s *TestSuite) StoreServiceEmissions() store.KVStoreService {
	return s.storeServiceEmissions
}

func (s *TestSuite) StoreServiceBank() store.KVStoreService {
	return s.storeServiceBank
}

func (s *TestSuite) WithBlockHeight(height int64) {
	s.ctx = s.ctx.WithBlockHeight(height)
}

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type (
	Option       func(s *customParams)
	customParams struct {
		block       int64
		topicID     uint64
		alphaRegret alloraMath.Dec
		epochLength,
		groundTruthLag,
		workerSubmissionWindow,
		epochLastEnded int64
		lossMethod,
		initialRegret string
		workerValues  []TestWorkerValue
		reputerValues []TestReputerValue
		reputerStake  *cosmosMath.Int
	}
)

func WithBlock(block int64) Option {
	return func(s *customParams) {
		s.block = block
	}
}

func WithTopicID(topicID uint64) Option {
	return func(s *customParams) {
		s.topicID = topicID
	}
}

func WithAlphaRegret(alphaRegret alloraMath.Dec) Option {
	return func(s *customParams) {
		s.alphaRegret = alphaRegret
	}
}

func WithEpochLength(epochLength int64) Option {
	return func(s *customParams) {
		s.epochLength = epochLength
	}
}

func WithGroundTruthLag(lag int64) Option {
	return func(s *customParams) {
		s.groundTruthLag = lag
	}
}

func WithWorkerSubmissionWindow(window int64) Option {
	return func(s *customParams) {
		s.workerSubmissionWindow = window
	}
}

func WithWorkerValues(workerValues []TestWorkerValue) Option {
	return func(s *customParams) {
		s.workerValues = workerValues
	}
}

func WithReputerValues(reputerValues []TestReputerValue) Option {
	return func(s *customParams) {
		s.reputerValues = reputerValues
	}
}

func WithReputerStake(reputerStake *cosmosMath.Int) Option {
	return func(s *customParams) {
		s.reputerStake = reputerStake
	}
}

func WithEpochLastEnded(epochLastEnded int64) Option {
	return func(s *customParams) {
		s.epochLastEnded = epochLastEnded
	}
}

func WithLossMethod(lossMethod string) Option {
	return func(s *customParams) {
		s.lossMethod = lossMethod
	}
}

func WithInitialRegret(initialRegret string) Option {
	return func(s *customParams) {
		s.initialRegret = initialRegret
	}
}

func (s *TestSuite) SetupTest() {
	var (
		keyEmissions        = storetypes.NewKVStoreKey("emissions")
		keyAccount          = storetypes.NewKVStoreKey("account")
		keyBank             = storetypes.NewKVStoreKey("bank")
		keyStaking          = storetypes.NewKVStoreKey("staking")
		keyMint             = storetypes.NewKVStoreKey("mint")
		storeServiceAccount = runtime.NewKVStoreService(keyAccount)
		storeServiceStaking = runtime.NewKVStoreService(keyStaking)
		storeServiceMint    = runtime.NewKVStoreService(keyMint)
	)
	s.storeServiceEmissions = runtime.NewKVStoreService(keyEmissions)
	s.storeServiceBank = runtime.NewKVStoreService(keyBank)
	testCtx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{
		"emissions": keyEmissions,
		"account":   keyAccount,
		"bank":      keyBank,
		"staking":   keyStaking,
		"mint":      keyMint,
	}, map[string]*storetypes.TransientStoreKey{
		"transient_test": storetypes.NewTransientStoreKey("transient_test"),
	}, nil).WithHeaderInfo(header.Info{Time: time.Now()})

	// Set logger to show logs from the module too
	logger := alloralog.NewTestLogger(s.T()).With("module", s.ModuleName)
	ctx := testCtx.WithHeaderInfo(header.Info{
		Height:  1,
		Hash:    []byte("1"),
		AppHash: []byte("1"),
		ChainID: "localnet",
		Time:    time.Now(),
	}).WithLogger(logger)
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	s.codec = encCfg.Codec

	maccPerms := map[string][]string{
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		"ecosystem":              {"minter"},
		"bonded_tokens_pool":     {"burner", "staking"},
		"not_bonded_tokens_pool": {"burner", "staking"},
		multiPerm:                {"burner", "minter", "staking"},
		randomPerm:               {"random"},
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
		s.storeServiceBank,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	emissionsKeeper := keeper.NewKeeper(
		encCfg.Codec,
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		s.storeServiceEmissions,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)
	stakingKeeper := stakingkeeper.NewKeeper(
		encCfg.Codec,
		storeServiceStaking,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress("gov").String(),
		codecAddress.NewBech32Codec(sdk.Bech32PrefixValAddr),
		codecAddress.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)
	mintKeeper := mintkeeper.NewKeeper(
		encCfg.Codec,
		storeServiceMint,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = &emissionsKeeper
	s.mintKeeper = mintKeeper
	s.stakingKeeper = stakingKeeper
	emissionsAppModule := module.NewAppModule(encCfg.Codec, emissionsKeeper)
	defaultEmissionsGenesis := emissionsAppModule.DefaultGenesis(encCfg.Codec)
	emissionsAppModule.InitGenesis(ctx, encCfg.Codec, defaultEmissionsGenesis)
	s.emissionsMsgServer = msgserver.NewMsgServerImpl(emissionsKeeper)
	s.emissionsQueryServer = queryserver.NewQueryServerImpl(*s.emissionsKeeper)
	s.emissionsAppModule = emissionsAppModule

	// Fund the rewards account generously
	s.FundAccount(10000000000, s.accountKeeper.GetModuleAddress(types.AlloraRewardsAccountName))

	s.privKeys, s.pubKeyHexStr, s.addrs, s.addrsStr = GenerateTestAccounts(50)
	for _, addr := range s.addrs {
		s.FundAccount(10000000000, addr)
	}

	// Add all tests addresses in whitelists
	for _, addr := range s.addrsStr {
		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.Require().NoError(err)

		err = s.emissionsKeeper.AddToTopicCreatorWhitelist(ctx, addr)
		s.Require().NoError(err)

		err = s.emissionsKeeper.AddToGlobalWhitelist(ctx, addr)
		s.Require().NoError(err)
	}

	// create first topic
	s.CreateTopic(WithEpochLength(100), WithGroundTruthLag(100), WithWorkerSubmissionWindow(100))
}

func (s *TestSuite) SetParamsForTest() {
	// Setup a sender address
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	err := s.emissionsKeeper.AddWhitelistAdmin(s.Ctx(), adminAddr.String())
	s.Require().NoError(err)

	newParams := &types.OptionalParams{
		MaxTopInferersToReward:  []uint64{24},
		MinEpochLength:          []int64{1},
		RegistrationFee:         []cosmosMath.Int{cosmosMath.NewInt(6)},
		MaxActiveTopicsPerBlock: []uint64{2},
		BlocksPerMonth:          []uint64{864000},
		// Exaggerated TopicRewardAlpha to compensate the effect of latest topic reward alpha vs
		// the dripping effect and separate epochs running multiple topics.
		TopicRewardAlpha: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.999375")},
	}

	updateMsg := &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	response, err := s.emissionsMsgServer.UpdateParams(s.Ctx(), updateMsg)
	s.Require().NoError(err)
	s.Require().NotNil(response)
}

func (s *TestSuite) FundAccount(amount int64, accAddress sdk.AccAddress) {
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(amount)))
	err := s.bankKeeper.MintCoins(s.Ctx(), types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, accAddress, initialStakeCoins)
	s.Require().NoError(err)
}

func (s *TestSuite) FundTopic(topicId uint64, funderAddr sdk.AccAddress, amount cosmosMath.Int) {
	s.MintTokensToAddress(funderAddr, amount)
	fundTopicMessage := types.FundTopicRequest{
		Sender:  funderAddr.String(),
		TopicId: topicId,
		Amount:  amount,
	}
	response, err := s.emissionsMsgServer.FundTopic(s.Ctx(), &fundTopicMessage)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response, "Response should not be nil")
}

func (s *TestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	err := s.bankKeeper.MintCoins(s.Ctx(), types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func (s *TestSuite) MintTokensToModule(moduleName string, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	err := s.bankKeeper.MintCoins(s.Ctx(), moduleName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func DecPtr(s string) *alloraMath.Dec {
	d := alloraMath.MustNewDecFromString(s)
	return &d
}

func GetWorkerValuesFromIndexes(indexes []int, value ...string) []TestWorkerValue {
	values := make([]TestWorkerValue, 0)
	for i, index := range indexes {
		values = append(values, TestWorkerValue{Index: index, Value: value[i%len(value)]})
	}
	return values
}

func (s *TestSuite) GetReputerValuesFromIndexes(reputerIndexes, workerIndexes []int, value ...string) []TestReputerValue {
	if len(value) == 0 {
		panic("value is empty")
	}
	addrs := make([]string, len(workerIndexes))
	for i, index := range workerIndexes {
		addrs[i] = s.addrsStr[index]
	}
	slices.Sort(addrs)
	values := make([]TestReputerValue, len(reputerIndexes))
	for i, repIdx := range reputerIndexes {
		for j, wrkIdx := range workerIndexes {
			if values[i].WorkerValues == nil {
				values[i].WorkerValues = make(map[string]string)
			}
			values[i].WorkerValues[addrs[wrkIdx%len(addrs)]] = value[j%len(value)]
			values[i].CombinedValue = value[repIdx%len(value)]
		}
	}
	return values
}

type TestWorkerValue struct {
	Index int
	Value string
}

func generateWorkerDataBundles(s *TestSuite, nonce int64, topicId uint64, workerIndexes []int, workerValues []TestWorkerValue) []*types.InputWorkerDataBundle {
	lwv := len(workerValues)
	hasWorkerValues := lwv > 0
	if hasWorkerValues && len(workerIndexes) != lwv {
		panic("invalid worker values length")
	}
	var bundles []*types.InputWorkerDataBundle
	totalAddresses := len(s.addrsStr)

	for i, workerIdx := range workerIndexes {
		// Generate random inference value between 0.1 and 0.25
		rand.Seed(int64(workerIdx) + nonce)
		inferenceValueStr := strconv.FormatFloat(0.1+rand.Float64()*0.15, 'f', 5, 64)
		if hasWorkerValues {
			inferenceValueStr = workerValues[i].Value
		}

		// Select forecast targets (next two workers in sequence, wrapping if needed)
		forecastTargets := []int{
			(workerIdx + 0) % totalAddresses,
			(workerIdx + 1) % totalAddresses,
			(workerIdx + 2) % totalAddresses,
		}

		// Create forecast elements
		forecastElements := []*types.InputForecastElement{
			{
				Inferer: s.addrs[forecastTargets[0]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
			{
				Inferer: s.addrs[forecastTargets[1]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
			{
				Inferer: s.addrs[forecastTargets[2]].String(),
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
			},
		}

		// Create inference-forecast bundle
		inferenceForecastBundle := &types.InputInferenceForecastBundle{
			Inference: &types.InputInference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     s.addrsStr[workerIdx],
				Value:       alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(inferenceValueStr)),
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &types.InputForecast{
				TopicId:          topicId,
				BlockHeight:      nonce,
				Forecaster:       s.addrsStr[workerIdx],
				ForecastElements: forecastElements,
				ExtraData:        nil,
			},
		}

		// Sign the bundle
		inputBundle, err := types.NewInferenceForecastBundleFromInput(inferenceForecastBundle)
		s.Require().NoError(err)
		signature, err := signInferenceForecastBundle(inputBundle, s.privKeys[workerIdx])
		s.Require().NoError(err)

		// Create the complete worker data bundle
		bundle := &types.InputWorkerDataBundle{
			Worker:                             s.addrsStr[workerIdx],
			Nonce:                              &types.Nonce{BlockHeight: nonce},
			TopicId:                            topicId,
			InferenceForecastsBundle:           inferenceForecastBundle,
			InferencesForecastsBundleSignature: signature,
			Pubkey:                             s.pubKeyHexStr[workerIdx],
		}

		bundles = append(bundles, bundle)
	}

	return bundles
}

func (s *TestSuite) signInputValueBundle(InputValueBundle *types.InputValueBundle, privateKey secp256k1.PrivKey) []byte {
	valueBundle, err := types.NewValueBundleFromInput(InputValueBundle)
	s.Require().NoError(err)
	return s.SignValueBundle(valueBundle, privateKey)
}

func signInferenceForecastBundle(
	inferenceForecastBundle *types.InferenceForecastBundle,
	privateKey secp256k1.PrivKey,
) ([]byte, error) {
	src := make([]byte, 0)
	src, err := inferenceForecastBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil, err
	}

	sig, err := privateKey.Sign(src)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

type TestReputerValue struct {
	CombinedValue   string
	WorkerValues    map[string]string
	OneOutInfValues map[string]string
}

func (s *TestSuite) generateLossBundles(
	nonce int64,
	topicId uint64,
	reputerIndexes []int,
	opts ...Option,
) (reputerValueBundles types.InputReputerValueBundles) {
	p := &customParams{}
	for _, opt := range opts {
		opt(p)
	}
	rv := len(p.reputerValues)
	hasReputerValues := rv > 0
	if hasReputerValues && len(reputerIndexes) != rv {
		panic("invalid reputer values length")
	}

	networkInferences, err := s.emissionsKeeper.GetLatestNetworkInferences(s.Ctx(), topicId, false)
	s.Require().NoError(err)

	if networkInferences == nil || len(networkInferences.InfererValues) == 0 {
		i := 0
		for range p.reputerValues[0].WorkerValues {
			networkInferences.InfererValues = append(networkInferences.InfererValues, &types.WorkerAttributedValue{
				Worker: s.addrsStr[i],
			})
			i++
		}
	}

	deltaVal := alloraMath.MustNewDecFromString("0.01988")

	for i, reputerIndex := range reputerIndexes {
		var val alloraMath.Dec
		combinedVal := alloraMath.MustNewDecFromString("0.1")
		if hasReputerValues {
			combinedVal = alloraMath.MustNewDecFromString(p.reputerValues[i].CombinedValue)
		}

		valueBundle := &types.InputValueBundle{
			TopicId: topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: nonce,
				},
			},
			Reputer:       s.addrsStr[reputerIndex],
			ExtraData:     nil,
			CombinedValue: alloraMath.MustNewBoundedExp40Dec(combinedVal),
			NaiveValue:    alloraMath.MustNewBoundedExp40Dec(combinedVal),
			InfererValues: fn.Map(networkInferences.InfererValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(p.reputerValues[i].WorkerValues[inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			ForecasterValues: fn.Map(networkInferences.ForecasterValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(p.reputerValues[i].WorkerValues[inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutInfererValues: fn.Map(networkInferences.OneOutInfererValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					valStr := p.reputerValues[i].WorkerValues[inf.Worker]
					if v, ok := p.reputerValues[i].OneOutInfValues[inf.Worker]; ok {
						valStr = v
					}
					val = alloraMath.MustNewDecFromString(valStr)
				}
				return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutForecasterValues: fn.Map(networkInferences.OneOutForecasterValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(p.reputerValues[i].WorkerValues[inf.Worker])
				}
				return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneInForecasterValues: fn.Map(networkInferences.OneInForecasterValues, func(inf *types.WorkerAttributedValue) *types.InputWorkerAttributedValue {
				val, _ = inf.Value.Sub(deltaVal)
				if hasReputerValues {
					val = alloraMath.MustNewDecFromString(p.reputerValues[i].WorkerValues[inf.Worker])
				}
				return &types.InputWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewBoundedExp40Dec(val)}
			}),
			OneOutInfererForecasterValues: fn.Map(networkInferences.OneOutInfererForecasterValues, func(inf *types.OneOutInfererForecasterValues) *types.InputOneOutInfererForecasterValues {
				return &types.InputOneOutInfererForecasterValues{Forecaster: inf.Forecaster, OneOutInfererValues: fn.Map(inf.OneOutInfererValues, func(inf *types.WithheldWorkerAttributedValue) *types.InputWithheldWorkerAttributedValue {
					val, _ = inf.Value.Sub(deltaVal)
					if hasReputerValues {
						valStr := p.reputerValues[i].WorkerValues[inf.Worker]
						if v, ok := p.reputerValues[i].OneOutInfValues[inf.Worker]; ok {
							valStr = v
						}
						val = alloraMath.MustNewDecFromString(valStr)
					}
					return &types.InputWithheldWorkerAttributedValue{Worker: inf.Worker, Value: alloraMath.MustNewCappedBoundedExp40Dec(val)}
				})}
			}),
		}

		sig := s.signInputValueBundle(valueBundle, s.privKeys[reputerIndex])

		bundle := &types.InputReputerValueBundle{
			Pubkey:      s.pubKeyHexStr[reputerIndex],
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

func (s *TestSuite) SignValueBundle(valueBundle *types.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil
	}

	valueBundleSignature, err := privateKey.Sign(src)
	s.Require().NoError(err)

	return valueBundleSignature
}

func (s *TestSuite) SetupTopic(creator sdk.AccAddress, opts ...Option) uint64 {
	// create topic
	topicId := s.CreateTopic(opts...)

	// fund topic
	initialStake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())
	s.FundTopic(topicId, creator, initialStake)
	s.Require().True(
		s.bankKeeper.HasBalance(
			s.Ctx(),
			s.accountKeeper.GetModuleAddress(minttypes.EcosystemModuleName),
			sdk.NewCoin(params.DefaultBondDenom, initialStake),
		),
		"ecosystem account should have something in it after funding",
	)

	return topicId
}

func (s *TestSuite) CreateTopic(opts ...Option) uint64 {
	newTopicMsg := s.MockTopicMsg()
	newTopicMsg.Creator = s.AddrsStr()[0]

	p := customParams{}
	for _, opt := range opts {
		opt(&p)
	}

	if !p.alphaRegret.IsZero() {
		newTopicMsg.AlphaRegret = p.alphaRegret
	}
	if p.epochLength > 0 {
		newTopicMsg.EpochLength = p.epochLength
	}
	if p.groundTruthLag > 0 {
		newTopicMsg.GroundTruthLag = p.groundTruthLag
	}
	if p.workerSubmissionWindow > 0 {
		newTopicMsg.WorkerSubmissionWindow = p.workerSubmissionWindow
	}
	if p.lossMethod != "" {
		newTopicMsg.LossMethod = p.lossMethod
	}

	if newTopicMsg.EpochLength < newTopicMsg.WorkerSubmissionWindow {
		newTopicMsg.WorkerSubmissionWindow = newTopicMsg.EpochLength - 1
	}
	if newTopicMsg.EpochLength < newTopicMsg.GroundTruthLag {
		prms, err := s.EmissionsKeeper().GetParams(s.Ctx())
		s.Require().NoError(err)
		maxGTL := prms.MaxUnfulfilledReputerRequests * uint64(newTopicMsg.EpochLength)
		if uint64(newTopicMsg.GroundTruthLag) > maxGTL {
			// TODO: on FullTopicPass they have to be the same. why?
			newTopicMsg.GroundTruthLag = newTopicMsg.EpochLength
		}
	}

	res, err := s.emissionsMsgServer.CreateNewTopic(s.Ctx(), newTopicMsg)
	s.Require().NoError(err)

	if p.epochLastEnded > 0 || p.initialRegret != "" {
		topic, err := s.emissionsKeeper.GetTopic(s.Ctx(), res.TopicId)
		s.Require().NoError(err)
		topic.EpochLastEnded = p.epochLastEnded
		topic.InitialRegret = alloraMath.MustNewDecFromString(p.initialRegret)
		err = s.emissionsKeeper.SetTopic(s.Ctx(), topic.Id, topic)
		s.Require().NoError(err)
	}

	return res.TopicId
}

func (s *TestSuite) MockTopic() *types.Topic {
	return &types.Topic{
		Id:                       1,
		Creator:                  s.addrs[0].String(),
		Metadata:                 "metadata",
		LossMethod:               "mse",
		EpochLength:              10800,
		EpochLastEnded:           0,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}
}

func (s *TestSuite) MockTopicMsg() *types.CreateNewTopicRequest {
	topic := s.MockTopic()
	return &types.CreateNewTopicRequest{
		Creator:                  topic.Creator,
		Metadata:                 topic.Metadata,
		LossMethod:               topic.LossMethod,
		EpochLength:              topic.EpochLength,
		GroundTruthLag:           topic.GroundTruthLag,
		PNorm:                    topic.PNorm,
		AlphaRegret:              topic.AlphaRegret,
		AllowNegative:            topic.AllowNegative,
		Epsilon:                  topic.Epsilon,
		WorkerSubmissionWindow:   topic.WorkerSubmissionWindow,
		MeritSortitionAlpha:      topic.MeritSortitionAlpha,
		ActiveInfererQuantile:    topic.ActiveInfererQuantile,
		ActiveForecasterQuantile: topic.ActiveForecasterQuantile,
		ActiveReputerQuantile:    topic.ActiveReputerQuantile,
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
}

// ReturnIndexes generates slice of consecutive numbers
func ReturnIndexes(start, count int) []int {
	res := make([]int, count)
	for ind := start; ind < start+count; ind++ {
		res[ind-start] = ind
	}
	return res
}

func (s *TestSuite) SetupParticipants(topicID uint64, indexes []int, isReputer bool, options ...Option) {
	p := &customParams{}
	for _, opt := range options {
		opt(p)
	}
	var addresses []sdk.AccAddress

	for _, index := range indexes {
		addresses = append(addresses, s.addrs[index])
		workerRegMsg := &types.RegisterRequest{
			Sender:    s.addrs[index].String(),
			TopicId:   topicID,
			IsReputer: isReputer,
			Owner:     s.addrs[index].String(),
		}
		_, err := s.emissionsMsgServer.Register(s.Ctx(), workerRegMsg)
		if !errors.Is(err, types.ErrAddressAlreadyRegisteredInATopic) {
			s.Require().NoError(err)
		} else {
			return
		}
	}

	if !isReputer {
		return
	}

	// Add Stake for reputers
	stakes := GenerateTestStakes(len(addresses))
	for i, addr := range addresses {
		stake := stakes[i]
		if p.reputerStake != nil {
			stake = *p.reputerStake
		}
		s.MintTokensToAddress(addr, stake)
		_, err := s.emissionsMsgServer.AddStake(s.Ctx(), &types.AddStakeRequest{
			Sender:  addr.String(),
			Amount:  stake,
			TopicId: topicID,
		})
		s.Require().NoError(err)
	}

	return
}

func (s *TestSuite) SetupInferences(topicID uint64, nonce int64, workerIndexes []int, workerValues ...TestWorkerValue) types.Nonce {
	if len(workerIndexes) == 0 {
		return types.Nonce{}
	}
	inferenceBundles := generateWorkerDataBundles(s, nonce, topicID, workerIndexes, workerValues)
	for _, payload := range inferenceBundles {
		_, err := s.emissionsMsgServer.InsertWorkerPayload(s.Ctx(), &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}
	return *inferenceBundles[0].Nonce
}

func (s *TestSuite) CloseWorkerNonce(topic types.Topic, workerNonce types.Nonce) {
	err := actorutils.CloseWorkerNonce(s.emissionsKeeper, s.Ctx(), topic, workerNonce)
	s.Require().NoError(err)
}

func (s *TestSuite) CloseReputerNonce(topic types.Topic, reputerNonce types.Nonce) {
	err := actorutils.CloseReputerNonce(s.emissionsKeeper, s.Ctx(), topic, reputerNonce)
	s.Require().NoError(err)
}

func (s *TestSuite) EndBlock() {
	err := s.emissionsKeeper.SetRewardCurrentBlockEmission(s.Ctx(), cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.Ctx())
	s.Require().NoError(err)
}

func (s *TestSuite) InsertReputerLossBundle(topicID uint64, nonce int64, reputerIndexes []int, opts ...Option) types.Nonce {
	if len(reputerIndexes) == 0 {
		return types.Nonce{}
	}
	lossBundles := s.generateLossBundles(nonce, topicID, reputerIndexes, opts...)
	for _, payload := range lossBundles.ReputerValueBundles {
		_, err := s.emissionsMsgServer.InsertReputerPayload(s.Ctx(), &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		s.Require().NoError(err)
	}
	return *lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce
}

func (s *TestSuite) FullTopicSetup(workerIndexes, reputerIndexes []int, options ...Option) types.Topic {
	p := &customParams{}
	for _, opt := range options {
		opt(p)
	}

	topicId := p.topicID
	// Create topic if not exists
	if topicId == 0 && len(reputerIndexes) > 0 {
		topicId = s.SetupTopic(s.addrs[reputerIndexes[0]], options...)
	}

	// Register workers
	if len(workerIndexes) > 0 {
		s.SetupParticipants(topicId, workerIndexes, false, options...)
	}

	// Register reputers
	if len(reputerIndexes) > 0 {
		s.SetupParticipants(topicId, reputerIndexes, true, options...)
	}

	topic, err := s.emissionsKeeper.GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	return topic
}

func (s *TestSuite) FullTopicPass(workerIndexes, reputerIndexes []int, options ...Option) (uint64, int64) {
	p := &customParams{}
	for _, opt := range options {
		opt(p)
	}

	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		options...,
	)

	p.topicID = topic.Id

	var err error
	nonce := p.block
	if nonce == 0 {
		nonce, _, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.Ctx(), p.topicID)
		s.Require().NoError(err)
		s.T().Logf("Moving nonce to post inferences for TopicId: %d, Next block: %v", p.topicID, nonce)
		s.WithBlockHeight(nonce)
		s.EndBlock()
	}

	topic, err = s.emissionsKeeper.GetTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)

	workerNonces, err := s.emissionsKeeper.GetUnfulfilledWorkerNonces(s.Ctx(), topic.Id)
	s.Require().NoError(err)

	if len(workerNonces.Nonces) > 0 {
		s.SetupInferences(p.topicID, workerNonces.Nonces[0].BlockHeight, workerIndexes, p.workerValues...)
		wswBlock := nonce + topic.WorkerSubmissionWindow
		s.T().Logf("Moving nonce to end of worker submission window for TopicId: %d, Next block: %v", p.topicID, wswBlock)
		s.WithBlockHeight(wswBlock)
		s.EndBlock()
	}

	epochEndBlock := nonce + topic.EpochLength
	s.T().Logf("Moving nonce to end of first epoch for TopicId: %d, Next block: %v", p.topicID, epochEndBlock)
	s.WithBlockHeight(epochEndBlock)
	s.EndBlock()

	reputerNonces, err := s.emissionsKeeper.GetUnfulfilledReputerNonces(s.Ctx(), p.topicID)
	s.Require().NoError(err)

	topic, err = s.emissionsKeeper.GetTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)

	// TODO: check this if statement
	if len(reputerNonces.Nonces) > 0 && topic.EpochLastEnded > topic.EpochLength {
		reputerTxBlockHeight := reputerNonces.Nonces[0].ReputerNonce.BlockHeight + topic.GroundTruthLag + 1
		s.T().Logf("Moving nonce to insert loss bundles from reputers for TopicId: %d, Next block: %v, Nonce: %v", p.topicID, reputerTxBlockHeight, nonce)
		s.WithBlockHeight(reputerTxBlockHeight)
		s.InsertReputerLossBundle(topic.GetId(), reputerNonces.Nonces[0].ReputerNonce.BlockHeight, reputerIndexes, options...)
	}

	rewardsBlockHeight := nonce + topic.GroundTruthLag + topic.EpochLength
	s.T().Logf("Moving nonce to rewards end blocker for TopicId: %d, Next block: %v", p.topicID, rewardsBlockHeight)
	s.WithBlockHeight(rewardsBlockHeight)
	s.EndBlock()

	return topic.GetId(), nonce
}

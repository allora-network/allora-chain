//nolint:exhaustruct
package testutil

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
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
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/utils/fn"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/keeper/queryserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
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
	emissionsMsgServer    types.MsgServiceServer
	accounts              []account
}

type account struct {
	addr         sdk.AccAddress
	addrStr      string
	pubKeyHexStr string
	privKey      secp256k1.PrivKey
}

func NewTestSuite(moduleName string) TestSuite {
	//nolint:exhaustruct
	return TestSuite{
		ModuleName: moduleName,
	}
}

func (s *TestSuite) Ctx() sdk.Context {
	return s.ctx
}

func (s *TestSuite) Accounts() []account {
	accounts := make([]account, len(s.accounts))
	copy(accounts, s.accounts)
	return accounts
}

func (s *TestSuite) Addrs(idx int) sdk.AccAddress {
	return s.accounts[idx].addr
}

func (s *TestSuite) AddrsStr(idx int) string {
	return s.accounts[idx].addrStr

}

func (s *TestSuite) PubKeyHexStr(idx int) string {
	return s.accounts[idx].pubKeyHexStr
}

func (s *TestSuite) PrivKeys(idx int) secp256k1.PrivKey {
	return s.accounts[idx].privKey
}

func (s *TestSuite) LenAccounts() int {
	return len(s.accounts)
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
		block             int64
		topicID           uint64
		initialTopicStake cosmosMath.Int
		alphaRegret       alloraMath.Dec
		epochLength,
		groundTruthLag,
		workerSubmissionWindow,
		epochLastEnded int64
		lossMethod,
		initialRegret string
		workerValues          []TestWorkerValue
		reputerValues         []TestReputerValue
		skipNetworkInferences bool
		outputArity           types.TopicOutputArity
		reputerStake          *cosmosMath.Int
		accounts              []account
	}
)

func WithAccounts(accounts []account) Option {
	return func(s *customParams) {
		s.accounts = accounts
	}
}

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

func WithInitialTopicStake(stake cosmosMath.Int) Option {
	return func(s *customParams) {
		s.initialTopicStake = stake
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

func WithSkipNetworkInferences() Option {
	return func(p *customParams) {
		p.skipNetworkInferences = true
	}
}

func WithOutputArity(outputArity types.TopicOutputArity) Option {
	return func(p *customParams) {
		p.outputArity = outputArity
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
	}, nil).WithHeaderInfo(header.Info{Time: time.Now()}) //nolint:exhaustruct

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

	const numAccounts = 50

	s.accounts = make([]account, numAccounts)

	privKeys, pubKeyHexStr, addrs, addrsStr := alloratestutil.GenerateTestAccounts(numAccounts)
	for i, addr := range addrs {
		s.accounts[i] = account{
			addr:         addr,
			addrStr:      addrsStr[i],
			pubKeyHexStr: pubKeyHexStr[i],
			privKey:      privKeys[i],
		}
	}

	s.FundAndWhitelistAccounts(ctx)

	// create first topic
	s.CreateTopic(WithEpochLength(100), WithGroundTruthLag(100), WithWorkerSubmissionWindow(100))
}

func (s *TestSuite) FundAndWhitelistAccounts(ctx context.Context) {
	// fund and add all tests addresses in whitelists
	for _, acc := range s.accounts {
		s.FundAccount(10000000000, acc.addr)

		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, acc.addrStr)
		s.Require().NoError(err)

		err = s.emissionsKeeper.AddToTopicCreatorWhitelist(ctx, acc.addrStr)
		s.Require().NoError(err)

		err = s.emissionsKeeper.AddToGlobalWhitelist(ctx, acc.addrStr)
		s.Require().NoError(err)
	}
}

func (s *TestSuite) SetParamsForTest() {
	// Setup a sender address
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	err := s.emissionsKeeper.AddWhitelistAdmin(s.Ctx(), adminAddr.String())
	s.Require().NoError(err)

	//nolint:exhaustruct
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

func GetWorkerMultiValuesFromIndexes(
	indexes []int,
	numLabels int,
	values ...string,
) []TestWorkerValue {
	if numLabels <= 0 {
		panic("numLabels must be > 0")
	}
	if len(values) == 0 {
		panic("values must not be empty")
	}
	out := make([]TestWorkerValue, 0, len(indexes))
	for i, index := range indexes {
		lbls := make([]TestLabeledValue, numLabels)
		for j := 0; j < numLabels; j++ {
			v := values[(i*numLabels+j)%len(values)]
			lbls[j] = TestLabeledValue{
				Label: fmt.Sprintf("L%d", j),
				Value: v,
			}
		}
		out = append(out, TestWorkerValue{
			Index:  index,
			Values: lbls,
		})
	}
	return out
}

func (s *TestSuite) GetReputerValuesFromIndexes(reputerIndexes, workerIndexes []int, value ...string) []TestReputerValue {
	if len(value) == 0 {
		panic("value is empty")
	}
	addrs := make([]string, len(workerIndexes))
	for i, index := range workerIndexes {
		addrs[i] = s.accounts[index].addrStr
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

type TestLabeledValue struct {
	Label string
	Value string
}

type TestWorkerValue struct {
	Index  int
	Value  string             // legacy / SINGLE convenience
	Values []TestLabeledValue // MULTI or explicit SINGLE
}

func generateWorkerDataBundles(
	s *TestSuite,
	nonce int64,
	topicId uint64,
	workerIndexes []int,
	workerValues []TestWorkerValue,
) []*types.InputWorkerDataBundle {
	lwv := len(workerValues)
	hasWorkerValues := lwv > 0
	if hasWorkerValues && len(workerIndexes) != lwv {
		panic("invalid worker values length")
	}

	toInputLabeledValues := func(tv TestWorkerValue) []*types.InputLabeledValue {
		if len(tv.Values) > 0 {
			out := make([]*types.InputLabeledValue, len(tv.Values))
			for i, v := range tv.Values {
				out[i] = &types.InputLabeledValue{
					Label: v.Label,
					Value: alloraMath.MustNewBoundedExp40DecFromString(v.Value),
				}
			}
			return out
		}
		// backward-compatible SINGLE default
		inferenceValueStr := tv.Value
		if inferenceValueStr == "" {
			//nolint:gosec
			inferenceValueStr = strconv.FormatFloat(0.1+rand.Float64()*0.15, 'f', 5, 64)
		}
		return []*types.InputLabeledValue{
			{
				Label: "",
				Value: alloraMath.MustNewBoundedExp40DecFromString(inferenceValueStr),
			},
		}
	}

	getForecastValueStr := func(tv TestWorkerValue) string {
		if len(tv.Values) > 0 {
			// forecasts are still scalar worker-loss forecasts; use first labeled value as default seed
			return tv.Values[0].Value
		}
		if tv.Value != "" {
			return tv.Value
		}
		//nolint:gosec
		return strconv.FormatFloat(0.1+rand.Float64()*0.15, 'f', 5, 64)
	}

	var bundles []*types.InputWorkerDataBundle
	totalAddresses := len(s.accounts)

	for i, workerIdx := range workerIndexes {
		var tv TestWorkerValue
		if hasWorkerValues {
			tv = workerValues[i]
		}

		inferenceValues := toInputLabeledValues(tv)
		forecastValueStr := getForecastValueStr(tv)

		forecastTargets := []int{
			(workerIdx + 0) % totalAddresses,
			(workerIdx + 1) % totalAddresses,
			(workerIdx + 2) % totalAddresses,
		}

		forecastElements := []*types.InputForecastElement{
			{
				Inferer: s.accounts[forecastTargets[0]].addrStr,
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(forecastValueStr)),
			},
			{
				Inferer: s.accounts[forecastTargets[1]].addrStr,
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(forecastValueStr)),
			},
			{
				Inferer: s.accounts[forecastTargets[2]].addrStr,
				Value:   alloraMath.MustNewBoundedExp40Dec(alloraMath.MustNewDecFromString(forecastValueStr)),
			},
		}

		inferenceForecastBundle := &types.InputInferenceForecastBundle{
			Inference: &types.InputInference{
				TopicId:     topicId,
				BlockHeight: nonce,
				Inferer:     s.accounts[workerIdx].addrStr,
				Values:      inferenceValues,
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &types.InputForecast{
				TopicId:          topicId,
				BlockHeight:      nonce,
				Forecaster:       s.accounts[workerIdx].addrStr,
				ForecastElements: forecastElements,
				ExtraData:        nil,
			},
		}

		// Sign the bundle
		signature, err := signInferenceForecastBundle(inferenceForecastBundle, s.privKeys[workerIdx])
		s.Require().NoError(err)

		// Create the complete worker data bundle
		bundle := &types.InputWorkerDataBundle{
			Worker:                   s.accounts[workerIdx].addrStr,
			Nonce:                    &types.Nonce{BlockHeight: nonce},
			TopicId:                  topicId,
			InferenceForecastsBundle: inferenceForecastBundle,
			InferencesForecastsBundleSignature: signature,
			Pubkey:                             s.accounts[workerIdx].pubKeyHexStr,
		}

		bundles = append(bundles, bundle)
	}

	return bundles
}

func signInferenceForecastBundle(
	inferenceForecastBundle *types.InputInferenceForecastBundle,
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
	CombinedValue    string
	WorkerValues     map[string]string
	ForecasterValues map[string]string
	OneOutInfValues  map[string]string
}

// GenerateLossBundles generates the reputer loss bundles for a given topic and nonce.
//
// Usage scenarios:
// - If network inferences exist:
//   - Will base output on existing network inferences
//   - Uses provided reputerValues to set specific worker inference values per worker
//   - If no reputerValues provided, applies arbitrary delta value to network inference entries
//
// - If no network inferences exist:
//   - Uses Worker/Forecaster/OneOutInf values directly from reputerValues
func (s *TestSuite) GenerateLossBundles(
	nonce int64,
	topicId uint64,
	reputerIndexes []int,
	opts ...Option,
) (reputerValueBundles types.InputReputerValueBundles) {
	p := new(customParams)
	for _, opt := range opts {
		opt(p)
	}

	rv := len(p.reputerValues)
	hasReputerValues := rv > 0
	if hasReputerValues && len(reputerIndexes) != rv {
		panic("invalid reputer values length")
	}

	// Get network inferences (may be empty)
	networkInferences := new(types.NetworkInferenceBundle)
	if !p.skipNetworkInferences {
		var err error
		networkInferences, err = s.emissionsKeeper.GetLatestNetworkInferences(s.Ctx(), topicId, false)
		s.Require().NoError(err)
	}

	for i, reputerIndex := range reputerIndexes {
		var reputerValue TestReputerValue
		if hasReputerValues {
			reputerValue = p.reputerValues[i]
		}

		base := NetworkInferenceBundleToLossBundleSkeleton(networkInferences)
		valueBundle := buildInputValueBundle(topicId, nonce, s.accounts[reputerIndex].addrStr, reputerValue, base, p.skipNetworkInferences)

		bundle := &types.InputReputerValueBundle{
			Pubkey:      s.pubKeyHexStr[reputerIndex],
			Signature:   sig,
			ValueBundle: valueBundle,
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	return reputerValueBundles
}

// buildInputValueBundle creates an InputValueBundle based on network inferences and/or test reputer values.
//
// Decision logic:
// - If network inferences exist and are non-empty, uses them as the base structure
// - If both network inferences AND reputer values exist, applies reputer values to override network values (useTestValues=true)
// - If only network inferences exist, applies deltaVal modification to network values
// - If only reputer values exist (no network inferences), builds structure entirely from reputer values
// - Special handling for OneOutInfererForecasterValues when no network inferences exist
func buildInputValueBundle(
	topicId uint64,
	nonce int64,
	reputerAddr string,
	reputerValue TestReputerValue,
	networkInferences *types.ValueBundle,
	skipNetworkInferences bool,
) *types.InputValueBundle {
	combinedValStr := "0.1"
	if reputerValue.CombinedValue != "" {
		combinedValStr = reputerValue.CombinedValue
	}
	combinedVal := alloraMath.MustNewBoundedExp40DecFromString(combinedValStr)

	useTestValues := len(reputerValue.WorkerValues) > 0 &&
		len(networkInferences.InfererValues) > 0 &&
		len(networkInferences.ForecasterValues) > 0 || // meaning there was a network loss
		skipNetworkInferences

	buildWorkerBundleValues := buildLossBundleValues[*types.WorkerAttributedValue, *types.InputWorkerAttributedValue]
	buildWithheldBundleValues := buildLossBundleValues[*types.WithheldWorkerAttributedValue, *types.InputWithheldWorkerAttributedValue]

	//nolint:exhaustruct
	return &types.InputValueBundle{
		TopicId:                topicId,
		ReputerRequestNonce:    &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: nonce}},
		Reputer:                reputerAddr,
		CombinedValue:          combinedVal,
		NaiveValue:             combinedVal,
		InfererValues:          buildWorkerBundleValues(networkInferences.InfererValues, reputerValue, "worker", useTestValues),
		ForecasterValues:       buildWorkerBundleValues(networkInferences.ForecasterValues, reputerValue, "forecaster", useTestValues),
		OneOutInfererValues:    buildWithheldBundleValues(networkInferences.OneOutInfererValues, reputerValue, "oneOutWorker", useTestValues),
		OneOutForecasterValues: buildWithheldBundleValues(networkInferences.OneOutForecasterValues, reputerValue, "forecaster", useTestValues),
		OneInForecasterValues:  buildWorkerBundleValues(networkInferences.OneInForecasterValues, reputerValue, "forecaster", useTestValues),
		OneOutInfererForecasterValues: fn.Map(networkInferences.OneOutInfererForecasterValues, func(inf *types.OneOutInfererForecasterValues) *types.InputOneOutInfererForecasterValues {
			return &types.InputOneOutInfererForecasterValues{
				Forecaster:          inf.Forecaster,
				OneOutInfererValues: buildWithheldBundleValues(inf.OneOutInfererValues, reputerValue, "oneOutWorker", useTestValues)}
		}),
	}
}

func buildLossBundleValues[I builderValueI, O builderValueO](baseValues []I, reputerValue TestReputerValue, valueType string, useTestValues bool) (values []O) {
	if len(baseValues) == 0 && useTestValues {
		for worker, valueStr := range getValueMap(reputerValue, valueType) {
			value := getWorkerValue(worker, alloraMath.MustNewDecFromString(valueStr), reputerValue, valueType)
			values = append(values, createValue[O](worker, value))
		}
		return
	}
	return fn.Map(baseValues, func(inf I) O {
		val := getWorkerValue(inf.GetWorker(), inf.GetValue(), reputerValue, valueType)
		return createValue[O](inf.GetWorker(), val)
	})
}

func getWorkerValue(worker string, baseValue alloraMath.Dec, reputerValue TestReputerValue, valueType string) (val alloraMath.Dec) {
	// Use base value minus delta as fallback
	deltaVal := alloraMath.MustNewDecFromString("0.01988")
	val, _ = baseValue.Sub(deltaVal)
	if valueMap := getValueMap(reputerValue, valueType); valueMap != nil {
		if v, ok := valueMap[worker]; ok {
			val = alloraMath.MustNewDecFromString(v)
		}
	}
	return
}

func getValueMap(reputerValue TestReputerValue, valueType string) map[string]string {
	switch valueType {
	case "worker":
		// InfererValues use WorkerValues
		return reputerValue.WorkerValues
	case "forecaster":
		// ForecasterValues, OneOutForecasterValues, OneInForecasterValues use ForecasterValues
		return reputerValue.ForecasterValues
	case "oneOutWorker":
		// OneOutInfererValues and OneOutInfererForecasterValues use OneOutInfValues first, then WorkerValues
		if len(reputerValue.OneOutInfValues) != 0 {
			return reputerValue.OneOutInfValues
		}
		return reputerValue.WorkerValues
	}
	return nil
}

type builderValueI interface {
	GetWorker() string
	GetValue() alloraMath.Dec
}

type builderValueO interface {
	SetWorker(string)
	SetValue(alloraMath.Dec)
}

//nolint:forcetypeassert
func createValue[O builderValueO](worker string, value alloraMath.Dec) (out O) {
	out = reflect.New(reflect.TypeOf(out).Elem()).Interface().(O)
	out.SetWorker(worker)
	out.SetValue(value)
	return out
}

func NetworkInferenceBundleToLossBundleSkeleton(b *types.NetworkInferenceBundle) *types.ValueBundle {
	if b == nil {
		return &types.ValueBundle{}
	}

	inferers := make([]*types.WorkerAttributedValue, 0, len(b.InfererValues))
	for _, w := range b.InfererValues {
		if w == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(w.Values) > 0 {
			val = w.Values[0].Value
		}
		inferers = append(inferers, &types.WorkerAttributedValue{
			Worker: w.Worker,
			Value:  val,
		})
	}

	forecasters := make([]*types.WorkerAttributedValue, 0, len(b.ForecasterValues))
	for _, w := range b.ForecasterValues {
		if w == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(w.Values) > 0 {
			val = w.Values[0].Value
		}
		forecasters = append(forecasters, &types.WorkerAttributedValue{
			Worker: w.Worker,
			Value:  val,
		})
	}

	oneOutInferers := make([]*types.WithheldWorkerAttributedValue, 0, len(b.OneOutInfererValues))
	for _, v := range b.OneOutInfererValues {
		if v == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(v.CombinedInference) != 0 {
			val = v.CombinedInference[0].Value
		}
		oneOutInferers = append(oneOutInferers, &types.WithheldWorkerAttributedValue{
			Worker: v.WithheldInferer,
			Value:  val,
		})
	}

	oneOutForecasters := make([]*types.WithheldWorkerAttributedValue, 0, len(b.OneOutForecasterValues))
	for _, v := range b.OneOutForecasterValues {
		if v == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(v.CombinedInference) != 0 {
			val = v.CombinedInference[0].Value
		}
		oneOutForecasters = append(oneOutForecasters, &types.WithheldWorkerAttributedValue{
			Worker: v.WithheldForecaster,
			Value:  val,
		})
	}

	oneInForecasters := make([]*types.WorkerAttributedValue, 0, len(b.OneInForecasterValues))
	for _, v := range b.OneInForecasterValues {
		if v == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(v.CombinedInference) != 0 {
			val = v.CombinedInference[0].Value
		}
		oneInForecasters = append(oneInForecasters, &types.WorkerAttributedValue{
			Worker: v.Forecaster,
			Value:  val,
		})
	}

	ooif := make([]*types.OneOutInfererForecasterValues, 0, len(b.OneOutInfererForecasterValues))
	for _, x := range b.OneOutInfererForecasterValues {
		if x == nil {
			continue
		}
		val := alloraMath.ZeroDec()
		if len(x.CombinedInference) > 0 {
			val = x.CombinedInference[0].Value
		}
		ooif = append(ooif, &types.OneOutInfererForecasterValues{
			Forecaster: x.Forecaster,
			OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
				{
					Worker: x.WithheldInferer,
					Value:  val,
				},
			},
		})
	}
	combVal := alloraMath.ZeroDec()
	if len(b.CombinedValue) > 0 {
		combVal = b.CombinedValue[0].Value
	}
	naiveVal := alloraMath.ZeroDec()
	if len(b.NaiveValue) > 0 {
		naiveVal = b.NaiveValue[0].Value
	}
	return &types.ValueBundle{
		TopicId:                       b.TopicId,
		CombinedValue:                 combVal,
		NaiveValue:                    naiveVal,
		InfererValues:                 inferers,
		ForecasterValues:              forecasters,
		OneOutInfererValues:           oneOutInferers,
		OneOutForecasterValues:        oneOutForecasters,
		OneInForecasterValues:         oneInForecasters,
		OneOutInfererForecasterValues: ooif,
	}
}

func (s *TestSuite) SetupTopic(creator sdk.AccAddress, opts ...Option) uint64 {
	// create topic
	p := new(customParams)
	for _, opt := range opts {
		opt(p)
	}
	topicId := s.CreateTopic(opts...)

	// fund topic
	initialStake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	if !p.initialTopicStake.IsNil() {
		initialStake = p.initialTopicStake
	}

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
	newTopicMsg.Creator = s.AddrsStr(0)

	p := new(customParams)
	for _, opt := range opts {
		opt(p)
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
			newTopicMsg.GroundTruthLag = newTopicMsg.EpochLength
		}
	}
	if p.outputArity != types.TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED {
		newTopicMsg.OutputArity = p.outputArity
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

func (s *TestSuite) MockTopic() types.Topic {
	return types.Topic{
		Id:                       1,
		Creator:                  s.accounts[0].addrStr,
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
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.1"),
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
		CNorm:                    topic.CNorm,
		TopicType:                topic.TopicType,
		OutputArity:              topic.OutputArity,
		RequireUnity:             topic.RequireUnity,
		UnityTolerance:           topic.UnityTolerance,
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
	p := new(customParams)
	for _, opt := range options {
		opt(p)
	}
	var addresses []sdk.AccAddress

	for _, index := range indexes {
		addresses = append(addresses, s.accounts[index].addr)
		workerRegMsg := &types.RegisterRequest{
			Sender:    s.accounts[index].addrStr,
			TopicId:   topicID,
			IsReputer: isReputer,
			Owner:     s.accounts[index].addrStr,
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
	stakes := alloratestutil.GenerateTestStakes(len(addresses))
	for i, addr := range addresses {
		stake := stakes[i]
		if p.reputerStake != nil {
			stake = *p.reputerStake
		}
		if stake.IsZero() {
			continue
		}
		s.MintTokensToAddress(addr, stake)
		_, err := s.emissionsMsgServer.AddStake(s.Ctx(), &types.AddStakeRequest{
			Sender:  addr.String(),
			Amount:  stake,
			TopicId: topicID,
		})
		s.Require().NoError(err)
	}
}

func (s *TestSuite) SetupInferences(topicID uint64, nonce int64, workerIndexes []int, workerValues ...TestWorkerValue) types.Nonce {
	if len(workerIndexes) == 0 {
		return types.Nonce{} //nolint:exhaustruct
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

func (s *TestSuite) InsertReputerLossBundle(topicID uint64, nonce int64, reputerIndexes []int, opts ...Option) error {
	if len(reputerIndexes) == 0 {
		return errors.New("no reputer indexes provided")
	}
	lossBundles := s.GenerateLossBundles(nonce, topicID, reputerIndexes, opts...)
	for _, payload := range lossBundles.ReputerValueBundles {
		_, err := s.emissionsMsgServer.InsertReputerPayload(s.Ctx(), &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TestSuite) FullTopicSetup(workerIndexes, reputerIndexes []int, options ...Option) types.Topic {
	p := new(customParams)
	for _, opt := range options {
		opt(p)
	}

	if len(p.accounts) > 0 {
		s.accounts = p.accounts
		s.FundAndWhitelistAccounts(s.Ctx())
	}

	topicId := p.topicID
	// Create topic if not exists
	if topicId == 0 && len(reputerIndexes) > 0 {
		topicId = s.SetupTopic(s.accounts[reputerIndexes[0]].addr, options...)
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

// FullTopicPass runs a full pass for a topic. It:
//   - takes the option and sets up the topic
//   - in case topicID is provided, it assumes that topic exists so it skips creating it,
//   - finds the next churning block for the topic and uses it as the nonce,
//   - submits inferences for the unfulfilled worker nonce,
//   - forwards the chain to the end of the epoch,
//   - submits reputer loss bundles for the unfulfilled reputer nonce,
//   - forwards the chain to the rewards end blocker,
//   - runs the end blocker for closing the reputer nonce and calculating and distributing rewards (when appropriate),
//   - when running on a newly created topic, there will be no losses, so no rewards will be distributed.
//
// Limitations:
//   - with the current design, `ground_truth_lag` needs to be the same as `epoch_length`.
func (s *TestSuite) FullTopicPass(workerIndexes, reputerIndexes []int, options ...Option) (uint64, int64) {
	p := new(customParams)
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

	if len(reputerNonces.Nonces) > 0 {
		reputerTxBlockHeight := reputerNonces.Nonces[0].ReputerNonce.BlockHeight + topic.GroundTruthLag + 1
		s.T().Logf("Moving nonce to insert loss bundles from reputers for TopicId: %d, Next block: %v, Nonce: %v", p.topicID, reputerTxBlockHeight, nonce)
		s.WithBlockHeight(reputerTxBlockHeight)
		err = s.InsertReputerLossBundle(topic.GetId(), reputerNonces.Nonces[0].ReputerNonce.BlockHeight, reputerIndexes, options...)
		s.Require().NoError(err)
	}

	rewardsBlockHeight := nonce + topic.GroundTruthLag + topic.EpochLength
	s.T().Logf("Moving nonce to rewards end blocker for TopicId: %d, Next block: %v", p.topicID, rewardsBlockHeight)
	s.WithBlockHeight(rewardsBlockHeight)
	s.EndBlock()

	return topic.GetId(), nonce
}

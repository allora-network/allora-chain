package rewards_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
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
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type (
	option       func(s *customParams)
	customParams struct {
		topicID       uint64
		alphaRegret   alloraMath.Dec
		epochLength   int64
		workerValues  []TestWorkerValue
		reputerValues []map[string]string
		reputerStake  *cosmosMath.Int
	}
)

func WithTopicID(topicID uint64) option {
	return func(s *customParams) {
		s.topicID = topicID
	}
}

func WithAlphaRegret(alphaRegret alloraMath.Dec) option {
	return func(s *customParams) {
		s.alphaRegret = alphaRegret
	}
}

func WithEpochLength(epochLength int64) option {
	return func(s *customParams) {
		s.epochLength = epochLength
	}
}

func WithWorkerValues(workerValues []TestWorkerValue) option {
	return func(s *customParams) {
		s.workerValues = workerValues
	}
}

func WithReputerValues(reputerValues []map[string]string) option {
	return func(s *customParams) {
		s.reputerValues = reputerValues
	}
}

func WithReputerStake(reputerStake *cosmosMath.Int) option {
	return func(s *customParams) {
		s.reputerStake = reputerStake
	}
}

type RewardsTestSuite struct {
	suite.Suite

	ctx                sdk.Context
	accountKeeper      authkeeper.AccountKeeper
	bankKeeper         bankkeeper.BaseKeeper
	emissionsKeeper    keeper.Keeper
	emissionsAppModule module.AppModule
	mintAppModule      mint.AppModule
	msgServer          types.MsgServiceServer
	key                *storetypes.KVStoreKey
	privKeys           []secp256k1.PrivKey
	addrs              []sdk.AccAddress
	addrsStr           []string
	pubKeyHexStr       []string
}

func (s *RewardsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	// Set logger to show logs from the rewards module too

	logger := alloralog.NewTestLogger(s.T()).With("module", "rewards")
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()}).WithLogger(logger) // nolint: exhaustruct
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})

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
		storeService,
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress("gov").String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	emissionsKeeper := keeper.NewKeeper(
		encCfg.Codec,
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)
	stakingKeeper := stakingkeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress("gov").String(),
		codecAddress.NewBech32Codec(sdk.Bech32PrefixValAddr),
		codecAddress.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)
	mintKeeper := mintkeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = emissionsKeeper
	s.key = key
	emissionsAppModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultEmissionsGenesis := emissionsAppModule.DefaultGenesis(encCfg.Codec)
	emissionsAppModule.InitGenesis(ctx, encCfg.Codec, defaultEmissionsGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.emissionsAppModule = emissionsAppModule
	mintAppModule := mint.NewAppModule(encCfg.Codec, mintKeeper, accountKeeper)
	defaultMintGenesis := mintAppModule.DefaultGenesis(encCfg.Codec)
	mintAppModule.InitGenesis(ctx, encCfg.Codec, defaultMintGenesis)
	s.mintAppModule = mintAppModule

	// Fund the rewards account generously
	s.FundAccount(10000000000, s.accountKeeper.GetModuleAddress(types.AlloraRewardsAccountName))

	s.privKeys, s.pubKeyHexStr, s.addrs, s.addrsStr = alloratestutil.GenerateTestAccounts(50)
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
	// topic id must not start at 0
	id, err := s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotEqual(0, id)
}

func TestRewardsTestSuite(t *testing.T) {
	suite.Run(t, new(RewardsTestSuite))
}

func (s *RewardsTestSuite) SetupTopic(creator sdk.AccAddress, opts ...option) uint64 {
	p := customParams{}
	for _, opt := range opts {
		opt(&p)
	}
	// create topic
	alphaRegret := alloraMath.NewDecFromInt64(1)
	if !p.alphaRegret.IsZero() {
		alphaRegret = p.alphaRegret
	}
	epochLength := int64(10800)
	if p.epochLength > 0 {
		epochLength = p.epochLength
	}
	workerSubmissionWindow := int64(10)
	if epochLength < workerSubmissionWindow {
		workerSubmissionWindow = epochLength
	}
	groundTruthLag := int64(10800)
	if epochLength < groundTruthLag {
		groundTruthLag = epochLength
	}
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  creator.String(),
		Metadata:                 "test",
		AllowNegative:            false,
		LossMethod:               "mse",
		EpochLength:              epochLength,
		GroundTruthLag:           groundTruthLag,
		WorkerSubmissionWindow:   workerSubmissionWindow,
		AlphaRegret:              alphaRegret,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// fund topic
	initialStake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())
	s.MintTokensToAddress(creator, initialStake)
	fundTopicMessage := types.FundTopicRequest{
		Sender:  creator.String(),
		TopicId: res.TopicId,
		Amount:  initialStake,
	}

	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)
	s.Require().True(
		s.bankKeeper.HasBalance(
			s.ctx,
			s.accountKeeper.GetModuleAddress(minttypes.EcosystemModuleName),
			sdk.NewCoin(params.DefaultBondDenom, initialStake),
		),
		"ecosystem account should have something in it after funding",
	)

	return res.TopicId
}

func (s *RewardsTestSuite) SetupParticipants(topicID uint64, indexes []int, isReputer bool, options ...option) {
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
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
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
		s.MintTokensToAddress(addr, stake)
		_, err := s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
			Sender:  addr.String(),
			Amount:  stake,
			TopicId: topicID,
		})
		s.Require().NoError(err)
	}

	return
}

func (s *RewardsTestSuite) SetupNonces(topicID uint64, block int64) {
	err := s.emissionsKeeper.AddWorkerNonce(s.ctx, topicID, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicID, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) SetupInferences(topicID uint64, block int64, workerIndexes []int, workerValues ...TestWorkerValue) types.Nonce {
	if len(workerIndexes) == 0 {
		return types.Nonce{}
	}
	inferenceBundles := generateWorkerDataBundles(s, block, topicID, workerIndexes, workerValues...)
	for _, payload := range inferenceBundles {
		_, err := s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}

	fmt.Println("Inference bundles:")
	printJSON(inferenceBundles)

	return *inferenceBundles[0].Nonce
}

func (s *RewardsTestSuite) CloseWorkerNonce(topic types.Topic, block int64, workerNonce types.Nonce) {
	err := actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, workerNonce)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) CloseReputerNonce(topic types.Topic, reputerNonce types.Nonce) {
	err := actorutils.CloseReputerNonce(&s.emissionsKeeper, s.ctx, topic, reputerNonce)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) EndBlock() {
	err := s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) InsertReputerLossBundle(topicID uint64, block int64, reputerIndexes []int, reputerValues ...map[string]string) types.Nonce {
	if len(reputerIndexes) == 0 {
		return types.Nonce{}
	}
	lossBundles := s.generateLossBundles(block, topicID, reputerIndexes, reputerValues...)
	for _, payload := range lossBundles.ReputerValueBundles {
		v := payload.ValueBundle.InfererValues[0].Value.String()
		_ = v
		_, _ = s.emissionsKeeper.FulfillWorkerNonce(s.ctx, topicID, payload.ValueBundle.ReputerRequestNonce.ReputerNonce)
		_ = s.emissionsKeeper.AddReputerNonce(s.ctx, topicID, payload.ValueBundle.ReputerRequestNonce.ReputerNonce)
		_, err := s.msgServer.InsertReputerPayload(s.ctx, &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		s.Require().NoError(err)
	}
	fmt.Println("Loss bundles:")
	printJSON(lossBundles.ReputerValueBundles[0].ValueBundle)

	return *lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce
}

func (s *RewardsTestSuite) GenerateRewards(topicId uint64, block int64) ([]types.TaskReward, alloraMath.Dec) {
	s.ctx = s.ctx.WithBlockHeight(block)
	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)
	p, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	rewardsDistribution, totalReputerReward, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		rewards.GenerateRewardsDistributionByTopicParticipantArgs{
			Ctx:          s.ctx,
			K:            s.emissionsKeeper,
			TopicId:      topicId,
			TopicReward:  &topicTotalRewards,
			BlockHeight:  block,
			ModuleParams: p,
		})
	s.Require().NoError(err)
	return rewardsDistribution, totalReputerReward
}

func (s *RewardsTestSuite) FullTopicSetup(block int64, workerIndexes, reputerIndexes []int, options ...option) (types.Topic, types.Nonce) {
	p := &customParams{}
	for _, opt := range options {
		opt(p)
	}

	topicId := p.topicID
	// Create topic if not exists
	if topicId == 0 && len(reputerIndexes) > 0 {
		topicId = s.SetupTopic(s.addrs[reputerIndexes[0]], options...)
	}
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Register workers
	if len(workerIndexes) > 0 {
		s.SetupParticipants(topicId, workerIndexes, false, options...)
	}

	// Register reputers
	if len(reputerIndexes) > 0 {
		s.SetupParticipants(topicId, reputerIndexes, true, options...)
	}

	// Insert unfulfilled nonces
	s.SetupNonces(topicId, block)

	// Insert inference from workers
	workerNonce := s.SetupInferences(topicId, block, workerIndexes, p.workerValues...)
	return topic, workerNonce
}

func (s *RewardsTestSuite) FullTopicPass(block int64, workerIndexes, reputerIndexes []int, options ...option) (uint64, int64) {
	p := &customParams{}
	for _, opt := range options {
		opt(p)
	}

	topic /*workerNonce*/, _ := s.FullTopicSetup(
		block,
		workerIndexes,
		reputerIndexes,
		options...,
	)

	p.topicID = topic.Id

	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, p.topicID)
	s.T().Logf("Moving nonce for TopicId: %d, Next block: %v", p.topicID, nextBlock)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)

	s.EndBlock()

	blockWrk := block + topic.WorkerSubmissionWindow
	s.ctx = s.ctx.WithBlockHeight(blockWrk)

	s.EndBlock()

	// Close worker nonce
	// s.CloseWorkerNonce(topic, block, workerNonce)

	// Move to end of worker submission window
	newBlockHeight := block + topic.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockHeight)

	// Insert loss bundle from reputers
	/*reputerNonce :=*/
	s.InsertReputerLossBundle(topic.GetId(), block, reputerIndexes, p.reputerValues...)

	s.EndBlock()

	// s.CloseReputerNonce(topic, reputerNonce)
	return topic.GetId(), newBlockHeight
}

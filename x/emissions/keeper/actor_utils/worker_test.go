package actorutils_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
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
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/suite"
)

type WorkerTestSuite struct {
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

func (s *WorkerTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	// Set logger to show logs from the rewards module too
	logger := log.NewTestLogger(s.T()).With("module", "rewards")
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

func (s *WorkerTestSuite) FundAccount(amount int64, accAddress sdk.AccAddress) {
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(amount)))
	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, accAddress, initialStakeCoins)
	s.Require().NoError(err)
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}

func (s *WorkerTestSuite) TestCloseWorkerNonce() {
	// Create a topic
	blockHeight := int64(101)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// Create topic using MsgServer like in rewards_test.go
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[0],
		Metadata:                 "test",
		LossMethod:               "mse",
		AllowNegative:            false,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
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
	topicId := res.TopicId

	// Get the topic
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Register workers using MsgServer
	worker0 := s.addrsStr[0]
	worker1 := s.addrsStr[1]

	workerRegMsg0 := &types.RegisterRequest{
		Sender:    worker0,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     s.addrsStr[4],
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg0)
	s.Require().NoError(err)

	workerRegMsg1 := &types.RegisterRequest{
		Sender:    worker1,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     s.addrsStr[4],
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg1)
	s.Require().NoError(err)

	// Add worker nonce
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &nonce)
	s.Require().NoError(err)

	// Create and insert inferences
	inferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Inferer:     worker0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     worker1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}
	err = s.emissionsKeeper.InsertInference(s.ctx, topicId, *inferences.Inferences[0])
	s.Require().NoError(err)
	err = s.emissionsKeeper.InsertInference(s.ctx, topicId, *inferences.Inferences[1])
	s.Require().NoError(err)

	// Artificially add the workers as active inferers
	err = s.emissionsKeeper.AddActiveInferer(s.ctx, topicId, worker0)
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddActiveInferer(s.ctx, topicId, worker1)
	s.Require().NoError(err)

	// ------------------------------------------------------------------------------------------------
	// Move the blockheight until end of wsw
	// ------------------------------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)

	// Test closing the worker nonce
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, nonce)
	s.Require().NoError(err)

	// Verify nonce is no longer unfulfilled
	isUnfulfilled, err := s.emissionsKeeper.IsWorkerNonceUnfulfilled(s.ctx, topicId, &nonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Nonce should no longer be unfulfilled")

	// Verify network inferences were created
	networkInferences, err := s.emissionsKeeper.GetNetworkInference(s.ctx, topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(networkInferences, "Network inferences should exist")

	// Verify outlier resistant network inferences were created
	outlierResistantInferences, err := s.emissionsKeeper.GetOutlierResistantNetworkInference(s.ctx, topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(outlierResistantInferences, "Outlier resistant network inferences should exist")
}

func (s *WorkerTestSuite) TestCloseWorkerNonceFailures() {
	blockHeight := int64(101)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// Create topic using MsgServer
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[0],
		Metadata:                 "test",
		LossMethod:               "mse",
		AllowNegative:            false,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
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
	topicId := res.TopicId

	// Get the topic
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Test 1: Closing without valid nonce
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrUnfulfilledNonceNotFound)

	// Test 2: Closing without active inferers
	topic.EpochLastEnded = blockHeight - 100 // Fix the window
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &nonce)
	s.Require().NoError(err)

	// Move to end of worker submission window
	s.ctx = s.ctx.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrNoQualifiedInferers)

	// Test 3: Closing an already fulfilled nonce
	// First register and activate a worker
	worker := s.addrsStr[0]
	workerRegMsg := &types.RegisterRequest{
		Sender:    worker,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     s.addrsStr[4],
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg)
	s.Require().NoError(err)

	// Add worker as active inferer
	err = s.emissionsKeeper.AddActiveInferer(s.ctx, topicId, worker)
	s.Require().NoError(err)

	// Insert an inference
	inference := types.Inference{
		Inferer:     worker,
		Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ExtraData:   nil,
		Proof:       "",
	}
	err = s.emissionsKeeper.InsertInference(s.ctx, topicId, inference)
	s.Require().NoError(err)

	// Close the nonce first time (should succeed)
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, nonce)
	s.Require().NoError(err)

	// Try to close the same nonce again (should fail)
	err = actorutils.CloseWorkerNonce(&s.emissionsKeeper, s.ctx, topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrUnfulfilledNonceNotFound)
}

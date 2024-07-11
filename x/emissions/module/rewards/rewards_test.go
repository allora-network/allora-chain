package rewards_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
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

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type RewardsTestSuite struct {
	suite.Suite

	ctx                sdk.Context
	accountKeeper      authkeeper.AccountKeeper
	bankKeeper         bankkeeper.BaseKeeper
	emissionsKeeper    keeper.Keeper
	emissionsAppModule module.AppModule
	mintAppModule      mint.AppModule
	msgServer          types.MsgServer
	key                *storetypes.KVStoreKey
	privKeys           map[string]secp256k1.PrivKey
	addrs              []sdk.AccAddress
	addrsStr           []string
}

func (s *RewardsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
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

	// Create accounts and fund it
	var addrs []sdk.AccAddress = make([]sdk.AccAddress, 0)
	var addrsStr []string = make([]string, 0)
	var privKeys = make(map[string]secp256k1.PrivKey)
	for i := 0; i < 50; i++ {
		senderPrivKey := secp256k1.GenPrivKey()
		pubkey := senderPrivKey.PubKey().Address()

		// Add coins to account module
		s.FundAccount(10000000000, sdk.AccAddress(pubkey))
		addrs = append(addrs, sdk.AccAddress(pubkey))
		addrsStr = append(addrsStr, addrs[i].String())
		privKeys[addrsStr[i]] = senderPrivKey
	}
	s.addrs = addrs
	s.addrsStr = addrsStr
	s.privKeys = privKeys

	// Add all tests addresses in whitelists
	for _, addr := range s.addrsStr {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func (s *RewardsTestSuite) FundAccount(amount int64, accAddress sdk.AccAddress) {
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(amount)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, accAddress, initialStakeCoins)
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(RewardsTestSuite))
}

func (s *RewardsTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
}

func (s *RewardsTestSuite) MintTokensToModule(moduleName string, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	s.bankKeeper.MintCoins(s.ctx, moduleName, creatorInitialBalanceCoins)
}

func (s *RewardsTestSuite) TestStandardRewardEmission() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)

	// Reputer Addresses
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Worker Addresses
	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 5 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	// Add Stake for reputers
	var stakes = []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(384623).Mul(cosmosOneE18),
		cosmosMath.NewInt(394676).Mul(cosmosOneE18),
		cosmosMath.NewInt(207999).Mul(cosmosOneE18),
		cosmosMath.NewInt(368582).Mul(cosmosOneE18),
	}
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles := GenerateWorkerDataBundles(s, block, topicId)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateLossBundles(s, block, topicId, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Trigger end block - rewards distribution
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) TestStandardRewardEmissionShouldRewardTopicsWithFulfilledNonces() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)

	// Reputer Addresses
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Worker Addresses
	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 5 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	// Add Stake for reputers
	var stakes = []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(384623).Mul(cosmosOneE18),
		cosmosMath.NewInt(394676).Mul(cosmosOneE18),
		cosmosMath.NewInt(207999).Mul(cosmosOneE18),
		cosmosMath.NewInt(368582).Mul(cosmosOneE18),
	}
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	initialStake := cosmosMath.NewInt(1000)
	s.MintTokensToAddress(reputerAddrs[0], initialStake)
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
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

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles := GenerateWorkerDataBundles(s, block, topicId)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateLossBundles(s, block, topicId, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	// Create topic 2
	// Reputer Addresses
	reputerAddrs = []sdk.AccAddress{
		s.addrs[10],
		s.addrs[11],
		s.addrs[12],
		s.addrs[13],
		s.addrs[14],
	}

	// Worker Addresses
	workerAddrs = []sdk.AccAddress{
		s.addrs[15],
		s.addrs[16],
		s.addrs[17],
		s.addrs[18],
		s.addrs[19],
	}

	// Create topic
	newTopicMsg = &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err = s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId2 := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId2,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 5 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId2,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId2,
		})
		s.Require().NoError(err)
	}

	initialStake = cosmosMath.NewInt(1000)
	s.MintTokensToAddress(reputerAddrs[0], initialStake)
	fundTopicMessage = types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId2,
		Amount:  initialStake,
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId2, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId2, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Do not send bundles for topic 2 yet

	beforeRewardsTopic1FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId)
	s.Require().NoError(err)
	beforeRewardsTopic2FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId2)
	s.Require().NoError(err)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Trigger end block - rewards distribution
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	afterRewardsTopic1FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId)
	s.Require().NoError(err)
	afterRewardsTopic2FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId2)
	s.Require().NoError(err)

	// Topic 1 should have less revenue after rewards distribution -> rewards distributed
	s.Require().True(
		beforeRewardsTopic1FeeRevenue.GT(afterRewardsTopic1FeeRevenue),
		"Topic 1 should lose influence of their fee revenue: %s > %s",
		beforeRewardsTopic1FeeRevenue.String(),
		afterRewardsTopic1FeeRevenue.String(),
	)
	// Topic 2 should also have less revenue after rewards distribution as topic rewards
	// are shared among all topics whose epoch lengths modulo the current block height are 0
	s.Require().True(
		beforeRewardsTopic2FeeRevenue.GT(afterRewardsTopic2FeeRevenue),
		"Topic 2 should lose influence of their fee revenue: %s > %s",
		beforeRewardsTopic2FeeRevenue.String(),
		afterRewardsTopic2FeeRevenue.String(),
	)
}

func (s *RewardsTestSuite) setUpTopic(
	blockHeight int64,
	workerAddrs []sdk.AccAddress,
	reputerAddrs []sdk.AccAddress,
	stake cosmosMath.Int,
	alphaRegret alloraMath.Dec,
) uint64 {
	return s.setUpTopicWithEpochLength(blockHeight, workerAddrs, reputerAddrs, stake, alphaRegret, 10800)
}

func (s *RewardsTestSuite) setUpTopicWithEpochLength(
	blockHeight int64,
	workerAddrs []sdk.AccAddress,
	reputerAddrs []sdk.AccAddress,
	stake cosmosMath.Int,
	alphaRegret alloraMath.Dec,
	epochLength int64,
) uint64 {
	require := s.Require()
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alphaRegret,
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	require.NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	for _, workerAddr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       workerAddr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        workerAddr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		require.NoError(err)
	}

	for _, reputerAddr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       reputerAddr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        reputerAddr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		require.NoError(err)
	}
	for _, reputerAddr := range reputerAddrs {
		s.MintTokensToAddress(reputerAddr, stake)
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  reputerAddr.String(),
			Amount:  stake,
			TopicId: topicId,
		})
		require.NoError(err)
	}

	var initialStake int64 = 1000
	s.MintTokensToAddress(reputerAddrs[0], cosmosMath.NewInt(initialStake))
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	require.NoError(err)

	return topicId
}

func (s *RewardsTestSuite) getRewardsDistribution(
	topicId uint64,
	blockHeight int64,
	workerValues []TestWorkerValue,
	reputerValues []TestWorkerValue,
	workerZeroAddress sdk.AccAddress,
	workerZeroOneOutInfererValue string,
	workerZeroInfererValue string,
) []types.TaskReward {
	require := s.Require()

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	require.NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(
		s.ctx,
		topicId,
		&types.Nonce{BlockHeight: blockHeight},
	)
	require.NoError(err)

	err = s.emissionsKeeper.AddReputerNonce(
		s.ctx,
		topicId,
		&types.Nonce{BlockHeight: blockHeight},
	)
	require.NoError(err)

	getAddrsFromValues := func(values []TestWorkerValue) []sdk.AccAddress {
		addrs := make([]sdk.AccAddress, 0)
		for _, value := range values {
			addrs = append(addrs, value.Address)
		}
		return addrs
	}

	workerAddrs := getAddrsFromValues(workerValues)
	reputerAddrs := getAddrsFromValues(reputerValues)

	// Insert inference from workers
	inferenceBundles := GenerateSimpleWorkerDataBundles(s, topicId, blockHeight, workerValues, reputerAddrs)

	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: blockHeight},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	require.NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateSimpleLossBundles(
		s,
		topicId,
		blockHeight,
		workerValues,
		reputerValues,
		workerZeroAddress,
		workerZeroOneOutInfererValue,
		workerZeroInfererValue,
	)

	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerValues[0].Address.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	require.NoError(err)

	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)

	rewardsDistributionByTopicParticipant, _, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		&topicTotalRewards,
		blockHeight,
		params,
	)
	require.NoError(err)

	return rewardsDistributionByTopicParticipant
}

func areTaskRewardsEqualIgnoringTopicId(s *RewardsTestSuite, A []types.TaskReward, B []types.TaskReward) bool {
	if len(A) != len(B) {
		s.Fail("Lengths are different")
	}

	for _, taskRewardA := range A {
		found := false
		for _, taskRewardB := range B {
			if taskRewardA.Address == taskRewardB.Address {
				if found {
					s.Fail("Worker %v found twice", taskRewardA.Address)
				}
				found = true
				if !alloraMath.InDelta(taskRewardA.Reward, taskRewardB.Reward, alloraMath.MustNewDecFromString("0.00001")) {
					return false
				}
				if taskRewardA.Type != taskRewardB.Type {
					return false
				}
			}
		}
		if !found {
			s.T().Logf("Worker %v not found", taskRewardA.Address)
			return false
		}
	}

	return true
}

// We have 2 trials with 2 epochs each, and the first worker does better in the 2nd epoch in both trials.
// We show that keeping the TaskRewardAlpha the same means that the worker is rewarded the same amount
// in both cases.
// This is a sanity test to ensure that we are isolating the effect of TaskRewardAlpha in subsequent tests.
func (s *RewardsTestSuite) TestFixingTaskRewardAlphaDoesNotChangePerformanceImportanceOfPastVsPresent() {
	/// SETUP
	require := s.Require()
	k := s.emissionsKeeper

	currentParams, err := k.GetParams(s.ctx)
	require.NoError(err)

	blockHeight0 := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	reputerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	stake := cosmosMath.NewInt(1000000000000000000).Mul(inference_synthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	topicId := s.setUpTopic(blockHeight0, workerAddrs, reputerAddrs, stake, alphaRegret)

	workerValues := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.1"},
		{Address: s.addrs[1], Value: "0.2"},
		{Address: s.addrs[2], Value: "0.3"},
	}

	reputerValues := []TestWorkerValue{
		{Address: s.addrs[3], Value: "0.1"},
		{Address: s.addrs[4], Value: "0.2"},
		{Address: s.addrs[5], Value: "0.3"},
	}

	currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString(("0.1"))
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	/// TEST 0 PART A

	rewardsDistribution0_0 := s.getRewardsDistribution(
		topicId,
		blockHeight0,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 0 PART B

	blockHeight1 := blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight1)

	rewardsDistribution0_1 := s.getRewardsDistribution(
		topicId,
		blockHeight1,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.2",
		"0.1",
	)

	/// TEST 1 PART A

	blockHeight2 := blockHeight1 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight2)

	topicId1 := s.setUpTopic(blockHeight0, workerAddrs, reputerAddrs, stake, alphaRegret)

	rewardsDistribution1_0 := s.getRewardsDistribution(
		topicId1,
		blockHeight2,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 1 PART B

	blockHeight3 := blockHeight2 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight3)

	rewardsDistribution1_1 := s.getRewardsDistribution(
		topicId1,
		blockHeight3,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.2",
		"0.1",
	)

	require.True(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistribution0_0, rewardsDistribution1_0))
	require.True(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistribution0_1, rewardsDistribution1_1))
}

// We have 2 trials with 2 epochs each, and the first worker does better in the 2nd epoch in both trials,
// due to a worse one out inferer value, indicating that the network is better off with the worker.
// We increase TaskRewardAlpha between the trials to show that weighting current performance more heavily
// means that the worker is rewarded more for their better performance in the 2nd epoch of the 2nd trial.
func (s *RewardsTestSuite) TestIncreasingTaskRewardAlphaIncreasesImportanceOfPresentPerformance() {
	require := s.Require()
	k := s.emissionsKeeper

	currentParams, err := k.GetParams(s.ctx)
	require.NoError(err)

	blockHeight0 := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	reputerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	stake := cosmosMath.NewInt(1000000000000000000).Mul(inference_synthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	topicId := s.setUpTopic(blockHeight0, workerAddrs, reputerAddrs, stake, alphaRegret)

	workerValues := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.1"},
		{Address: s.addrs[1], Value: "0.2"},
		{Address: s.addrs[2], Value: "0.3"},
	}

	reputerValues := []TestWorkerValue{
		{Address: s.addrs[3], Value: "0.1"},
		{Address: s.addrs[4], Value: "0.2"},
		{Address: s.addrs[5], Value: "0.3"},
	}

	currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString("0.1")
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	/// TEST 0 PART A

	rewardsDistribution0_0 := s.getRewardsDistribution(
		topicId,
		blockHeight0,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 0 PART B

	blockHeight1 := blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight1)

	rewardsDistribution0_1 := s.getRewardsDistribution(
		topicId,
		blockHeight1,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.2",
		"0.1",
	)

	/// CHANGE TASK REWARD ALPHA

	currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString(("0.2"))
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	/// TEST 1 PART A

	blockHeight2 := blockHeight1 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight2)

	topicId1 := s.setUpTopic(blockHeight2, workerAddrs, reputerAddrs, stake, alphaRegret)

	rewardsDistribution1_0 := s.getRewardsDistribution(
		topicId1,
		blockHeight2,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 1 PART B

	blockHeight3 := blockHeight2 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight3)

	rewardsDistribution1_1 := s.getRewardsDistribution(
		topicId1,
		blockHeight3,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.2",
		"0.1",
	)

	require.True(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistribution0_0, rewardsDistribution1_0))
	require.False(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistribution0_1, rewardsDistribution1_1))

	var workerReward_0_0_1_Reward alloraMath.Dec
	found := false
	for _, reward := range rewardsDistribution0_1 {
		if reward.Address == workerAddrs[0].String() {
			found = true
			workerReward_0_0_1_Reward = reward.Reward
		}
	}
	if !found {
		require.Fail("Worker not found")
	}

	var workerReward_0_1_1_Reward alloraMath.Dec
	found = false
	for _, reward := range rewardsDistribution1_1 {
		if reward.Address == workerAddrs[0].String() {
			found = true
			workerReward_0_1_1_Reward = reward.Reward
		}
	}
	if !found {
		require.Fail("Worker not found")
	}

	require.True(workerReward_0_0_1_Reward.Lt(workerReward_0_1_1_Reward))
}

// We have 2 trials with 2 epochs each, and the first worker does worse in 2nd epoch in both trials,
// enacted by their increasing loss between epochs.
// We increase alpha between the trials to prove that their worsening performance decreases regret.
// This is somewhat counterintuitive, but can be explained by the following passage from the litepaper:
// "A positive regret implies that the inference of worker j is expected by worker k to outperform
// the network’s previously reported accuracy, whereas a negative regret indicates that the network
// is expected to be more accurate."
func (s *RewardsTestSuite) TestIncreasingAlphaRegretIncreasesPresentEffectOnRegret() {
	/// SETUP
	require := s.Require()
	k := s.emissionsKeeper

	currentParams, err := k.GetParams(s.ctx)
	require.NoError(err)

	blockHeight0 := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	reputerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	stake := cosmosMath.NewInt(1000000000000000000).Mul(inference_synthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	topicId0 := s.setUpTopic(blockHeight0, workerAddrs, reputerAddrs, stake, alphaRegret)

	workerValues := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.1"},
		{Address: s.addrs[1], Value: "0.2"},
		{Address: s.addrs[2], Value: "0.3"},
	}

	reputerValues := []TestWorkerValue{
		{Address: s.addrs[3], Value: "0.1"},
		{Address: s.addrs[4], Value: "0.2"},
		{Address: s.addrs[5], Value: "0.3"},
	}

	topic, err := k.GetTopic(s.ctx, topicId0)
	s.Require().NoError(err)
	topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
	err = k.SetTopic(s.ctx, topicId0, topic)
	require.NoError(err)

	worker0_0, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[0].String())
	require.NoError(err)
	require.True(notFound)

	worker1_0, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[1].String())
	require.NoError(err)
	require.True(notFound)

	worker2_0, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[2].String())
	require.NoError(err)
	require.True(notFound)

	/// TEST 0 PART A

	s.getRewardsDistribution(
		topicId0,
		blockHeight0,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 0 PART B

	blockHeight1 := blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight1)

	s.getRewardsDistribution(
		topicId0,
		blockHeight1,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.2",
	)

	worker0_0, notFound, err = k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[0].String())
	require.NoError(err)
	require.False(notFound)

	worker1_0, notFound, err = k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[1].String())
	require.NoError(err)
	require.False(notFound)

	worker2_0, notFound, err = k.GetInfererNetworkRegret(s.ctx, topicId0, workerAddrs[2].String())
	require.NoError(err)
	require.False(notFound)

	/// INCREASE ALPHA REGRET

	alphaRegret = alloraMath.MustNewDecFromString(("0.2"))
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	/// TEST 1 PART A

	blockHeight2 := blockHeight1 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight2)

	topicId1 := s.setUpTopic(blockHeight2, workerAddrs, reputerAddrs, stake, alphaRegret)

	s.getRewardsDistribution(
		topicId1,
		blockHeight2,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	/// TEST 1 PART B

	blockHeight3 := blockHeight2 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight3)

	s.getRewardsDistribution(
		topicId1,
		blockHeight3,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.2",
	)

	blockHeight4 := blockHeight3 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight4)

	worker0_1, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId1, workerAddrs[0].String())
	require.NoError(err)
	require.False(notFound)

	worker1_1, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId1, workerAddrs[1].String())
	require.NoError(err)
	require.False(notFound)

	worker2_1, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId1, workerAddrs[2].String())
	require.NoError(err)
	require.False(notFound)

	require.True(worker0_0.Value.Gt(worker0_1.Value))
	require.True(alloraMath.InDelta(worker1_0.Value, worker1_1.Value, alloraMath.MustNewDecFromString("0.00001")))
	require.True(alloraMath.InDelta(worker2_0.Value, worker2_1.Value, alloraMath.MustNewDecFromString("0.00001")))
}

func (s *RewardsTestSuite) TestGenerateTasksRewardsShouldIncreaseRewardShareIfMoreParticipants() {
	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	stakes := []cosmosMath.Int{
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 3 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	var initialStake int64 = 1000
	s.FundAccount(initialStake, reputerAddrs[0])
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles := GenerateWorkerDataBundles(s, block, topicId)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateLossBundles(s, block, topicId, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	firstRewardsDistribution, firstTotalReputerReward, err := rewards.GenerateRewardsDistributionByTopicParticipant(s.ctx, s.emissionsKeeper, topicId, &topicTotalRewards, block, params)
	s.Require().NoError(err)

	calcFirstTotalReputerReward := alloraMath.ZeroDec()
	for _, reward := range firstRewardsDistribution {
		if reward.Type == types.ReputerAndDelegatorRewardType {
			calcFirstTotalReputerReward, err = calcFirstTotalReputerReward.Add(reward.Reward)
			s.Require().NoError(err)
		}
	}
	s.Require().True(
		alloraMath.InDelta(
			firstTotalReputerReward,
			calcFirstTotalReputerReward,
			alloraMath.MustNewDecFromString("0.0001"),
		),
		"expected: %s, got: %s",
		firstTotalReputerReward.String(),
		calcFirstTotalReputerReward.String(),
	)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Add new reputers and stakes
	newReputerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
	}
	reputerAddrs = append(reputerAddrs, newReputerAddrs...)

	// Add Stake for new reputers
	newStakes := []cosmosMath.Int{
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
	}
	stakes = append(stakes, newStakes...)

	// Create new topic
	newTopicMsg = &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err = s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId = res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 5 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	s.FundAccount(initialStake, reputerAddrs[0])

	fundTopicMessage = types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles = GenerateWorkerDataBundles(s, block, topicId)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles = GenerateLossBundles(s, block, topicId, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	secondRewardsDistribution, secondTotalReputerReward, err := rewards.GenerateRewardsDistributionByTopicParticipant(s.ctx, s.emissionsKeeper, topicId, &topicTotalRewards, block, params)
	s.Require().NoError(err)

	calcSecondTotalReputerReward := alloraMath.ZeroDec()
	for _, reward := range secondRewardsDistribution {
		if reward.Type == types.ReputerAndDelegatorRewardType {
			calcSecondTotalReputerReward, err = calcSecondTotalReputerReward.Add(reward.Reward)
			s.Require().NoError(err)
		}
	}
	s.Require().True(
		alloraMath.InDelta(
			secondTotalReputerReward,
			calcSecondTotalReputerReward,
			alloraMath.MustNewDecFromString("0.0001"),
		),
		"expected: %s, got: %s",
		secondTotalReputerReward.String(),
		calcSecondTotalReputerReward.String(),
	)

	// Check if the reward share increased
	s.Require().True(secondTotalReputerReward.Gt(firstTotalReputerReward))
}

func (s *RewardsTestSuite) TestRewardsIncreasesBalance() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)
	epochLength := int64(10800)
	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(10000000000))

	// Reputer Addresses
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Worker Addresses
	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 5 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	// Add Stake for reputers
	var stakes = []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(984623).Mul(cosmosOneE18),
		cosmosMath.NewInt(994676).Mul(cosmosOneE18),
		cosmosMath.NewInt(907999).Mul(cosmosOneE18),
		cosmosMath.NewInt(868582).Mul(cosmosOneE18),
	}
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	initialStake := cosmosMath.NewInt(1000)
	s.MintTokensToAddress(reputerAddrs[0], initialStake)
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  initialStake,
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	reputerBalances := make([]sdk.Coin, 5)
	reputerStake := make([]cosmosMath.Int, 5)
	for i, addr := range reputerAddrs {
		reputerBalances[i] = s.bankKeeper.GetBalance(s.ctx, addr, params.DefaultBondDenom)
		reputerStake[i], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, addr.String())
		s.Require().NoError(err)
	}

	workerBalances := make([]sdk.Coin, 5)
	for i, addr := range workerAddrs {
		workerBalances[i] = s.bankKeeper.GetBalance(s.ctx, addr, params.DefaultBondDenom)
	}

	// Insert inference from workers
	inferenceBundles := GenerateWorkerDataBundles(s, block, topicId)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateLossBundles(s, block, topicId, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	block += epochLength * 3
	s.ctx = s.ctx.WithBlockHeight(block)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	// Trigger end block - rewards distribution
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	for i, addr := range reputerAddrs {
		reputerStakeCurrent, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, addr.String())
		s.Require().NoError(err)
		s.Require().True(
			reputerStakeCurrent.GT(reputerStake[i]),
			"Reputer %s stake did not increase: %s | %s",
			addr.String(),
			reputerStakeCurrent.String(),
			reputerStake[i].String(),
		)
		s.Require().True(
			s.bankKeeper.GetBalance(
				s.ctx, addr, params.DefaultBondDenom).Amount.Equal(reputerBalances[i].Amount))
	}

	for i, addr := range workerAddrs {
		s.Require().True(s.bankKeeper.GetBalance(s.ctx, addr, params.DefaultBondDenom).Amount.GT(workerBalances[i].Amount))
	}
}

func (s *RewardsTestSuite) TestRewardsHandleStandardDeviationOfZero() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)
	epochLength := int64(10800)

	// Reputer Addresses
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Worker Addresses
	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	// Create first topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		GroundTruthLag:  10,
		EpochLength:     epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	// Get Topic Id for first topic
	topicId1 := res.TopicId
	res, err = s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	topicId2 := res.TopicId

	// Register 5 workers, first 3 for topic 1 and last 2 for topic 2
	for i, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId1,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		if i > 2 {
			workerRegMsg.TopicId = topicId2
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)

	}

	// Register 5 reputers, first 3 for topic 1 and last 2 for topic 2
	for i, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			Owner:        addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId1,
			IsReputer:    true,
		}
		if i > 2 {
			reputerRegMsg.TopicId = topicId2
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	// Add Stake for reputers
	var stakes = []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(384623).Mul(cosmosOneE18),
		cosmosMath.NewInt(394676).Mul(cosmosOneE18),
		cosmosMath.NewInt(207999).Mul(cosmosOneE18),
		cosmosMath.NewInt(368582).Mul(cosmosOneE18),
	}
	for i, addr := range reputerAddrs {
		addStakeMsg := &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId1,
		}
		if i > 2 {
			addStakeMsg.TopicId = topicId2
		}
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, addStakeMsg)
		s.Require().NoError(err)
	}

	// fund topic 1
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, reputerAddrs[0], initialStakeCoins)
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId1,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	// fund topic 2
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, reputerAddrs[0], initialStakeCoins)
	fundTopicMessage.TopicId = topicId2
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId1, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId1, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId2, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId2, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	reputerBalances := make([]sdk.Coin, 5)
	reputerStake := make([]cosmosMath.Int, 5)
	for i, addr := range reputerAddrs {
		reputerBalances[i] = s.bankKeeper.GetBalance(s.ctx, addr, params.DefaultBondDenom)
		if i > 2 {
			reputerStake[i], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId2, addr.String())
			s.Require().NoError(err)
		} else {
			reputerStake[i], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, addr.String())
			s.Require().NoError(err)
		}
	}

	workerBalances := make([]sdk.Coin, 5)
	for i, addr := range workerAddrs {
		workerBalances[i] = s.bankKeeper.GetBalance(s.ctx, addr, params.DefaultBondDenom)
	}

	// Insert inference from workers
	inferenceBundles := GenerateWorkerDataBundles(s, block, topicId1)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId1,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)
	inferenceBundles2 := GenerateWorkerDataBundles(s, block, topicId2)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId2,
		WorkerDataBundles: inferenceBundles2,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateLossBundles(s, block, topicId1, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId1,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)
	lossBundles2 := GenerateLossBundles(s, block, topicId2, reputerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId2,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles2.ReputerValueBundles,
	})
	s.Require().NoError(err)

	block += epochLength * 3
	s.ctx = s.ctx.WithBlockHeight(block)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(10000000000))

	// Trigger end block - rewards distribution
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) TestStandardRewardEmissionWithOneInfererAndOneReputer() {
	blockHeight := int64(600)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)
	epochLength := int64(10800)

	// Reputer Addresses
	reputer := s.addrs[0]
	// Worker Addresses
	worker := s.addrs[5]

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputer.String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	// Get Topic Id
	topicId := res.TopicId

	// Register 1 worker
	workerRegMsg := &types.MsgRegister{
		Sender:       worker.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    false,
		Owner:        worker.String(),
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg)
	s.Require().NoError(err)

	// Register 1 reputer
	reputerRegMsg := &types.MsgRegister{
		Sender:       reputer.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputer.String(),
	}
	_, err = s.msgServer.Register(s.ctx, reputerRegMsg)
	s.Require().NoError(err)

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	s.MintTokensToAddress(reputer, cosmosMath.NewInt(1176644).Mul(cosmosOneE18))
	// Add Stake for reputer
	_, err = s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
		Sender:  reputer.String(),
		Amount:  cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		TopicId: topicId,
	})
	s.Require().NoError(err)

	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, reputer, initialStakeCoins)
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputer.String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)
	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: blockHeight,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: blockHeight,
	})
	s.Require().NoError(err)

	// Insert inference from worker
	worker1InferenceForecastBundle := &types.InferenceForecastBundle{
		Inference: &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker.String(),
			Value:       alloraMath.MustNewDecFromString("0.01127"),
		},
	}
	worker1Sig, err := GenerateWorkerSignature(s, worker1InferenceForecastBundle, worker)
	s.Require().NoError(err)
	worker1Bundle := &types.WorkerDataBundle{
		Worker:                             worker.String(),
		InferenceForecastsBundle:           worker1InferenceForecastBundle,
		InferencesForecastsBundleSignature: worker1Sig,
		Pubkey:                             GetAccPubKey(s, worker),
	}
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            worker.String(),
		Nonce:             &types.Nonce{BlockHeight: blockHeight},
		TopicId:           topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{worker1Bundle},
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputer
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: blockHeight,
			},
		},
		Reputer:                reputer.String(),
		CombinedValue:          alloraMath.MustNewDecFromString("0.01127"),
		NaiveValue:             alloraMath.MustNewDecFromString("0.0116"),
		InfererValues:          []*types.WorkerAttributedValue{{Worker: worker.String(), Value: alloraMath.MustNewDecFromString("0.0112")}},
		ForecasterValues:       []*types.WorkerAttributedValue{},
		OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{},
		OneInForecasterValues:  []*types.WorkerAttributedValue{},
	}
	sig, err := GenerateReputerSignature(s, valueBundle, reputer)
	s.Require().NoError(err)
	reputerBundle := &types.ReputerValueBundle{
		Pubkey:      GetAccPubKey(s, reputer),
		Signature:   sig,
		ValueBundle: valueBundle,
	}
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputer.String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: blockHeight,
			},
		},
		ReputerValueBundles: []*types.ReputerValueBundle{reputerBundle},
	})
	s.Require().NoError(err)

	blockHeight += epochLength * 3
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(10000000000))

	// Trigger end block - rewards distribution
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) SetParamsForTest(numInferers uint64) {
	// Setup a sender address
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	s.emissionsKeeper.AddWhitelistAdmin(s.ctx, adminAddr.String())

	newParams := &types.OptionalParams{
		MaxTopInferersToReward: []uint64{24},
		MinEpochLength:         []int64{1},
		RegistrationFee:        []cosmosMath.Int{cosmosMath.NewInt(6)},
	}

	updateMsg := &types.MsgUpdateParams{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	response, err := s.msgServer.UpdateParams(s.ctx, updateMsg)
	s.Require().NoError(err)
	s.Require().NotNil(response)
}

func (s *RewardsTestSuite) TestOnlyFewTopActorsGetReward() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)
	epochLength := int64(10800)

	// Reputer Addresses
	var reputerAddrs = make([]sdk.AccAddress, 0)
	var workerAddrs = make([]sdk.AccAddress, 0)
	var stakes = make([]cosmosMath.Int, 0)
	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	s.SetParamsForTest(24)

	for i := 0; i < 25; i++ {
		reputerAddrs = append(reputerAddrs, s.addrs[i])
		workerAddrs = append(workerAddrs, s.addrs[i+25])
		stakes = append(stakes, cosmosMath.NewInt(int64(1000*(i+1))).Mul(cosmosOneE18))
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 25 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 25 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	var initialStake int64 = 1000
	s.FundAccount(initialStake, reputerAddrs[0])

	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	// Insert unfullfiled nonces
	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles := GenerateHugeWorkerDataBundles(s, block, topicId, workerAddrs)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateHugeLossBundles(s, block, topicId, reputerAddrs, workerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	//scoresAtBlock, err := s.emissionsKeeper.GetReputersScoresAtBlock(s.ctx, topicId, block)
	//s.Require().Equal(len(scoresAtBlock.Scores), int(params.GetMaxTopReputersToReward()), "Only few Top reputers can get reward")

	networkLossBundles, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err)

	infererScores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	forecasterScores, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	s.Require().Equal(len(infererScores), int(params.GetMaxTopInferersToReward()), "Only few Top inferers can get reward")
	s.Require().Equal(len(forecasterScores), int(params.GetMaxTopForecastersToReward()), "Only few Top forecasters can get reward")
}

func (s *RewardsTestSuite) TestTotalInferersRewardFractionGrowsWithMoreInferers() {
	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	cosmosOneE18 := inference_synthesis.CosmosIntOneE18()

	stakes := []cosmosMath.Int{
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
		cosmosMath.NewInt(1000000000000000000).Mul(cosmosOneE18),
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	// Register 5 workers
	for _, addr := range workerAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 3 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	var initialStake int64 = 1000
	s.FundAccount(initialStake, reputerAddrs[0])
	fundTopicMessage := types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles := GenerateHugeWorkerDataBundles(s, block, topicId, workerAddrs)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            workerAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := GenerateHugeLossBundles(s, block, topicId, reputerAddrs, workerAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	firstRewardsDistribution, _, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		s.ctx, s.emissionsKeeper, topicId, &topicTotalRewards, block, params)
	s.Require().NoError(err)

	totalInferersReward := alloraMath.ZeroDec()
	totalForecastersReward := alloraMath.ZeroDec()
	totalReputersReward := alloraMath.ZeroDec()
	for _, reward := range firstRewardsDistribution {
		if reward.Type == types.WorkerInferenceRewardType {
			totalInferersReward, _ = totalInferersReward.Add(reward.Reward)
		} else if reward.Type == types.WorkerForecastRewardType {
			totalForecastersReward, _ = totalForecastersReward.Add(reward.Reward)
		} else if reward.Type == types.ReputerAndDelegatorRewardType {
			totalReputersReward, _ = totalReputersReward.Add(reward.Reward)
		}
	}
	totalNonInfererReward, err := totalForecastersReward.Add(totalReputersReward)
	s.Require().NoError(err)
	totalReward, err := totalNonInfererReward.Add(totalInferersReward)
	s.Require().NoError(err)

	firstInfererFraction, err := totalInferersReward.Quo(totalReward)
	s.Require().NoError(err)
	firstForecasterFraction, err := totalForecastersReward.Quo(totalReward)
	s.Require().NoError(err)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Add new worker(inferer) and stakes
	newSecondWorkersAddrs := []sdk.AccAddress{
		s.addrs[10],
		s.addrs[11],
	}
	newSecondWorkersAddrs = append(workerAddrs, newSecondWorkersAddrs...)

	// Create new topic
	newTopicMsg = &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err = s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId = res.TopicId

	// Register 7 workers with 2 new inferers
	for _, addr := range newSecondWorkersAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 3 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	s.FundAccount(initialStake, reputerAddrs[0])

	fundTopicMessage = types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles = GenerateHugeWorkerDataBundles(s, block, topicId, newSecondWorkersAddrs)
	// Add more inferer
	newInferenceBundles := GenerateMoreInferencesDataBundles(s, block, topicId)
	inferenceBundles = append(inferenceBundles, newInferenceBundles...)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            newSecondWorkersAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles = GenerateHugeLossBundles(s, block, topicId, reputerAddrs, newSecondWorkersAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	topicTotalRewards = alloraMath.NewDecFromInt64(1000000)
	secondRewardsDistribution, _, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		s.ctx, s.emissionsKeeper, topicId, &topicTotalRewards, block, params)
	s.Require().NoError(err)

	totalInferersReward = alloraMath.ZeroDec()
	totalReward = alloraMath.ZeroDec()
	for _, reward := range secondRewardsDistribution {
		if reward.Type == types.WorkerInferenceRewardType {
			totalInferersReward, _ = totalInferersReward.Add(reward.Reward)
		}
		totalReward, _ = totalReward.Add(reward.Reward)
	}
	secondInfererFraction, err := totalInferersReward.Quo(totalReward)
	s.Require().NoError(err)
	s.Require().True(
		firstInfererFraction.Lt(secondInfererFraction),
		"Second inference fraction must be bigger than first fraction %s < %s",
		firstInfererFraction,
		secondInfererFraction,
	)

	// Add new worker(forecsater) and stakes
	newThirdWorkersAddrs := []sdk.AccAddress{
		s.addrs[10],
		s.addrs[11],
	}
	newThirdWorkersAddrs = append(workerAddrs, newThirdWorkersAddrs...)

	// Create new topic
	newTopicMsg = &types.MsgCreateNewTopic{
		Creator:         reputerAddrs[0].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err = s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)

	// Get Topic Id
	topicId = res.TopicId

	// Register 7 workers with 2 new forecasters
	for _, addr := range newThirdWorkersAddrs {
		workerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    false,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		s.Require().NoError(err)
	}

	// Register 3 reputers
	for _, addr := range reputerAddrs {
		reputerRegMsg := &types.MsgRegister{
			Sender:       addr.String(),
			LibP2PKey:    "test",
			MultiAddress: "test",
			TopicId:      topicId,
			IsReputer:    true,
			Owner:        addr.String(),
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		s.MintTokensToAddress(addr, stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	s.FundAccount(initialStake, reputerAddrs[0])

	fundTopicMessage = types.MsgFundTopic{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)

	// Insert inference from workers
	inferenceBundles = GenerateHugeWorkerDataBundles(s, block, topicId, newThirdWorkersAddrs)
	// Add more inferer
	newInferenceBundles = GenerateMoreForecastersDataBundles(s, block, topicId)
	inferenceBundles = append(inferenceBundles, newInferenceBundles...)
	_, err = s.msgServer.InsertBulkWorkerPayload(s.ctx, &types.MsgInsertBulkWorkerPayload{
		Sender:            newThirdWorkersAddrs[0].String(),
		Nonce:             &types.Nonce{BlockHeight: block},
		TopicId:           topicId,
		WorkerDataBundles: inferenceBundles,
	})
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles = GenerateHugeLossBundles(s, block, topicId, reputerAddrs, newThirdWorkersAddrs)
	_, err = s.msgServer.InsertBulkReputerPayload(s.ctx, &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddrs[0].String(),
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	topicTotalRewards = alloraMath.NewDecFromInt64(1000000)
	thirdRewardsDistribution, _, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		s.ctx, s.emissionsKeeper, topicId, &topicTotalRewards, block, params)
	s.Require().NoError(err)

	totalForecastersReward = alloraMath.ZeroDec()
	totalReward = alloraMath.ZeroDec()
	for _, reward := range thirdRewardsDistribution {
		if reward.Type == types.WorkerForecastRewardType {
			totalForecastersReward, _ = totalForecastersReward.Add(reward.Reward)
		}
		totalReward, _ = totalReward.Add(reward.Reward)
	}
	thirdForecasterFraction, err := totalForecastersReward.Quo(totalReward)
	s.Require().NoError(err)
	s.Require().True(firstForecasterFraction.Lt(thirdForecasterFraction), "Third forecaster fraction must be bigger than first fraction")
}

func (s *RewardsTestSuite) TestRewardForTopicGoesUpWhenRelativeStakeGoesUp() {
	// setup
	require := s.Require()

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest(24)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	// setup topics
	stake := cosmosMath.NewInt(1000).Mul(inference_synthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, 1)
	topicId1 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, 1)

	// setup values to be identical for both topics
	reputerValues := []TestWorkerValue{
		{Address: reputerAddrs[0], Value: "0.2"},
		{Address: reputerAddrs[1], Value: "0.2"},
		{Address: reputerAddrs[2], Value: "0.2"},
	}

	workerValues := []TestWorkerValue{
		{Address: workerAddrs[0], Value: "0.2"},
		{Address: workerAddrs[1], Value: "0.2"},
		{Address: workerAddrs[2], Value: "0.2"},
	}

	// record the stakes on each topic so we can see the reward differences
	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[0].String())
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[1].String())
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[2].String())
	require.NoError(err)

	reputer3_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[0].String())
	require.NoError(err)
	reputer4_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[1].String())
	require.NoError(err)
	reputer5_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[2].String())
	require.NoError(err)

	// do work on the topics to earn rewards
	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	s.getRewardsDistribution(
		topicId1,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	// force rewards to be distributed
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	const topicFundAmount int64 = 1000

	fundTopic := func(topicId uint64, funderAddr sdk.AccAddress, amount int64) {
		s.MintTokensToAddress(funderAddr, cosmosMath.NewInt(amount))
		fundTopicMessage := types.MsgFundTopic{
			Sender:  funderAddr.String(),
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		}
		_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
		require.NoError(err)
	}

	fundTopic(topicId0, s.addrs[0], topicFundAmount)
	fundTopic(topicId1, s.addrs[3], topicFundAmount)

	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[0].String())
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[1].String())
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[2].String())
	require.NoError(err)

	reputer3_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[0].String())
	require.NoError(err)
	reputer4_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[1].String())
	require.NoError(err)
	reputer5_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[2].String())
	require.NoError(err)

	reputer0_Reward0 := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1_Reward0 := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2_Reward0 := reputer2_Stake1.Sub(reputer2_Stake0)
	reputer3_Reward0 := reputer3_Stake1.Sub(reputer3_Stake0)
	reputer4_Reward0 := reputer4_Stake1.Sub(reputer4_Stake0)
	reputer5_Reward0 := reputer5_Stake1.Sub(reputer5_Stake0)

	topic0RewardTotal0 := reputer0_Reward0.Add(reputer1_Reward0).Add(reputer2_Reward0)
	topic1RewardTotal0 := reputer3_Reward0.Add(reputer4_Reward0).Add(reputer5_Reward0)

	require.Equal(topic0RewardTotal0, topic1RewardTotal0)

	// Now, in second trial, increase stake for first reputer in topic1
	s.MintTokensToAddress(reputerAddrs[0], stake)
	_, err = s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
		Sender:  reputerAddrs[0].String(),
		Amount:  stake,
		TopicId: topicId1,
	})
	require.NoError(err)

	// record the updated stakes
	reputer3_Stake1, err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrs[3].String())
	require.NoError(err)

	// do work on the topics to earn rewards
	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	s.getRewardsDistribution(
		topicId1,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	// force rewards to be distributed
	block++
	s.ctx = s.ctx.WithBlockHeight(block)

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	// record the stakes after
	reputer0_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[0].String())
	require.NoError(err)
	reputer1_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[1].String())
	require.NoError(err)
	reputer2_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, reputerAddrs[2].String())
	require.NoError(err)

	reputer3_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[0].String())
	require.NoError(err)
	reputer4_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[1].String())
	require.NoError(err)
	reputer5_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, reputerAddrs[2].String())
	require.NoError(err)

	// calculate rewards
	reputer0_Reward1 := reputer0_Stake2.Sub(reputer0_Stake1)
	reputer1_Reward1 := reputer1_Stake2.Sub(reputer1_Stake1)
	reputer2_Reward1 := reputer2_Stake2.Sub(reputer2_Stake1)

	reputer3_Reward1 := reputer3_Stake2.Sub(reputer3_Stake1)
	reputer4_Reward1 := reputer4_Stake2.Sub(reputer4_Stake1)
	reputer5_Reward1 := reputer5_Stake2.Sub(reputer5_Stake1)

	// calculate total rewards for each topic
	topic0RewardTotal1 := reputer0_Reward1.Add(reputer1_Reward1).Add(reputer2_Reward1)
	topic1RewardTotal1 := reputer3_Reward1.Add(reputer4_Reward1).Add(reputer5_Reward1)

	// in the first round, the rewards should be equal for each topic
	require.True(topic0RewardTotal0.Equal(topic1RewardTotal0))
	// for topic 0, the rewards should be less in the second round
	require.True(topic0RewardTotal0.GT(topic0RewardTotal1))
	// in the second round, the rewards should be greater for topic 1
	require.True(topic0RewardTotal1.LT(topic1RewardTotal1))
	// the rewards for topic 1 should be greater in the second round
	require.True(topic1RewardTotal0.LT(topic1RewardTotal1))
}

func (s *RewardsTestSuite) TestReputerAboveConsensusGetsLessRewards() {
	require := s.Require()

	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	s.SetParamsForTest(24)

	reputer0Addrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
	}

	stake := cosmosMath.NewInt(1000).Mul(inference_synthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, workerAddrs, reputer0Addrs, stake, alphaRegret, 1)

	reputer0Values := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.1"},
		{Address: s.addrs[1], Value: "0.1"},
		{Address: s.addrs[2], Value: "0.1"},
		{Address: s.addrs[3], Value: "0.1"},
		{Address: s.addrs[4], Value: "0.1"},
		{Address: s.addrs[5], Value: "0.9"},
	}

	workerValues := []TestWorkerValue{
		{Address: s.addrs[6], Value: "0.1"},
		{Address: s.addrs[7], Value: "0.1"},
		{Address: s.addrs[8], Value: "0.1"},
	}

	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)
	reputer3_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[3].String())
	require.NoError(err)
	reputer4_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[4].String())
	require.NoError(err)
	reputer5_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[5].String())
	require.NoError(err)

	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputer0Values,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)
	reputer3_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[3].String())
	require.NoError(err)
	reputer4_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[4].String())
	require.NoError(err)
	reputer5_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[5].String())
	require.NoError(err)

	reputer0Reward := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1Reward := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2Reward := reputer2_Stake1.Sub(reputer2_Stake0)
	reputer3Reward := reputer3_Stake1.Sub(reputer3_Stake0)
	reputer4Reward := reputer4_Stake1.Sub(reputer4_Stake0)
	reputer5Reward := reputer5_Stake1.Sub(reputer5_Stake0)

	require.True(reputer0Reward.Equal(reputer1Reward))
	require.True(reputer1Reward.Equal(reputer2Reward))
	require.True(reputer2Reward.Equal(reputer3Reward))
	require.True(reputer3Reward.Equal(reputer4Reward))
	require.True(reputer5Reward.LT(reputer1Reward))
}

func (s *RewardsTestSuite) TestReputerBelowConsensusGetsLessRewards() {
	require := s.Require()

	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	s.SetParamsForTest(24)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
	}

	stake := cosmosMath.NewInt(1000).Mul(inference_synthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, 1)

	reputerValues := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.9"},
		{Address: s.addrs[1], Value: "0.9"},
		{Address: s.addrs[2], Value: "0.9"},
		{Address: s.addrs[3], Value: "0.9"},
		{Address: s.addrs[4], Value: "0.9"},
		{Address: s.addrs[5], Value: "0.1"},
	}

	workerValues := []TestWorkerValue{
		{Address: s.addrs[6], Value: "0.9"},
		{Address: s.addrs[7], Value: "0.9"},
		{Address: s.addrs[8], Value: "0.9"},
	}

	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)
	reputer3_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[3].String())
	require.NoError(err)
	reputer4_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[4].String())
	require.NoError(err)
	reputer5_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[5].String())
	require.NoError(err)

	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
		"0.9",
		"0.9",
	)

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)
	reputer3_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[3].String())
	require.NoError(err)
	reputer4_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[4].String())
	require.NoError(err)
	reputer5_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[5].String())
	require.NoError(err)

	reputer0Reward := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1Reward := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2Reward := reputer2_Stake1.Sub(reputer2_Stake0)
	reputer3Reward := reputer3_Stake1.Sub(reputer3_Stake0)
	reputer4Reward := reputer4_Stake1.Sub(reputer4_Stake0)
	reputer5Reward := reputer5_Stake1.Sub(reputer5_Stake0)

	require.True(reputer0Reward.Equal(reputer1Reward))
	require.True(reputer1Reward.Equal(reputer2Reward))
	require.True(reputer2Reward.Equal(reputer3Reward))
	require.True(reputer3Reward.Equal(reputer4Reward))
	require.True(reputer5Reward.LT(reputer1Reward))
}

func (s *RewardsTestSuite) TestRewardForRemainingParticipantsGoUpWhenParticipantDropsOut() {
	// SETUP
	require := s.Require()

	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	s.SetParamsForTest(24)

	reputer0Addrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
	}

	workerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
		s.addrs[5],
	}

	stake := cosmosMath.NewInt(1000).Mul(inference_synthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, workerAddrs, reputer0Addrs, stake, alphaRegret, 1)

	// Define values to test
	reputer0Values := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.2"},
		{Address: s.addrs[1], Value: "0.2"},
		{Address: s.addrs[2], Value: "0.2"},
	}

	workerValues := []TestWorkerValue{
		{Address: s.addrs[3], Value: "0.2"},
		{Address: s.addrs[4], Value: "0.2"},
		{Address: s.addrs[5], Value: "0.2"},
	}

	// Define second round values to test with one less reputer
	reputer1Values := []TestWorkerValue{
		{Address: s.addrs[0], Value: "0.2"},
		{Address: s.addrs[1], Value: "0.2"},
	}

	// record the stakes before rewards
	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// do work on the current block
	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputer0Values,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	// create tokens to reward with
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	// force rewards to be distributed
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	// record the updated stakes after rewards
	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// calculate the rewards for each reputer
	reputer0_Reward0 := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1_Reward0 := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2_Reward0 := reputer2_Stake1.Sub(reputer2_Stake0)

	// fund the topic again for future rewards
	const topicFundAmount int64 = 1000

	fundTopic := func(topicId uint64, funderAddr sdk.AccAddress, amount int64) {
		s.MintTokensToAddress(funderAddr, cosmosMath.NewInt(amount))
		fundTopicMessage := types.MsgFundTopic{
			Sender:  funderAddr.String(),
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		}
		_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
		require.NoError(err)
	}

	fundTopic(topicId0, s.addrs[0], topicFundAmount)

	// do work on the current block, but with one less reputer
	s.getRewardsDistribution(
		topicId0,
		block,
		workerValues,
		reputer1Values,
		workerAddrs[0],
		"0.1",
		"0.1",
	)

	// increase the block height
	block++
	s.ctx = s.ctx.WithBlockHeight(block)

	// create tokens to reward with
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	// force rewards to be distributed
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	// check the updated stakes after rewards
	reputer0_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// calculate the rewards for each reputer
	reputer0_Reward1 := reputer0_Stake2.Sub(reputer0_Stake1)
	reputer1_Reward1 := reputer1_Stake2.Sub(reputer1_Stake1)
	reputer2_Reward1 := reputer2_Stake2.Sub(reputer2_Stake1)

	// sanity check that participating reputer rewards went up, but non participating reputer
	// rewards went to zero
	require.True(reputer0_Reward1.GT(reputer0_Reward0))
	require.True(reputer1_Reward1.GT(reputer1_Reward0))
	require.True(reputer2_Reward0.GT(cosmosMath.ZeroInt()))
	require.True(reputer2_Reward1.Equal(cosmosMath.ZeroInt()))
}

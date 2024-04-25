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
	"github.com/cometbft/cometbft/crypto/secp256k1"
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

type RewardsTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       types.MsgServer
	key             *storetypes.KVStoreKey
	privKeys        map[string]secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
}

func (s *RewardsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                 {"minter"},
		"mint":                          {"minter"},
		types.AlloraStakingAccountName:  {"burner", "minter", "staking"},
		types.AlloraRequestsAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName:  {"minter"},
		"ecosystem":                     {"minter"},
		"bonded_tokens_pool":            {"burner", "staking"},
		"not_bonded_tokens_pool":        {"burner", "staking"},
		multiPerm:                       {"burner", "minter", "staking"},
		randomPerm:                      {"random"},
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
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = emissionsKeeper
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.appModule = appModule

	// Add coins to account module
	amount := sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(10000000000))
	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraRewardsAccountName, sdk.NewCoins(amount))
	s.Require().NoError(err)

	// Create accounts and fund it
	var addrs []sdk.AccAddress = make([]sdk.AccAddress, 0)
	var addrsStr []string = make([]string, 0)
	var privKeys = make(map[string]secp256k1.PrivKey)
	for i := 0; i < 10; i++ {
		senderPrivKey := secp256k1.GenPrivKey()
		pubkey := senderPrivKey.PubKey().Address()
		err := s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraRewardsAccountName, sdk.AccAddress(pubkey), sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(10000000))))
		s.Require().NoError(err)
		addrs = append(addrs, sdk.AccAddress(pubkey))
		addrsStr = append(addrsStr, addrs[i].String())
		privKeys[addrsStr[i]] = senderPrivKey
	}
	s.addrs = addrs
	s.addrsStr = addrsStr
	s.privKeys = privKeys

	// Add all tests addresses in whitelists
	for _, addr := range s.addrs {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(RewardsTestSuite))
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
		Creator:          reputerAddrs[0].String(),
		Metadata:         "test",
		LossLogic:        "logic",
		EpochLength:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		DefaultArg:       "ETH",
		AlphaRegret:      alloraMath.NewDecFromInt64(10),
		PrewardReputer:   alloraMath.NewDecFromInt64(11),
		PrewardInference: alloraMath.NewDecFromInt64(12),
		PrewardForecast:  alloraMath.NewDecFromInt64(13),
		FTolerance:       alloraMath.NewDecFromInt64(14),
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
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}

	cosmosOneE18 := inference_synthesis.CosmosUintOneE18()

	// Add Stake for reputers
	var stakes = []cosmosMath.Uint{
		cosmosMath.NewUint(1176644).Mul(cosmosOneE18),
		cosmosMath.NewUint(384623).Mul(cosmosOneE18),
		cosmosMath.NewUint(394676).Mul(cosmosOneE18),
		cosmosMath.NewUint(207999).Mul(cosmosOneE18),
		cosmosMath.NewUint(368582).Mul(cosmosOneE18),
	}
	for i, addr := range reputerAddrs {
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
	}, &types.Nonce{
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
			WorkerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Trigger end block - rewards distribution
	err = s.appModule.EndBlock(s.ctx)
	s.Require().NoError(err)
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

	stakes := []cosmosMath.Uint{
		cosmosMath.NewUint(1000000000000000000),
		cosmosMath.NewUint(1000000000000000000),
		cosmosMath.NewUint(1000000000000000000),
	}

	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          reputerAddrs[0].String(),
		Metadata:         "test",
		LossLogic:        "logic",
		EpochLength:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		DefaultArg:       "ETH",
		AlphaRegret:      alloraMath.NewDecFromInt64(10),
		PrewardReputer:   alloraMath.NewDecFromInt64(11),
		PrewardInference: alloraMath.NewDecFromInt64(12),
		PrewardForecast:  alloraMath.NewDecFromInt64(13),
		FTolerance:       alloraMath.NewDecFromInt64(14),
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
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	}, &types.Nonce{
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
			WorkerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	firstTaskReputerReward, _, _, err := rewards.GenerateTasksRewards(s.ctx, s.emissionsKeeper, topicId, topicTotalRewards, block, params)
	s.Require().NoError(err)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// Add new reputers and stakes
	newReputerAddrs := []sdk.AccAddress{
		s.addrs[3],
		s.addrs[4],
	}
	reputerAddrs = append(reputerAddrs, newReputerAddrs...)

	// Add Stake for new reputers
	newStakes := []cosmosMath.Uint{
		cosmosMath.NewUint(1000000000000000000),
		cosmosMath.NewUint(1000000000000000000),
	}
	stakes = append(stakes, newStakes...)

	// Create new topic
	newTopicMsg = &types.MsgCreateNewTopic{
		Creator:          reputerAddrs[0].String(),
		Metadata:         "test",
		LossLogic:        "logic",
		EpochLength:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		DefaultArg:       "ETH",
		AlphaRegret:      alloraMath.NewDecFromInt64(10),
		PrewardReputer:   alloraMath.NewDecFromInt64(11),
		PrewardInference: alloraMath.NewDecFromInt64(12),
		PrewardForecast:  alloraMath.NewDecFromInt64(13),
		FTolerance:       alloraMath.NewDecFromInt64(14),
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
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		s.Require().NoError(err)
	}
	// Add Stake for reputers
	for i, addr := range reputerAddrs {
		_, err := s.msgServer.AddStake(s.ctx, &types.MsgAddStake{
			Sender:  addr.String(),
			Amount:  stakes[i],
			TopicId: topicId,
		})
		s.Require().NoError(err)
	}

	err = s.emissionsKeeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{
		BlockHeight: block,
	}, &types.Nonce{
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
			WorkerNonce: &types.Nonce{
				BlockHeight: block,
			},
		},
		ReputerValueBundles: lossBundles.ReputerValueBundles,
	})
	s.Require().NoError(err)

	secondTaskReputerReward, _, _, err := rewards.GenerateTasksRewards(s.ctx, s.emissionsKeeper, topicId, topicTotalRewards, block, params)
	s.Require().NoError(err)

	// Check if the reward share increased
	s.Require().True(secondTaskReputerReward.Gt(firstTaskReputerReward))
}

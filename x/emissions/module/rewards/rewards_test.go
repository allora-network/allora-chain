package rewards_test

import (
	"fmt"
	"testing"
	"time"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"

	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
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

	var addrs []sdk.AccAddress = make([]sdk.AccAddress, 0)
	var addrsStr []string = make([]string, 0)
	pubkeys := simtestutil.CreateTestPubKeys(10)
	for i := 0; i < 10; i++ {
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
		authtypes.FeeCollectorName)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrs {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.emissionsKeeper.AddToTopicCreationWhitelist(ctx, addr)
		s.emissionsKeeper.AddToReputerWhitelist(ctx, addr)
	}
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(RewardsTestSuite))
}

/// HELPER FUNCTIONS

const (
	reputer1StartAmount = 1337
	reputer2StartAmount = 6969
	worker1StartAmount  = 4242
	worker2StartAmount  = 1111
)

// mock mint coins to participants
func mockMintRewardCoins(s *RewardsTestSuite, amount []cosmosMath.Int, target []sdk.AccAddress) error {
	if len(amount) != len(target) {
		return fmt.Errorf("amount and target must be the same length")
	}
	for i, addr := range target {
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount[i]))
		s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, coins)
		s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, addr, coins)
	}
	return nil
}

// give some reputers coins, have them stake those coins
func mockSomeReputers(s *RewardsTestSuite, topicId uint64) ([]sdk.AccAddress, error) {
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
	}
	reputerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(reputer1StartAmount),
		cosmosMath.NewInt(reputer2StartAmount),
	}
	err := mockMintRewardCoins(
		s,
		reputerAmounts,
		reputerAddrs,
	)
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      reputerAddrs[0].String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[0].BigInt()),
		IsReputer:    true,
	})
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      reputerAddrs[1].String(),
		LibP2PKey:    "libp2pkeyReputer2",
		MultiAddress: "multiaddressReputer2",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[1].BigInt()),
		IsReputer:    true,
	})
	if err != nil {
		return nil, err
	}
	return reputerAddrs, nil
}

// give some workers coins, have them stake those coins
func mockSomeWorkers(s *RewardsTestSuite, topicId uint64) ([]sdk.AccAddress, error) {
	workerAddrs := []sdk.AccAddress{
		s.addrs[2],
		s.addrs[3],
	}
	workerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(worker1StartAmount),
		cosmosMath.NewInt(worker2StartAmount),
	}
	err := mockMintRewardCoins(
		s,
		workerAmounts,
		workerAddrs,
	)
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      workerAddrs[0].String(),
		LibP2PKey:    "libp2pkeyWorker1",
		MultiAddress: "multiaddressWorker1",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(workerAmounts[0].BigInt()),
		Owner:        workerAddrs[0].String(),
	})
	if err != nil {
		return nil, err
	}
	_, err = s.msgServer.Register(s.ctx, &types.MsgRegister{
		Creator:      workerAddrs[1].String(),
		LibP2PKey:    "libp2pkeyWorker2",
		MultiAddress: "multiaddressWorker2",
		TopicIds:     []uint64{topicId},
		InitialStake: cosmosMath.NewUintFromBigInt(workerAmounts[1].BigInt()),
		Owner:        workerAddrs[1].String(),
	})
	if err != nil {
		return nil, err
	}
	return workerAddrs, nil
}

// create a topic
func mockCreateTopics(s *RewardsTestSuite, numToCreate uint64) ([]uint64, error) {
	ret := make([]uint64, 0)
	var i uint64
	for i = 0; i < numToCreate; i++ {
		topicMessage := types.MsgCreateNewTopic{
			Creator:          s.addrsStr[0],
			Metadata:         "metadata",
			LossLogic:        "logic",
			LossMethod:       "whatever",
			InferenceLogic:   "morelogic",
			InferenceMethod:  "whatever2",
			EpochLength:      10800,
			DefaultArg:       "default",
			Pnorm:            2,
			AlphaRegret:      "0.1",
			PrewardReputer:   "0.1",
			PrewardInference: "0.1",
			PrewardForecast:  "0.1",
			FTolerance:       "0.1",
		}

		response, err := s.msgServer.CreateNewTopic(s.ctx, &topicMessage)
		if err != nil {
			return nil, err
		}
		ret = append(ret, response.TopicId)
	}
	return ret, nil
}

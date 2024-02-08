package module_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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
	state "github.com/upshot-tech/protocol-state-machine-module"
	"github.com/upshot-tech/protocol-state-machine-module/keeper"
	"github.com/upshot-tech/protocol-state-machine-module/module"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

var (
	accAddrs = []sdk.AccAddress{
		sdk.AccAddress([]byte("addr1_______________")),
		sdk.AccAddress([]byte("addr2_______________")),
		sdk.AccAddress([]byte("addr3_______________")),
		sdk.AccAddress([]byte("addr4_______________")),
		sdk.AccAddress([]byte("addr5_______________")),
	}
)

type ModuleTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	accountKeeper keeper.AccountKeeper
	bankKeeper    keeper.BankKeeper
	upshotKeeper  keeper.Keeper
	appModule     module.AppModule
	msgServer     state.MsgServer
	key           *storetypes.KVStoreKey
}

func (s *ModuleTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("upshot")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec("cosmos")

	maccPerms := map[string][]string{
		"fee_collector":          nil,
		"mint":                   {"minter"},
		"upshot":                 {"burner", "minter", "staking"},
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
		authcodec.NewBech32Codec("cosmos"),
		"cosmos",
		authtypes.NewModuleAddress("gov").String(),
	)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{accAddrs[4].String(): true},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.upshotKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, accountKeeper, bankKeeper)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.upshotKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = keeper.NewMsgServerImpl(s.upshotKeeper)
	s.appModule = appModule
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestRegisterReputer() {
	topicId, addr, amount := registerCommonBefore(s)
	response, err := s.msgServer.RegisterReputer(s.ctx, &state.MsgRegisterReputer{
		Creator:      addr.String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicId:      topicId,
		InitialStake: amount,
	})
	s.Require().NoError(err)
	expected := state.MsgRegisterReputerResponse{
		Success: true,
		Message: "Reputer node successfully registered",
	}
	s.Require().Equal(response, &expected, "RegisterReputer should return a success message")

	isReputerRegistered, err := s.upshotKeeper.IsReputerRegistered(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().True(isReputerRegistered, "Expect reputer to be registered")

	registerCommonAfter(s, topicId, addr, amount)
}

func (s *ModuleTestSuite) TestRegisterWorker() {
	topicId, addr, amount := registerCommonBefore(s)
	response, err := s.msgServer.RegisterWorker(s.ctx, &state.MsgRegisterWorker{
		Creator:      addr.String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicId:      topicId,
		InitialStake: amount,
	})
	s.Require().NoError(err)
	expected := state.MsgRegisterWorkerResponse{
		Success: true,
		Message: "Worker node successfully registered",
	}
	s.Require().Equal(response, &expected, "RegisterWorker should return a success message")

	isWorkerRegistered, err := s.upshotKeeper.IsWorkerRegistered(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().True(isWorkerRegistered, "Expect reputer to be registered")
	registerCommonAfter(s, topicId, addr, amount)
}

/********************************************
*            Helper functions               *
*********************************************/

func registerCommonBefore(s *ModuleTestSuite) (uint64, sdk.AccAddress, cosmosMath.Uint) {
	topicId, err := mockCreateTopic(s)
	s.Require().NoError(err)
	s.Require().Equal(topicId, uint64(1))
	reputerAddrs := []sdk.AccAddress{
		sdk.AccAddress([]byte("actor________________")),
	}
	reputerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(1000),
	}
	err = mockMintRewardCoins(
		s,
		reputerAmounts,
		reputerAddrs,
	)
	s.Require().NoError(err)
	return topicId, reputerAddrs[0], cosmosMath.NewUintFromBigInt(reputerAmounts[0].BigInt())
}

func registerCommonAfter(s *ModuleTestSuite, topicId uint64, addr sdk.AccAddress, amount cosmosMath.Uint) {

	stakeAmountAfter, err := s.upshotKeeper.GetDelegatorStake(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		stakeAmountAfter,
		amount,
		"Expect stake amount to be equal to the initial stake amount after registration")
	bondAmountAfter, err := s.upshotKeeper.GetBond(s.ctx, addr, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		bondAmountAfter,
		amount,
		"Expect bond amount to be equal to the initial stake amount after registration")
	targetStakeAfter, err := s.upshotKeeper.GetStakePlacedUponTarget(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		targetStakeAfter,
		amount,
		"Expect target stake amount to be equal to the initial stake amount after registration")

	topicStake, err := s.upshotKeeper.GetTopicStake(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(
		topicStake,
		amount,
		"Expect topic stake amount to be equal to the initial stake amount after registration")

	totalStake, err := s.upshotKeeper.GetTotalStake(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(
		totalStake,
		amount,
		"Expect total stake amount to be equal to the initial stake amount after registration")
}

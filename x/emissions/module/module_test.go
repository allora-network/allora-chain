package module_test

import (
	"testing"
	"time"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module"
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

type ModuleTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       state.MsgServer
	key             *storetypes.KVStoreKey
	addrs           []sdk.AccAddress
	addrsStr        []string
}

func (s *ModuleTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                nil,
		"mint":                         {"minter"},
		state.AlloraStakingModuleName:  {"burner", "minter", "staking"},
		state.AlloraRequestsModuleName: {"burner", "minter", "staking"},
		"bonded_tokens_pool":           {"burner", "staking"},
		"not_bonded_tokens_pool":       {"burner", "staking"},
		multiPerm:                      {"burner", "minter", "staking"},
		randomPerm:                     {"random"},
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
	s.emissionsKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, accountKeeper, bankKeeper)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = keeper.NewMsgServerImpl(s.emissionsKeeper)
	s.appModule = appModule
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestRegisterReputer() {
	topicId, addr, amount := registerCommonBefore(s)
	response, err := s.msgServer.Register(s.ctx, &state.MsgRegister{
		Creator:      addr.String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicsIds:    []uint64{topicId},
		InitialStake: amount,
		IsReputer:    true,
	})
	s.Require().NoError(err)
	expected := state.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}
	s.Require().Equal(response, &expected, "RegisterReputer should return a success message")

	registeredTopics, err := s.emissionsKeeper.GetRegisteredTopicsIdsByReputerAddress(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().True(len(registeredTopics) > 0, "Expect reputer to be registered")

	registerCommonAfter(s, topicId, addr, amount)
}

func (s *ModuleTestSuite) TestRegisterWorker() {
	topicId, addr, amount := registerCommonBefore(s)
	response, err := s.msgServer.Register(s.ctx, &state.MsgRegister{
		Creator:      addr.String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicsIds:    []uint64{topicId},
		InitialStake: amount,
		Owner:        addr.String(),
	})
	s.Require().NoError(err)
	expected := state.MsgRegisterResponse{
		Success: true,
		Message: "Node successfully registered",
	}
	s.Require().Equal(response, &expected, "RegisterWorker should return a success message")

	registeredTopics, err := s.emissionsKeeper.GetRegisteredTopicsIdsByWorkerAddress(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().True(len(registeredTopics) > 0, "Expect reputer to be registered")
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
		s.addrs[0],
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

	stakeAmountAfter, err := s.emissionsKeeper.GetDelegatorStake(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		stakeAmountAfter,
		amount,
		"Expect stake amount to be equal to the initial stake amount after registration")
	bondAmountAfter, err := s.emissionsKeeper.GetBond(s.ctx, addr, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		bondAmountAfter,
		amount,
		"Expect bond amount to be equal to the initial stake amount after registration")
	targetStakeAfter, err := s.emissionsKeeper.GetStakePlacedUponTarget(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(
		targetStakeAfter,
		amount,
		"Expect target stake amount to be equal to the initial stake amount after registration")

	topicStake, err := s.emissionsKeeper.GetTopicStake(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(
		topicStake,
		amount,
		"Expect topic stake amount to be equal to the initial stake amount after registration")

	totalStake, err := s.emissionsKeeper.GetTotalStake(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(
		totalStake,
		amount,
		"Expect total stake amount to be equal to the initial stake amount after registration")
}

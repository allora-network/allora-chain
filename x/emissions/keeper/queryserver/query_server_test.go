package queryserver_test

import (
	"crypto/ed25519"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/keeper/queryserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
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

type ChainKey struct {
	pubKey ed25519.PublicKey
	priKey ed25519.PrivateKey
}

var (
	PKS     = simtestutil.CreateTestPubKeys(10)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = GeneratePrivateKeys(10)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       types.MsgServer
	queryServer     types.QueryServer
	key             *storetypes.KVStoreKey
	addrs           []sdk.AccAddress
	addrsStr        []string
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
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
		mintTypes.EcosystemModuleName:  {"burner", "minter", "staking"},
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
		authtypes.FeeCollectorName)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.queryServer = queryserver.NewQueryServerImpl(s.emissionsKeeper)

	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrsStr {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func GeneratePrivateKeys(numKeys int) []ChainKey {
	testAddrs := make([]ChainKey, numKeys)
	for i := 0; i < numKeys; i++ {
		pk, prk, _ := ed25519.GenerateKey(nil)
		testAddrs[i] = ChainKey{
			pubKey: pk,
			priKey: prk,
		}
	}

	return testAddrs
}

func (s *KeeperTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
}

func (s *KeeperTestSuite) CreateOneTopic() uint64 {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a topic first
	metadata := "Some metadata for the new topic"
	// Create a MsgCreateNewTopic message

	creator := sdk.AccAddress(PKS[0].Address())

	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:                creator.String(),
		Metadata:               metadata,
		LossMethod:             "method",
		EpochLength:            10800,
		GroundTruthLag:         10800,
		WorkerSubmissionWindow: 10,
		AlphaRegret:            alloraMath.NewDecFromInt64(1),
		PNorm:                  alloraMath.NewDecFromInt64(3),
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
	}

	s.MintTokensToAddress(creator, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	return result.TopicId
}

func (s *KeeperTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	// Create a MsgCreateNewTopic message

	creator := sdk.AccAddress(PKS[0].Address())

	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:                creator.String(),
		Metadata:               metadata,
		LossMethod:             "mse",
		EpochLength:            10800,
		GroundTruthLag:         10800,
		WorkerSubmissionWindow: 10,
		AlphaRegret:            alloraMath.NewDecFromInt64(1),
		PNorm:                  alloraMath.NewDecFromInt64(3),
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
	}

	creatorInitialBalance := types.DefaultParams().CreateTopicFee.Mul(cosmosMath.NewInt(3))
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, creatorInitialBalance))

	s.bankKeeper.MintCoins(ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, creator, creatorInitialBalanceCoins)

	initialTopicId, err := s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(initialTopicId)

	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+1, result)

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+2, result)
}

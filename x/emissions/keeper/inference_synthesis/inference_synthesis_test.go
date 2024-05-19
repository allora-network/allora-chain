package inference_synthesis_test

import (
	"testing"
	"time"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"

	// cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
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

type InferenceSynthesisTestSuite struct {
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

func (s *InferenceSynthesisTestSuite) SetupTest() {
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
		authtypes.FeeCollectorName,
	)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrsStr {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func (s *InferenceSynthesisTestSuite) inEpsilon(value alloraMath.Dec, target string, epsilon string) {
	require := s.Require()
	epsilonDec := alloraMath.MustNewDecFromString(epsilon)
	targetDec := alloraMath.MustNewDecFromString(target)
	one := alloraMath.MustNewDecFromString("1")

	lowerMultiplier, err := one.Sub(epsilonDec)
	require.NoError(err)
	lowerBound, err := targetDec.Mul(lowerMultiplier)
	require.NoError(err)

	upperMultiplier, err := one.Add(epsilonDec)
	require.NoError(err)
	upperBound, err := targetDec.Mul(upperMultiplier)
	require.NoError(err)

	if lowerBound.Lt(upperBound) {
		require.True(value.Gte(lowerBound))
		require.True(value.Lte(upperBound))
	} else {
		require.True(value.Lte(lowerBound))
		require.True(value.Gte(upperBound))
	}
}

func (s *InferenceSynthesisTestSuite) inEpsilon2(value alloraMath.Dec, target string) {
	s.inEpsilon(value, target, "0.01")
}

func (s *InferenceSynthesisTestSuite) inEpsilon3(value alloraMath.Dec, target string) {
	s.inEpsilon(value, target, "0.001")
}

func (s *InferenceSynthesisTestSuite) inEpsilon4(value alloraMath.Dec, target string) {
	s.inEpsilon(value, target, "0.0001")
}

func (s *InferenceSynthesisTestSuite) inEpsilon5(value alloraMath.Dec, target string) {
	s.inEpsilon(value, target, "0.00001")
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(InferenceSynthesisTestSuite))
}

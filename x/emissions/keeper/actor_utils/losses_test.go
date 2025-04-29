package actorutils_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	clog "cosmossdk.io/log"

	alloratestutil "github.com/allora-network/allora-chain/test/testutil"

	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/codec/address"
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
	"github.com/stretchr/testify/suite"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type ActorUtilsTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	key             *storetypes.KVStoreKey
	privKeys        []secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
	pubKeyHexStr    []string
}

func (a *ActorUtilsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(a.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()}) // nolint: exhaustruct
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                         {"minter"},
		"mint":                                  {"minter"},
		emissionstypes.AlloraStakingAccountName: {"burner", "minter", "staking"},
		emissionstypes.AlloraRewardsAccountName: {"minter"},
		emissionstypes.AlloraPendingRewardForDelegatorAccountName: {"minter"},
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

	a.privKeys, a.pubKeyHexStr, a.addrs, a.addrsStr = alloratestutil.GenerateTestAccounts(50)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		clog.NewNopLogger(),
	)

	a.ctx = ctx
	a.accountKeeper = accountKeeper
	a.bankKeeper = bankKeeper
	a.emissionsKeeper = keeper.NewKeeper(
		encCfg.Codec,
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	a.key = key
	appModule := module.NewAppModule(encCfg.Codec, a.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	a.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range a.addrsStr {
		err := a.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		a.Require().NoError(err)
	}

	err := a.emissionsKeeper.SetTopic(a.ctx, 1, emissionstypes.Topic{
		Id:                       1,
		Creator:                  a.addrsStr[49],
		Metadata:                 "metadata",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.0001"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.0001"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.0001"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.0001"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.0001"),
	})
	a.Require().NoError(err)
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ActorUtilsTestSuite))
}

func (a *ActorUtilsTestSuite) signValueBundle(valueBundle *emissionstypes.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := a.Require()
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	return valueBundleSignature
}

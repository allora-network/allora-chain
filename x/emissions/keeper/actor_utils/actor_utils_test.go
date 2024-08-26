package actorutils_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
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

type TestWorkerValue struct {
	Address sdk.AccAddress
	Value   string
}

type ActorUtilsTestSuite struct {
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

func (s *ActorUtilsTestSuite) SetupTest() {
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

func (s *ActorUtilsTestSuite) FundAccount(amount int64, accAddress sdk.AccAddress) {
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(amount)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, accAddress, initialStakeCoins)
}

func (s *ActorUtilsTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ActorUtilsTestSuite))
}

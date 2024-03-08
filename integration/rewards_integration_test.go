package integration_test

import (
	"fmt"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	params "github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	alloraEmissionsKeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type IntegrationTestSuite struct {
	suite.Suite
}

func (s *IntegrationTestSuite) SetupTest() {
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestEmitRewardsSimple() {
	// in this example we are testing the integration of the following modules:
	// - mint, which directly depends on auth, bank and staking
	encodingCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, emissions.AppModule{})
	keys := storetypes.NewKVStoreKeys(authtypes.StoreKey, banktypes.StoreKey, minttypes.StoreKey, state.ModuleName)
	authority := authtypes.NewModuleAddress("gov").String()

	// replace the logger by testing values in a real test case (e.g. log.NewTestLogger(t))
	logger := log.NewTestLogger(s.T())

	cms := integration.CreateMultiStore(keys, logger)
	newCtx := sdk.NewContext(cms, cmtproto.Header{}, true, logger)

	maccPerms := map[string][]string{
		"fee_collector":                 {"minter"},
		"mint":                          {"minter"},
		state.AlloraStakingAccountName:  {"burner", "minter", "staking"},
		state.AlloraRequestsAccountName: {"burner", "minter", "staking"},
		state.AlloraRewardsAccountName:  {"minter"},
		"bonded_tokens_pool":            {"burner", "staking"},
		"not_bonded_tokens_pool":        {"burner", "staking"},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authority,
	)
	// subspace is nil because we don't test params (which is legacy anyway)
	authModule := auth.NewAppModule(encodingCfg.Codec, accountKeeper, authsims.RandomGenesisAccounts, nil)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	bankModule := bank.NewAppModule(
		encodingCfg.Codec,
		bankKeeper,
		accountKeeper,
		nil,
	)

	mintKeeper := mintkeeper.NewKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		nil,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authority,
	)
	mintModule := mint.NewAppModule(encodingCfg.Codec, mintKeeper, accountKeeper)

	emissionsKeeper := alloraEmissionsKeeper.NewKeeper(
		encodingCfg.Codec,
		addresscodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		runtime.NewKVStoreService(keys[state.ModuleName]),
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	emissionsModule := emissions.NewAppModule(encodingCfg.Codec, emissionsKeeper)

	// create the application and register all the modules from the previous step
	integrationApp := integration.NewIntegrationApp(
		newCtx,
		logger,
		keys,
		encodingCfg.Codec,
		map[string]appmodule.AppModule{
			authtypes.ModuleName: authModule,
			banktypes.ModuleName: bankModule,
			minttypes.ModuleName: mintModule,
			state.ModuleName:     emissionsModule,
		},
	)

	// register the message and query servers
	authtypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), authkeeper.NewMsgServerImpl(accountKeeper))
	authtypes.RegisterQueryServer(integrationApp.QueryHelper(), authkeeper.NewQueryServer(accountKeeper))
	banktypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), bankkeeper.NewMsgServerImpl(bankKeeper))
	banktypes.RegisterQueryServer(integrationApp.QueryHelper(), bankKeeper)
	minttypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), mintkeeper.NewMsgServerImpl(mintKeeper))
	minttypes.RegisterQueryServer(integrationApp.QueryHelper(), mintkeeper.NewQueryServerImpl(mintKeeper))
	state.RegisterMsgServer(integrationApp.MsgServiceRouter(), alloraEmissionsKeeper.NewMsgServerImpl(emissionsKeeper))
	state.RegisterQueryServer(integrationApp.QueryHelper(), alloraEmissionsKeeper.NewQueryServerImpl(emissionsKeeper))

	params := minttypes.DefaultParams()
	params.BlocksPerYear = 10000

	// now we can use the application to test a mint message
	result, err := integrationApp.RunMsg(&minttypes.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	})
	s.Require().NoError(err)

	// in this example the result is an empty response, a nil check is enough
	// in other cases, it is recommended to check the result value.
	s.Require().NotNil(result)

	// we now check the result
	resp := minttypes.MsgUpdateParamsResponse{}
	err = encodingCfg.Codec.Unmarshal(result.Value, &resp)
	s.Require().NoError(err)

	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	// we should also check the state of the application
	got, err := mintKeeper.Params.Get(sdkCtx)
	s.Require().NoError(err)

	s.Require().Equal(params, got)

	fmt.Println(got.BlocksPerYear)
	// Output: 10000
}

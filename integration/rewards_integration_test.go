package integration_test

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/allora-network/allora-chain/integration"
	cosmosintegration "github.com/cosmos/cosmos-sdk/testutil/integration"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cosmosMath "cosmossdk.io/math"

	params "github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	alloraEmissionsKeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"

	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
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
	// emissions (also referred to as state), which directly depends on auth, bank, and mint
	encodingCfg := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distribution.AppModuleBasic{},
		emissions.AppModule{},
	)
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		minttypes.StoreKey,
		state.ModuleName,
	)
	authority := authtypes.NewModuleAddress("gov").String()

	logger := log.NewTestLogger(s.T())

	cms := integration.CreateMultiStore(keys, logger)
	newCtx := sdk.NewContext(cms, cmtproto.Header{}, true, logger)

	// Create keeper and module objects
	maccPerms := map[string][]string{
		"fee_collector":                 {"minter"},
		"mint":                          {"minter"},
		state.AlloraStakingAccountName:  {"burner", "minter", "staking"},
		state.AlloraRequestsAccountName: {"burner", "minter", "staking"},
		state.AlloraRewardsAccountName:  {"minter"},
		"bonded_tokens_pool":            {"burner", "staking"},
		"distribution":                  {"minter", "burner"},
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

	stakingKeeper := stakingkeeper.NewKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authority,
		addresscodec.NewBech32Codec(params.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(params.Bech32PrefixConsAddr),
	)
	stakingModule := staking.NewAppModule(
		encodingCfg.Codec,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		nil,
	)

	distrKeeper := distrkeeper.NewKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
		authority,
	)
	distrModule := distribution.NewAppModule(
		encodingCfg.Codec,
		distrKeeper,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		nil,
	)

	mintKeeper := mintkeeper.NewKeeper(
		encodingCfg.Codec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		stakingKeeper,
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
			authtypes.ModuleName:    authModule,
			banktypes.ModuleName:    bankModule,
			stakingtypes.ModuleName: stakingModule,
			minttypes.ModuleName:    mintModule,
			distrtypes.ModuleName:   distrModule,
			state.ModuleName:        emissionsModule,
		},
		[]string{
			state.ModuleName,
			distrtypes.ModuleName,
			stakingtypes.ModuleName,
			minttypes.ModuleName,
		},
		[]string{
			stakingtypes.ModuleName,
			state.ModuleName,
		},
	)

	// register the message and query servers
	authtypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), authkeeper.NewMsgServerImpl(accountKeeper))
	authtypes.RegisterQueryServer(integrationApp.QueryHelper(), authkeeper.NewQueryServer(accountKeeper))
	banktypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), bankkeeper.NewMsgServerImpl(bankKeeper))
	banktypes.RegisterQueryServer(integrationApp.QueryHelper(), bankKeeper)
	stakingtypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), stakingkeeper.NewMsgServerImpl(stakingKeeper))
	stakingtypes.RegisterQueryServer(integrationApp.QueryHelper(), stakingkeeper.NewQuerier(stakingKeeper))
	minttypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), mintkeeper.NewMsgServerImpl(mintKeeper))
	minttypes.RegisterQueryServer(integrationApp.QueryHelper(), mintkeeper.NewQueryServerImpl(mintKeeper))
	state.RegisterMsgServer(integrationApp.MsgServiceRouter(), alloraEmissionsKeeper.NewMsgServerImpl(emissionsKeeper))
	state.RegisterQueryServer(integrationApp.QueryHelper(), alloraEmissionsKeeper.NewQueryServerImpl(emissionsKeeper))

	// get a test account set up
	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())
	pubkeys := simtestutil.CreateTestPubKeys(5)
	testerAddr := sdk.AccAddress(pubkeys[0].Address())
	testerAddrStr := testerAddr.String()

	// give test account some money
	startCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(100000)))
	bankKeeper.MintCoins(sdkCtx, state.AlloraStakingAccountName, startCoins)
	bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, state.AlloraStakingAccountName, testerAddr, startCoins)

	// run the init genesis for the modules, because integrationApp isn't running them correctly
	genesisState := state.GenesisState{
		Params: state.DefaultParams(),
		CoreTeamAddresses: []string{
			testerAddrStr,
		},
	}
	emissionsKeeper.InitGenesis(newCtx, &genesisState)
	mintKeeper.InitGenesis(newCtx, accountKeeper, minttypes.DefaultGenesisState())
	// make sure test account is on various whitelists
	emissionsKeeper.AddWhitelistAdmin(sdkCtx, testerAddr)
	emissionsKeeper.AddToTopicCreationWhitelist(sdkCtx, testerAddr)
	emissionsKeeper.AddToWeightSettingWhitelist(sdkCtx, testerAddr)

	// Set emissions module params to expected values for test
	emissionsKeeper.SetParams(sdkCtx, state.Params{
		Version:                       "0.0.3",                                   // version of the protocol should be in lockstep with github release tag version
		EpochLength:                   int64(5),                                  // length of an "epoch" for rewards payouts in blocks
		MinTopicUnmetDemand:           cosmosMath.NewUint(100),                   // total unmet demand for a topic < this => don't run inference solicatation or weight-adjustment
		MaxTopicsPerBlock:             uint64(1000),                              // max number of topics to run cadence for per block
		MinRequestUnmetDemand:         cosmosMath.NewUint(1),                     // delete requests if they have below this demand remaining
		MaxMissingInferencePercent:    cosmosMath.LegacyMustNewDecFromStr("0.1"), // if a worker has this percentage of inferences missing, they are penalized
		RequiredMinimumStake:          cosmosMath.NewUint(1),                     // minimum stake required to be a worker
		RemoveStakeDelayWindow:        uint64(172800),                            // 2 days in seconds
		MinRequestCadence:             uint64(60),                                // 1 minute in seconds
		MinWeightCadence:              uint64(10800),                             // 3 hours in seconds
		MaxInferenceRequestValidity:   uint64(60 * 60 * 24 * 7 * 24),             // 24 weeks approximately 6 months in seconds
		MaxRequestCadence:             uint64(60 * 60 * 24 * 7 * 24),             // 24 weeks approximately 6 months in seconds
		PercentRewardsReputersWorkers: cosmosMath.LegacyMustNewDecFromStr("0.5"), // 50% of rewards go to workers and reputers, 50% to cosmos validators
	})

	// Create a topic to test with
	topicMessage := state.MsgCreateNewTopic{
		Creator:          testerAddrStr,
		Metadata:         "metadata",
		WeightLogic:      "logic",
		WeightMethod:     "whatever",
		WeightCadence:    10800,
		InferenceLogic:   "morelogic",
		InferenceMethod:  "whatever2",
		InferenceCadence: 60,
	}

	result, err := integrationApp.RunMsg(
		&topicMessage,
		cosmosintegration.WithAutomaticFinalizeBlock(),
		cosmosintegration.WithAutomaticCommit(),
	)
	s.Require().NoError(err)

	// verify that the begin and end blocker were called
	// verifying the block height
	s.Require().Equal(int64(2), integrationApp.LastBlockHeight())

	// we now check the result
	response := &state.MsgCreateNewTopicResponse{}
	expectedTopicId := uint64(1)
	expectedResponse := &state.MsgCreateNewTopicResponse{TopicId: expectedTopicId}
	err = encodingCfg.Codec.Unmarshal(result.Value, response)
	s.Require().NoError(err)
	s.Require().Equal(expectedResponse, response)

	nextTopicId, err := emissionsKeeper.GetNumTopics(sdkCtx)
	s.Require().NoError(err)
	s.Require().Equal(expectedTopicId+1, nextTopicId)
}

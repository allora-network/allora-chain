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
	Ctx             sdk.Context
	Addr            []sdk.AccAddress
	AddrStr         []string
	EncodingCfg     moduletestutil.TestEncodingConfig
	IntegrationApp  *integration.App
	EmissionsKeeper alloraEmissionsKeeper.Keeper
	BankKeeper      bankkeeper.Keeper
	AccountKeeper   authkeeper.AccountKeeper
}

func (s *IntegrationTestSuite) SetupTest() {
	// in this example we are testing the integration of the following modules:
	// emissions (also referred to as state), which directly depends on auth, bank, and mint
	s.EncodingCfg = moduletestutil.MakeTestEncodingConfig(
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

	s.AccountKeeper = authkeeper.NewAccountKeeper(
		s.EncodingCfg.Codec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authority,
	)
	// subspace is nil because we don't test params (which is legacy anyway)
	authModule := auth.NewAppModule(s.EncodingCfg.Codec, s.AccountKeeper, authsims.RandomGenesisAccounts, nil)

	s.BankKeeper = bankkeeper.NewBaseKeeper(
		s.EncodingCfg.Codec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		s.AccountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	bankModule := bank.NewAppModule(
		s.EncodingCfg.Codec,
		s.BankKeeper,
		s.AccountKeeper,
		nil,
	)

	stakingKeeper := stakingkeeper.NewKeeper(
		s.EncodingCfg.Codec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		s.AccountKeeper,
		s.BankKeeper,
		authority,
		addresscodec.NewBech32Codec(params.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(params.Bech32PrefixConsAddr),
	)
	stakingModule := staking.NewAppModule(
		s.EncodingCfg.Codec,
		stakingKeeper,
		s.AccountKeeper,
		s.BankKeeper,
		nil,
	)

	distrKeeper := distrkeeper.NewKeeper(
		s.EncodingCfg.Codec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		s.AccountKeeper,
		s.BankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
		authority,
	)
	distrModule := distribution.NewAppModule(
		s.EncodingCfg.Codec,
		distrKeeper,
		s.AccountKeeper,
		s.BankKeeper,
		stakingKeeper,
		nil,
	)

	mintKeeper := mintkeeper.NewKeeper(
		s.EncodingCfg.Codec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		stakingKeeper,
		s.AccountKeeper,
		s.BankKeeper,
		authtypes.FeeCollectorName,
		authority,
	)
	mintModule := mint.NewAppModule(s.EncodingCfg.Codec, mintKeeper, s.AccountKeeper)

	s.EmissionsKeeper = alloraEmissionsKeeper.NewKeeper(
		s.EncodingCfg.Codec,
		addresscodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		runtime.NewKVStoreService(keys[state.ModuleName]),
		s.AccountKeeper,
		s.BankKeeper,
		authtypes.FeeCollectorName,
	)
	emissionsModule := emissions.NewAppModule(s.EncodingCfg.Codec, s.EmissionsKeeper)

	// create the application and register all the modules from the previous step
	s.IntegrationApp = integration.NewIntegrationApp(
		newCtx,
		logger,
		keys,
		s.EncodingCfg.Codec,
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
	authtypes.RegisterMsgServer(s.IntegrationApp.MsgServiceRouter(), authkeeper.NewMsgServerImpl(s.AccountKeeper))
	authtypes.RegisterQueryServer(s.IntegrationApp.QueryHelper(), authkeeper.NewQueryServer(s.AccountKeeper))
	banktypes.RegisterMsgServer(s.IntegrationApp.MsgServiceRouter(), bankkeeper.NewMsgServerImpl(s.BankKeeper))
	banktypes.RegisterQueryServer(s.IntegrationApp.QueryHelper(), s.BankKeeper)
	stakingtypes.RegisterMsgServer(s.IntegrationApp.MsgServiceRouter(), stakingkeeper.NewMsgServerImpl(stakingKeeper))
	stakingtypes.RegisterQueryServer(s.IntegrationApp.QueryHelper(), stakingkeeper.NewQuerier(stakingKeeper))
	minttypes.RegisterMsgServer(s.IntegrationApp.MsgServiceRouter(), mintkeeper.NewMsgServerImpl(mintKeeper))
	minttypes.RegisterQueryServer(s.IntegrationApp.QueryHelper(), mintkeeper.NewQueryServerImpl(mintKeeper))
	state.RegisterMsgServer(s.IntegrationApp.MsgServiceRouter(), alloraEmissionsKeeper.NewMsgServerImpl(s.EmissionsKeeper))
	state.RegisterQueryServer(s.IntegrationApp.QueryHelper(), alloraEmissionsKeeper.NewQueryServerImpl(s.EmissionsKeeper))

	// get a test account set up
	s.Ctx = sdk.UnwrapSDKContext(s.IntegrationApp.Context())
	pubkeys := simtestutil.CreateTestPubKeys(5)
	s.Addr = make([]sdk.AccAddress, 5)
	s.AddrStr = make([]string, 5)
	startCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(100000)))
	for i := 0; i < 5; i++ {
		s.Addr[i] = sdk.AccAddress(pubkeys[i].Address())
		s.AddrStr[i] = s.Addr[i].String()
		// give test account some money
		s.BankKeeper.MintCoins(s.Ctx, state.AlloraStakingAccountName, startCoins)
		s.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, state.AlloraStakingAccountName, s.Addr[i], startCoins)
	}

	// run the init genesis for the modules, because s.IntegrationApp isn't running them correctly
	genesisState := state.GenesisState{
		Params: state.Params{
			Version:                       "0.0.3",                                   // version of the protocol should be in lockstep with github release tag version
			EpochLength:                   int64(9),                                  // length of an "epoch" for rewards payouts in blocks
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
		},
		CoreTeamAddresses: []string{
			s.AddrStr[0],
		},
	}
	s.AccountKeeper.InitGenesis(newCtx, *authtypes.DefaultGenesisState())
	s.BankKeeper.InitGenesis(newCtx, banktypes.DefaultGenesisState())
	s.EmissionsKeeper.InitGenesis(newCtx, &genesisState)
	mintGenesisState := minttypes.DefaultGenesisState()
	mintGenesisState.Params.MintDenom = params.DefaultBondDenom
	mintKeeper.InitGenesis(newCtx, s.AccountKeeper, mintGenesisState)
	// make sure test account is on various whitelists
	s.EmissionsKeeper.AddWhitelistAdmin(s.Ctx, s.Addr[0])
	s.EmissionsKeeper.AddToTopicCreationWhitelist(s.Ctx, s.Addr[0])
	s.EmissionsKeeper.AddToWeightSettingWhitelist(s.Ctx, s.Addr[0])
	s.EmissionsKeeper.AddToWeightSettingWhitelist(s.Ctx, s.Addr[1])

}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestAlloraRewardsReceivesFunds() {

	rewardsModuleAddr := s.AccountKeeper.GetModuleAddress(state.AlloraRewardsAccountName)
	balanceBefore := s.BankKeeper.GetBalance(s.Ctx, rewardsModuleAddr, params.DefaultBondDenom)

	// Create a topic to test with
	topicMessage := state.MsgCreateNewTopic{
		Creator:          s.AddrStr[0],
		Metadata:         "metadata",
		WeightLogic:      "logic",
		WeightMethod:     "whatever",
		WeightCadence:    10800,
		InferenceLogic:   "morelogic",
		InferenceMethod:  "whatever2",
		InferenceCadence: 60,
	}
	_, err := s.IntegrationApp.RunMsg(
		&topicMessage,
		cosmosintegration.WithAutomaticFinalizeBlock(),
		cosmosintegration.WithAutomaticCommit(),
	)
	s.Require().NoError(err)

	_, err = s.IntegrationApp.RunMsg(
		&topicMessage,
		cosmosintegration.WithAutomaticFinalizeBlock(),
		cosmosintegration.WithAutomaticCommit(),
	)
	s.Require().NoError(err)

	// verify that the begin and end blocker were called
	// verifying the block height
	s.Require().Equal(int64(3), s.IntegrationApp.LastBlockHeight())

	// on block 3 rewards get paid for what was minted on block 2
	balanceAfter := s.BankKeeper.GetBalance(s.Ctx, rewardsModuleAddr, params.DefaultBondDenom)

	s.Require().True(balanceAfter.Amount.GT(balanceBefore.Amount))

}

func (s *IntegrationTestSuite) TestEmitRewardsSimple() {
	// Create a topic to test with
	topicMessage := state.MsgCreateNewTopic{
		Creator:          s.AddrStr[0],
		Metadata:         "metadata",
		WeightLogic:      "logic",
		WeightMethod:     "whatever",
		WeightCadence:    10800,
		InferenceLogic:   "morelogic",
		InferenceMethod:  "whatever2",
		InferenceCadence: 60,
	}

	result, err := s.IntegrationApp.RunMsg(
		&topicMessage,
		cosmosintegration.WithAutomaticFinalizeBlock(),
		cosmosintegration.WithAutomaticCommit(),
	)
	s.Require().NoError(err)

	// verify that the begin and end blocker were called
	// verifying the block height
	s.Require().Equal(int64(2), s.IntegrationApp.LastBlockHeight())

	// we now check the result
	response := &state.MsgCreateNewTopicResponse{}
	expectedTopicId := uint64(1)
	expectedResponse := &state.MsgCreateNewTopicResponse{TopicId: expectedTopicId}
	err = s.EncodingCfg.Codec.Unmarshal(result.Value, response)
	s.Require().NoError(err)
	s.Require().Equal(expectedResponse, response)

	nextTopicId, err := s.EmissionsKeeper.GetNumTopics(s.Ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedTopicId+1, nextTopicId)

	// add 2 reputers, then add 2 workers
	reputerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(100),
		cosmosMath.NewInt(200),
		cosmosMath.NewInt(300),
		cosmosMath.NewInt(400),
	}
	for i := 0; i < 2; i++ {
		reputerAddrStr := s.AddrStr[i]
		registrationMsg := &state.MsgRegister{
			Creator:      reputerAddrStr,
			LibP2PKey:    "libp2pkeyReputer" + reputerAddrStr,
			MultiAddress: "multiaddressReputer" + reputerAddrStr,
			TopicIds:     []uint64{expectedTopicId},
			InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[i].BigInt()),
			IsReputer:    true,
		}

		result, err := s.IntegrationApp.RunMsg(
			registrationMsg,
			cosmosintegration.WithAutomaticFinalizeBlock(),
			cosmosintegration.WithAutomaticCommit(),
		)
		s.Require().NoError(err)
		response := &state.MsgRegisterResponse{}
		expectedResponse := &state.MsgRegisterResponse{
			Success: true,
			Message: "Node successfully registered",
		}
		err = s.EncodingCfg.Codec.Unmarshal(result.Value, response)
		s.Require().NoError(err)
		s.Require().Equal(expectedResponse, response)
	}
	for i := 2; i < 4; i++ {
		workerAddrStr := s.AddrStr[i]
		registrationMsg := &state.MsgRegister{
			Creator:      workerAddrStr,
			LibP2PKey:    "libp2pkeyWorker" + workerAddrStr,
			MultiAddress: "multiaddressWorker" + workerAddrStr,
			TopicIds:     []uint64{expectedTopicId},
			InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[i].BigInt()),
			Owner:        s.AddrStr[0],
			IsReputer:    false,
		}
		result, err := s.IntegrationApp.RunMsg(
			registrationMsg,
			cosmosintegration.WithAutomaticFinalizeBlock(),
			cosmosintegration.WithAutomaticCommit(),
		)
		s.Require().NoError(err)
		response := &state.MsgRegisterResponse{}
		expectedResponse := &state.MsgRegisterResponse{
			Success: true,
			Message: "Node successfully registered",
		}
		err = s.EncodingCfg.Codec.Unmarshal(result.Value, response)
		s.Require().NoError(err)
		s.Require().Equal(expectedResponse, response)
	}

	topicStake, err := s.EmissionsKeeper.GetTopicStake(s.Ctx, expectedTopicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewUint(1000), topicStake)
	s.Require().Equal(int64(6), s.IntegrationApp.LastBlockHeight())

	weightsUint64 := [2][4]uint64{{10, 20, 60, 100}, {30, 40, 50, 70}}
	// have reputers set weights
	for i := 0; i < 2; i++ {
		// test panicked: runtime error: invalid memory address or nil pointer dereference
		// weights := make([]*state.Weight, 4)
		// for j := 0; j < 4; j++ {
		// 	weights = append(weights, &state.Weight{
		// 		TopicId: expectedTopicId,
		// 		Reputer: s.AddrStr[i],
		// 		Worker:  s.AddrStr[j],
		// 		Weight:  cosmosMath.NewUint(weightsUint64[i][j]),
		// 	})
		// }
		weights := []*state.Weight{
			{
				TopicId: expectedTopicId,
				Reputer: s.AddrStr[i],
				Worker:  s.AddrStr[0],
				Weight:  cosmosMath.NewUint(weightsUint64[i][0]),
			},
			{
				TopicId: expectedTopicId,
				Reputer: s.AddrStr[i],
				Worker:  s.AddrStr[1],
				Weight:  cosmosMath.NewUint(weightsUint64[i][1]),
			},
			{
				TopicId: expectedTopicId,
				Reputer: s.AddrStr[i],
				Worker:  s.AddrStr[2],
				Weight:  cosmosMath.NewUint(weightsUint64[i][2]),
			},
			{
				TopicId: expectedTopicId,
				Reputer: s.AddrStr[i],
				Worker:  s.AddrStr[3],
				Weight:  cosmosMath.NewUint(weightsUint64[i][3]),
			},
		}
		weightMessage := state.MsgSetWeights{
			Sender:  s.AddrStr[i],
			Weights: weights,
		}
		resultA, err := s.IntegrationApp.RunMsg(
			&weightMessage,
			cosmosintegration.WithAutomaticFinalizeBlock(),
			cosmosintegration.WithAutomaticCommit(),
		)
		s.Require().NoError(err)
		responseA := &state.MsgSetWeightsResponse{}
		expectedResponseA := &state.MsgSetWeightsResponse{}
		err = s.EncodingCfg.Codec.Unmarshal(resultA.Value, response)
		s.Require().NoError(err)
		s.Require().Equal(expectedResponseA, responseA)
	}
	s.Require().Equal(int64(8), s.IntegrationApp.LastBlockHeight())

	// here we check stake
}

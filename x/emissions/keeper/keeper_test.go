package keeper_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"cosmossdk.io/collections"
	cosmosAddress "cosmossdk.io/core/address"
	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	codec           codec.Codec
	addressCodec    cosmosAddress.Codec
	storeService    store.KVStoreService
	accountKeeper   authkeeper.AccountKeeper
	bankKeeper      bankkeeper.BaseKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       types.MsgServiceServer
	key             *storetypes.KVStoreKey
	privKeys        []secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
	pubKeyHexStr    []string
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()}) // nolint: exhaustruct
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	s.codec = encCfg.Codec
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)
	s.addressCodec = addressCodec

	maccPerms := map[string][]string{
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		minttypes.EcosystemModuleName:                    nil,
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

	s.privKeys, s.pubKeyHexStr, s.addrs, s.addrsStr = alloratestutil.GenerateTestAccounts(10)

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
	for _, addr := range s.addrsStr {
		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.Require().NoError(err)
	}
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) mockTopic() types.Topic {
	return types.Topic{
		Id:                       123,
		Creator:                  s.addrsStr[9],
		Metadata:                 "metadata",
		LossMethod:               "method",
		EpochLastEnded:           123,
		EpochLength:              135,
		GroundTruthLag:           135,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.5"),
		WorkerSubmissionWindow:   135,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}
}

func (s *KeeperTestSuite) signValueBundle(valueBundle *types.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := s.Require()
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	return valueBundleSignature
}

func (s *KeeperTestSuite) CreateOneTopic(epochLen int64) uint64 {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a topic first
	metadata := "Some metadata for the new topic"
	// Create a CreateNewTopicRequest message

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[0],
		Metadata:                 metadata,
		LossMethod:               "method",
		EpochLength:              epochLen,
		GroundTruthLag:           epochLen,
		WorkerSubmissionWindow:   epochLen,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}

	s.MintTokensToAddress(s.addrs[0], types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	return result.TopicId
}

/// WORKER NONCE TESTS

func (s *KeeperTestSuite) TestAddWorkerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	unfulfilledNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Empty(unfulfilledNonces.Nonces, "Unfulfilled nonces should be empty")

	// Set worker nonce
	newNonce := &types.Nonce{BlockHeight: 42}
	err = keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(newNonce.BlockHeight, unfulfilledNonces.Nonces[0].BlockHeight, "Unfulfilled nonces should contain the new nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedWorkerNonceIsUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	isUnfulfilled, err := keeper.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "non existent nonce should not be listed as unfulfilled")

	// Set worker nonce
	err = keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	isUnfulfilled, err = keeper.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "new nonce should be unfulfilled")
}

func (s *KeeperTestSuite) TestCanFulfillNewWorkerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	// Set worker nonce
	err := keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	isUnfulfilled, err := keeper.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "new nonce should not be unfulfilled")

	// Fulfill the nonce
	success, err := keeper.FulfillWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(success, "nonce should be able to be fulfilled")

	// Check that the nonce is no longer unfulfilled
	isUnfulfilled, err = keeper.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "new nonce should be fulfilled")
}

func (s *KeeperTestSuite) TestGetMultipleUnfulfilledWorkerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *KeeperTestSuite) TestGetAndFulfillMultipleUnfulfilledWorkerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}
	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues), "Should match the number of unfulfilled nonces")

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		success, err := keeper.FulfillWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().True(success, "Nonce should be successfully fulfilled")
		s.Require().NoError(err, "Error fulfilling nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err = keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{46, 44, 42} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.BlockHeight, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestWorkerNonceLimitEnforcement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	maxUnfulfilledRequests := 3
	// Set the maximum number of unfulfilled worker nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledWorkerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, maxUnfulfilledRequests, "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.BlockHeight, "Nonce should match the expected recent nonce")
	}
}

/// REPUTER NONCE TESTS

func (s *KeeperTestSuite) TestAddReputerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	unfulfilledNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Empty(unfulfilledNonces.Nonces, "Unfulfilled nonces should be empty")

	// Set reputer nonce
	newReputerNonce := &types.Nonce{BlockHeight: 42}
	err = keeper.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(
		newReputerNonce.BlockHeight,
		unfulfilledNonces.Nonces[0].ReputerNonce.BlockHeight,
		"Unfulfilled nonces should contain the new reputer nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedReputerNonceIsUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{BlockHeight: 42}

	isUnfulfilled, err := keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Non-existent nonce should not be listed as unfulfilled")

	// Set reputer nonce
	err = keeper.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	isUnfulfilled, err = keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "New nonce should be unfulfilled")
}

func (s *KeeperTestSuite) TestCanFulfillNewReputerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{BlockHeight: 42}

	// Set reputer nonce
	err := keeper.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	// Check that the nonce is the correct nonce
	isUnfulfilled, err := keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "New nonce should be unfulfilled")

	// Fulfill the nonce
	nonceIsUnfulfilled, err := keeper.FulfillReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(nonceIsUnfulfilled, "Nonce should be able to be fulfilled")

	// Check that the nonce is no longer unfulfilled
	isUnfulfilled, err = keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "New nonce should be fulfilled")
}

func (s *KeeperTestSuite) TestGetAndFulfillMultipleUnfulfilledReputerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple reputer nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		nonceIsUnfulfilled, err := keeper.FulfillReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Error fulfilling nonce")
		s.Require().True(nonceIsUnfulfilled, "Nonce should be able to be fulfilled")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{46, 44, 42} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.ReputerNonce.BlockHeight, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestReputerNonceLimitEnforcement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	maxUnfulfilledRequests := 3

	// Set the maximum number of unfulfilled reputer nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledReputerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled reputer nonces via the SetParams method
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Failed to set parameters")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err := keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, maxUnfulfilledRequests, "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.ReputerNonce.BlockHeight, "Nonce should match the expected recent nonce")
	}
}

/// REGRET TESTS

func (s *KeeperTestSuite) TestSetAndGetInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[1]
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	// Set Inferer Network Regret
	err := keeper.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	gotRegret, _, err := keeper.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[3] // Assuming sdk.AccAddress is initialized with a string representing the address

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	// Set Forecaster Network Regret
	err := keeper.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	gotRegret, _, err := keeper.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestSetAndGetOneInForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := s.addrsStr[3]
	inferer := s.addrsStr[1]

	regret := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(30)}

	// Set One-In Forecaster Network Regret
	err := keeper.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One-In Forecaster Network Regret
	gotRegret, _, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentInfererRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := s.addrsStr[1]

	// Topic IDs
	topicId1 := s.CreateOneTopic(10800)
	topicId2 := s.CreateOneTopic(10800)

	// Zero regret for initial check
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := keeper.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := keeper.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err = keeper.SetInfererNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetInfererNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = keeper.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = keeper.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentForecasterRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := s.addrsStr[1]

	// Topic IDs
	topicId1 := s.CreateOneTopic(10800)
	topicId2 := s.CreateOneTopic(10800)

	// Regrets
	noRagret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	gotRegret1, _, err := keeper.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRagret, gotRegret1)

	// Set regrets for the same worker under different topic IDs
	err = keeper.SetForecasterNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetForecasterNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, _, err = keeper.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err := keeper.GetForecasterNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentOneInForecasterNetworkRegrets() {
	ctx := s.ctx
	s.CreateOneTopic(10800) // Topic 1
	s.CreateOneTopic(10800) // Topic 2
	keeper := s.emissionsKeeper
	forecaster := s.addrsStr[3]
	inferer := s.addrsStr[1]

	// Topic IDs
	topicId1 := uint64(1)
	topicId2 := uint64(2)

	// Zero regret for initial checks
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same forecaster-inferer pair under different topic IDs
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer, regret1)
	s.Require().NoError(err)
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = keeper.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = keeper.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestSetAndGetNaiveInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	inferer := s.addrsStr[1]

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	err := keeper.SetNaiveInfererNetworkRegret(ctx, topicId, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := keeper.GetNaiveInfererNetworkRegret(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(15)}

	err := keeper.SetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := keeper.GetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	inferer := s.addrsStr[1]
	forecaster := s.addrsStr[3]

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	err := keeper.SetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster, regret)
	s.Require().NoError(err)

	gotRegret, _, err := keeper.GetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := s.addrsStr[3]
	inferer := s.addrsStr[1]

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(25)}

	err := keeper.SetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := keeper.GetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster1 := s.addrsStr[3]
	forecaster2 := s.addrsStr[4]

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(30)}

	err := keeper.SetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := keeper.GetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

// / PARAMS TESTS
func (s *KeeperTestSuite) TestSetGetMaxTopicsPerBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(100)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxActiveTopicsPerBlock
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetRemoveStakeDelayWindow() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(50)

	// Set the parameter
	params := types.DefaultParams()
	params.RemoveStakeDelayWindow = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RemoveStakeDelayWindow
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetValidatorsVsAlloraPercentReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := alloraMath.MustNewDecFromString("0.25") // Assume a function to create LegacyDec

	// Set the parameter
	params := types.DefaultParams()
	params.ValidatorsVsAlloraPercentReward = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.ValidatorsVsAlloraPercentReward
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinTopicUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := alloraMath.NewDecFromInt64(300)

	// Set the parameter
	params := types.DefaultParams()
	params.MinTopicWeight = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinTopicWeight
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsRequiredMinimumStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue, ok := cosmosMath.NewIntFromString("500")
	s.Require().True(ok)

	// Set the parameter
	params := types.DefaultParams()
	params.RequiredMinimumStake = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RequiredMinimumStake
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinEpochLength() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(720)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLength = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLength
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsEpsilon() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := alloraMath.MustNewDecFromString("0.1234")

	// Set the parameter
	params := types.DefaultParams()
	params.EpsilonReputer = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.EpsilonReputer
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsTopicCreationFee() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.NewInt(1000)

	// Set the parameter
	params := types.DefaultParams()
	params.CreateTopicFee = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.CreateTopicFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsRegistrationFee() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.NewInt(500)

	// Set the parameter
	params := types.DefaultParams()
	params.RegistrationFee = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RegistrationFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsMaxSamplesToScaleScores() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(1500)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSamplesToScaleScores
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxTopInferersToReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopInferersToReward = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopInferersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopInferersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecastersToReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopForecastersToReward = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopForecastersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopForecastersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecasterElementToSubmit() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxElementsPerForecast = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxElementsPerForecast
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxElementsPerForecast should match the expected value")
}

func (s *KeeperTestSuite) TestGetMinEpochLengthRecordLimit() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := int64(10)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLengthRecordLimit = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLengthRecordLimit
	s.Require().Equal(expectedValue, actualValue, "The retrieved MinEpochLengthRecordLimit should be equal to the expected value")
}

func (s *KeeperTestSuite) TestGetMaxSerializedMsgLength() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := int64(2048)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSerializedMsgLength = expectedValue
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSerializedMsgLength
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxSerializedMsgLength should be equal to the expected value")
}

/// INFERENCES, FORECASTS

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(2),
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertActiveInferences correctly sets up inferences
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := keeper.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	// Retrieve inferences
	actualInferences, err := keeper.GetInferencesAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, actualInferences)
}

func (s *KeeperTestSuite) TestGetLatestTopicInferences() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topicId := uint64(1)

	// Initially, there should be no inferences, so we expect an empty result
	emptyInferences, emptyBlockHeight, err := keeper.GetLatestTopicInferences(ctx, topicId)
	s.Require().NoError(err, "Retrieving latest inferences when none exist should not result in an error")
	s.Require().Equal(&types.Inferences{Inferences: []*types.Inference{}}, emptyInferences, "Expected no inferences initially")
	s.Require().Equal(types.BlockHeight(0), emptyBlockHeight, "Expected block height to be zero initially")

	// Insert first set of inferences
	blockHeight1 := types.BlockHeight(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = keeper.InsertActiveInferences(ctx, topicId, nonce1.BlockHeight, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t",
		Value:       alloraMath.MustNewDecFromString("20"),
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = keeper.InsertActiveInferences(ctx, topicId, nonce2.BlockHeight, inferences2)
	s.Require().NoError(err, "Inserting second set of inferences should not fail")

	// Retrieve the latest inferences
	latestInferences, latestBlockHeight, err := keeper.GetLatestTopicInferences(ctx, topicId)
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}

func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	workerAccStr := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"

	_, err := keeper.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().Error(err, "Retrieving an inference that does not exist should result in an error")

	blockHeight1 := int64(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = keeper.InsertInference(ctx, topicId, newInference1)
	s.Require().NoError(err, "Inserting inferences should not fail")

	blockHeight2 := int64(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Value:       alloraMath.MustNewDecFromString("10"),
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = keeper.InsertInference(ctx, topicId, newInference2)
	s.Require().NoError(err, "Inserting inferences should not fail")

	retrievedInference, err := keeper.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().NoError(err, "Retrieving an existing inference should not fail")
	s.Require().Equal(newInference2, retrievedInference, "Retrieved inference should match the inserted one")
}

func (s *KeeperTestSuite) TestGetForecastsAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}

	// Assume InsertActiveForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: block}
	err := keeper.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	// Retrieve forecasts
	actualForecasts, err := keeper.GetForecastsAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedForecasts, actualForecasts)
}

func (s *KeeperTestSuite) TestInsertActiveReputerLosses() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: block},
		},
		Reputer:                       s.addrsStr[0],
		ExtraData:                     []byte("data"),
		CombinedValue:                 alloraMath.MustNewDecFromString("123"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("123"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature := s.signValueBundle(valueBundle, s.privKeys[0])
	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: valueBundle,
				Signature:   signature,
				Pubkey:      s.pubKeyHexStr[0],
			},
		},
	}

	// Test inserting data
	err := s.emissionsKeeper.InsertActiveReputerLosses(ctx, topicId, block, reputerLossBundles)
	require.NoError(err, "InsertActiveReputerLosses should not return an error")

	// Retrieve data to verify insertion
	result, err := s.emissionsKeeper.GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetReputerLossBundlesAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Test getting data before any insert, should return error or nil
	result, err := s.emissionsKeeper.GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.Empty(result.ReputerValueBundles, "Result should be empty for non-existent data")
}

func (s *KeeperTestSuite) TestInsertNetworkLossBundleAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	lossBundle := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: block},
		},
		Reputer:                       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:                     []byte("data"),
		CombinedValue:                 alloraMath.MustNewDecFromString("123"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("123"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}

	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)
	require.NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Verify the insertion
	result, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

// this unit test needs to be completely rewritten, PROTO-2575
func (s *KeeperTestSuite) TestGetNetworkLossBundleAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Attempt to retrieve before insertion
	result, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err, "Should not return error for empty loss bundle")
	require.NotNil(result)
	require.Equal(topicId, result.TopicId, "Result should be not be nil for non-existent data")
}

func (s *KeeperTestSuite) TestGetLatestNetworkLossBundle() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)

	// Initially, there should be no loss bundle, so we expect a zero result
	emptyLossBundle, err := keeper.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().ErrorIs(err, types.ErrNotFound)
	s.Require().Nil(emptyLossBundle, "Expected no network loss bundle initially")

	// Insert first network loss bundle
	blockHeight1 := types.BlockHeight(100)
	lossBundle1 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight1},
		},
		Reputer:                       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:                     []byte("data"),
		CombinedValue:                 alloraMath.MustNewDecFromString("123"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("123"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	err = keeper.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight1, lossBundle1)
	s.Require().NoError(err, "Inserting first network loss bundle should not fail")

	// Insert second network loss bundle
	blockHeight2 := types.BlockHeight(200)
	lossBundle2 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight2},
		},
		Reputer:                       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:                     []byte("data"),
		CombinedValue:                 alloraMath.MustNewDecFromString("456"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("123"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	err = keeper.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight2, lossBundle2)
	s.Require().NoError(err, "Inserting second network loss bundle should not fail")

	// Retrieve the latest network loss bundle
	latestLossBundle, err := keeper.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().NoError(err, "Retrieving latest network loss bundle should not fail")
	s.Require().Equal(&lossBundle2, latestLossBundle, "Latest network loss bundle should match the second inserted set")
}

// ########################################
// #           Staking tests              #
// ########################################

func (s *KeeperTestSuite) TestGetSetTotalStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Set total stake
	newTotalStake := cosmosMath.NewInt(1000)
	err := keeper.SetTotalStake(ctx, newTotalStake)
	s.Require().NoError(err)

	// Check total stake
	totalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(newTotalStake, totalStake)
}

func (s *KeeperTestSuite) TestAddReputerStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := s.addrs[0].String()
	stakeAmount := cosmosMath.NewInt(500)

	// Initial Values
	initialTotalStake := cosmosMath.NewInt(0)
	initialTopicStake := cosmosMath.NewInt(0)

	// Add stake
	err := keeper.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, delegatorStake, "Delegator stake should be equal to stake amount after addition")

	// Check updated topic stake
	topicStake, err := keeper.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(initialTopicStake.Add(stakeAmount), topicStake, "Topic stake should be incremented by stake amount after addition")

	// Check updated total stake
	totalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Add(stakeAmount), totalStake, "Total stake should be incremented by stake amount after addition")
}

func (s *KeeperTestSuite) TestAddDelegateStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := s.addrsStr[0]
	reputerAddr := s.addrsStr[1]
	initialStakeAmount := cosmosMath.NewInt(500)
	additionalStakeAmount := cosmosMath.NewInt(300)

	// Setup initial stake
	err := keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount, delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")

	// Add additional stake
	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err = keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
}

func (s *KeeperTestSuite) TestAddReputerStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := s.addrs[0].String()
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Try to add zero stake
	err := keeper.AddReputerStake(ctx, topicId, delegatorAddr, zeroStakeAmount)
	s.Require().ErrorIs(err, types.ErrInvalidValue)
}

func (s *KeeperTestSuite) TestRemoveStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := s.addrs[0].String()
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = keeper.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes after adding stake
	initialTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)

	// make a request to remove stake
	err = keeper.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake
	err = keeper.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator after removal
	delegatorStake, err := keeper.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), delegatorStake, "Delegator stake should be zero after removal")

	// Check updated topic stake after removal
	topicStake, err := keeper.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")

	// Check updated total stake after removal
	finalTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().True(initialTotalStake.Sub(stakeAmount).Equal(finalTotalStake), "Total stake should be decremented by stake amount after removal")
}

func (s *KeeperTestSuite) TestRemovePartialStakeFromDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := s.addrsStr[0]
	reputerAddr := s.addrsStr[1]
	initialStakeAmount := cosmosMath.NewInt(1000)
	removeStakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = keeper.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                removeStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := keeper.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := s.addrsStr[0]
	reputerAddr := s.addrsStr[1]
	initialStakeAmount := cosmosMath.NewInt(1000)
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = keeper.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                initialStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for Reputer
	stakeUponReputer, err := keeper.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := s.addrsStr[0]
	initialStakeAmount := cosmosMath.NewInt(500)
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Setup initial stake
	err := keeper.AddReputerStake(ctx, topicId, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Try to remove zero stake
	err = keeper.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, reputerAddr, zeroStakeAmount)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	nonExistingDelegatorAddr := s.addrsStr[0]
	stakeAmount := cosmosMath.NewInt(500)

	// Try to remove stake with non-existing delegator or target
	err := keeper.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, nonExistingDelegatorAddr, stakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGetAllStakeForDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	delegatorAddr := s.addrsStr[0]

	// Mock setup
	topicId := uint64(1)
	targetAddr := s.addrsStr[1]
	stakeAmount := cosmosMath.NewInt(500)

	// Add stake to create bonds
	err := keeper.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Add stake to create bonds
	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount.Mul(cosmosMath.NewInt(2)))
	s.Require().NoError(err)

	// Get all bonds for delegator
	amount, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)

	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
	s.Require().Equal(stakeAmount.Mul(cosmosMath.NewInt(3)), amount, "The total amount is incorrect")
}

func (s *KeeperTestSuite) TestSetGetDeleteStakeRemovalByAddressWithDetailedPlacement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic0 := uint64(101)
	reputer0 := "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n"

	topic1 := uint64(102)
	reputer1 := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	// Create a sample stake removal information
	removalInfo0 := types.StakeRemovalInfo{
		BlockRemovalStarted:   12,
		BlockRemovalCompleted: 13,
		TopicId:               topic0,
		Reputer:               reputer0,
		Amount:                cosmosMath.NewInt(100),
	}
	removalInfo1 := types.StakeRemovalInfo{
		BlockRemovalStarted:   13,
		BlockRemovalCompleted: 14,
		TopicId:               topic1,
		Reputer:               reputer1,
		Amount:                cosmosMath.NewInt(200),
	}

	// Set stake removal information
	err := keeper.SetStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = keeper.SetStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 101

	// Retrieve the stake removal information
	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 1)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 1, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit, "The limit should not be hit")
	s.Require().Equal(removalInfo0.BlockRemovalStarted, retrievedInfo[0].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo0.BlockRemovalCompleted, retrievedInfo[0].BlockRemovalCompleted, "Block removal completed should match")
	s.Require().Equal(removalInfo0.TopicId, retrievedInfo[0].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo0.Reputer, retrievedInfo[0].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo0.Amount, retrievedInfo[0].Amount, "Amounts should match for all placements")

	// Topic 102

	// Retrieve the stake removal information
	retrievedInfo, limitHit, err = keeper.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 2)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit, "The limit should not be hit")
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.BlockRemovalCompleted, retrievedInfo[1].BlockRemovalCompleted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = keeper.DeleteStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")

	// delete 102
	err = keeper.DeleteStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err = keeper.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Attempt to retrieve stake removal info for an address with no set info
	removals, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, 202, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitPreviousBlocks() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.addrsStr[2]
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := keeper.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+1, // note how we are getting a block AFTER blockRemovalsEnd
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitExactBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.addrsStr[2]
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := keeper.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd,
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitGreaterThanNumRemovals() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	numRemovals := int64(5)
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.addrsStr[2]
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := keeper.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(numRemovals),
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, int(numRemovals))
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitLessThanNumRemovals() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	numRemovals := int64(5)
	limitRemovals := numRemovals - 2
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.addrsStr[2]
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := keeper.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(limitRemovals),
	)
	s.Require().NoError(err)
	s.Require().True(limitHit)
	s.Require().Len(retrievedInfo, int(limitRemovals))
}

func (s *KeeperTestSuite) TestSetGetDeleteDelegateStakeRemovalByAddress() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic0 := uint64(201)
	reputer0 := "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n"
	delegator0 := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	topic1 := uint64(202)
	reputer1 := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"
	delegator1 := "allo16skpmhw8etsu70kknkmxquk5ut7lsewgtqqtlu"

	// Create sample delegate stake removal information
	removalInfo0 := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   12,
		BlockRemovalCompleted: 13,
		TopicId:               topic0,
		Reputer:               reputer0,
		Delegator:             delegator0,
		Amount:                cosmosMath.NewInt(300),
	}
	removalInfo1 := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   13,
		BlockRemovalCompleted: 14,
		TopicId:               topic1,
		Reputer:               reputer1,
		Delegator:             delegator1,
		Amount:                cosmosMath.NewInt(400),
	}

	// Set delegate stake removal information
	err := keeper.SetDelegateStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = keeper.SetDelegateStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 201

	// Retrieve the delegate stake removal information
	retrievedInfo, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 1, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit)
	s.Require().Equal(removalInfo0.BlockRemovalStarted, retrievedInfo[0].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo0.TopicId, retrievedInfo[0].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo0.Reputer, retrievedInfo[0].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo0.Delegator, retrievedInfo[0].Delegator, "Delegator addresses should match for all placements")
	s.Require().Equal(removalInfo0.Amount, retrievedInfo[0].Amount, "Amounts should match for all placements")

	// Topic 202

	// Retrieve the delegate stake removal information
	retrievedInfo, limitHit, err = keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2)
	s.Require().False(limitHit)
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Delegator, retrievedInfo[1].Delegator, "Delegator addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = keeper.DeleteDelegateStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer, removalInfo0.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)

	// delete 102
	err = keeper.DeleteDelegateStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer, removalInfo1.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err = keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)
}

func (s *KeeperTestSuite) TestGetDeleteDelegateStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Create sample delegate stake removal information
	removalInfo := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   int64(12),
		BlockRemovalCompleted: int64(13),
		TopicId:               uint64(201),
		Reputer:               "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n",
		Delegator:             "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		Amount:                cosmosMath.NewInt(300),
	}

	// Set delegate stake removal information
	err := keeper.SetDelegateStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	_, err = keeper.GetDelegateStakeRemoval(ctx,
		removalInfo.BlockRemovalStarted,
		removalInfo.TopicId,
		removalInfo.Delegator,
		removalInfo.Reputer,
	)
	// index is on BlockRemovalCompleted not BlockRemovalStarted
	s.Require().Error(err)

	retrievedInfo, err := keeper.GetDelegateStakeRemoval(ctx,
		removalInfo.BlockRemovalCompleted,
		removalInfo.TopicId,
		removalInfo.Delegator,
		removalInfo.Reputer,
	)
	s.Require().NoError(err)

	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted)
	s.Require().Equal(removalInfo.TopicId, retrievedInfo.TopicId)
	s.Require().Equal(removalInfo.Reputer, retrievedInfo.Reputer)
	s.Require().Equal(removalInfo.Delegator, retrievedInfo.Delegator)
	s.Require().Equal(removalInfo.Amount, retrievedInfo.Amount)
}

func (s *KeeperTestSuite) TestGetDelegateStakeRemovalByAddressNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Attempt to retrieve delegate stake removal info for an address with no set info
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, 201, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestSetParams() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	params := types.DefaultParams()
	// Set params
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Check params
	paramsFromKeeper, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, paramsFromKeeper, "Params should be equal to the set params")
}

// / REPUTERS AND WORKER
func (s *KeeperTestSuite) TestInsertWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: worker,
	}

	// Attempt to insert the worker for multiple topics
	err := keeper.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)

	node, err := keeper.GetWorkerInfo(ctx, worker)

	s.Require().NoError(err)
	s.Require().Equal(workerInfo.Owner, node.Owner)
	s.Require().Equal(workerInfo.NodeAddress, node.NodeAddress)
}

func (s *KeeperTestSuite) TestRemoveWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401) // Assume the worker is associated with this topicId initially

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: "allo195jgulwj7vd292m0fth5gwqu4r2447dnarunmx",
	}

	// Insert the worker
	insertErr := keeper.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(insertErr, "Failed to insert worker initially")

	// Verify the worker is registered in the topic
	isRegisteredPre, preErr := keeper.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(preErr, "Failed to check worker registration before removal")
	s.Require().True(isRegisteredPre, "Worker should be registered in the topic before removal")

	// Perform the removal
	removeErr := keeper.RemoveWorker(ctx, topicId, worker)
	s.Require().NoError(removeErr, "Failed to remove worker")

	// Verify the worker is no longer registered in the topic
	isRegisteredPost, postErr := keeper.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(postErr, "Failed to check worker registration after removal")
	s.Require().False(isRegisteredPost, "Worker should not be registered in the topic after removal")
}

func (s *KeeperTestSuite) TestInsertReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputer := s.addrsStr[0]
	topicId := uint64(501)

	// Define sample OffchainNode information for a reputer
	reputerInfo := types.OffchainNode{
		Owner:       s.addrsStr[1],
		NodeAddress: s.addrsStr[2],
	}

	// Attempt to insert the reputer for multiple topics
	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	// Optionally check if reputer is registered in each topic using an assumed IsReputerRegisteredInTopic method
	isRegistered, regErr := keeper.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(regErr, "Checking reputer registration should not fail")
	s.Require().True(isRegistered, "Reputer should be registered in each topic")
}

func (s *KeeperTestSuite) TestGetReputerInfo() {
	ctx := s.ctx
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)
	keeper := s.emissionsKeeper
	reputerInfo := types.OffchainNode{
		Owner:       s.addrsStr[2],
		NodeAddress: reputer,
	}

	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	actualReputer, err := keeper.GetReputerInfo(ctx, reputer)
	s.Require().NoError(err)
	s.Require().Equal(reputerInfo, actualReputer)

	nonExistentKey := "nonExistentKey123"
	_, err = keeper.GetReputerInfo(ctx, nonExistentKey)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)

	// Pre-setup: Insert the reputer for initial setup
	err := keeper.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		Owner:       s.addrsStr[1],
		NodeAddress: reputer,
	})
	s.Require().NoError(err, "InsertReputer failed during setup")

	// Verify the reputer is registered in the topic
	isRegisteredPre, preErr := keeper.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(preErr, "Failed to check reputer registration before removal")
	s.Require().True(isRegisteredPre, "Reputer should be registered in the topic before removal")

	// Perform the removal
	removeErr := keeper.RemoveReputer(ctx, topicId, reputer)
	s.Require().NoError(removeErr, "Failed to remove reputer")

	// Verify the reputer is no longer registered in the topic
	isRegisteredPost, postErr := keeper.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(postErr, "Failed to check reputer registration after removal")
	s.Require().False(isRegisteredPost, "Reputer should not be registered in the topic after removal")
}

/// TOPICS

func (s *KeeperTestSuite) TestSetAndGetPreviousTopicWeight() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := alloraMath.NewDecFromInt64(10)
	err := keeper.SetPreviousTopicWeight(ctx, topicId, weightToSet)
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Get the previously set topic weight
	retrievedWeight, noPrior, err := keeper.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err, "Getting previous topic weight should not fail")
	s.Require().Equal(weightToSet, retrievedWeight, "Retrieved weight should match the set weight")
	s.Require().False(noPrior, "Should indicate prior weight for a set topic")
}

func (s *KeeperTestSuite) TestGetPreviousTopicWeightNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(2)

	// Attempt to get a weight for a topic that has no set weight
	retrievedWeight, noPrior, err := keeper.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err, "Getting weight for an unset topic should not error but return zero value")
	s.Require().True(alloraMath.ZeroDec().Equal(retrievedWeight), "Weight for an unset topic should be zero")
	s.Require().True(noPrior, "Should indicate no prior weight for an unset topic")
}

func (s *KeeperTestSuite) TestInactivateAndActivateTopic() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(3)

	maxActiveTopicsNum := uint64(5)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Assume topic initially active
	initialTopic := s.mockTopic()
	err = keeper.SetTopic(ctx, topicId, initialTopic)
	s.Require().NoError(err, "Setting topic should not fail")

	// Activate the topic
	err = keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active
	topicActive, err := keeper.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")

	// Inactivate the topic
	err = keeper.InactivateTopic(ctx, topicId)
	s.Require().NoError(err, "Inactivating topic should not fail")

	// Check if topic is inactive
	topicActive, err = keeper.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after inactivation")
	s.Require().False(topicActive, "Topic should be inactive")

	// Activate the topic
	err = keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active again
	topicActive, err = keeper.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")
}

func (s *KeeperTestSuite) TestGetActiveTopicIdsAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	maxActiveTopicsNum := uint64(2)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	params.MaxPageLimit = 100
	params.MinEpochLength = 1
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	topic1 := s.mockTopic()
	topic1.Id = 1
	topic1.EpochLength = 5
	topic1.WorkerSubmissionWindow = topic1.EpochLength
	topic2 := s.mockTopic()
	topic2.Id = 2
	topic2.EpochLength = 5
	topic2.WorkerSubmissionWindow = topic2.EpochLength
	topic3 := s.mockTopic()
	topic3.Id = 3
	topic3.EpochLength = 15
	topic3.WorkerSubmissionWindow = topic3.EpochLength

	err = keeper.SetTopic(ctx, topic1.Id, topic1)
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.ActivateTopic(ctx, topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")
	err = keeper.SetTopic(ctx, topic2.Id, topic2) // Inactive topic
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.SetTopic(ctx, topic3.Id, topic3)
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.ActivateTopic(ctx, topic3.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	// Fetch only active topics
	activeTopics, err := keeper.GetActiveTopicIdsAtBlock(ctx, 5)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 1, "Should retrieve exactly one active topic")

	activeTopics, err = keeper.GetActiveTopicIdsAtBlock(ctx, 15)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 1, "Should retrieve exactly one active topic")
	s.Require().Equal(activeTopics.TopicIds[0], topic3.Id, "The details of topic 1 should match")
}

func (s *KeeperTestSuite) TestTopicGoesInactivateOnEpochEndBlockIfLowWeight() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = uint64(3)
	params.MaxPageLimit = uint64(100)
	params.MinEpochLength = 1
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.MustNewDecFromString("1")
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	topic1 := s.mockTopic()
	topic1.Id = 1
	topic1.EpochLength = 15
	topic1.WorkerSubmissionWindow = topic1.EpochLength
	topic2 := s.mockTopic()
	topic2.Id = 2
	topic2.EpochLength = 15
	topic2.WorkerSubmissionWindow = topic2.EpochLength
	topic3 := s.mockTopic()
	topic3.Id = 3
	topic3.EpochLength = 5
	topic3.WorkerSubmissionWindow = topic3.EpochLength
	topic4 := s.mockTopic()
	topic4.Id = 4
	topic4.EpochLength = 5
	topic4.WorkerSubmissionWindow = topic4.EpochLength

	setTopicWeight := func(topicId uint64, revenue, stake int64) {
		err := keeper.AddTopicFeeRevenue(ctx, topicId, cosmosMath.NewInt(revenue))
		s.Require().NoError(err, "Adding topic fee revenue should not fail")
		err = keeper.SetTopicStake(ctx, topicId, cosmosMath.NewInt(stake))
		s.Require().NoError(err, "Setting topic stake should not fail")
	}

	setTopicWeight(topic1.Id, 10, 10)
	err = keeper.SetTopic(ctx, topic1.Id, topic1)
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.ActivateTopic(ctx, topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	setTopicWeight(topic2.Id, 20, 10)
	err = keeper.SetTopic(ctx, topic2.Id, topic2)
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.ActivateTopic(ctx, topic2.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	// Fetch next page -- should only return topic 5
	activeTopics, err := keeper.GetActiveTopicIdsAtBlock(ctx, 15)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 2, "Should retrieve exactly two active topics")

	ctx = s.ctx.WithBlockHeight(15)
	err = keeper.AttemptTopicReactivation(ctx, topic1.Id)
	s.Require().NoError(err, "Attempting to reactivate topic should not fail")
	err = keeper.AttemptTopicReactivation(ctx, topic2.Id)
	s.Require().NoError(err, "Attempting to reactivate topic should not fail")

	ctx = s.ctx.WithBlockHeight(25)
	setTopicWeight(topic3.Id, 50, 10)
	err = keeper.SetTopic(ctx, topic3.Id, topic3)
	s.Require().NoError(err, "Setting topic should not fail")
	err = keeper.ActivateTopic(ctx, topic3.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err = keeper.GetActiveTopicIdsAtBlock(ctx, 30)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 3, "Should retrieve exactly two active topics")
	s.Require().Equal(uint64(1), activeTopics.TopicIds[0])
	s.Require().Equal(uint64(2), activeTopics.TopicIds[1])
	s.Require().Equal(uint64(3), activeTopics.TopicIds[2])

	ctx = s.ctx.WithBlockHeight(30)
	setTopicWeight(topic4.Id, 1, 1)
	isActive, err := keeper.IsTopicActive(ctx, topic4.Id)
	s.Require().NoError(err, "Is topic active should not produce an error")
	s.Require().False(isActive, "Topic4 should not be activated")
}
func (s *KeeperTestSuite) TestIncrementTopicId() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Initial check for the current topic ID
	initialTopicId, err := keeper.IncrementTopicId(ctx)
	s.Require().NoError(err, "Getting initial topic ID should not fail")

	// Increment the topic ID
	newTopicId, err := keeper.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")
	s.Require().Equal(initialTopicId+1, newTopicId, "New topic ID should be one more than the initial topic ID")
}

func (s *KeeperTestSuite) TestGetNumTopicsWithActualTopicCreation() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	nextTopicIdStart, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the number of topics should not fail")

	// Create multiple topics to simulate actual usage
	topicsToCreate := 5
	for i := 1; i <= topicsToCreate; i++ {
		topicId, err := keeper.IncrementTopicId(ctx)
		s.Require().NoError(err, "Incrementing topic ID should not fail")

		newTopic := s.mockTopic()
		newTopic.Id = topicId

		err = keeper.SetTopic(ctx, topicId, newTopic)
		s.Require().NoError(err, "Setting a new topic should not fail")
	}

	// Now retrieve the total number of topics
	nextTopicIdEnd, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the number of topics should not fail")
	s.Require().Equal(uint64(topicsToCreate), nextTopicIdEnd-nextTopicIdStart)
}

func (s *KeeperTestSuite) TestUpdateAndGetTopicEpochLastEnded() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	epochLastEnded := types.BlockHeight(100)

	// Setup a topic initially
	initialTopic := s.mockTopic()
	err := keeper.SetTopic(ctx, topicId, initialTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Update the epoch last ended
	err = keeper.UpdateTopicEpochLastEnded(ctx, topicId, epochLastEnded)
	s.Require().NoError(err, "Updating topic epoch last ended should not fail")

	// Retrieve the last ended epoch for the topic
	topic, err := keeper.GetTopic(ctx, topicId)
	s.Require().NoError(err, "Retrieving topic epoch last ended should not fail")
	s.Require().Equal(epochLastEnded, topic.EpochLastEnded, "The retrieved epoch last ended should match the updated value")
}

func (s *KeeperTestSuite) TestTopicExists() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test a topic ID that does not exist
	nonExistentTopicId := uint64(999) // Assuming this ID has not been used
	exists, err := keeper.TopicExists(ctx, nonExistentTopicId)
	s.Require().NoError(err, "Checking existence for a non-existent topic should not fail")
	s.Require().False(exists, "No topic should exist for an unused topic ID")

	// Create a topic to test existence
	existentTopicId, err := keeper.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")

	newTopic := s.mockTopic()
	err = keeper.SetTopic(ctx, existentTopicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test the newly created topic ID
	exists, err = keeper.TopicExists(ctx, existentTopicId)
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

func (s *KeeperTestSuite) TestGetTopic() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	_, err := keeper.GetTopic(ctx, topicId)
	s.Require().Error(err, "Retrieving a non-existent topic should result in an error")

	newTopic := s.mockTopic()

	err = keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	retrievedTopic, err := keeper.GetTopic(ctx, topicId)
	s.Require().NoError(err, "Retrieving an existent topic should not fail")
	s.Require().Equal(newTopic, retrievedTopic, "Retrieved topic should match the set topic")
	s.Require().Equal(newTopic.Metadata, retrievedTopic.Metadata, "Retrieved topic should match the set topic")
}

/// FEE REVENUE

func (s *KeeperTestSuite) TestGetTopicFeeRevenue() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	newTopic := s.mockTopic()
	err := keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test getting revenue for a topic with no existing revenue
	feeRev, err := keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when revenue does not exist")
	s.Require().Equal(cosmosMath.ZeroInt(), feeRev, "Revenue should be zero for non-existing entries")

	// Setup a topic with some revenue
	initialRevenue := cosmosMath.NewInt(100)
	initialRevenueInt := cosmosMath.NewInt(100)
	err = keeper.AddTopicFeeRevenue(ctx, topicId, initialRevenue)
	s.Require().NoError(err, "Adding initial revenue should not fail")

	// Test getting revenue for a topic with existing revenue
	feeRev, err = keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when retrieving existing revenue")
	s.Require().Equal(feeRev.String(), initialRevenueInt.String(), "Revenue should match the initial setup")
}

func (s *KeeperTestSuite) TestAddTopicFeeRevenue() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	block := int64(100)

	newTopic := types.Topic{
		Creator:                  s.addrs[0].String(),
		Id:                       topicId,
		Metadata:                 "metadata",
		LossMethod:               "LOSS",
		EpochLastEnded:           10,
		EpochLength:              20,
		GroundTruthLag:           20,
		PNorm:                    alloraMath.MustNewDecFromString("3.5"),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.5"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.5"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.5"),
		WorkerSubmissionWindow:   20,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.5"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.5"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.5"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.5"),
	}
	err := keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")
	err = keeper.DripTopicFeeRevenue(ctx, topicId, block)
	s.Require().NoError(err, "Resetting topic fee revenue should not fail")

	// Add initial revenue
	initialAmount := cosmosMath.NewInt(100)
	err = keeper.AddTopicFeeRevenue(ctx, topicId, initialAmount)
	s.Require().NoError(err, "Adding initial revenue should not fail")

	// Verify initial revenue
	feeRev, err := keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Getting topic fee revenue should not fail")
	s.Require().Equal(initialAmount, feeRev, "Initial revenue should be correctly recorded")
}

/// SCORES

func (s *KeeperTestSuite) TestGetScoreEmas() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"

	// Test getting latest scores when none are set
	infererScore, err := keeper.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching latest inferer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     worker,
		Score:       alloraMath.ZeroDec(),
	}, infererScore, "Inferer score should be zero if not set")

	forecasterScore, err := keeper.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err, "Fetching latest forecaster score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     forecaster,
		Score:       alloraMath.ZeroDec(),
	}, forecasterScore, "Forecaster score should be empty if not set")

	reputerScore, err := keeper.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching latest reputer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     reputer,
		Score:       alloraMath.ZeroDec(),
	}, reputerScore, "Reputer score should be empty if not set")
}

func (s *KeeperTestSuite) TestSetScoreEmas() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"
	score := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	err := keeper.SetInfererScoreEma(ctx, topicId, worker, score)
	s.Require().NoError(err)
	infererScore, err := keeper.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, infererScore.Score, "Newer inferer score should be set")

	// Set a new score for forecaster
	err = keeper.SetForecasterScoreEma(ctx, topicId, forecaster, score)
	s.Require().NoError(err)
	forecasterScore, err := keeper.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, forecasterScore.Score, "Newer forecaster score should be set")

	// Set a new score for reputer
	err = keeper.SetReputerScoreEma(ctx, topicId, reputer, score)
	s.Require().NoError(err)
	reputerScore, err := keeper.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, reputerScore.Score, "Newer reputer score should be set")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ {
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore2() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		scoreValue := alloraMath.NewDecFromInt64(int64(90 + i)) // Increment score value to simulate variation
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       scoreValue,
		}
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")

	// Check that the retained scores are the last five inserted
	for idx, score := range scores.Scores {
		expectedScoreValue := alloraMath.NewDecFromInt64(int64(92 + idx)) // Expecting the last 5 scores: 94, 95, 96, 97
		s.Require().Equal(expectedScoreValue, score.Score, "Score should match the expected last scores")
	}
}

func (s *KeeperTestSuite) TestGetInferenceScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	workerAddress := s.addrs[0]
	blockHeight := int64(105)

	// Insert scores for different workers and blocks
	for blockHeight := int64(100); blockHeight <= 110; blockHeight++ {
		// Scores for the targeted worker
		scoreForWorker := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     workerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(blockHeight),
		}
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, scoreForWorker)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Get scores for the worker up to block 105
	scores, err := keeper.GetInferenceScoresUntilBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching worker inference scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")

	// Verify that the scores are correct and ordered as expected (descending block number)
	expectedBlock := blockHeight
	for _, score := range scores {
		s.Require().Equal(workerAddress.String(), score.Address, "Only scores for the specified worker should be returned")
		s.Require().Equal(expectedBlock, score.BlockHeight, "Scores should be returned in descending order by block")
		s.Require().Equal(alloraMath.NewDecFromInt64(expectedBlock), score.Score, "Score value should match expected")
		expectedBlock--
	}
}

func (s *KeeperTestSuite) TestInsertWorkerForecastScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetForecastScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(105)

	// Insert scores for the worker at various blocks
	for i := int64(100); i <= 110; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: i,
			Score:       alloraMath.NewDecFromInt64(i),
			Address:     s.addrsStr[0],
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, i, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Get forecast scores for the worker up to block 105
	scores, err := keeper.GetForecastScoresUntilBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching worker forecast scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.addrsStr[i],
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := keeper.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestInsertReputerScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := keeper.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetReputersScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert multiple scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.addrsStr[i],
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := keeper.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestSetListeningCoefficient() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Define a listening coefficient
	coefficient := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(10),
	}

	// Set the listening coefficient
	err := keeper.SetListeningCoefficient(ctx, topicId, reputer, coefficient)
	s.Require().NoError(err, "Setting listening coefficient should not fail")

	// Retrieve the set coefficient to verify it was set correctly
	retrievedCoef, err := keeper.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching listening coefficient should not fail")
	s.Require().Equal(coefficient.Coefficient, retrievedCoef.Coefficient, "The retrieved coefficient should match the set value")
}

func (s *KeeperTestSuite) TestGetListeningCoefficient() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Attempt to fetch a coefficient before setting it
	defaultCoef, err := keeper.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail when not set")
	s.Require().Equal(alloraMath.NewDecFromInt64(1), defaultCoef.Coefficient, "Should return the default coefficient when not set")

	// Now set a specific coefficient
	setCoef := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(5),
	}
	err = keeper.SetListeningCoefficient(ctx, topicId, reputer, setCoef)
	s.Require().NoError(err, "Setting listening coefficient should not fail")
	// Fetch and verify the coefficient after setting
	fetchedCoef, err := keeper.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail after setting")
	s.Require().Equal(setCoef.Coefficient, fetchedCoef.Coefficient, "The fetched coefficient should match the set value")
}

/// REWARD FRACTION

func (s *KeeperTestSuite) TestSetPreviousReputerRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := s.addrsStr[2]

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(75) // Assuming 0.75 as a fraction example

	// Set the reward fraction
	err := keeper.SetPreviousReputerRewardFraction(ctx, topicId, reputer, rewardFraction)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := keeper.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousReputerRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := s.addrsStr[2]

	// Attempt to fetch a reward fraction before setting it
	defaultReward, _, err := keeper.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(50) // Assuming 0.50 as a fraction example
	err = keeper.SetPreviousReputerRewardFraction(ctx, topicId, reputer, setReward)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := keeper.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousInferenceRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[1]

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(25)

	// Set the reward fraction
	err := keeper.SetPreviousInferenceRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := keeper.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousInferenceRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[1]

	// Attempt to fetch a reward fraction before setting it
	defaultReward, noPrior, err := keeper.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75)
	err = keeper.SetPreviousInferenceRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")
	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := keeper.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousForecastRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[3]

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(50) // Assume setting the fraction to 0.50

	// Set the forecast reward fraction
	err := keeper.SetPreviousForecastRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Verify by fetching the set value
	fetchedReward, noPrior, err := keeper.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set forecast reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousForecastRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[3]

	// Attempt to fetch the reward fraction before setting it, expecting default value
	defaultReward, noPrior, err := keeper.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero forecast reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75) // Assume setting it to 0.75
	err = keeper.SetPreviousForecastRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := keeper.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetGetPreviousPercentageRewardToStakedReputers() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	previousPercentageReward := alloraMath.NewDecFromInt64(50)

	// Set the previous percentage reward to staked reputers
	err := keeper.SetPreviousPercentageRewardToStakedReputers(ctx, previousPercentageReward)
	s.Require().NoError(err, "Setting previous percentage reward to staked reputers should not fail")

	// Get the previous percentage reward to staked reputers
	fetchedPercentageReward, err := keeper.GetPreviousPercentageRewardToStakedReputers(ctx)
	s.Require().NoError(err, "Fetching previous percentage reward to staked reputers should not fail")
	s.Require().Equal(previousPercentageReward, fetchedPercentageReward, "The fetched percentage reward should match the set value")
}

/// WHITELISTS

func (s *KeeperTestSuite) TestWhitelistAdminOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	adminAddress := "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h"

	// Test Adding to whitelist
	err := keeper.AddWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Adding whitelist admin should not fail")

	// Test Checking whitelist
	isAdmin, err := keeper.IsWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Checking if address is an admin should not fail")
	s.Require().True(isAdmin, "Address should be an admin after being added")

	// Test Removing from whitelist
	err = keeper.RemoveWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Removing whitelist admin should not fail")

	// Verify removal
	isAdmin, err = keeper.IsWhitelistAdmin(ctx, adminAddress)
	s.Require().NoError(err, "Checking admin status after removal should not fail")
	s.Require().False(isAdmin, "Address should not be an admin after being removed")
}

/// TOPIC REWARD NONCE

func (s *KeeperTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test Get on an unset topicId, should return 0
	nonce, err := keeper.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting an unset topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce for an unset topicId should be 0")

	// Test Set
	expectedNonce := int64(12345)
	err = keeper.SetTopicRewardNonce(ctx, topicId, expectedNonce)
	s.Require().NoError(err, "Setting topic reward nonce should not fail")

	// Test Get after Set, should return the set value
	nonce, err = keeper.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting set topic reward nonce should not fail")
	s.Require().Equal(expectedNonce, nonce, "Nonce should match the value set earlier")

	// Test Delete
	err = keeper.DeleteTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Deleting topic reward nonce should not fail")

	// Test Get after Delete, should return 0
	nonce, err = keeper.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting deleted topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce should be 0 after deletion")
}

/// UTILS

func (s *KeeperTestSuite) TestCalcAppropriatePaginationForUint64Cursor() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	defaultLimit := uint64(20)
	maxLimit := uint64(50)

	params := types.DefaultParams()
	params.DefaultPageLimit = defaultLimit
	params.MaxPageLimit = maxLimit
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting default and max limit parameters should not fail")

	paramsActual, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(maxLimit, paramsActual.MaxPageLimit, "Max limit should be set correctly")
	s.Require().Equal(defaultLimit, paramsActual.DefaultPageLimit, "Default limit should be set correctly")

	// Test 1: Pagination request is nil
	limit, cursor, err := keeper.CalcAppropriatePaginationForUint64Cursor(ctx, nil)
	s.Require().NoError(err, "Should handle nil pagination request without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key nil")

	// Test 2: Pagination Key is empty and Limit is zero
	pagination := &types.SimpleCursorPaginationRequest{Key: []byte{}, Limit: 0}
	limit, cursor, err = keeper.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Should handle empty key and zero limit without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key is empty")

	// Test 3: Valid key and non-zero limit within bounds
	validKey := binary.BigEndian.AppendUint64(nil, uint64(12345)) // Convert 12345 to big-endian byte slice
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 30}
	limit, cursor, err = keeper.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling valid key and valid limit should not fail")
	s.Require().Equal(uint64(30), limit, "Limit should be as specified")
	s.Require().Equal(uint64(12345), cursor, "Cursor should decode correctly from key")

	// Test 4: Limit exceeds maximum limit
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 60}
	limit, _, err = keeper.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling limit exceeding maximum should not fail")
	s.Require().Equal(maxLimit, limit, "Limit should be capped at the maximum limit")
}

// STATE MANAGEMENT

func (s *KeeperTestSuite) TestPruneRecordsAfterRewards() {
	// Set infereces, forecasts, and reputations for a topic
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     s.addrsStr[0],
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Value:       alloraMath.NewDecFromInt64(2),
				Inferer:     s.addrsStr[1],
			},
		},
	}
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err, "Inserting inferences should not fail")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.addrsStr[6],
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.addrsStr[0],
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: s.addrsStr[1],
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.addrsStr[7],
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.addrsStr[0],
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: s.addrsStr[1],
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}
	err = s.emissionsKeeper.InsertActiveForecasts(s.ctx, topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{},
	}
	err = s.emissionsKeeper.InsertActiveReputerLosses(s.ctx, topicId, block, reputerLossBundles)
	s.Require().NoError(err, "InsertActiveReputerLosses should not return an error")

	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: block},
	}
	networkLosses := types.ValueBundle{
		Reputer:                       s.addrsStr[4],
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000117005278862668"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, block, networkLosses)
	s.Require().NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Check if the records are set
	_, err = s.emissionsKeeper.GetInferencesAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting inferences should not fail")
	_, err = s.emissionsKeeper.GetForecastsAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	lossBundles, err := s.emissionsKeeper.GetReputerLossBundlesAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().NotNil(lossBundles)
	_, err = s.emissionsKeeper.GetNetworkLossBundleAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail")

	// Prune records in the subsequent block
	err = s.emissionsKeeper.PruneRecordsAfterRewards(s.ctx, topicId, block+1)
	s.Require().NoError(err, "Pruning records after rewards should not fail")

	// Check if the records are pruned
	inferences, err := s.emissionsKeeper.GetInferencesAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting inferences should not fail")
	s.Require().Empty(inferences.Inferences, "Must be pruned")
	forecasts, err := s.emissionsKeeper.GetForecastsAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	s.Require().Empty(forecasts.Forecasts, "Must be pruned")
	lossbundles, err := s.emissionsKeeper.GetReputerLossBundlesAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().Empty(lossbundles.ReputerValueBundles, "Must be pruned")
	networkBundles, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail but be empty")
	s.Require().Equal(topicId, networkBundles.TopicId, "topic id returned")
	s.Require().Empty(networkBundles.InfererValues, "inferer values is empty")
	s.Require().Empty(networkBundles.ForecasterValues, "forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererValues, "one out inferer values is empty")
	s.Require().Empty(networkBundles.OneOutForecasterValues, "one out forecaster values is empty")
	s.Require().Empty(networkBundles.OneInForecasterValues, "one in forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererForecasterValues, "one out inferer forecaster values is empty")
	s.Require().Equal("0", networkBundles.CombinedValue.String(), "Must be pruned as evidenced by empty combined value")
	s.Require().Equal("0", networkBundles.NaiveValue.String(), "Must be pruned as evidenced by empty naive value")
}

func (s *KeeperTestSuite) TestPruneWorkerNoncesLogicNoNonces() {
	keeper := s.emissionsKeeper
	topicId1 := uint64(1)
	blockHeightThreshold := int64(10)
	err := keeper.DeleteUnfulfilledWorkerNonces(s.ctx, topicId1)
	s.Require().NoError(err, "Failed to delete unfulfilled worker nonces, topicId1")

	// Call pruneWorkerNonces
	err = s.emissionsKeeper.PruneWorkerNonces(s.ctx, topicId1, blockHeightThreshold)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	// Check remaining nonces
	nonces, err := s.emissionsKeeper.GetUnfulfilledWorkerNonces(s.ctx, topicId1)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestPruneWorkerNoncesLogicCorrectness() {
	tests := []struct {
		name                 string
		blockHeightThreshold int64
		nonces               []*types.Nonce
		expectedNonces       []*types.Nonce
	}{

		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: All Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 7}},
			expectedNonces:       []*types.Nonce{},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: Some Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 15}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 15}},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: Nonces Pruned on the Edge",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 10}, {BlockHeight: 15}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 10}, {BlockHeight: 15}},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: No Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 15}, {BlockHeight: 20}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 15}, {BlockHeight: 20}},
		},
	}
	keeper := s.emissionsKeeper
	topicId1 := uint64(1)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := keeper.DeleteUnfulfilledWorkerNonces(s.ctx, topicId1)
			s.Require().NoError(err, "Failed to delete unfulfilled worker nonces, topicId1")
			// Set multiple worker nonces
			for _, val := range tt.nonces {
				err := keeper.AddWorkerNonce(s.ctx, topicId1, val)
				s.Require().NoError(err, "Failed to add worker nonce, topicId1")
			}

			// Call pruneWorkerNonces
			err = s.emissionsKeeper.PruneWorkerNonces(s.ctx, topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := s.emissionsKeeper.GetUnfulfilledWorkerNonces(s.ctx, topicId1)
			s.Require().NoError(err)
			// for loop nonces
			for _, nonce := range nonces.Nonces {
				s.Require().Contains(tt.expectedNonces, nonce)
			}
			for _, nonce := range tt.expectedNonces {
				s.Require().Contains(nonces.Nonces, nonce)
			}
		})
	}
}

func (s *KeeperTestSuite) TestPruneReputerNoncesLogicCorrectness() {
	tests := []struct {
		name                 string
		blockHeightThreshold int64
		nonces               []*types.ReputerRequestNonce
		expectedNonces       []*types.ReputerRequestNonce
	}{
		{
			name:                 "No nonces",
			blockHeightThreshold: 10,
			nonces:               []*types.ReputerRequestNonce{},
			expectedNonces:       []*types.ReputerRequestNonce{},
		},
		{
			name:                 "All nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 7}}},
			expectedNonces: []*types.ReputerRequestNonce{},
		},
		{
			name:                 "Some nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
			},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
		},
		{
			name:                 "Nonces pruned on the edge",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 10}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 10}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
		},
		{
			name:                 "No nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
				{ReputerNonce: &types.Nonce{BlockHeight: 20}}},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
				{ReputerNonce: &types.Nonce{BlockHeight: 20}}},
		},
	}
	keeper := s.emissionsKeeper
	topicId1 := uint64(1)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := keeper.DeleteUnfulfilledReputerNonces(s.ctx, topicId1)
			s.Require().NoError(err, "Failed to delete unfulfilled reputer nonces, topicId1")
			// Set multiple reputer nonces
			for _, val := range tt.nonces {
				err := keeper.AddReputerNonce(s.ctx, topicId1, val.ReputerNonce)
				s.Require().NoError(err, "Failed to add reputer nonce, topicId1")
			}

			// Call PruneReputerNonces
			err = s.emissionsKeeper.PruneReputerNonces(s.ctx, topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := s.emissionsKeeper.GetUnfulfilledReputerNonces(s.ctx, topicId1)
			s.Require().NoError(err)
			// for loop nonces
			for _, nonce := range nonces.Nonces {
				s.Require().Contains(tt.expectedNonces, nonce)
			}
			for _, nonce := range tt.expectedNonces {
				s.Require().Contains(nonces.Nonces, nonce)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetTargetWeight() {
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	if err != nil {
		s.T().Fatalf("Failed to get parameters: %v", err)
	}

	dec, err := alloraMath.NewDecFromString("22.36067977499789696409173668731276")
	s.Require().NoError(err)

	testCases := []struct {
		name             string
		topicStake       alloraMath.Dec
		topicEpochLength int64
		topicFeeRevenue  alloraMath.Dec
		stakeImportance  alloraMath.Dec
		feeImportance    alloraMath.Dec
		want             alloraMath.Dec
		expectError      bool
	}{
		{
			name:             "Basic valid inputs",
			topicStake:       alloraMath.NewDecFromInt64(100),
			topicEpochLength: 10,
			topicFeeRevenue:  alloraMath.NewDecFromInt64(50),
			stakeImportance:  params.TopicRewardStakeImportance,
			feeImportance:    params.TopicRewardFeeRevenueImportance,
			want:             dec,
			expectError:      false,
		},
		{
			name:             "Zero epoch length",
			topicStake:       alloraMath.NewDecFromInt64(100),
			topicEpochLength: 0,
			topicFeeRevenue:  alloraMath.NewDecFromInt64(50),
			stakeImportance:  params.TopicRewardStakeImportance,
			feeImportance:    params.TopicRewardFeeRevenueImportance,
			want:             alloraMath.Dec{},
			expectError:      true,
		},
		{
			name:             "Negative stake",
			topicStake:       alloraMath.NewDecFromInt64(-100),
			topicEpochLength: 10,
			topicFeeRevenue:  alloraMath.NewDecFromInt64(50),
			stakeImportance:  params.TopicRewardStakeImportance,
			feeImportance:    params.TopicRewardFeeRevenueImportance,
			want:             alloraMath.Dec{},
			expectError:      true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			got, err := s.emissionsKeeper.GetTargetWeight(tc.topicStake, tc.topicEpochLength, tc.topicFeeRevenue, tc.stakeImportance, tc.feeImportance)
			if tc.expectError {
				s.Require().Error(err, "Expected an error for case: %s", tc.name)
			} else {
				s.Require().NoError(err, "Did not expect an error for case: %s", tc.name)
				s.Require().True(tc.want.Equal(got), "Expected %s, got %s for case %s", tc.want.String(), got.String(), tc.name)
			}
		})
	}
}

func (s *KeeperTestSuite) TestDeleteUnfulfilledWorkerNonces() {
	topicId := uint64(1)
	keeper := s.emissionsKeeper
	// Setup initial nonces
	err := keeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{BlockHeight: 10})
	s.Require().NoError(err)
	err = keeper.AddWorkerNonce(s.ctx, topicId, &types.Nonce{BlockHeight: 20})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = s.emissionsKeeper.DeleteUnfulfilledWorkerNonces(s.ctx, topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := s.emissionsKeeper.GetUnfulfilledWorkerNonces(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestDeleteUnfulfilledreputerNonces() {
	topicId := uint64(1)
	keeper := s.emissionsKeeper
	// Setup initial nonces
	err := keeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{BlockHeight: 50})
	s.Require().NoError(err)
	err = keeper.AddReputerNonce(s.ctx, topicId, &types.Nonce{BlockHeight: 60})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = s.emissionsKeeper.DeleteUnfulfilledReputerNonces(s.ctx, topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := s.emissionsKeeper.GetUnfulfilledReputerNonces(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestGetFirstStakeRemovalForReputerAndTopicId() {
	k := s.emissionsKeeper
	ctx := s.ctx
	reputer := s.addrsStr[2]
	topicId := uint64(1)

	// Create a stake removal info
	stakeRemovalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(100),
		BlockRemovalCompleted: 30,
	}
	anotherStakeRemoval := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               "allo13c8tjxmlv32s6d76f9anh6cu6c767v0ddnn0uh",
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(200),
		BlockRemovalCompleted: 30,
	}

	// Set the stake removal info in the keeper
	err := k.SetStakeRemoval(ctx, stakeRemovalInfo)
	s.Require().NoError(err)
	err = k.SetStakeRemoval(ctx, anotherStakeRemoval)
	s.Require().NoError(err)

	// Get the first stake removal for the reputer and topic ID
	result, found, err := k.GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(stakeRemovalInfo, result)
}

func (s *KeeperTestSuite) TestGetFirstStakeRemovalForReputerAndTopicIdNotFound() {
	k := s.emissionsKeeper
	ctx := s.ctx
	reputer := s.addrsStr[2]
	topicId := uint64(1)

	_, found, err := k.GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetFirstDelegateStakeRemovalForDelegatorReputerAndTopicId() {
	k := s.emissionsKeeper
	ctx := s.ctx
	delegator := s.addrsStr[5]
	reputer := s.addrsStr[2]
	reputer2 := s.addrsStr[3]
	topicId := uint64(1)

	// Create a stake removal info
	stakeRemovalInfo := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer,
		Delegator:             delegator,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(100),
		BlockRemovalCompleted: 30,
	}
	anotherStakeRemoval := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer2,
		Delegator:             delegator,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(200),
		BlockRemovalCompleted: 30,
	}

	// Set the stake removal info in the keeper
	err := k.SetDelegateStakeRemoval(ctx, stakeRemovalInfo)
	s.Require().NoError(err)
	err = k.SetDelegateStakeRemoval(ctx, anotherStakeRemoval)
	s.Require().NoError(err)

	// Get the first stake removal for the reputer and topic ID
	result, found, err := k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(stakeRemovalInfo, result)
}

func (s *KeeperTestSuite) TestGetFirstDelegateStakeRemovalForDelegatorReputerAndTopicIdNotFound() {
	k := s.emissionsKeeper
	ctx := s.ctx
	delegator := "delegator"
	reputer := s.addrsStr[2]
	topicId := uint64(1)

	_, found, err := k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestAppendInference() {
	ctx := s.ctx
	k := s.emissionsKeeper
	// Topic IDs
	topicId := s.CreateOneTopic(10800)
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Set previous topic quantile inferer score ema
	err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("1000"))
	s.Require().NoError(err)

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	worker1 := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	worker2 := "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t"
	worker3 := "allo1wha0sj6pldjwjasc0pm28htgpqa5uf69kafe5y"
	worker4 := "allo19n8h9zwsqawpfn9qk9v773zj6rcjmqt28a2gyd"
	worker5 := "allo18d5wvwlhc08kfu27l25q9zr2shhanlh9fvt225"
	ogWorker2Score := alloraMath.MustNewDecFromString("90")

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: ogWorker2Score}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	score4 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker4, Score: alloraMath.NewDecFromInt64(91)}
	score5 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker5, Score: alloraMath.NewDecFromInt64(96)}
	err = k.SetInfererScoreEma(ctx, topicId, worker1, score1)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker2, score2)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker3, score3)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker4, score4)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker5, score5)
	s.Require().NoError(err)

	// Ensure that the number of top inferers is capped at the max top inferers to reward
	// New high-score entrant should replace earlier low-score entrant
	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err = k.SetParams(ctx, params)
	s.Require().NoError(err)

	allInferences := types.Inferences{
		Inferences: []*types.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker4,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker5,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}

	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	// New high-score entrant should replace earlier low-score entrant
	worker5OgScore, err := k.GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	worker5Found := false
	for _, address := range activeInferers {
		if address == worker5 {
			worker5Found = true
		}
	}
	s.Require().True(worker5Found)

	// Ensure EMA score of active set is not yet updated
	// This will happen later during epoch reward calculation, not here
	worker5NewScore, err := k.GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	s.Require().True(worker5OgScore.Score.Equal(worker5NewScore.Score))
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(worker5OgScore.BlockHeight, worker5NewScore.BlockHeight)

	// Ensure EMA score of actor moved to passive set is updated
	updatedWorker2Score, err := k.GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	updatedWorker2ScoreVal, err := updatedWorker2Score.Score.Int64()
	s.Require().NoError(err)
	ogWorker2ScoreVal, err := ogWorker2Score.Int64()
	s.Require().NoError(err)
	worker5OgScoreVal, err := worker5OgScore.Score.Int64()
	s.Require().NoError(err)
	s.Require().Greater(updatedWorker2ScoreVal, ogWorker2ScoreVal, "worker2 score should go up given large ema value")
	s.Require().Greater(updatedWorker2ScoreVal, worker5OgScoreVal, "worker2 could not overtake worker5, but not in this epoch")
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(nonce.BlockHeight, updatedWorker2Score.BlockHeight)

	// Ensure passive set participant can't update their score within the same epoch
	blockHeightInferences = blockHeightInferences + 1 // within the same epoch => no update
	newInference2 = types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker2,
		Value:       alloraMath.MustNewDecFromString("0.52"),
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().Error(err, types.ErrCantUpdateEmaMoreThanOncePerWindow.Error())
	// Confirm no change in EMA score
	updateAttemptForWorker2, err := k.GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	updateAttemptForWorker2Val, err := updateAttemptForWorker2.Score.Int64()
	s.Require().NoError(err)
	s.Require().Equal(updateAttemptForWorker2Val, updatedWorker2ScoreVal, "unchanged score")
	s.Require().Equal(updateAttemptForWorker2.BlockHeight, updatedWorker2Score.BlockHeight, "unchanged height")
}

func mockUninitializedParams() types.Params {
	return types.Params{
		Version:                             "v2",
		MinTopicWeight:                      alloraMath.MustNewDecFromString("0"),
		RequiredMinimumStake:                cosmosMath.NewInt(0),
		RemoveStakeDelayWindow:              0,
		MinEpochLength:                      1,
		BetaEntropy:                         alloraMath.MustNewDecFromString("0"),
		LearningRate:                        alloraMath.MustNewDecFromString("0.0001"),
		GradientDescentMaxIters:             uint64(100),
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.0001"),
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0"),
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.0001"),
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.0001"),
		MaxUnfulfilledWorkerRequests:        uint64(0),
		MaxUnfulfilledReputerRequests:       uint64(0),
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0"),
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0"),
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.1"),
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1"),
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0"),
		MaxSamplesToScaleScores:             uint64(10),
		MaxTopInferersToReward:              uint64(0),
		MaxTopForecastersToReward:           uint64(0),
		MaxTopReputersToReward:              uint64(0),
		CreateTopicFee:                      cosmosMath.NewInt(0),
		RegistrationFee:                     cosmosMath.NewInt(0),
		DefaultPageLimit:                    uint64(0),
		MaxPageLimit:                        uint64(0),
		MinEpochLengthRecordLimit:           int64(0),
		MaxSerializedMsgLength:              int64(0),
		BlocksPerMonth:                      uint64(864000),
		PRewardInference:                    alloraMath.NewDecFromInt64(1),
		PRewardForecast:                     alloraMath.NewDecFromInt64(1),
		PRewardReputer:                      alloraMath.NewDecFromInt64(1),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.1"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.1"),
		CNorm:                               alloraMath.MustNewDecFromString("0.1"),
		HalfMaxProcessStakeRemovalsEndBlock: uint64(1),
		DataSendingFee:                      cosmosMath.NewInt(0),
		MaxElementsPerForecast:              uint64(0),
		MaxActiveTopicsPerBlock:             uint64(0),
		MaxStringLength:                     uint64(0),
		InitialRegretQuantile:               alloraMath.ZeroDec(),
		PNormSafeDiv:                        alloraMath.ZeroDec(),
	}
}

func (s *KeeperTestSuite) TestAppendForecast() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	worker1 := s.addrsStr[0]
	worker2 := s.addrsStr[1]
	worker3 := s.addrsStr[2]
	worker4 := s.addrsStr[3]
	worker5 := s.addrsStr[4]

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	score4 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker4, Score: alloraMath.NewDecFromInt64(91)}
	score5 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker5, Score: alloraMath.NewDecFromInt64(96)}
	err := k.SetForecasterScoreEma(ctx, topicId, worker1, score1)
	s.Require().NoError(err)
	err = k.SetForecasterScoreEma(ctx, topicId, worker2, score2)
	s.Require().NoError(err)
	err = k.SetForecasterScoreEma(ctx, topicId, worker3, score3)
	s.Require().NoError(err)
	err = k.SetForecasterScoreEma(ctx, topicId, worker4, score4)
	s.Require().NoError(err)
	err = k.SetForecasterScoreEma(ctx, topicId, worker5, score5)
	s.Require().NoError(err)

	params := mockUninitializedParams()
	params.MaxTopForecastersToReward = 4
	err = k.SetParams(ctx, params)
	s.Require().NoError(err)

	allForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker1,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker2,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker3,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker4,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
		},
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)
	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newForecast2 := types.Forecast{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Forecaster:  worker5,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: worker1,
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
			{
				Inferer: worker2,
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
		ExtraData: nil,
	}
	// MaxTopInferersToReward is 4, so this should not be added
	err = k.AppendForecast(ctx, topic, nonce.BlockHeight, &newForecast2, params.MaxTopForecastersToReward)
	s.Require().NoError(err)

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))
}

func (s *KeeperTestSuite) TestAppendReputerLoss() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)
	blockHeight := int64(10)
	nonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	reputer1 := s.addrsStr[0]
	reputer2 := s.addrsStr[1]
	reputer3 := s.addrsStr[2]
	reputer4 := s.addrsStr[3]
	reputer5 := s.addrsStr[4]

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer3, Score: alloraMath.NewDecFromInt64(99)}
	score4 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer4, Score: alloraMath.NewDecFromInt64(91)}
	score5 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer5, Score: alloraMath.NewDecFromInt64(96)}
	err := k.SetReputerScoreEma(ctx, topicId, reputer1, score1)
	s.Require().NoError(err)
	err = k.SetReputerScoreEma(ctx, topicId, reputer2, score2)
	s.Require().NoError(err)
	err = k.SetReputerScoreEma(ctx, topicId, reputer3, score3)
	s.Require().NoError(err)
	err = k.SetReputerScoreEma(ctx, topicId, reputer4, score4)
	s.Require().NoError(err)
	err = k.SetReputerScoreEma(ctx, topicId, reputer5, score5)
	s.Require().NoError(err)

	params := types.DefaultParams()
	params.MaxTopReputersToReward = 4
	err = k.SetParams(ctx, params)
	s.Require().NoError(err)

	valueBundleReputer1 := types.ValueBundle{
		Reputer:                       reputer1,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000117005278862668"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature := s.signValueBundle(&valueBundleReputer1, s.privKeys[0])
	reputerValueBundle1 := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer1,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[0],
	}
	valueBundleReputer2 := types.ValueBundle{
		Reputer:                       reputer2,
		CombinedValue:                 alloraMath.MustNewDecFromString(".00000962701954026944"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature = s.signValueBundle(&valueBundleReputer2, s.privKeys[1])
	reputerValueBundle2 := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer2,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[1],
	}
	valueBundleReputer3 := types.ValueBundle{
		Reputer:                       reputer3,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature = s.signValueBundle(&valueBundleReputer3, s.privKeys[2])
	reputerValueBundle3 := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer3,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[2],
	}

	allReputerLosses := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			&reputerValueBundle1,
			&reputerValueBundle2,
			&reputerValueBundle3,
		},
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)
	for _, reputerValueBundle := range allReputerLosses.ReputerValueBundles {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundle)
		s.Require().NoError(err)
	}

	valueBundleReputer4 := types.ValueBundle{
		Reputer:                       reputer4,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature = s.signValueBundle(&valueBundleReputer4, s.privKeys[3])
	reputerValueBundle4 := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer4,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[3],
	}

	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, &reputerValueBundle4)
	s.Require().NoError(err)
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))

	valueBundleReputer5 := types.ValueBundle{
		Reputer:                       reputer5,
		CombinedValue:                 alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce:           reputerRequestNonce,
		TopicId:                       topicId,
		ExtraData:                     nil,
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature = s.signValueBundle(&valueBundleReputer5, s.privKeys[4])
	reputerValueBundle5 := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer5,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[4],
	}
	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, &reputerValueBundle5)
	s.Require().NoError(err)
	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))
}

func (s *KeeperTestSuite) TestDripTopicFeeRevenue() {
	// Initialize the test environment
	ctx := s.ctx
	k := s.emissionsKeeper
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	block := int64(100)
	// Calculated expected drip with these values: 26
	expectedDrip := cosmosMath.NewInt(26)
	initialRevenue := cosmosMath.NewInt(1000000) // 0.001 in Int representation (assuming 6 decimal places)

	params := types.DefaultParams()
	params.MinEpochLength = 1
	err := k.SetParams(ctx, params)
	require.NoError(err, "Setting a new topic should not fail")

	// Create and activate a topic
	topic := s.mockTopic()
	topic.EpochLength = 5
	topic.WorkerSubmissionWindow = 5
	err = k.SetTopic(ctx, topicId, topic)
	require.NoError(err, "Setting a new topic should not fail")

	err = k.ActivateTopic(ctx, topicId)
	require.NoError(err, "Activating the topic should not fail")

	// Set up initial topic fee revenue
	err = k.AddTopicFeeRevenue(ctx, topicId, initialRevenue)
	require.NoError(err, "Setting initial topic fee revenue should not fail")

	// Call the function under test
	err = k.DripTopicFeeRevenue(ctx, topicId, block)
	require.NoError(err, "DripTopicFeeRevenue should not return an error")

	// Retrieve the updated topic fee revenue
	updatedTopicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	require.NoError(err, "Getting topic fee revenue should not fail")

	// Assert the expected results
	require.True(updatedTopicFeeRevenue.LT(initialRevenue),
		"The topic fee revenue should have decreased after dripping")

	// Calculate expected revenue (this may need adjustment based on your actual implementation)
	expectedRevenue := initialRevenue.Sub(expectedDrip)
	require.Equal(expectedRevenue.String(), updatedTopicFeeRevenue.String(),
		"The topic fee revenue should match the expected value after dripping")
}

func (s *KeeperTestSuite) TestActiveInfererFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	inferer := s.addrsStr[0]

	err := k.AddActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeInferers, 1)
	s.Require().Equal(inferer, activeInferers[0])
}

func (s *KeeperTestSuite) TestActiveForecasterFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := s.addrsStr[1]

	err := k.AddActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err := k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeForecasters, 1)
	s.Require().Equal(forecaster, activeForecasters[0])

	err = k.RemoveActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err = k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestLowestScoreEmaFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	address := s.addrsStr[2]

	lowestInfererScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 100,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(50),
	}
	err := k.SetLowestInfererScoreEma(ctx, topicId, lowestInfererScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestInfererScore, retrievedScore)

	lowestForecasterScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 200,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(75),
	}
	err = k.SetLowestForecasterScoreEma(ctx, topicId, lowestForecasterScore)
	s.Require().NoError(err)

	retrievedScore, found, err = k.GetLowestForecasterScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestForecasterScore, retrievedScore)
}

func (s *KeeperTestSuite) TestActiveReputerFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	reputer := s.addrsStr[3]

	err := k.AddActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeReputers, 1)
	s.Require().Equal(reputer, activeReputers[0])

	err = k.RemoveActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err = k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestResetFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)

	err := k.AddActiveReputer(ctx, topicId, s.addrsStr[0])
	s.Require().NoError(err)
	err = k.AddActiveInferer(ctx, topicId, s.addrsStr[1])
	s.Require().NoError(err)
	err = k.AddActiveForecaster(ctx, topicId, s.addrsStr[2])
	s.Require().NoError(err)

	err = k.ResetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeReputers)

	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeInferers)
	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeForecasters)

	err = k.ResetReputersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)

	err = k.ResetWorkersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestLowestReputerScoreEmaFunctions() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	address := s.addrsStr[4]

	lowestReputerScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 300,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(60),
	}
	err := k.SetLowestReputerScoreEma(ctx, topicId, lowestReputerScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestReputerScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestReputerScore, retrievedScore)
}

func (s *KeeperTestSuite) TestRemoveReputerLoss() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)
	reputer := s.addrsStr[0]

	// Create a reputer loss bundle
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: 100},
		},
		Reputer:                       reputer,
		ExtraData:                     []byte("data"),
		CombinedValue:                 alloraMath.MustNewDecFromString("123"),
		InfererValues:                 nil,
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.MustNewDecFromString("123"),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	signature := s.signValueBundle(valueBundle, s.privKeys[0])
	reputerLossBundle := types.ReputerValueBundle{
		ValueBundle: valueBundle,
		Signature:   signature,
		Pubkey:      s.pubKeyHexStr[0],
	}

	// Insert the reputer loss bundle
	err := k.InsertReputerLoss(ctx, topicId, reputerLossBundle)
	s.Require().NoError(err)

	// Verify the reputer loss was added
	retrievedLoss, err := k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(reputerLossBundle, retrievedLoss)

	// Remove the reputer loss
	err = k.RemoveReputerLoss(ctx, topicId, reputer)
	s.Require().NoError(err)

	// Verify the reputer loss was removed
	_, err = k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().Error(err) // Expect an error since the loss should be removed
}

func (s *KeeperTestSuite) TestRemoveForecast() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)
	forecaster := s.addrsStr[0]

	// Create a forecast
	forecast := types.Forecast{
		TopicId:     topicId,
		BlockHeight: 100,
		Forecaster:  forecaster,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   alloraMath.MustNewDecFromString("1"),
			},
			{
				Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Value:   alloraMath.MustNewDecFromString("2"),
			},
		},
		ExtraData: []byte("data"),
	}

	// Insert the forecast
	err := k.InsertForecast(ctx, topicId, forecast)
	s.Require().NoError(err)

	// Verify the forecast was added
	retrievedForecast, err := k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(forecast, retrievedForecast)

	// Remove the forecast
	err = k.RemoveForecast(ctx, topicId, forecaster)
	s.Require().NoError(err)

	// Verify the forecast was removed
	_, err = k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().Error(err) // Expect an error since the forecast should be removed
}

func (s *KeeperTestSuite) TestRemoveInference() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := s.CreateOneTopic(10800)
	inferer := s.addrsStr[0]

	// Create an inference
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: 100,
		Value:       alloraMath.NewDecFromInt64(1),
		Inferer:     inferer,
		ExtraData:   []byte("data"),
		Proof:       "",
	}

	// Insert the inference
	err := k.InsertInference(ctx, topicId, inference)
	s.Require().NoError(err)

	// Verify the inference was added
	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(inference, retrievedInference)

	// Remove the inference
	err = k.RemoveInference(ctx, topicId, inferer)
	s.Require().NoError(err)

	// Verify the inference was removed
	_, err = k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().Error(err) // Expect an error since the inference should be removed
}

func (s *KeeperTestSuite) TestGetCountInfererInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.ctx
	k := s.emissionsKeeper
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	inferer1 := s.addrsStr[0]
	inferer2 := s.addrsStr[1]
	err := k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

func (s *KeeperTestSuite) TestGetCountForecasterInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.ctx
	k := s.emissionsKeeper
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	forecaster1 := s.addrsStr[0]
	forecaster2 := s.addrsStr[1]
	err := k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

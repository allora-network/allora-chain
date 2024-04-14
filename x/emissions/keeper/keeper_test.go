package keeper_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	// "cosmossdk.io/collections"
	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

var (
	nonAdminAccounts = simtestutil.CreateRandomAccounts(4)
	// TODO: Change PKS to accounts here and in all the tests (like the above line)
	PKS     = simtestutil.CreateTestPubKeys(4)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = sdk.ValAddress(Addr)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	bankKeeper      *emissionstestutil.MockBankKeeper
	authKeeper      *emissionstestutil.MockAccountKeeper
	emissionsKeeper keeper.Keeper
	msgServer       types.MsgServer
	mockCtrl        *gomock.Controller
	key             *storetypes.KVStoreKey
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)
	ctrl := gomock.NewController(s.T())

	s.bankKeeper = emissionstestutil.NewMockBankKeeper(ctrl)
	s.authKeeper = emissionstestutil.NewMockAccountKeeper(ctrl)

	s.ctx = ctx
	s.emissionsKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, s.authKeeper, s.bankKeeper, "fee_collector")
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.mockCtrl = ctrl
	s.key = key

	// Add all tests addresses in whitelists
	for _, addr := range PKS {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, sdk.AccAddress(addr.Address()))
		s.emissionsKeeper.AddToTopicCreationWhitelist(ctx, sdk.AccAddress(addr.Address()))
		s.emissionsKeeper.AddToReputerWhitelist(ctx, sdk.AccAddress(addr.Address()))
	}
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

//////////////////////////////////////////////////////////////
//                 WORKER NONCE TESTS                       //
//////////////////////////////////////////////////////////////

func (s *KeeperTestSuite) TestAddWorkerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	unfulfilledNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Len(unfulfilledNonces.Nonces, 0, "Unfulfilled nonces should be empty")

	// Set worker nonce
	newNonce := &types.Nonce{Nonce: 42}
	err = keeper.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(newNonce.Nonce, unfulfilledNonces.Nonces[0].Nonce, "Unfulfilled nonces should contain the new nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedWorkerNonceIsUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{Nonce: 42}

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
	newNonce := &types.Nonce{Nonce: 42}

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
	s.Require().Len(initialNonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(nonceValues[i], nonce.Nonce, "Nonce value should match the expected value")
	}
}

func (s *KeeperTestSuite) TestGetAndFulfillMultipleUnfulfilledWorkerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Len(initialNonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		success, err := keeper.FulfillWorkerNonce(ctx, topicId, &types.Nonce{Nonce: val})
		s.Require().True(success, "Nonce should be successfully fulfilled")
		s.Require().NoError(err, "Error fulfilling nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{42, 44, 46} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.Nonce, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestWorkerNonceLimitEnforcement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	maxUnfulfilledRequests := uint64(3)
	// Set the maximum number of unfulfilled worker nonces
	params := types.Params{
		MaxUnfulfilledWorkerRequests: maxUnfulfilledRequests,
	}

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err := keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, int(maxUnfulfilledRequests), "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{30, 40, 50} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.Nonce, "Nonce should match the expected recent nonce")
	}
}

//////////////////////////////////////////////////////////////
//                 REPUTER NONCE TESTS                      //
//////////////////////////////////////////////////////////////

func (s *KeeperTestSuite) TestAddReputerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	unfulfilledNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Len(unfulfilledNonces.Nonces, 0, "Unfulfilled nonces should be empty")

	// Set reputer nonce
	newReputerNonce := &types.Nonce{Nonce: 42}
	newWorkerNonce := &types.Nonce{Nonce: 43}
	err = keeper.AddReputerNonce(ctx, topicId, newReputerNonce, newWorkerNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(
		newReputerNonce.Nonce,
		unfulfilledNonces.Nonces[0].ReputerNonce.Nonce,
		"Unfulfilled nonces should contain the new reputer nonce")
	s.Require().Equal(
		newWorkerNonce.Nonce,
		unfulfilledNonces.Nonces[0].WorkerNonce.Nonce,
		"Unfulfilled nonces should contain the new worker nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedReputerNonceIsUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{Nonce: 42}
	newWorkerNonce := &types.Nonce{Nonce: 43}

	isUnfulfilled, err := keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Non-existent nonce should not be listed as unfulfilled")

	// Set reputer nonce
	err = keeper.AddReputerNonce(ctx, topicId, newReputerNonce, newWorkerNonce)
	s.Require().NoError(err)

	isUnfulfilled, err = keeper.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "New nonce should be unfulfilled")
}

func (s *KeeperTestSuite) TestCanFulfillNewReputerNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{Nonce: 42}
	newWorkerNonce := &types.Nonce{Nonce: 43}

	// Set reputer nonce
	err := keeper.AddReputerNonce(ctx, topicId, newReputerNonce, newWorkerNonce)
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
	s.Require().Len(initialNonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple reputer nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = keeper.AddReputerNonce(ctx, topicId, &types.Nonce{Nonce: val}, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		nonceIsUnfulfilled, err := keeper.FulfillReputerNonce(ctx, topicId, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Error fulfilling nonce")
		s.Require().True(nonceIsUnfulfilled, "Nonce should be able to be fulfilled")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{42, 44, 46} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.ReputerNonce.Nonce, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestReputerNonceLimitEnforcement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	maxUnfulfilledRequests := uint64(3)

	// Set the maximum number of unfulfilled reputer nonces
	params := types.Params{
		MaxUnfulfilledReputerRequests: maxUnfulfilledRequests,
	}

	// Set the maximum number of unfulfilled reputer nonces via the SetParams method
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Failed to set parameters")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err := keeper.AddReputerNonce(ctx, topicId, &types.Nonce{Nonce: val}, &types.Nonce{Nonce: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, int(maxUnfulfilledRequests), "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{30, 40, 50} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.ReputerNonce.Nonce, "Nonce should match the expected recent nonce")
	}
}

//////////////////////////////////////////////////////////////
//                     REGRET TESTS                         //
//////////////////////////////////////////////////////////////

func (s *KeeperTestSuite) TestSetAndGetInfererNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("worker-address")
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	// Set Inferer Network Regret
	err := keeper.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	gotRegret, err := keeper.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("forecaster-address") // Assuming sdk.AccAddress is initialized with a string representing the address

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	// Set Forecaster Network Regret
	err := keeper.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	gotRegret, err := keeper.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetOneInForecasterNetworkRegret() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := sdk.AccAddress("forecaster-address")
	inferer := sdk.AccAddress("inferer-address")

	regret := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(30)}

	// Set One-In Forecaster Network Regret
	err := keeper.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One-In Forecaster Network Regret
	gotRegret, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestGetInfererNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("nonexistent-inferer-address")

	// Attempt to get Inferer Network Regret for a nonexistent worker
	regret, err := keeper.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(1)}, regret, "Default regret value should be returned for nonexistent inferer")
}

func (s *KeeperTestSuite) TestGetForecasterNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("nonexistent-forecaster-address")

	// Attempt to get Forecaster Network Regret for a nonexistent worker
	regret, err := keeper.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(1)}, regret, "Default regret value should be returned for nonexistent forecaster")
}

func (s *KeeperTestSuite) TestGetOneInForecasterNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := sdk.AccAddress("nonexistent-forecaster-address")
	inferer := sdk.AccAddress("nonexistent-inferer-address")

	// Attempt to get One-In Forecaster Network Regret for a nonexistent forecaster-inferer pair
	regret, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(1)}, regret, "Default regret value should be returned for nonexistent forecaster-inferer pair")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentInfererRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("worker-address")

	// Topic IDs
	topicId1 := uint64(1)
	topicId2 := uint64(2)

	// Regrets
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err := keeper.SetInfererNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetInfererNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, err := keeper.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, err := keeper.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentForecasterRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("forecaster-address")

	// Topic IDs
	topicId1 := uint64(1)
	topicId2 := uint64(2)

	// Regrets
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err := keeper.SetForecasterNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetForecasterNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, err := keeper.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, err := keeper.GetForecasterNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentOneInForecasterNetworkRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	forecaster := sdk.AccAddress("forecaster-address")
	inferer := sdk.AccAddress("inferer-address")

	// Topic IDs
	topicId1 := uint64(1)
	topicId2 := uint64(2)

	// Regrets
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same forecaster-inferer pair under different topic IDs
	err := keeper.SetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer, regret1)
	s.Require().NoError(err)
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

//////////////////////////////////////////////////////////////
//                     PARAMS TESTS                         //
//////////////////////////////////////////////////////////////

func (s *KeeperTestSuite) TestSetGetMaxTopicsPerBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(100)

	// Set the parameter
	params := types.Params{MaxTopicsPerBlock: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxTopicsPerBlock(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetMinRequestUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.NewUint(1000)

	// Set the parameter
	params := types.Params{MinRequestUnmetDemand: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMinRequestUnmetDemand(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetRemoveStakeDelayWindow() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(50)

	// Set the parameter
	params := types.Params{RemoveStakeDelayWindow: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsRemoveStakeDelayWindow(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetValidatorsVsAlloraPercentReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.LegacyMustNewDecFromStr("0.25") // Assume a function to create LegacyDec

	// Set the parameter
	params := types.Params{ValidatorsVsAlloraPercentReward: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsValidatorsVsAlloraPercentReward(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinTopicUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.NewUintFromString("300")

	// Set the parameter
	params := types.Params{MinTopicUnmetDemand: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMinTopicUnmetDemand(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsRequiredMinimumStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := cosmosMath.NewUintFromString("500")

	// Set the parameter
	params := types.Params{RequiredMinimumStake: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsRequiredMinimumStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxInferenceRequestValidity() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(1000)

	// Set the parameter
	params := types.Params{MaxInferenceRequestValidity: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxInferenceRequestValidity(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinEpochLength() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(720)

	// Set the parameter
	params := types.Params{MinEpochLength: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMinEpochLength(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxRequestCadence() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := types.BlockHeight(360)

	// Set the parameter
	params := types.Params{MaxRequestCadence: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxRequestCadence(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsStakeAndFeeRevenueImportance() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedStakeImportance := alloraMath.NewDecFromInt64(2) // Example value
	expectedFeeImportance := alloraMath.NewDecFromInt64(3)   // Example value

	// Set the parameter
	params := types.Params{
		TopicRewardStakeImportance:      expectedStakeImportance,
		TopicRewardFeeRevenueImportance: expectedFeeImportance,
	}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualStakeImportance, actualFeeImportance, err := keeper.GetParamsStakeAndFeeRevenueImportance(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedStakeImportance, actualStakeImportance)
	s.Require().Equal(expectedFeeImportance, actualFeeImportance)
}

func (s *KeeperTestSuite) TestGetParamsTopicRewardAlpha() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := alloraMath.NewDecFromInt64(1) // Assuming it's a value like 0.1 formatted correctly for your system

	// Set the parameter
	params := types.Params{TopicRewardAlpha: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsTopicRewardAlpha(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxSamplesToScaleScores() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(1500)

	// Set the parameter
	params := types.Params{MaxSamplesToScaleScores: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxSamplesToScaleScores(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue)
}

//////////////////////////////////////////////////////////////
//                 INFERENCES, FORECASTS                    //
//////////////////////////////////////////////////////////////

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1), // Assuming NewDecFromInt64 exists and is appropriate
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				Value:   alloraMath.NewDecFromInt64(2),
				Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertInferences correctly sets up inferences
	nonce := types.Nonce{Nonce: int64(block)} // Assuming block type cast to int64 if needed
	err := keeper.InsertInferences(ctx, topicId, nonce, expectedInferences)
	s.Require().NoError(err)

	// Retrieve inferences
	actualInferences, err := keeper.GetInferencesAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, actualInferences)
}

func (s *KeeperTestSuite) TestGetForecastsAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:    topicId,
				Forecaster: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertForecasts correctly sets up forecasts
	nonce := types.Nonce{Nonce: int64(block)}
	err := keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	// Retrieve forecasts
	actualForecasts, err := keeper.GetForecastsAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedForecasts, actualForecasts)
}

func (s *KeeperTestSuite) TestGetInferencesAtOrAfterBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	otherTopicId := uint64(2)
	block0 := types.BlockHeight(100)
	block1 := types.BlockHeight(102)
	block2 := types.BlockHeight(105)
	block3 := types.BlockHeight(110)
	block4 := types.BlockHeight(112)

	inferences0 := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(1),
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
		},
	}

	inferences1 := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(2),
				Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	inferences2 := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(3),
				Inferer: "allo16skpmhw8etsu70kknkmxquk5ut7lsewgtqqtlu",
			},
		},
	}

	inferences3 := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(4),
				Inferer: "allo1743237sr8yhkj3q558tyv5wdcthj34g4jdhfuf",
			},
		},
	}

	inferences4 := types.Inferences{
		Inferences: []*types.Inference{
			{
				Value:   alloraMath.NewDecFromInt64(5),
				Inferer: "allo19af6agncfgj2adly0hydykm4h0ctdcvev7u5fx",
			},
			{
				Value:   alloraMath.NewDecFromInt64(6),
				Inferer: "allo1w89qy7xpeg3tn6rtm9rj9awc9jwv7hsc20crft",
			},
		},
	}

	nonce0 := types.Nonce{Nonce: int64(block0)}
	nonce1 := types.Nonce{Nonce: int64(block1)}
	nonce2 := types.Nonce{Nonce: int64(block2)}
	nonce3 := types.Nonce{Nonce: int64(block3)}
	nonce4 := types.Nonce{Nonce: int64(block4)}

	// Assume latest inference is correctly set up
	err := keeper.InsertInferences(ctx, topicId, nonce3, inferences3)
	s.Require().NoError(err)
	err = keeper.InsertInferences(ctx, topicId, nonce0, inferences0)
	s.Require().NoError(err)
	err = keeper.InsertInferences(ctx, otherTopicId, nonce2, inferences2)
	s.Require().NoError(err)
	err = keeper.InsertInferences(ctx, topicId, nonce1, inferences1)
	s.Require().NoError(err)
	err = keeper.InsertInferences(ctx, topicId, nonce4, inferences4) // make sure that it filters on topicId
	s.Require().NoError(err)

	// Retrieve latest inferences at or after the given block
	actualInferences, actualBlock, err := keeper.GetInferencesAtOrAfterBlock(ctx, topicId, 105)
	s.Require().NoError(err)
	s.Require().Equal(1, len(actualInferences.Inferences))
	s.Require().Equal(types.BlockHeight(110), actualBlock)

	s.Require().Equal(inferences3.Inferences[0].Value, actualInferences.Inferences[0].Value)
	s.Require().Equal(inferences3.Inferences[0].Inferer, actualInferences.Inferences[0].Inferer)
}

func (s *KeeperTestSuite) TestGetForecastsAtOrAfterBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	otherTopicId := uint64(2)
	block0 := types.BlockHeight(100)
	block1 := types.BlockHeight(102)
	block2 := types.BlockHeight(105)
	block3 := types.BlockHeight(110)
	block4 := types.BlockHeight(112)

	forecasts0 := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
		},
	}

	forecasts1 := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	forecasts2 := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo1743237sr8yhkj3q558tyv5wdcthj34g4jdhfuf",
			},
		},
	}

	forecasts3 := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo19af6agncfgj2adly0hydykm4h0ctdcvev7u5fx",
			},
			{
				TopicId:    topicId,
				Forecaster: "allo1w89qy7xpeg3tn6rtm9rj9awc9jwv7hsc20crft",
			},
		},
	}

	forecasts4 := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo1huwe2zxpve35z5esw4s0msg7h0kajevchdwyjp",
			},
		},
	}

	nonce0 := types.Nonce{Nonce: int64(block0)}
	nonce1 := types.Nonce{Nonce: int64(block1)}
	nonce2 := types.Nonce{Nonce: int64(block2)}
	nonce3 := types.Nonce{Nonce: int64(block3)}
	nonce4 := types.Nonce{Nonce: int64(block4)}

	// Insert forecasts into the system
	err := keeper.InsertForecasts(ctx, topicId, nonce3, forecasts3)
	s.Require().NoError(err)
	err = keeper.InsertForecasts(ctx, topicId, nonce0, forecasts0)
	s.Require().NoError(err)
	err = keeper.InsertForecasts(ctx, otherTopicId, nonce2, forecasts2)
	s.Require().NoError(err)
	err = keeper.InsertForecasts(ctx, topicId, nonce1, forecasts1)
	s.Require().NoError(err)
	err = keeper.InsertForecasts(ctx, topicId, nonce4, forecasts4)
	s.Require().NoError(err)

	// Retrieve forecasts at or after the specified block
	actualForecasts, actualBlock, err := keeper.GetForecastsAtOrAfterBlock(ctx, topicId, 105)
	s.Require().NoError(err)
	s.Require().Equal(2, len(actualForecasts.Forecasts))
	s.Require().Equal(types.BlockHeight(110), actualBlock)

	// Validate the retrieved forecasts
	s.Require().Equal(forecasts3.Forecasts[0].Forecaster, actualForecasts.Forecasts[0].Forecaster)
	s.Require().Equal(forecasts3.Forecasts[1].Forecaster, actualForecasts.Forecasts[1].Forecaster)
}

func (s *KeeperTestSuite) TestInsertReputerLossBundlesAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	reputerLossBundles := types.ReputerValueBundles{}

	// Test inserting data
	err := s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, block, reputerLossBundles)
	require.NoError(err, "InsertReputerLossBundlesAtBlock should not return an error")

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
	require.Error(err, "Should return error for non-existent data")
	require.Nil(result, "Result should be nil for non-existent data")
}

func (s *KeeperTestSuite) TestInsertNetworkLossBundleAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	lossBundle := types.ValueBundle{}

	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)
	require.NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Verify the insertion
	result, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetNetworkLossBundleAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Attempt to retrieve before insertion
	result, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.Error(err, "Should return error for non-existent data")
	require.Nil(result, "Result should be nil for non-existent data")
}

func (s *KeeperTestSuite) TestGetNetworkLossBundleAtOrBeforeBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	lossBundle := types.ValueBundle{}

	// Insert data at a specific block
	s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)

	// Get the bundle at or before the specific block
	result, blockResult, err := s.emissionsKeeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetReputerReportedLossesAtOrBeforeBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	reputerLossBundles := types.ReputerValueBundles{}

	// Insert data at a specific block
	s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, block, reputerLossBundles)

	// Get the losses at or before the specific block
	result, blockResult, err := s.emissionsKeeper.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetNetworkLossBundleAtOrBeforeBlockComplex() {
	ctx := s.ctx
	require := s.Require()

	topicId := uint64(3)
	otherTopicId := uint64(4)

	earlierBlock := types.BlockHeight(250)
	block := types.BlockHeight(300)
	unrelatedBlock := types.BlockHeight(310)
	laterBlock := types.BlockHeight(350)

	earlierLossBundle := types.ValueBundle{}
	lossBundle := types.ValueBundle{}
	unrelatedLossBundle := types.ValueBundle{}
	laterLossBundle := types.ValueBundle{}

	// Insert data for different blocks and topics
	s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, earlierBlock, earlierLossBundle)
	s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)
	s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, otherTopicId, unrelatedBlock, unrelatedLossBundle)
	s.emissionsKeeper.InsertNetworkLossBundleAtBlock(ctx, topicId, laterBlock, laterLossBundle)

	// Test the retrieval logic
	result, blockResult, err := s.emissionsKeeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")

	// Test the retrieval logic for a block before any data is inserted for that block
	result, blockResult, err = s.emissionsKeeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, earlierBlock-10)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(int64(0), blockResult, "No block should be returned")

	// Test the retrieval logic for blocks after inserted data
	result, blockResult, err = s.emissionsKeeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, laterBlock+10)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(laterBlock, blockResult, "Block returned should be the latest available block")

	// Ensure it does not return data for a different topic
	result, blockResult, err = s.emissionsKeeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, 320)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetReputerReportedLossesAtOrBeforeBlockComplex() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(3)
	otherTopicId := uint64(4)

	earlierBlock := types.BlockHeight(250)
	block := types.BlockHeight(300)
	unrelatedBlock := types.BlockHeight(310)
	laterBlock := types.BlockHeight(350)

	earlierReputerLossBundles := types.ReputerValueBundles{}
	reputerLossBundles := types.ReputerValueBundles{}
	unrelatedReputerLossBundles := types.ReputerValueBundles{}
	laterReputerLossBundles := types.ReputerValueBundles{}

	// Insert data at various blocks and topics
	s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, earlierBlock, earlierReputerLossBundles)
	s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, block, reputerLossBundles)
	s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, otherTopicId, unrelatedBlock, unrelatedReputerLossBundles)
	s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, laterBlock, laterReputerLossBundles)

	// Test the retrieval logic for the specified block
	result, blockResult, err := s.emissionsKeeper.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")

	// Test the retrieval logic for a block before any data is inserted for that block
	result, blockResult, err = s.emissionsKeeper.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, earlierBlock-10)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(0, len(result.ReputerValueBundles))
	require.Equal(types.BlockHeight(0), blockResult, "No block should be returned")

	// Test the retrieval logic for blocks after inserted data
	result, blockResult, err = s.emissionsKeeper.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, laterBlock+10)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(laterBlock, blockResult, "Block returned should be the latest available block")

	// Ensure it does not return data for a different topic at a nearby block
	result, blockResult, err = s.emissionsKeeper.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, 320)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(block, blockResult, "Block returned should match the requested block")
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")
}

// ########################################
// #           Staking tests              #
// ########################################

func (s *KeeperTestSuite) TestGetSetTotalStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Set total stake
	newTotalStake := cosmosMath.NewUint(1000)
	err := keeper.SetTotalStake(ctx, newTotalStake)
	s.Require().NoError(err)

	// Check total stake
	totalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(newTotalStake, totalStake)
}

func (s *KeeperTestSuite) TestAddStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Initial Values
	initialTotalStake := cosmosMath.NewUint(0)
	initialTopicStake := cosmosMath.NewUint(0)

	// Add stake
	err := keeper.AddStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetStakeOnTopicFromReputer(ctx, topicId, reputerAddr)
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

func (s *KeeperTestSuite) TestAddDelegatedStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(500)
	additionalStakeAmount := cosmosMath.NewUint(300)

	// Setup initial stake
	err := keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetStakeFromDelegator(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount, delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")

	// Add additional stake
	err = keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err = keeper.GetStakeFromDelegator(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
}

func (s *KeeperTestSuite) TestAddStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	zeroStakeAmount := cosmosMath.NewUint(0)

	// Try to add zero stake
	err := keeper.AddStake(ctx, topicId, delegatorAddr, zeroStakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes after adding stake
	initialTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)

	// Remove stake
	err = keeper.RemoveStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator after removal
	delegatorStake, err := keeper.GetStakeOnTopicFromReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroUint(), delegatorStake, "Delegator stake should be zero after removal")

	// Check updated topic stake after removal
	topicStake, err := keeper.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroUint(), topicStake, "Topic stake should be zero after removal")

	// Check updated total stake after removal
	finalTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Sub(stakeAmount), finalTotalStake, "Total stake should be decremented by stake amount after removal")
}

func (s *KeeperTestSuite) TestRemovePartialStakeFromDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(1000)
	removeStakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetStakeFromDelegator(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := keeper.GetDelegatedStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(1000)

	// Setup initial stake
	err := keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetStakeFromDelegator(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroUint(), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := keeper.GetDelegatedStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroUint(), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	initialStakeAmount := cosmosMath.NewUint(500)
	zeroStakeAmount := cosmosMath.NewUint(0)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicId, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Try to remove zero stake
	err = keeper.RemoveStake(ctx, topicId, reputerAddr, zeroStakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	nonExistingDelegatorAddr := sdk.AccAddress(PKS[0].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Try to remove stake with non-existing delegator or target
	err := keeper.RemoveStake(ctx, topicId, nonExistingDelegatorAddr, stakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGetAllStakeForDelegator() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	delegatorAddr := sdk.AccAddress(PKS[2].Address())

	// Mock setup
	topicId := uint64(1)
	targetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Add stake to create bonds
	err := keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Add stake to create bonds
	err = keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount.Mul(cosmosMath.NewUint(2)))
	s.Require().NoError(err)

	// Get all bonds for delegator
	amount, err := keeper.GetStakeFromDelegator(ctx, topicId, delegatorAddr)

	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
	s.Require().Equal(stakeAmount.Mul(cosmosMath.NewUint(3)), amount, "The total amount is incorrect")
}

func (s *KeeperTestSuite) TestSetAndGetStakeRemovalQueueByAddressWithDetailedPlacement() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	address := sdk.AccAddress("sampleAddress1")

	// Create sample stake placement information with multiple topics and reputers
	placements := []*types.StakePlacement{
		{
			TopicId: 101,
			Reputer: "reputer1",
			Amount:  cosmosMath.NewUint(100),
		},
		{
			TopicId: 102,
			Reputer: "reputer2",
			Amount:  cosmosMath.NewUint(200),
		},
	}

	// Create a sample stake removal information
	removalInfo := types.StakeRemoval{
		BlockRemovalStarted: time.Now().Unix(),
		Placements:          placements,
	}

	// Set stake removal information
	err := keeper.SetStakeRemovalQueueForAddress(ctx, address, removalInfo)
	s.Require().NoError(err)

	// Retrieve the stake removal information
	retrievedInfo, err := keeper.GetStakeRemovalQueueByAddress(ctx, address)
	s.Require().NoError(err)
	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(len(removalInfo.Placements), len(retrievedInfo.Placements), "Number of placements should match")

	// Detailed check on each placement
	for i, placement := range retrievedInfo.Placements {
		s.Require().Equal(removalInfo.Placements[i].TopicId, placement.TopicId, "Topic IDs should match for all placements")
		s.Require().Equal(removalInfo.Placements[i].Reputer, placement.Reputer, "Reputer addresses should match for all placements")
		s.Require().Equal(removalInfo.Placements[i].Amount, placement.Amount, "Amounts should match for all placements")
	}
}

func (s *KeeperTestSuite) TestGetStakeRemovalQueueByAddressNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	address := sdk.AccAddress("sampleAddress2")

	// Attempt to retrieve stake removal info for an address with no set info
	_, err := keeper.GetStakeRemovalQueueByAddress(ctx, address)
	s.Require().Error(err)
	s.Require().True(errors.Is(err, collections.ErrNotFound), "Should return not found error for missing stake removal information")
}

func (s *KeeperTestSuite) TestGetStakePlacementsByReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputerAddr := sdk.AccAddress("reputerAddress1")

	// Set up stakes for the reputer
	topicId1 := uint64(101)
	topicId2 := uint64(102)
	stake1 := cosmosMath.NewUint(100)
	stake2 := cosmosMath.NewUint(200)

	// Add stakes to two different topics for the same reputer
	err := keeper.AddStake(ctx, topicId1, reputerAddr, stake1)
	s.Require().NoError(err)
	err = keeper.AddStake(ctx, topicId2, reputerAddr, stake2)
	s.Require().NoError(err)

	// Retrieve the stake placements for the reputer
	stakes, err := keeper.GetStakePlacementsByReputer(ctx, reputerAddr)
	s.Require().NoError(err)
	s.Require().Len(stakes, 2, "Should return two stake placements")

	// Check that the returned stakes contain the correct topic IDs and amounts
	for _, stake := range stakes {
		s.Require().True(stake.TopicId == topicId1 || stake.TopicId == topicId2, "Topic ID should be either of the two added")
		if stake.TopicId == topicId1 {
			s.Require().Equal(stake1, stake.Amount, "Amount should match the stake added for TopicId1")
		} else {
			s.Require().Equal(stake2, stake.Amount, "Amount should match the stake added for TopicId2")
		}
	}
}

func (s *KeeperTestSuite) TestGetStakePlacementsByTopic() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(101)

	// Reputer addresses
	reputerAddr1 := sdk.AccAddress("reputerAddress1")
	reputerAddr2 := sdk.AccAddress("reputerAddress2")

	// Stake amounts
	stake1 := cosmosMath.NewUint(100)
	stake2 := cosmosMath.NewUint(200)

	// Add stakes for different reputers under the same topic
	err := keeper.AddStake(ctx, topicId, reputerAddr1, stake1)
	s.Require().NoError(err)
	err = keeper.AddStake(ctx, topicId, reputerAddr2, stake2)
	s.Require().NoError(err)

	// Retrieve the stake placements for the topic
	stakes, err := keeper.GetStakePlacementsByTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(stakes, 2, "Should return two stake placements")

	// Validate the correctness of the data retrieved
	foundStake1 := false
	foundStake2 := false
	for _, stake := range stakes {
		s.Require().Equal(topicId, stake.TopicId, "Topic ID should match the one queried")
		if stake.Reputer == reputerAddr1.String() && stake.Amount.Equal(stake1) {
			foundStake1 = true
		} else if stake.Reputer == reputerAddr2.String() && stake.Amount.Equal(stake2) {
			foundStake2 = true
		}
	}
	s.Require().True(foundStake1, "Should find stake placement for Reputer1")
	s.Require().True(foundStake2, "Should find stake placement for Reputer2")
}

func (s *KeeperTestSuite) TestGetStakePlacementsByTopicWithNoStakes() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(102)

	// Ensure no stakes are set for this topic
	stakes, err := keeper.GetStakePlacementsByTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(stakes, "Should return an empty slice when no stakes are found for the topic")
}

func (s *KeeperTestSuite) TestRewardsUpdate() {
	noInitLastRewardsUpdate, err := s.emissionsKeeper.GetLastRewardsUpdate(s.ctx)
	s.NoError(err, "error getting un-initialized")
	s.Require().Equal(int64(0), noInitLastRewardsUpdate, "Last rewards update should be zero")

	err = s.emissionsKeeper.SetLastRewardsUpdate(s.ctx, 100)
	s.NoError(err, "error setting")

	lastRewardsUpdate, err := s.emissionsKeeper.GetLastRewardsUpdate(s.ctx)
	s.NoError(err, "error getting")
	s.Require().Equal(int64(100), lastRewardsUpdate, "Last rewards update should be 100")
}

func (s *KeeperTestSuite) TestSetRequestDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	amount := cosmosMath.NewUint(1000)
	requestId := "0xa948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"

	// Set demand
	err := keeper.SetRequestDemand(ctx, requestId, amount)
	s.Require().NoError(err)

	// Check demand
	demand, err := keeper.GetRequestDemand(ctx, requestId)
	s.Require().NoError(err)
	s.Require().Equal(amount, demand, "Demand should be equal to the set amount")
}

func (s *KeeperTestSuite) TestAddToMempool() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	inferenceRequest := types.InferenceRequest{
		Sender:               sdk.AccAddress(PKS[0].Address()).String(),
		Nonce:                1,
		TopicId:              1,
		Cadence:              60 * 60 * 24,
		MaxPricePerInference: cosmosMath.NewUint(1000),
		BidAmount:            cosmosMath.NewUint(1446),
		BlockValidUntil:      0x14,
		BlockLastChecked:     0,
		ExtraData:            []byte("extra data"),
	}
	requestId, err := inferenceRequest.GetRequestId()
	s.Require().NoError(err, "error getting request id")

	// Add to mempool
	err = keeper.AddToMempool(ctx, inferenceRequest)
	s.Require().NoError(err, "Error adding to mempool")

	// Check mempool
	mempool, err := keeper.GetMempoolInferenceRequestById(ctx, inferenceRequest.TopicId, requestId)
	s.Require().NoError(err)
	s.Require().Equal(inferenceRequest, mempool, "Mempool should contain the added inference request")
}

func (s *KeeperTestSuite) TestGetMempoolInferenceRequestsForTopicSimple() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	var i uint64
	var inferenceRequestMap = make(map[string]types.InferenceRequest)
	for i = 0; i < 10; i++ {
		inferenceRequest := types.InferenceRequest{
			Sender:               sdk.AccAddress(PKS[0].Address()).String(),
			Nonce:                i,
			TopicId:              1,
			Cadence:              60 * 60 * 24,
			MaxPricePerInference: cosmosMath.NewUint(1000 * i),
			BidAmount:            cosmosMath.NewUint(1446 * i),
			BlockValidUntil:      0x14,
			BlockLastChecked:     0x0,
			ExtraData:            []byte(fmt.Sprintf("%d extra data", i)),
		}
		// Add to mempool
		err := keeper.AddToMempool(ctx, inferenceRequest)
		s.Require().NoError(err, "Error adding to mempool")
		requestId, err := inferenceRequest.GetRequestId()
		s.Require().NoError(err, "error getting request id 1")
		inferenceRequestMap[requestId] = inferenceRequest
	}

	requestsForTopic, err := keeper.GetMempoolInferenceRequestsForTopic(ctx, 1)
	s.Require().NoError(err, "error getting requests for topic")
	for _, request := range requestsForTopic {
		requestId, err := request.GetRequestId()
		s.Require().NoError(err, "error getting request id 2")
		s.Require().Contains(inferenceRequestMap, requestId, "Mempool should contain the added inference request id")
		expected := inferenceRequestMap[requestId]
		s.Require().Equal(expected, request, "Mempool should contain the added inference request")
	}
}

func (s *KeeperTestSuite) TestGetMempoolSimple() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	var i uint64
	var inferenceRequestMap = make(map[string]types.InferenceRequest)
	for i = 0; i < 10; i++ {
		inferenceRequest := types.InferenceRequest{
			Sender:               sdk.AccAddress(PKS[0].Address()).String(),
			Nonce:                i,
			TopicId:              i,
			Cadence:              60 * 60 * 24,
			MaxPricePerInference: cosmosMath.NewUint(1000 * i),
			BidAmount:            cosmosMath.NewUint(1446 * i),
			BlockValidUntil:      0x14,
			BlockLastChecked:     0x0,
			ExtraData:            []byte(fmt.Sprintf("%d extra data", i)),
		}
		// Add to mempool
		err := keeper.AddToMempool(ctx, inferenceRequest)
		s.Require().NoError(err, "Error adding to mempool")
		requestId, err := inferenceRequest.GetRequestId()
		s.Require().NoError(err, "error getting request id 1")
		inferenceRequestMap[requestId] = inferenceRequest
	}

	mempool, err := keeper.GetMempool(ctx)
	s.Require().NoError(err, "error getting mempool")

	for _, request := range mempool {
		requestId, err := request.GetRequestId()
		s.Require().NoError(err, "error getting request id 2")
		s.Require().Contains(inferenceRequestMap, requestId, "Mempool should contain the added inference request id")
		expected := inferenceRequestMap[requestId]
		s.Require().Equal(expected, request, "Mempool should contain the added inference request")
	}
}

func (s *KeeperTestSuite) TestSetParams() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	params := types.Params{
		Version:                     "v1.0.0",
		RewardCadence:               60 * 60 * 24 * 7 * 24,
		MinTopicUnmetDemand:         cosmosMath.NewUint(100),
		MaxTopicsPerBlock:           1000,
		MinRequestUnmetDemand:       cosmosMath.NewUint(1),
		MaxMissingInferencePercent:  alloraMath.NewDecFromInt64(10),
		RequiredMinimumStake:        cosmosMath.NewUint(1),
		RemoveStakeDelayWindow:      172800,
		MinEpochLength:              60,
		MaxInferenceRequestValidity: 60 * 60 * 24 * 7 * 24,
		MaxRequestCadence:           60 * 60 * 24 * 7 * 24,
		MaxWorkersPerTopicRequest:   10,
		MaxReputersPerTopicRequest:  10,
	}

	// Set params
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Check params
	paramsFromKeeper, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(params.Version, paramsFromKeeper.Version, "Params should be equal to the set params: Version")
	s.Require().Equal(params.RewardCadence, paramsFromKeeper.RewardCadence, "Params should be equal to the set params: EpochLength")
	s.Require().True(params.MinTopicUnmetDemand.Equal(paramsFromKeeper.MinTopicUnmetDemand), "Params should be equal to the set params: MinTopicUnmetDemand")
	s.Require().Equal(params.MaxTopicsPerBlock, paramsFromKeeper.MaxTopicsPerBlock, "Params should be equal to the set params: MaxTopicsPerBlock")
	s.Require().True(params.MinRequestUnmetDemand.Equal(paramsFromKeeper.MinRequestUnmetDemand), "Params should be equal to the set params: MinRequestUnmetDemand")
	s.Require().Equal(params.MaxMissingInferencePercent, paramsFromKeeper.MaxMissingInferencePercent, "Params should be equal to the set params: MaxMissingInferencePercent")
	s.Require().True(params.RequiredMinimumStake.Equal(paramsFromKeeper.RequiredMinimumStake), "Params should be equal to the set params: RequiredMinimumStake")
	s.Require().Equal(params.RemoveStakeDelayWindow, paramsFromKeeper.RemoveStakeDelayWindow, "Params should be equal to the set params: RemoveStakeDelayWindow")
	s.Require().Equal(params.MinEpochLength, paramsFromKeeper.MinEpochLength, "Params should be equal to the set params: MinEpochLength")
	s.Require().Equal(params.MaxInferenceRequestValidity, paramsFromKeeper.MaxInferenceRequestValidity, "Params should be equal to the set params: MaxInferenceRequestValidity")
	s.Require().Equal(params.MaxRequestCadence, paramsFromKeeper.MaxRequestCadence, "Params should be equal to the set params: MaxRequestCadence")
	s.Require().Equal(params.MaxWorkersPerTopicRequest, paramsFromKeeper.MaxWorkersPerTopicRequest, "Params should be equal to the set params: MaxWorkersPerTopicRequest")
	s.Require().Equal(params.MaxReputersPerTopicRequest, paramsFromKeeper.MaxReputersPerTopicRequest, "Params should be equal to the set params: MaxReputersPerTopicRequest")
}

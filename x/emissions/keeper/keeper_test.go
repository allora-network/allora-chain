package keeper_test

import (
	"errors"
	"fmt"
	"strconv"
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
	"github.com/cosmos/cosmos-sdk/types/query"
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
	s.Require().Len(initialNonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

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
	s.Require().Len(initialNonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		success, err := keeper.FulfillWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().True(success, "Nonce should be successfully fulfilled")
		s.Require().NoError(err, "Error fulfilling nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
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
		err := keeper.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, int(maxUnfulfilledRequests), "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.BlockHeight, "Nonce should match the expected recent nonce")
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
	newReputerNonce := &types.Nonce{BlockHeight: 42}
	newWorkerNonce := &types.Nonce{BlockHeight: 43}
	err = keeper.AddReputerNonce(ctx, topicId, newReputerNonce, newWorkerNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(
		newReputerNonce.BlockHeight,
		unfulfilledNonces.Nonces[0].ReputerNonce.BlockHeight,
		"Unfulfilled nonces should contain the new reputer nonce")
	s.Require().Equal(
		newWorkerNonce.BlockHeight,
		unfulfilledNonces.Nonces[0].WorkerNonce.BlockHeight,
		"Unfulfilled nonces should contain the new worker nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedReputerNonceIsUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{BlockHeight: 42}
	newWorkerNonce := &types.Nonce{BlockHeight: 43}

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
	newReputerNonce := &types.Nonce{BlockHeight: 42}
	newWorkerNonce := &types.Nonce{BlockHeight: 43}

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
		err = keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val}, &types.Nonce{BlockHeight: val})
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
		err := keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val}, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := keeper.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, int(maxUnfulfilledRequests), "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.ReputerNonce.BlockHeight, "Nonce should match the expected recent nonce")
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
	gotRegret, _, err := keeper.GetInfererNetworkRegret(ctx, topicId, worker)
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
	gotRegret, noPrior, err := keeper.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
	s.Require().Equal(noPrior, false)
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
	gotRegret, noPrior, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
	s.Require().Equal(noPrior, false)
}

func (s *KeeperTestSuite) TestGetInfererNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("nonexistent-inferer-address")

	// Attempt to get Inferer Network Regret for a nonexistent worker
	regret, noPrior, err := keeper.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}, regret, "Default regret value should be returned for nonexistent inferer")
	s.Require().Equal(noPrior, true, "No prior regret should be returned for nonexistent inferer")
}

func (s *KeeperTestSuite) TestGetForecasterNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("nonexistent-forecaster-address")

	// Attempt to get Forecaster Network Regret for a nonexistent worker
	regret, noPrior, err := keeper.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}, regret, "Default regret value should be returned for nonexistent forecaster")
	s.Require().Equal(noPrior, true, "No prior regret should be returned for nonexistent forecaster")
}

func (s *KeeperTestSuite) TestGetOneInForecasterNetworkRegretNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	forecaster := sdk.AccAddress("nonexistent-forecaster-address")
	inferer := sdk.AccAddress("nonexistent-inferer-address")

	// Attempt to get One-In Forecaster Network Regret for a nonexistent forecaster-inferer pair
	regret, noPrior, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}, regret, "Default regret value should be returned for nonexistent forecaster-inferer pair")
	s.Require().Equal(noPrior, true, "No prior regret should be returned for nonexistent forecaster-inferer pair")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentInfererRegrets() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("worker-address")

	// Topic IDs
	topicId1 := uint64(1)
	topicId2 := uint64(2)

	// Zero regret for initial check
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, noPrior1, err := keeper.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")
	s.Require().Equal(true, noPrior1, "Should return true for no prior regret on Topic ID 1")

	gotRegret2, noPrior2, err := keeper.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")
	s.Require().Equal(true, noPrior2, "Should return true for no prior regret on Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err = keeper.SetInfererNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetInfererNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, noPrior1, err = keeper.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)
	s.Require().Equal(false, noPrior1, "Should return false indicating prior regret is now set for Topic ID 1")

	gotRegret2, noPrior2, err = keeper.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)
	s.Require().Equal(false, noPrior2, "Should return false indicating prior regret is now set for Topic ID 2")

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
	noRagret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	gotRegret1, noPrior1, err := keeper.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRagret, gotRegret1)
	s.Require().Equal(noPrior1, true)

	// Set regrets for the same worker under different topic IDs
	err = keeper.SetForecasterNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = keeper.SetForecasterNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, noPrior1, err = keeper.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)
	s.Require().Equal(noPrior1, false)

	gotRegret2, noPrior2, err := keeper.GetForecasterNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)
	s.Require().Equal(noPrior2, false)

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

	// Zero regret for initial checks
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, noPrior1, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")
	s.Require().Equal(true, noPrior1, "Should return true indicating no prior regret for Topic ID 1")

	gotRegret2, noPrior2, err := keeper.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")
	s.Require().Equal(true, noPrior2, "Should return true indicating no prior regret for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same forecaster-inferer pair under different topic IDs
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer, regret1)
	s.Require().NoError(err)
	err = keeper.SetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, noPrior1, err = keeper.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)
	s.Require().Equal(false, noPrior1, "Should return false now that prior regret is set for Topic ID 1")

	gotRegret2, noPrior2, err = keeper.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)
	s.Require().Equal(false, noPrior2, "Should return false now that prior regret is set for Topic ID 2")

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
	expectedValue := alloraMath.MustNewDecFromString("0.25") // Assume a function to create LegacyDec

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

func (s *KeeperTestSuite) TestGetParamsMaxTopWorkersToReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.Params{MaxTopWorkersToReward: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxTopWorkersToReward(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopWorkersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxRetriesToFulfilNoncesWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := int64(5) // Example expected value

	// Set the parameter
	params := types.Params{MaxRetriesToFulfilNoncesWorker: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxRetriesToFulfilNoncesWorker(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxRetriesToFulfilNoncesWorker should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxRetriesToFulfilNoncesReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedValue := int64(5) // Example expected value

	// Set the parameter
	params := types.Params{MaxRetriesToFulfilNoncesReputer: expectedValue}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	actualValue, err := keeper.GetParamsMaxRetriesToFulfilNoncesReputer(ctx)
	s.Require().NoError(err)
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxRetriesToFulfilNoncesReputer should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsRewardCadence() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	expectedCadence := int64(86400) // Assume a daily cadence (in seconds if applicable)

	// Set the parameter first to ensure there is something to retrieve
	params := types.Params{RewardCadence: expectedCadence}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Now test getting the reward cadence
	actualCadence, err := keeper.GetParamsRewardCadence(ctx)
	s.Require().NoError(err, "Getting reward cadence should not fail")
	s.Require().Equal(expectedCadence, actualCadence, "The retrieved reward cadence should match the expected value")
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
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
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
	nonce := types.Nonce{BlockHeight: int64(block)}
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

	nonce0 := types.Nonce{BlockHeight: block0}
	nonce1 := types.Nonce{BlockHeight: block1}
	nonce2 := types.Nonce{BlockHeight: block2}
	nonce3 := types.Nonce{BlockHeight: block3}
	nonce4 := types.Nonce{BlockHeight: block4}

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

	nonce0 := types.Nonce{BlockHeight: block0}
	nonce1 := types.Nonce{BlockHeight: block1}
	nonce2 := types.Nonce{BlockHeight: block2}
	nonce3 := types.Nonce{BlockHeight: block3}
	nonce4 := types.Nonce{BlockHeight: block4}

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
	delegatorStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount, delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")

	// Add additional stake
	err = keeper.AddDelegatedStake(ctx, topicId, delegatorAddr, reputerAddr, additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err = keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
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
	remainingStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
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
	remainingStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
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
	amount, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)

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

func (s *KeeperTestSuite) TestSetAndGetDelegatedStakeRemovalQueueByAddress() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	address := sdk.AccAddress("sampleAddress1")

	// Create sample delegated stake removal information
	removalInfo := types.DelegatedStakeRemoval{
		BlockRemovalStarted: time.Now().Unix(),
		Placements: []*types.DelegatedStakePlacement{
			{
				TopicId:   201,
				Reputer:   "reputer3",
				Delegator: "delegator3",
				Amount:    cosmosMath.NewUint(300),
			},
			{
				TopicId:   202,
				Reputer:   "reputer4",
				Delegator: "delegator4",
				Amount:    cosmosMath.NewUint(400),
			},
		},
	}

	// Set delegated stake removal information
	err := keeper.SetDelegatedStakeRemovalQueueForAddress(ctx, address, removalInfo)
	s.Require().NoError(err)

	// Retrieve the delegated stake removal information
	retrievedInfo, err := keeper.GetDelegatedStakeRemovalQueueByAddress(ctx, address)
	s.Require().NoError(err)
	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(len(removalInfo.Placements), len(retrievedInfo.Placements), "Number of delegated placements should match")

	// Detailed check on each delegated placement
	for i, placement := range retrievedInfo.Placements {
		s.Require().Equal(removalInfo.Placements[i].TopicId, placement.TopicId, "Topic IDs should match for all placements")
		s.Require().Equal(removalInfo.Placements[i].Reputer, placement.Reputer, "Reputer addresses should match for all placements")
		s.Require().Equal(removalInfo.Placements[i].Delegator, placement.Delegator, "Delegator addresses should match for all placements")
		s.Require().Equal(removalInfo.Placements[i].Amount, placement.Amount, "Amounts should match for all placements")
	}
}

func (s *KeeperTestSuite) TestGetDelegatedStakeRemovalQueueByAddressNotFound() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	address := sdk.AccAddress("sampleAddress2")

	// Attempt to retrieve delegated stake removal info for an address with no set info
	_, err := keeper.GetDelegatedStakeRemovalQueueByAddress(ctx, address)
	s.Require().Error(err)
	s.Require().True(errors.Is(err, collections.ErrNotFound), "Should return not found error for missing delegated stake removal information")
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

	pagination := &query.PageRequest{
		Key:        nil,
		Offset:     0,
		Limit:      10,
		CountTotal: false,
	}
	requestsForTopic, _, err := keeper.GetMempoolInferenceRequestsForTopic(ctx, 1, pagination)
	s.Require().NoError(err, "error getting requests for topic")
	for _, request := range requestsForTopic {
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

// / REPUTERS AND WORKER
func (s *KeeperTestSuite) TestInsertWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("sampleWorkerAddress")
	key := "worker-libp2p-key-sample"
	topicId := uint64(401)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    key,
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
	}

	// Attempt to insert the worker for multiple topics
	err := keeper.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)

	node, err := keeper.GetWorkerByLibp2pKey(ctx, key)

	s.Require().NoError(err)
	s.Require().Equal(workerInfo.LibP2PKey, node.LibP2PKey)
	s.Require().Equal(workerInfo.MultiAddress, node.MultiAddress)
	s.Require().Equal(workerInfo.Owner, node.Owner)
	s.Require().Equal(workerInfo.NodeAddress, node.NodeAddress)
	s.Require().Equal(workerInfo.NodeId, node.NodeId)
}

func (s *KeeperTestSuite) TestGetWorkerAddressByP2PKey() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("sampleWorkerAddress")
	topicId := uint64(401)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
	}

	// Attempt to insert the worker for multiple topics
	err := keeper.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)

	// Call the function to get the worker address using the P2P key
	retrievedAddress, err := keeper.GetWorkerAddressByP2PKey(ctx, workerInfo.LibP2PKey)
	s.Require().NoError(err)
	workerAddress, err := sdk.AccAddressFromBech32(workerInfo.Owner)
	s.Require().NoError(err)
	s.Require().Equal(workerAddress, retrievedAddress)
}

func (s *KeeperTestSuite) TestRemoveWorker() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	worker := sdk.AccAddress("sampleWorkerAddress")
	topicId := uint64(401) // Assume the worker is associated with this topicId initially

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		LibP2PKey:    "worker-libp2p-key-sample",
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
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
	reputer := sdk.AccAddress("sampleReputerAddress")
	topicId := uint64(501)

	// Define sample OffchainNode information for a reputer
	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	// Attempt to insert the reputer for multiple topics
	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	// Optionally check if reputer is registered in each topic using an assumed IsReputerRegisteredInTopic method
	isRegistered, regErr := keeper.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(regErr, "Checking reputer registration should not fail")
	s.Require().True(isRegistered, "Reputer should be registered in each topic")
}

func (s *KeeperTestSuite) TestRemoveReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputer := sdk.AccAddress("sampleReputerAddress")
	topicId := uint64(501)

	// Pre-setup: Insert the reputer for initial setup
	err := keeper.InsertReputer(ctx, topicId, reputer, types.OffchainNode{Owner: "sample-owner"})
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

func (s *KeeperTestSuite) TestGetReputerAddressByP2PKey() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputer := sdk.AccAddress("sampleReputerAddress")
	topicId := uint64(501)

	// Define sample OffchainNode information for a reputer
	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	// Insert the reputer for multiple topics
	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	// Retrieve the reputer address using the P2P key
	retrievedAddress, err := keeper.GetReputerAddressByP2PKey(ctx, reputerInfo.LibP2PKey)
	s.Require().NoError(err)
	expectedAddress, err := sdk.AccAddressFromBech32(reputerInfo.Owner)
	s.Require().NoError(err)
	s.Require().Equal(expectedAddress, retrievedAddress, "The retrieved address should match the expected address")
}

/// TOPICS

func (s *KeeperTestSuite) TestSetAndGetPreviousTopicWeight() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := types.PreviousTopicWeight{Weight: alloraMath.NewDecFromInt64(10), Epoch: 5}
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
	s.Require().Equal(types.PreviousTopicWeight{Weight: alloraMath.ZeroDec(), Epoch: 0}, retrievedWeight, "Weight for an unset topic should be zero")
	s.Require().True(noPrior, "Should indicate no prior weight for an unset topic")
}

func (s *KeeperTestSuite) TestInactivateAndActivateTopic() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(3)

	// Assume topic initially active
	initialTopic := types.Topic{Id: topicId}
	_ = keeper.SetTopic(ctx, topicId, initialTopic)

	// Activate the topic
	err := keeper.ActivateTopic(ctx, topicId)
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

func (s *KeeperTestSuite) TestGetActiveTopics() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Clear existing topics (if possible)
	// This step is hypothetical and depends on your system's capabilities
	// _ = keeper.ClearAllTopics(ctx)

	// Create sample topics with mixed active states
	topic1 := types.Topic{Id: 1}
	topic2 := types.Topic{Id: 2}
	topic3 := types.Topic{Id: 3}

	// Set topics in the system
	_ = keeper.SetTopic(ctx, topic1.Id, topic1)
	_ = keeper.ActivateTopic(ctx, topic1.Id)
	_ = keeper.SetTopic(ctx, topic2.Id, topic2) // Inactive topic
	_ = keeper.SetTopic(ctx, topic3.Id, topic3)
	_ = keeper.ActivateTopic(ctx, topic1.Id)

	// Fetch only active topics
	pagination := &query.PageRequest{
		Key:        nil,
		Offset:     0,
		Limit:      10,
		CountTotal: false,
	}
	activeTopics, _, err := keeper.GetActiveTopics(ctx, pagination)
	s.Require().NoError(err, "Fetching active topics should not produce an error")

	// Verify the correct number of active topics is retrieved
	s.Require().Len(activeTopics, 2, "Should retrieve exactly two active topics")

	// Verify the correctness of the data retrieved, specifically checking active status
	for _, topic := range activeTopics {
		isActive, err := keeper.IsTopicActive(ctx, topic.Id)
		s.Require().NoError(err, "Checking topic activity should not fail")
		s.Require().True(isActive, "Only active topics should be returned")
		switch topic.Id {
		case 1:
			s.Require().Equal(topic1, *topic, "The details of topic 1 should match")
		case 3:
			s.Require().Equal(topic3, *topic, "The details of topic 3 should match")
		default:
			s.Fail("Unexpected topic ID retrieved")
		}
	}
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

		newTopic := types.Topic{Id: topicId}

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
	initialTopic := types.Topic{Id: topicId}
	_ = keeper.SetTopic(ctx, topicId, initialTopic)

	// Update the epoch last ended
	err := keeper.UpdateTopicEpochLastEnded(ctx, topicId, epochLastEnded)
	s.Require().NoError(err, "Updating topic epoch last ended should not fail")

	// Retrieve the last ended epoch for the topic
	retrievedEpoch, err := keeper.GetTopicEpochLastEnded(ctx, topicId)
	s.Require().NoError(err, "Retrieving topic epoch last ended should not fail")
	s.Require().Equal(epochLastEnded, retrievedEpoch, "The retrieved epoch last ended should match the updated value")
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

	newTopic := types.Topic{Id: existentTopicId}

	err = keeper.SetTopic(ctx, existentTopicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test the newly created topic ID
	exists, err = keeper.TopicExists(ctx, existentTopicId)
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

/// FEE REVENUE

func (s *KeeperTestSuite) TestGetTopicFeeRevenue() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test getting revenue for a topic with no existing revenue
	feeRev, err := keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when revenue does not exist")
	s.Require().Equal(cosmosMath.ZeroInt(), feeRev.Revenue, "Revenue should be zero for non-existing entries")
	s.Require().Equal(uint64(0), feeRev.Epoch, "Epoch should be zero for non-existing entries")

	// Setup a topic with some revenue
	initialRevenue := cosmosMath.NewUint(100)
	initialRevenueInt := cosmosMath.NewInt(100)
	keeper.AddTopicFeeRevenue(ctx, topicId, initialRevenue)

	// Test getting revenue for a topic with existing revenue
	feeRev, err = keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when retrieving existing revenue")
	s.Require().True(feeRev.Revenue.Equal(initialRevenueInt), "Revenue should match the initial setup")
}

func (s *KeeperTestSuite) TestAddTopicFeeRevenueAndIncrementEpoch() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Add initial revenue in the first epoch
	initialAmount := cosmosMath.NewUint(100)
	err := keeper.AddTopicFeeRevenue(ctx, topicId, initialAmount)
	s.Require().NoError(err, "Adding initial revenue should not fail")

	// Verify initial revenue
	feeRev, _ := keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().Equal(initialAmount.BigInt(), feeRev.Revenue.BigInt(), "Initial revenue should be correctly recorded")

	// Increment fee revenue epoch
	err = keeper.IncrementFeeRevenueEpoch(ctx)
	s.Require().NoError(err, "Incrementing fee revenue epoch should not fail")

	// Add more revenue in the new epoch
	additionalAmount := cosmosMath.NewUint(200)
	err = keeper.AddTopicFeeRevenue(ctx, topicId, additionalAmount)
	s.Require().NoError(err, "Adding additional revenue in new epoch should not fail")

	// Verify updated revenue in the new epoch
	updatedFeeRev, _ := keeper.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NotEqual(feeRev.Epoch, updatedFeeRev.Epoch, "Epoch should be updated")
	s.Require().Equal(additionalAmount.BigInt(), updatedFeeRev.Revenue.BigInt(), "Revenue in new epoch should match the additional amount")
}

func (s *KeeperTestSuite) TestGetFeeRevenueEpoch() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Check initial epoch
	initialEpoch, err := keeper.GetFeeRevenueEpoch(ctx)
	s.Require().NoError(err, "Fetching initial fee revenue epoch should not fail")

	// Increment the epoch
	err = keeper.IncrementFeeRevenueEpoch(ctx)
	s.Require().NoError(err, "Incrementing fee revenue epoch should not fail")

	// Check updated epoch
	updatedEpoch, err := keeper.GetFeeRevenueEpoch(ctx)
	s.Require().NoError(err, "Fetching updated fee revenue epoch should not fail")
	s.Require().Equal(initialEpoch+1, updatedEpoch, "Updated epoch should be incremented by one")
}

/// MEMPOOL & INFERENCE REQUESTS

func (s *KeeperTestSuite) TestAddUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	addAmount := cosmosMath.NewUint(50)

	// Initial add should set demand since it starts at zero
	err := keeper.AddUnmetDemand(ctx, topicId, addAmount)
	s.Require().NoError(err, "Adding unmet demand should not fail")

	// Verify the addition
	demand, err := keeper.GetTopicUnmetDemand(ctx, topicId)
	s.Require().NoError(err, "Fetching unmet demand should not fail after addition")
	s.Require().Equal(addAmount, demand, "Unmet demand should match the added amount")

	// Add more to the existing demand
	additionalAmount := cosmosMath.NewUint(30)
	err = keeper.AddUnmetDemand(ctx, topicId, additionalAmount)
	s.Require().NoError(err, "Adding more unmet demand should not fail")

	// Verify new demand
	newDemand, err := keeper.GetTopicUnmetDemand(ctx, topicId)
	s.Require().NoError(err, "Fetching new unmet demand should not fail")
	expectedDemand := addAmount.Add(additionalAmount)
	s.Require().Equal(expectedDemand, newDemand, "Unmet demand should be correctly accumulated")
}

func (s *KeeperTestSuite) TestRemoveUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(2)
	initialDemand := cosmosMath.NewUint(100)
	removeAmount := cosmosMath.NewUint(50)

	// Set initial unmet demand
	_ = keeper.SetTopicUnmetDemand(ctx, topicId, initialDemand)

	// Remove some demand
	err := keeper.RemoveUnmetDemand(ctx, topicId, removeAmount)
	s.Require().NoError(err, "Removing unmet demand should not fail")

	// Verify removal
	remainingDemand, err := keeper.GetTopicUnmetDemand(ctx, topicId)
	s.Require().NoError(err, "Fetching remaining unmet demand should not fail")
	s.Require().Equal(initialDemand.Sub(removeAmount), remainingDemand, "Unmet demand should be correctly subtracted")

	// Attempt to remove more than exists, should fail
	largeRemoveAmount := cosmosMath.NewUint(200)
	err = keeper.RemoveUnmetDemand(ctx, topicId, largeRemoveAmount)
	s.Require().Error(err, "Should error when removing more demand than exists")
	s.Require().IsType(types.ErrIntegerUnderflowUnmetDemand, err, "Error should be of type ErrIntegerUnderflowUnmetDemand")
}

func (s *KeeperTestSuite) TestSetTopicUnmetDemand() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(3)
	setAmount := cosmosMath.NewUint(120)

	// Set specific unmet demand
	err := keeper.SetTopicUnmetDemand(ctx, topicId, setAmount)
	s.Require().NoError(err, "Setting unmet demand should not fail")

	// Verify set demand
	demand, err := keeper.GetTopicUnmetDemand(ctx, topicId)
	s.Require().NoError(err, "Fetching set unmet demand should not fail")
	s.Require().Equal(setAmount, demand, "Unmet demand should exactly match the set amount")

	// Set to zero to test removal
	err = keeper.SetTopicUnmetDemand(ctx, topicId, cosmosMath.NewUint(0))
	s.Require().NoError(err, "Setting unmet demand to zero should not fail")

	// Verify removal
	zeroDemand, err := keeper.GetTopicUnmetDemand(ctx, topicId)
	s.Require().NoError(err, "Fetching unmet demand after setting to zero should not fail")
	s.Require().Equal(cosmosMath.ZeroUint(), zeroDemand, "Unmet demand should be zero after being set to zero")
}

func (s *KeeperTestSuite) TestAddToMempool2() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	request := types.InferenceRequest{
		Sender:    "allo1zf8q4lzfnavru3etczeamd3esu3yalzj2puq5p",
		Nonce:     123,
		TopicId:   topicId,
		BidAmount: cosmosMath.NewUint(100),
	}
	requestId, err := request.GetRequestId()
	s.Require().NoError(err, "Getting request ID should not fail")

	// Add request to the mempool
	err = keeper.AddToMempool(ctx, request)
	s.Require().NoError(err, "Adding to mempool should not fail")

	// Check if the request is now in the mempool
	exists, err := keeper.IsRequestInMempool(ctx, topicId, requestId)
	s.Require().NoError(err, "Checking if request is in mempool should not fail")
	s.Require().True(exists, "Request should exist in the mempool after being added")
}

func (s *KeeperTestSuite) TestRemoveFromMempool() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	request := types.InferenceRequest{
		Sender:    "allo1zf8q4lzfnavru3etczeamd3esu3yalzj2puq5p",
		Nonce:     123,
		TopicId:   topicId,
		BidAmount: cosmosMath.NewUint(100),
	}

	requestId, err := request.GetRequestId()
	s.Require().NoError(err, "Getting request ID should not fail")

	// Assume the request is already in the mempool
	_ = keeper.AddToMempool(ctx, request)

	// Remove the request from the mempool
	err = keeper.RemoveFromMempool(ctx, request)
	s.Require().NoError(err, "Removing from mempool should not fail")

	// Check if the request is still in the mempool
	exists, err := keeper.IsRequestInMempool(ctx, topicId, requestId)
	s.Require().NoError(err, "Checking if request is in mempool should not fail after removal")
	s.Require().False(exists, "Request should not exist in the mempool after being removed")
}

func (s *KeeperTestSuite) TestSetChurnReadyTopics() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Define a list of topics to set as churn ready
	topicList := types.TopicList{
		Topics: []*types.Topic{
			{Id: 1, Creator: "Creator1"},
			{Id: 2, Creator: "Creator2"},
		},
	}

	// Set churn ready topics
	err := keeper.SetChurnReadyTopics(ctx, topicList)
	s.Require().NoError(err, "Setting churn ready topics should not fail")

	// Retrieve and verify the set topics
	retrievedList, err := keeper.GetChurnReadyTopics(ctx)
	s.Require().NoError(err, "Fetching churn ready topics should not fail")
	s.Require().Equal(len(topicList.Topics), len(retrievedList.Topics), "The number of topics in the list should match")
	s.Require().Equal(topicList.Topics[0].Id, retrievedList.Topics[0].Id, "The IDs of the churn ready topics should match")
}

func (s *KeeperTestSuite) TestGetChurnReadyTopics() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Attempt to fetch churn ready topics when none are set
	retrievedList, err := keeper.GetChurnReadyTopics(ctx)
	s.Require().NoError(err, "Fetching churn ready topics should not fail even if none are set")
	s.Require().Empty(retrievedList.Topics, "No topics should be returned when none are set")
}

func (s *KeeperTestSuite) TestResetChurnReadyTopics() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Set some churn ready topics first
	topicList := types.TopicList{
		Topics: []*types.Topic{
			{Id: 1, Creator: "Creator1"},
			{Id: 2, Creator: "Creator2"},
		},
	}
	_ = keeper.SetChurnReadyTopics(ctx, topicList)

	// Now reset the churn ready topics
	err := keeper.ResetChurnReadyTopics(ctx)
	s.Require().NoError(err, "Resetting churn ready topics should not fail")

	// Verify the reset by fetching the topics
	retrievedList, err := keeper.GetChurnReadyTopics(ctx)
	s.Require().NoError(err, "Fetching churn ready topics after reset should not fail")
	s.Require().Empty(retrievedList.Topics, "Churn ready topics should be empty after reset")
}

/// SCORES

func (s *KeeperTestSuite) TestGetLatestScores() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	forecaster := "forecaster1"
	reputer := "reputer1"

	// Test getting latest scores when none are set
	infererScore, err := keeper.GetLatestInfererScore(ctx, topicId, sdk.AccAddress(worker))
	s.Require().NoError(err, "Fetching latest inferer score should not fail")
	s.Require().Equal(types.Score{}, infererScore, "Inferer score should be empty if not set")

	forecasterScore, err := keeper.GetLatestForecasterScore(ctx, topicId, sdk.AccAddress(forecaster))
	s.Require().NoError(err, "Fetching latest forecaster score should not fail")
	s.Require().Equal(types.Score{}, forecasterScore, "Forecaster score should be empty if not set")

	reputerScore, err := keeper.GetLatestReputerScore(ctx, topicId, sdk.AccAddress(reputer))
	s.Require().NoError(err, "Fetching latest reputer score should not fail")
	s.Require().Equal(types.Score{}, reputerScore, "Reputer score should be empty if not set")
}

func (s *KeeperTestSuite) TestSetLatestScores() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("worker1")
	forecaster := sdk.AccAddress("forecaster1")
	reputer := sdk.AccAddress("reputer1")
	oldScore := types.Score{TopicId: topicId, BlockNumber: 1, Address: worker.String(), Score: alloraMath.NewDecFromInt64(90)}
	newScore := types.Score{TopicId: topicId, BlockNumber: 2, Address: worker.String(), Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	_ = keeper.SetLatestInfererScore(ctx, topicId, worker, newScore)
	err := keeper.SetLatestInfererScore(ctx, topicId, worker, oldScore)
	s.Require().NoError(err, "Setting an older inferer score should not fail but should not update")
	updatedScore, _ := keeper.GetLatestInfererScore(ctx, topicId, worker)
	s.Require().NotEqual(oldScore.Score, updatedScore.Score, "Older score should not replace newer score")

	// Set a new score for forecaster
	_ = keeper.SetLatestForecasterScore(ctx, topicId, forecaster, newScore)
	forecasterScore, _ := keeper.GetLatestForecasterScore(ctx, topicId, forecaster)
	s.Require().Equal(newScore.Score, forecasterScore.Score, "Newer forecaster score should be set")

	// Set a new score for reputer
	_ = keeper.SetLatestReputerScore(ctx, topicId, reputer, newScore)
	reputerScore, _ := keeper.GetLatestReputerScore(ctx, topicId, reputer)
	s.Require().Equal(newScore.Score, reputerScore.Score, "Newer reputer score should be set")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)
	score := types.Score{
		TopicId:     topicId,
		BlockNumber: blockNumber,
		Address:     "worker1",
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := uint64(5)
	params := types.Params{MaxSamplesToScaleScores: maxNumScores}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < int(maxNumScores+2); i++ {
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockNumber, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, int(maxNumScores), "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore2() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := uint64(5)
	params := types.Params{MaxSamplesToScaleScores: maxNumScores}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < int(maxNumScores+2); i++ { // Inserting 7 scores where the limit is 5
		scoreValue := alloraMath.NewDecFromInt64(int64(90 + i)) // Increment score value to simulate variation
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     "worker1",
			Score:       scoreValue,
		}
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockNumber, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, int(maxNumScores), "Scores should not exceed the maximum limit")

	// Check that the retained scores are the last five inserted
	for idx, score := range scores.Scores {
		expectedScoreValue := alloraMath.NewDecFromInt64(int64(92 + idx)) // Expecting the last 5 scores: 94, 95, 96, 97
		s.Require().Equal(expectedScoreValue, score.Score, "Score should match the expected last scores")
	}
}

func (s *KeeperTestSuite) TestGetWorkerInferenceScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	workerAddress := sdk.AccAddress("allo16jmt7f7r4e6j9k4ds7jgac2t4k4cz0wthv4u88")
	otherWorkerAddress := sdk.AccAddress("allo1k2u0wy9436pz5spxww22vr95lrfr4hhuznamva")
	blockNumber := int64(105)

	// Insert scores for different workers and blocks
	for blockNumber := int64(100); blockNumber <= 110; blockNumber++ {
		// Scores for the targeted worker
		scoreForWorker := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     workerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(blockNumber),
		}
		_ = keeper.InsertWorkerInferenceScore(ctx, topicId, blockNumber, scoreForWorker)

		// Scores for another worker to test filtering
		scoreForOtherWorker := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     otherWorkerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(blockNumber),
		}
		_ = keeper.InsertWorkerInferenceScore(ctx, topicId, blockNumber, scoreForOtherWorker)
	}

	// Get scores for the worker up to block 105
	scores, err := keeper.GetWorkerInferenceScoresUntilBlock(ctx, topicId, blockNumber, workerAddress)
	s.Require().NoError(err, "Fetching worker inference scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")

	// Verify that the scores are correct and ordered as expected (descending block number)
	expectedBlock := blockNumber
	for _, score := range scores {
		s.Require().Equal(workerAddress.String(), score.Address, "Only scores for the specified worker should be returned")
		s.Require().Equal(expectedBlock, score.BlockNumber, "Scores should be returned in descending order by block")
		s.Require().Equal(alloraMath.NewDecFromInt64(expectedBlock), score.Score, "Score value should match expected")
		expectedBlock--
	}
}

func (s *KeeperTestSuite) TestInsertWorkerForecastScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := uint64(5)
	params := types.Params{MaxSamplesToScaleScores: maxNumScores}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < int(maxNumScores+2); i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     "worker1",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, blockNumber, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetWorkerForecastScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, int(maxNumScores), "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	workerAddress := sdk.AccAddress("allo16jmt7f7r4e6j9k4ds7jgac2t4k4cz0wthv4u88")
	blockNumber := int64(105)

	// Insert scores for the worker at various blocks
	for i := int64(100); i <= 110; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: i,
			Address:     workerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(i),
		}
		_ = keeper.InsertWorkerForecastScore(ctx, topicId, i, score)
	}

	// Get forecast scores for the worker up to block 105
	scores, err := keeper.GetWorkerForecastScoresUntilBlock(ctx, topicId, blockNumber, workerAddress)
	s.Require().NoError(err, "Fetching worker forecast scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)

	// Insert scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     "worker" + strconv.Itoa(i+1),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertWorkerForecastScore(ctx, topicId, blockNumber, score)
	}

	// Fetch scores at the specific block
	scores, err := keeper.GetWorkerForecastScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestInsertReputerScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := uint64(5)
	params := types.Params{MaxSamplesToScaleScores: maxNumScores}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < int(maxNumScores+2); i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     "reputer1",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := keeper.InsertReputerScore(ctx, topicId, blockNumber, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, int(maxNumScores), "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetReputersScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockNumber := int64(100)

	// Insert multiple scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockNumber: blockNumber,
			Address:     "reputer" + strconv.Itoa(i+1),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertReputerScore(ctx, topicId, blockNumber, score)
	}

	// Fetch scores at the specific block
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, blockNumber)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestSetListeningCoefficient() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := sdk.AccAddress("sampleReputerAddress")

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
	reputer := sdk.AccAddress("sampleReputerAddress")

	// Attempt to fetch a coefficient before setting it
	defaultCoef, err := keeper.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail when not set")
	s.Require().Equal(alloraMath.NewDecFromInt64(1), defaultCoef.Coefficient, "Should return the default coefficient when not set")

	// Now set a specific coefficient
	setCoef := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(5),
	}
	_ = keeper.SetListeningCoefficient(ctx, topicId, reputer, setCoef)

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
	reputer := sdk.AccAddress("reputerAddressExample")

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
	reputer := sdk.AccAddress("reputerAddressExample")

	// Attempt to fetch a reward fraction before setting it
	defaultReward, _, err := keeper.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(50) // Assuming 0.50 as a fraction example
	_ = keeper.SetPreviousReputerRewardFraction(ctx, topicId, reputer, setReward)

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
	worker := sdk.AccAddress("workerAddressExample")

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
	worker := sdk.AccAddress("workerAddressExample")

	// Attempt to fetch a reward fraction before setting it
	defaultReward, noPrior, err := keeper.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75)
	_ = keeper.SetPreviousInferenceRewardFraction(ctx, topicId, worker, setReward)

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
	worker := sdk.AccAddress("forecastWorkerAddress")

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
	worker := sdk.AccAddress("forecastWorkerAddress")

	// Attempt to fetch the reward fraction before setting it, expecting default value
	defaultReward, noPrior, err := keeper.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero forecast reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75) // Assume setting it to 0.75
	_ = keeper.SetPreviousForecastRewardFraction(ctx, topicId, worker, setReward)

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := keeper.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

/// TAX for REWARD

func (s *KeeperTestSuite) TestSetAverageWorkerReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("uniqueWorkerAddress")

	// Define an average worker reward to set
	averageReward := types.AverageWorkerReward{
		Count: 5,
		Value: alloraMath.NewDecFromInt64(100), // Assume the average reward value is 100
	}

	// Set the average worker reward
	err := keeper.SetAverageWorkerReward(ctx, topicId, worker, averageReward)
	s.Require().NoError(err, "Setting average worker reward should not fail")

	// Verify by fetching the set value
	fetchedReward, err := keeper.GetAverageWorkerReward(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set average worker reward should not fail")
	s.Require().Equal(averageReward, fetchedReward, "The fetched average worker reward should match the set value")
}

func (s *KeeperTestSuite) TestGetAverageWorkerReward() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := sdk.AccAddress("uniqueWorkerAddress")

	// Attempt to fetch the average reward before setting it, expecting a default value
	defaultReward, err := keeper.GetAverageWorkerReward(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching average worker reward should not fail when not set")
	s.Require().Equal(types.AverageWorkerReward{Count: 0, Value: alloraMath.ZeroDec()}, defaultReward, "Should return default average worker reward when not set")

	// Now set a specific average worker reward
	setReward := types.AverageWorkerReward{
		Count: 10,
		Value: alloraMath.NewDecFromInt64(200), // Assume setting it to 200
	}
	_ = keeper.SetAverageWorkerReward(ctx, topicId, worker, setReward)

	// Fetch and verify the average worker reward after setting
	fetchedReward, err := keeper.GetAverageWorkerReward(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching average worker reward should not fail after setting")
	s.Require().Equal(setReward, fetchedReward, "The fetched average worker reward should match the set value")
}

/// WHITELISTS

func (s *KeeperTestSuite) TestWhitelistAdminOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	adminAddress := sdk.AccAddress("adminAddressExample")

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

func (s *KeeperTestSuite) TestTopicCreationWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	address := sdk.AccAddress("creatorAddressExample")

	// Test Adding to whitelist
	err := keeper.AddToTopicCreationWhitelist(ctx, address)
	s.Require().NoError(err, "Adding to topic creation whitelist should not fail")

	// Test Checking whitelist
	isInWhitelist, err := keeper.IsInTopicCreationWhitelist(ctx, address)
	s.Require().NoError(err, "Checking if address is in topic creation whitelist should not fail")
	s.Require().True(isInWhitelist, "Address should be in the topic creation whitelist after being added")

	// Test Removing from whitelist
	err = keeper.RemoveFromTopicCreationWhitelist(ctx, address)
	s.Require().NoError(err, "Removing from topic creation whitelist should not fail")

	// Verify removal
	isInWhitelist, err = keeper.IsInTopicCreationWhitelist(ctx, address)
	s.Require().NoError(err, "Checking topic creation whitelist status after removal should not fail")
	s.Require().False(isInWhitelist, "Address should not be in the topic creation whitelist after being removed")
}

func (s *KeeperTestSuite) TestReputerWhitelistOperations() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	reputerAddress := sdk.AccAddress("reputerAddressExample")

	// Test Adding to whitelist
	err := keeper.AddToReputerWhitelist(ctx, reputerAddress)
	s.Require().NoError(err, "Adding to reputer whitelist should not fail")

	// Test Checking whitelist
	isInWhitelist, err := keeper.IsInReputerWhitelist(ctx, reputerAddress)
	s.Require().NoError(err, "Checking if address is in reputer whitelist should not fail")
	s.Require().True(isInWhitelist, "Address should be in the reputer whitelist after being added")

	// Test Removing from whitelist
	err = keeper.RemoveFromReputerWhitelist(ctx, reputerAddress)
	s.Require().NoError(err, "Removing from reputer whitelist should not fail")

	// Verify removal
	isInWhitelist, err = keeper.IsInReputerWhitelist(ctx, reputerAddress)
	s.Require().NoError(err, "Checking reputer whitelist status after removal should not fail")
	s.Require().False(isInWhitelist, "Address should not be in the reputer whitelist after being removed")
}

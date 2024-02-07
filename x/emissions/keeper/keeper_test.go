package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"

	// simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	state "github.com/upshot-tech/protocol-state-machine-module"
	"github.com/upshot-tech/protocol-state-machine-module/keeper"
	emissionstestutil "github.com/upshot-tech/protocol-state-machine-module/testutil"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx          sdk.Context
	bankKeeper   *emissionstestutil.MockBankKeeper
	authKeeper   *emissionstestutil.MockAccountKeeper
	upshotKeeper keeper.Keeper
	msgServer    state.MsgServer
	mockCtrl     *gomock.Controller
	key          *storetypes.KVStoreKey
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("upshot")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := address.NewBech32Codec("upt")
	ctrl := gomock.NewController(s.T())

	s.bankKeeper = emissionstestutil.NewMockBankKeeper(ctrl)
	s.authKeeper = emissionstestutil.NewMockAccountKeeper(ctrl)

	s.ctx = ctx
	s.upshotKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, s.authKeeper, s.bankKeeper)
	s.msgServer = keeper.NewMsgServerImpl(s.upshotKeeper)
	s.mockCtrl = ctrl
	s.key = key
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// ########################################
// #           Staking tests              #
// ########################################

func (s *KeeperTestSuite) TestGetSetTotalStake() {
	ctx := s.ctx
	keeper := s.upshotKeeper

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
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Initial Values
	initialTotalStake := cosmosMath.NewUint(0)
	initialTopicStake := cosmosMath.NewUint(0)
	initialTargetStake := cosmosMath.NewUint(0)

	// Add stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, delegatorStake, "Delegator stake should be equal to stake amount after addition")

	// Check updated bond stake for delegator and target
	bondStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, bondStake, "Bond stake should be equal to stake amount after addition")

	// Check updated stake placed upon target
	targetStake, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialTargetStake.Add(stakeAmount), targetStake, "Target stake should be incremented by stake amount after addition")

	// Check updated topic stake
	topicStake, err := keeper.GetTopicStake(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(initialTopicStake.Add(stakeAmount), topicStake, "Topic stake should be incremented by stake amount after addition")

	// Check updated total stake
	totalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Add(stakeAmount), totalStake, "Total stake should be incremented by stake amount after addition")
}

func (s *KeeperTestSuite) TestAddStakeExistingDelegatorAndTarget() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(500)
	additionalStakeAmount := cosmosMath.NewUint(300)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
	s.Require().NoError(err)

	// Add additional stake
	err = keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
}

func (s *KeeperTestSuite) TestAddStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	zeroStakeAmount := cosmosMath.NewUint(0)

	// Try to add zero stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), zeroStakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveStakeFromBond() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes after adding stake
	initialTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)

	// Remove stake
	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator after removal
	_, err = keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check updated bond stake for delegator and target after removal
	_, err = keeper.GetBond(ctx, delegatorAddr, targetAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check updated stake placed upon target after removal
	_, err = keeper.GetStakePlacedUponTarget(ctx, targetAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check updated topic stake after removal
	_, err = keeper.GetTopicStake(ctx, topicID)
	s.Require().Error(err, collections.ErrNotFound)

	// Check updated total stake after removal
	finalTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Sub(stakeAmount), finalTotalStake, "Total stake should be decremented by stake amount after removal")
}

func (s *KeeperTestSuite) TestRemoveStakePartialFromDelegatorAndTarget() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(1000)
	removeStakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining bond stake for delegator and target
	remainingBondStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingBondStake, "Remaining bond stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegatorAndTarget() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
	s.Require().NoError(err)

	// Remove entire stake
	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator should be zero
	_, err = keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().Error(collections.ErrNotFound)

	// Check remaining bond stake for delegator and target should be zero
	_, err = keeper.GetBond(ctx, delegatorAddr, targetAddr)
	s.Require().Error(collections.ErrNotFound)
}

func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	initialStakeAmount := cosmosMath.NewUint(500)
	zeroStakeAmount := cosmosMath.NewUint(0)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
	s.Require().NoError(err)

	// Try to remove zero stake
	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, zeroStakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	nonExistingDelegatorAddr := sdk.AccAddress(PKS[0].Address())
	nonExistingTargetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Try to remove stake with non-existing delegator or target
	err := keeper.RemoveStakeFromBond(ctx, topicID, nonExistingDelegatorAddr, nonExistingTargetAddr, stakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGetAllBondsForDelegator() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	delegatorAddr := sdk.AccAddress(PKS[2].Address())

	// Mock setup
	topicID := uint64(1)
	targetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Add stake to create bonds
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
	s.Require().NoError(err)

	// Get all bonds for delegator
	targets, amounts, err := keeper.GetAllBondsForDelegator(ctx, delegatorAddr)

	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
	s.Require().NotEmpty(targets, "Targets should not be empty")
	s.Require().NotEmpty(amounts, "Amounts should not be empty")
	s.Require().Equal(len(targets), len(amounts), "The lengths of targets and amounts should match")
}

func (s *KeeperTestSuite) TestWalkAllTopicStake() {
	ctx := s.ctx
	keeper := s.upshotKeeper

	// Mock setup for multiple topics and stakes
	for i := 1; i <= 3; i++ {
		topicID := uint64(i)
		stakeAmount := cosmosMath.NewUint(uint64(100 * i))
		keeper.SetTopicStake(ctx, topicID, stakeAmount)
	}

	// Define a walk function to collect stakes
	var collectedStakes []cosmosMath.Uint
	walkFunc := func(topicID uint64, stake cosmosMath.Uint) (stop bool, err error) {
		collectedStakes = append(collectedStakes, stake)
		return false, nil
	}

	// Walk all topic stakes
	err := keeper.WalkAllTopicStake(ctx, walkFunc)

	s.Require().NoError(err, "Walking all topic stakes should not return an error")
	s.Require().Equal(3, len(collectedStakes), "The number of collected stakes should match the number of topics")
}

func (s *KeeperTestSuite) TestRemoveStakeFromBondMissingTotalOrTopicStake() {
	ctx := s.ctx
	keeper := s.upshotKeeper
	topicID := uint64(1)
	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	targetAddr := sdk.AccAddress(PKS[1].Address())
	stakeAmount := cosmosMath.NewUint(500)

	// Setup initial stake
	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes
	initialTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	initialTopicStake, err := keeper.GetTopicStake(ctx, topicID)
	s.Require().NoError(err)

	// Remove stake without updating total or topic stake
	err = keeper.RemoveStakeFromBondMissingTotalOrTopicStake(ctx, topicID, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Check stakeOwnedByDelegator after removal
	_, err = keeper.GetDelegatorStake(ctx, delegatorAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check stakePlacement after removal
	_, err = keeper.GetBond(ctx, delegatorAddr, targetAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check stakePlacedUponTarget after removal
	_, err = keeper.GetStakePlacedUponTarget(ctx, targetAddr)
	s.Require().Error(err, collections.ErrNotFound)

	// Check totalStake did not change
	finalTotalStake, err := keeper.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake, finalTotalStake, "Total stake should not change")

	// Check topicStake did not change
	finalTopicStake, err := keeper.GetTopicStake(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(initialTopicStake, finalTopicStake, "Topic stake should not change")
}

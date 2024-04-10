package keeper_test

import (
	"fmt"
	"testing"
	"time"

	// "cosmossdk.io/collections"
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

// func (s *KeeperTestSuite) TestAddStake() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	stakeAmount := cosmosMath.NewUint(500)

// 	// Initial Values
// 	initialTotalStake := cosmosMath.NewUint(0)
// 	initialTopicStake := cosmosMath.NewUint(0)
// 	initialTargetStake := cosmosMath.NewUint(0)

// 	// Add stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
// 	s.Require().NoError(err)

// 	// Check updated stake for delegator
// 	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(stakeAmount, delegatorStake, "Delegator stake should be equal to stake amount after addition")

// 	// Check updated bond stake for delegator and target
// 	bondStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(stakeAmount, bondStake, "Bond stake should be equal to stake amount after addition")

// 	// Check updated stake placed upon target
// 	targetStake, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTargetStake.Add(stakeAmount), targetStake, "Target stake should be incremented by stake amount after addition")

// 	// Check updated topic stake
// 	topicStake, err := keeper.GetTopicStake(ctx, topicID[0])
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTopicStake.Add(stakeAmount), topicStake, "Topic stake should be incremented by stake amount after addition")

// 	// Check updated total stake
// 	totalStake, err := keeper.GetTotalStake(ctx)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTotalStake.Add(stakeAmount), totalStake, "Total stake should be incremented by stake amount after addition")
// }

// func (s *KeeperTestSuite) TestAddStakeExistingDelegatorAndTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)
// 	additionalStakeAmount := cosmosMath.NewUint(300)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Add additional stake
// 	err = keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), additionalStakeAmount)
// 	s.Require().NoError(err)

// 	// Check updated stake for delegator
// 	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
// }

// func (s *KeeperTestSuite) TestAddStakeZeroAmount() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	zeroStakeAmount := cosmosMath.NewUint(0)

// 	// Try to add zero stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), zeroStakeAmount)
// 	s.Require().Error(err)
// }

// func (s *KeeperTestSuite) TestRemoveStakeFromBond() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	stakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
// 	s.Require().NoError(err)

// 	// Capture the initial total and topic stakes after adding stake
// 	initialTotalStake, err := keeper.GetTotalStake(ctx)
// 	s.Require().NoError(err)

// 	// Remove stake
// 	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, stakeAmount)
// 	s.Require().NoError(err)

// 	// Check updated stake for delegator after removal
// 	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), delegatorStake, "Delegator stake should be zero after removal")

// 	// Check updated bond stake for delegator and target after removal
// 	bond, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), bond, "Bond stake should be zero after removal")

// 	// Check updated stake placed upon target after removal
// 	stakePlacedUponTarget, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), stakePlacedUponTarget, "Stake placed upon target should be zero after removal")

// 	// Check updated topic stake after removal
// 	topicStake, err := keeper.GetTopicStake(ctx, topicID[0])
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), topicStake, "Topic stake should be zero after removal")

// 	// Check updated total stake after removal
// 	finalTotalStake, err := keeper.GetTotalStake(ctx)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTotalStake.Sub(stakeAmount), finalTotalStake, "Total stake should be decremented by stake amount after removal")
// }

// func (s *KeeperTestSuite) TestRemoveStakePartialFromDelegatorAndTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(1000)
// 	removeStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Remove a portion of stake
// 	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, removeStakeAmount)
// 	s.Require().NoError(err)

// 	// Check remaining stake for delegator
// 	remainingStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

// 	// Check remaining bond stake for delegator and target
// 	remainingBondStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingBondStake, "Remaining bond stake should be initial minus removed amount")
// }

// func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegatorAndTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Remove entire stake
// 	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Check remaining stake for delegator should be zero
// 	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), delegatorStake, "Delegator stake should be zero after removal")

// 	// Check remaining bond stake for delegator and target should be zero
// 	bond, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), bond, "Bond stake should be zero after removal")
// }

// func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)
// 	zeroStakeAmount := cosmosMath.NewUint(0)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Try to remove zero stake
// 	err = keeper.RemoveStakeFromBond(ctx, topicID, delegatorAddr, targetAddr, zeroStakeAmount)
// 	s.Require().Error(err)
// }

// func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	nonExistingDelegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	nonExistingTargetAddr := sdk.AccAddress(PKS[1].Address())
// 	stakeAmount := cosmosMath.NewUint(500)

// 	// Try to remove stake with non-existing delegator or target
// 	err := keeper.RemoveStakeFromBond(ctx, topicID, nonExistingDelegatorAddr, nonExistingTargetAddr, stakeAmount)
// 	s.Require().Error(err)
// }

// func (s *KeeperTestSuite) TestGetAllBondsForDelegator() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	delegatorAddr := sdk.AccAddress(PKS[2].Address())

// 	// Mock setup
// 	topicID := []uint64{1}
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	stakeAmount := cosmosMath.NewUint(500)

// 	// Add stake to create bonds
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
// 	s.Require().NoError(err)

// 	// Get all bonds for delegator
// 	targets, amounts, err := keeper.GetAllBondsForDelegator(ctx, delegatorAddr)

// 	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
// 	s.Require().NotEmpty(targets, "Targets should not be empty")
// 	s.Require().NotEmpty(amounts, "Amounts should not be empty")
// 	s.Require().Equal(len(targets), len(amounts), "The lengths of targets and amounts should match")
// }

// func (s *KeeperTestSuite) TestWalkAllTopicStake() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper

// 	//rather than calling keeper.InitGenesis, we just increment the topic id for 0 manually
// 	topic0, err := keeper.IncrementTopicId(ctx)
// 	s.Require().NoError(err)
// 	s.Require().Equal(uint64(0), topic0)
// 	// Mock setup for multiple topics and stakes
// 	for i := 1; i <= 3; i++ {
// 		topicID := uint64(i)
// 		stakeAmount := cosmosMath.NewUint(uint64(100 * i))
// 		keeper.SetTopicStake(ctx, topicID, stakeAmount)
// 		keeper.IncrementTopicId(ctx)
// 	}

// 	// Define a walk function to collect stakes
// 	var collectedStakes []cosmosMath.Uint
// 	walkFunc := func(topicID uint64, stake cosmosMath.Uint) (stop bool, err error) {
// 		collectedStakes = append(collectedStakes, stake)
// 		return false, nil
// 	}

// 	// Walk all topic stakes
// 	err = keeper.WalkAllTopicStake(ctx, walkFunc)

// 	s.Require().NoError(err, "Walking all topic stakes should not return an error")
// 	s.Require().Equal(3, len(collectedStakes), "The number of collected stakes should match the number of topics")
// }

// func (s *KeeperTestSuite) TestRemoveStakeFromBondMissingTotalOrTopicStake() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	stakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), stakeAmount)
// 	s.Require().NoError(err)

// 	// Capture the initial total and topic stakes
// 	initialTotalStake, err := keeper.GetTotalStake(ctx)
// 	s.Require().NoError(err)
// 	initialTopicStake, err := keeper.GetTopicStake(ctx, topicID[0])
// 	s.Require().NoError(err)

// 	// Remove stake without updating total or topic stake
// 	err = keeper.RemoveStakeFromBondMissingTotalOrTopicStake(ctx, delegatorAddr, targetAddr, stakeAmount)
// 	s.Require().NoError(err)

// 	// Check stakeOwnedByDelegator after removal
// 	delegatorStake, err := keeper.GetDelegatorStake(ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.ZeroUint(), delegatorStake, "Delegator stake should be zero after removal")

// 	// Check stakePlacement after removal
// 	bond, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err, "Stake placement should be removed")
// 	s.Require().Equal(cosmosMath.ZeroUint(), bond, "Stake placement should be removed")

// 	targetStake, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err, "Stake placed upon target should be removed")
// 	s.Require().Equal(cosmosMath.ZeroUint(), targetStake, "Stake placed upon target should be removed")

// 	// Check totalStake did not change
// 	finalTotalStake, err := keeper.GetTotalStake(ctx)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTotalStake, finalTotalStake, "Total stake should not change")

// 	// Check topicStake did not change
// 	finalTopicStake, err := keeper.GetTopicStake(ctx, topicID[0])
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialTopicStake, finalTopicStake, "Topic stake should not change")
// }

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

// func (s *KeeperTestSuite) TestSubStakePlacement() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Sub stake
// 	subAmount := cosmosMath.NewUint(400)
// 	err = keeper.SubStakePlacement(ctx, delegatorAddr, targetAddr, subAmount)
// 	s.Require().NoError(err)

// 	// Check remaining stake for delegator
// 	remainingStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Sub(subAmount), remainingStake, "Remaining bond stake should be initial minus sub amount")
// }

// func (s *KeeperTestSuite) TestSubStakePlacementErr() {
// 	ctx := s.ctx
// 	k := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := k.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Sub stake
// 	subAmount := cosmosMath.NewUint(600)
// 	err = k.SubStakePlacement(ctx, delegatorAddr, targetAddr, subAmount)
// 	s.Require().ErrorIs(err, state.ErrIntegerUnderflowBonds)

// 	// Check remaining stake for delegator
// 	remainingStake, err := k.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount, remainingStake, "Remaining bond stake should be same after error")
// }

// func (s *KeeperTestSuite) TestAddStakePlacement() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Add stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	additionalStakeAmount := cosmosMath.NewUint(300)

// 	// Add additional stake
// 	err = keeper.AddStakePlacement(ctx, delegatorAddr, targetAddr, additionalStakeAmount)
// 	s.Require().NoError(err)

// 	// Check updated stake for delegator
// 	finalStake, err := keeper.GetBond(ctx, delegatorAddr, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), finalStake, "Final stake should be added to initial stake amount after addition")
// }

// func (s *KeeperTestSuite) TestSubStakePlacedUponTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := keeper.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Sub stake
// 	subAmount := cosmosMath.NewUint(400)
// 	err = keeper.SubStakePlacedUponTarget(ctx, targetAddr, subAmount)
// 	s.Require().NoError(err)

// 	// Check remaining stake for delegator
// 	remainingStake, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Sub(subAmount), remainingStake, "Remaining bond stake should be initial minus sub amount")
// }

// func (s *KeeperTestSuite) TestSubStakePlacedUponTargetErr() {
// 	ctx := s.ctx
// 	k := s.emissionsKeeper
// 	topicID := []uint64{1}
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Setup initial stake
// 	err := k.AddStake(ctx, topicID, delegatorAddr.String(), targetAddr.String(), initialStakeAmount)
// 	s.Require().NoError(err)

// 	// Sub stake
// 	subAmount := cosmosMath.NewUint(600)
// 	err = k.SubStakePlacedUponTarget(ctx, targetAddr, subAmount)
// 	s.Require().ErrorIs(err, state.ErrIntegerUnderflowTarget)

// 	// Check remaining stake for delegator
// 	remainingStake, err := k.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount, remainingStake, "Remaining bond stake should be the same after error")
// }

// func (s *KeeperTestSuite) TestAddStakePlacedUponTarget() {
// 	ctx := s.ctx
// 	keeper := s.emissionsKeeper
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	initialStakeAmount := cosmosMath.NewUint(500)

// 	// Add stake
// 	err := keeper.AddStakePlacedUponTarget(ctx, targetAddr, initialStakeAmount)
// 	s.Require().NoError(err)

// 	additionalStakeAmount := cosmosMath.NewUint(300)

// 	// Add additional stake
// 	err = keeper.AddStakePlacedUponTarget(ctx, targetAddr, additionalStakeAmount)
// 	s.Require().NoError(err)

// 	// Check updated stake for target
// 	finalStake, err := keeper.GetStakePlacedUponTarget(ctx, targetAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), finalStake, "Final stake should be added to initial stake amount after addition")
// }

// func (s *KeeperTestSuite) TestSetStakeRemovalQueueForAddress() {
// 	delegatorAddr := sdk.AccAddress(PKS[0].Address())
// 	targetAddr := sdk.AccAddress(PKS[1].Address())
// 	placement := state.StakeRemovalPlacement{
// 		TopicIds: []uint64{1},
// 		Target:   targetAddr.String(),
// 		Amount:   cosmosMath.NewUint(500),
// 	}
// 	placements := []*state.StakeRemovalPlacement{&placement}
// 	removalInfo := state.StakeRemoval{
// 		TimestampRemovalStarted: uint64(time.Now().Unix()),
// 		Placements:              placements,
// 	}

// 	_, err := s.emissionsKeeper.GetStakeRemovalQueueByAddress(s.ctx, delegatorAddr)
// 	s.Require().ErrorIs(err, collections.ErrNotFound)

// 	// Set stake removal queue
// 	err = s.emissionsKeeper.SetStakeRemovalQueueForAddress(s.ctx, delegatorAddr, removalInfo)
// 	s.Require().NoError(err)

// 	// Check stake removal queue
// 	stakeRemovalQueue, err := s.emissionsKeeper.GetStakeRemovalQueueByAddress(s.ctx, delegatorAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(removalInfo, stakeRemovalQueue, "Stake removal queue should be equal to the set removal info")
// }

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
		Version:                       "v1.0.0",
		RewardCadence:                 60 * 60 * 24 * 7 * 24,
		MinTopicUnmetDemand:           cosmosMath.NewUint(100),
		MaxTopicsPerBlock:             1000,
		MinRequestUnmetDemand:         cosmosMath.NewUint(1),
		MaxMissingInferencePercent:    alloraMath.NewDecFromInt64(10),
		RequiredMinimumStake:          cosmosMath.NewUint(1),
		RemoveStakeDelayWindow:        172800,
		MinEpochLength:                60,
		MaxInferenceRequestValidity:   60 * 60 * 24 * 7 * 24,
		MaxRequestCadence:             60 * 60 * 24 * 7 * 24,
		PercentRewardsReputersWorkers: alloraMath.NewDecFromInt64(50),
		MaxWorkersPerTopicRequest:     10,
		MaxReputersPerTopicRequest:    10,
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
	s.Require().Equal(params.PercentRewardsReputersWorkers, paramsFromKeeper.PercentRewardsReputersWorkers, "Params should be equal to the set params: PercentRewardsReputersWorkers")
	s.Require().Equal(params.MaxWorkersPerTopicRequest, paramsFromKeeper.MaxWorkersPerTopicRequest, "Params should be equal to the set params: MaxWorkersPerTopicRequest")
	s.Require().Equal(params.MaxReputersPerTopicRequest, paramsFromKeeper.MaxReputersPerTopicRequest, "Params should be equal to the set params: MaxReputersPerTopicRequest")
}

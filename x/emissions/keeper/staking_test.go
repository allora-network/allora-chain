package keeper_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetSetTotalStake() {
	ctx := s.Ctx()
	k := s.StakingKeeper()

	// Set total stake
	newTotalStake := cosmosMath.NewInt(1000)
	err := k.SetTotalStake(ctx, newTotalStake)
	s.Require().NoError(err)

	// Check total stake
	totalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(newTotalStake, totalStake)
}

func (s *KeeperTestSuite) TestAddReputerStake() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)

	// Initial Values
	initialTotalStake := cosmosMath.NewInt(0)
	initialTopicStake := cosmosMath.NewInt(0)

	// Add stake
	err := k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := k.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, delegatorStake, "Delegator stake should be equal to stake amount after addition")

	// Check updated topic stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(initialTopicStake.Add(stakeAmount), topicStake, "Topic stake should be incremented by stake amount after addition")

	// Check updated total stake
	totalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Add(stakeAmount), totalStake, "Total stake should be incremented by stake amount after addition")
}

func (s *KeeperTestSuite) TestAddDelegateStake() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(500)
	additionalStakeAmount := cosmosMath.NewInt(300)

	// Setup initial stake
	err := k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount, delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")

	// Add additional stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err = k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
}

func (s *KeeperTestSuite) TestAddReputerStakeZeroAmount() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Try to add zero stake
	err := k.AddReputerStake(ctx, topicId, delegatorAddr, zeroStakeAmount)
	s.Require().ErrorIs(err, types.ErrInvalidValue)
}

func (s *KeeperTestSuite) TestRemoveStake() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes after adding stake
	initialTotalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator after removal
	delegatorStake, err := k.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), delegatorStake, "Delegator stake should be zero after removal")

	// Check updated topic stake after removal
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")

	// Check updated total stake after removal
	finalTotalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().True(initialTotalStake.Sub(stakeAmount).Equal(finalTotalStake), "Total stake should be decremented by stake amount after removal")
}

func (s *KeeperTestSuite) TestRemovePartialStakeFromDelegator() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(1000)
	removeStakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                removeStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegator() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(1000)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                initialStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for Reputer
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	initialStakeAmount := cosmosMath.NewInt(500)
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Setup initial stake
	err := k.AddReputerStake(ctx, topicId, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Try to remove zero stake
	err = k.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, reputerAddr, zeroStakeAmount)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	nonExistingDelegatorAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)

	// Try to remove stake with non-existing delegator or target
	err := k.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, nonExistingDelegatorAddr, stakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGetAllStakeForDelegator() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	delegatorAddr := s.AddrsStr(0)

	// Mock setup
	topicId := uint64(1)
	targetAddr := s.AddrsStr(1)
	stakeAmount := cosmosMath.NewInt(500)

	// Add stake to create bonds
	err := k.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Add stake to create bonds
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount.Mul(cosmosMath.NewInt(2)))
	s.Require().NoError(err)

	// Get all bonds for delegator
	amount, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)

	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
	s.Require().Equal(stakeAmount.Mul(cosmosMath.NewInt(3)), amount, "The total amount is incorrect")
}

func (s *KeeperTestSuite) TestSetGetDeleteStakeRemovalByAddressWithDetailedPlacement() {
	ctx := s.Ctx()
	k := s.StakingKeeper()

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
	err := k.SetStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = k.SetStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 101

	// Retrieve the stake removal information
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 1)
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
	retrievedInfo, limitHit, err = k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 2)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit, "The limit should not be hit")
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.BlockRemovalCompleted, retrievedInfo[1].BlockRemovalCompleted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = k.DeleteStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")

	// delete 102
	err = k.DeleteStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err = k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockNotFound() {
	ctx := s.Ctx()
	k := s.StakingKeeper()

	// Attempt to retrieve stake removal info for an address with no set info
	removals, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, 202, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitPreviousBlocks() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.AddrsStr(2)
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := k.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+1, // note how we are getting a block AFTER blockRemovalsEnd
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitExactBlock() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.AddrsStr(2)
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := k.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd,
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitGreaterThanNumRemovals() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	numRemovals := int64(5)
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.AddrsStr(2)
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := k.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(numRemovals),
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, int(numRemovals))
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitLessThanNumRemovals() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	numRemovals := int64(5)
	limitRemovals := numRemovals - 2
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.AddrsStr(2)
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := k.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(limitRemovals),
	)
	s.Require().NoError(err)
	s.Require().True(limitHit)
	s.Require().Len(retrievedInfo, int(limitRemovals))
}

func (s *KeeperTestSuite) TestSetGetDeleteDelegateStakeRemovalByAddress() {
	ctx := s.Ctx()
	k := s.StakingKeeper()

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
	err := k.SetDelegateStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = k.SetDelegateStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 201

	// Retrieve the delegate stake removal information
	retrievedInfo, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
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
	retrievedInfo, limitHit, err = k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2)
	s.Require().False(limitHit)
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Delegator, retrievedInfo[1].Delegator, "Delegator addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = k.DeleteDelegateStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer, removalInfo0.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)

	// delete 102
	err = k.DeleteDelegateStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer, removalInfo1.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err = k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)
}

func (s *KeeperTestSuite) TestGetDeleteDelegateStake() {
	ctx := s.Ctx()
	k := s.StakingKeeper()

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
	err := k.SetDelegateStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	_, err = k.GetDelegateStakeRemoval(ctx,
		removalInfo.BlockRemovalStarted,
		removalInfo.TopicId,
		removalInfo.Delegator,
		removalInfo.Reputer,
	)
	// index is on BlockRemovalCompleted not BlockRemovalStarted
	s.Require().Error(err)

	retrievedInfo, err := k.GetDelegateStakeRemoval(ctx,
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
	ctx := s.Ctx()
	k := s.StakingKeeper()

	// Attempt to retrieve delegate stake removal info for an address with no set info
	removals, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, 201, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

// TestActiveTopicStakeRemoval tests that when a stake is removed from an active topic,
// it correctly updates the stake
func (s *KeeperTestSuite) TestActiveTopicStakeRemoval() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic and activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Verify the topic is active
	isActive, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive, "Topic should be active")

	// Add some stake to the topic
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for stake removal
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestDelegateStakeRemoval tests that when a delegate stake is removed,
// it correctly updates the stake
func (s *KeeperTestSuite) TestDelegateStakeRemoval() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(1)
	delegatorAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic and activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Verify the topic is active
	isActive, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive, "Topic should be active")

	// Add some delegate stake to the topic
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for delegate stake removal
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove delegate stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestInactiveTopicStakeRemoval tests that when a stake is removed from an inactive topic,
// it still correctly updates the stake but does not affect the total weight calculations
func (s *KeeperTestSuite) TestInactiveTopicStakeRemoval() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic but do NOT activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)

	// Verify the topic is not active
	isActive, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(isActive, "Topic should not be active")

	// Add some stake to the topic
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for stake removal
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake first (this should work even for inactive topics)
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestTopicWeightRecalculationAfterStakeRemoval tests that when a stake is removed,
// the topic weight is recalculated correctly based on the remaining stake
func (s *KeeperTestSuite) TestTopicWeightRecalculationAfterStakeRemoval() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	topicId := uint64(2)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(1000)
	feeRevenue := cosmosMath.NewInt(100)
	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create and activate topic
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Add stake and fee revenue
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)
	err = s.TopicKeeper().AddTopicFeeRevenue(ctx, topicId, feeRevenue)
	s.Require().NoError(err)

	// Get initial weight and set it
	initialWeight, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
		ctx,
		topicId,
		epochLength,
		moduleParams.TopicRewardAlpha,
		moduleParams.TopicRewardStakeImportance,
		moduleParams.TopicRewardFeeRevenueImportance,
		moduleParams.BlocksPerMonth,
	)
	s.Require().NoError(err)
	err = s.TopicKeeper().SetPreviousTopicWeight(ctx, topicId, initialWeight)
	s.Require().NoError(err)

	// Remove half the stake
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount.QuoRaw(2),
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount.QuoRaw(2))
	s.Require().NoError(err)

	// Verify weight and stake changes
	newWeight, noPrior, err := s.TopicKeeper().GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(noPrior, "Should still have a prior weight")
	s.Require().True(newWeight.Lt(initialWeight), "New weight should be less than initial weight")

	remainingStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount.QuoRaw(2), remainingStake, "Remaining stake should be half of initial stake")
}

// TestTopicWeightRecalculationWithMultipleTopics tests that when stakes are removed,
// the weights of multiple topics are recalculated correctly and the total sum is updated
func (s *KeeperTestSuite) TestTopicWeightRecalculationWithMultipleTopics() {
	ctx := s.Ctx()
	k := s.StakingKeeper()
	reputerAddr := s.AddrsStr(0)

	// Set up test parameters
	stakeAmount1 := cosmosMath.NewInt(1000)
	stakeAmount2 := cosmosMath.NewInt(2000)
	epochLength := int64(100)
	feeRevenue1 := cosmosMath.NewInt(100)
	feeRevenue2 := cosmosMath.NewInt(200)

	// Set up params
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = uint64(5)
	params.MinTopicWeight = alloraMath.MustNewDecFromString("1")
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// Create and activate both topics
	topicId1, topicId2 :=
		s.CreateTopic(testutil.WithEpochLength(epochLength)),
		s.CreateTopic(testutil.WithEpochLength(epochLength))
	err = s.TopicKeeper().ActivateTopic(ctx, topicId1)
	s.Require().NoError(err)
	err = s.TopicKeeper().ActivateTopic(ctx, topicId2)
	s.Require().NoError(err)

	// Add stake and fee revenue to both topics
	err = k.AddReputerStake(ctx, topicId1, reputerAddr, stakeAmount1)
	s.Require().NoError(err)
	err = k.AddReputerStake(ctx, topicId2, reputerAddr, stakeAmount2)
	s.Require().NoError(err)
	err = s.TopicKeeper().AddTopicFeeRevenue(ctx, topicId1, feeRevenue1)
	s.Require().NoError(err)
	err = s.TopicKeeper().AddTopicFeeRevenue(ctx, topicId2, feeRevenue2)
	s.Require().NoError(err)

	// Calculate and set initial weights for both topics
	for id := range map[uint64]struct{}{topicId1: {}, topicId2: {}} {
		weight, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
			ctx,
			id,
			epochLength, // epochLength
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			params.BlocksPerMonth,
		)
		s.Require().NoError(err)
		s.Require().True(weight.Gt(params.MinTopicWeight), "Initial weight should be greater than minimum weight")
		err = s.TopicKeeper().SetPreviousTopicWeight(ctx, id, weight)
		s.Require().NoError(err)
	}

	// Get total sum before stake removal
	totalSumBefore, err := s.TopicKeeper().GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)

	// Remove stake from first topic
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + params.RemoveStakeDelayWindow
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId1,
		Reputer:               reputerAddr,
		Amount:                stakeAmount1,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)
	err = k.RemoveReputerStake(ctx, endBlock, topicId1, reputerAddr, stakeAmount1)
	s.Require().NoError(err)

	// Verify total sum was updated correctly
	totalSumAfter, err := s.TopicKeeper().GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(totalSumAfter.Lt(totalSumBefore), "Total sum should decrease after stake removal")
}

func (s *KeeperTestSuite) TestGetFirstStakeRemovalForReputerAndTopicId() {
	k := s.StakingKeeper()
	ctx := s.Ctx()
	reputer := s.AddrsStr(2)
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
	k := s.StakingKeeper()
	ctx := s.Ctx()
	reputer := s.AddrsStr(2)
	topicId := uint64(1)

	_, found, err := k.GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetFirstDelegateStakeRemovalForDelegatorReputerAndTopicId() {
	k := s.StakingKeeper()
	ctx := s.Ctx()
	delegator := s.AddrsStr(5)
	reputer := s.AddrsStr(2)
	reputer2 := s.AddrsStr(3)
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
	k := s.StakingKeeper()
	ctx := s.Ctx()
	delegator := "delegator"
	reputer := s.AddrsStr(2)
	topicId := uint64(1)

	_, found, err := k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

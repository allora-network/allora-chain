package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// TestInactivateTopicWithoutMinWeightReset_TopicNotFoundInBlockRemovesAllTopics
// This test demonstrates the bug where if a topic is not found in the block's active topics list,
// all topics at that block are incorrectly removed instead of performing a no-op.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_TopicNotFoundInBlockRemovesAllTopics() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	block := int64(100)
	topicId1 := uint64(1)
	topicId2 := uint64(2)
	topicId3 := uint64(3)
	nonExistentInBlockTopicId := uint64(999)

	// Set up 3 active topics at the block
	activeTopics := types.TopicIds{
		TopicIds: []uint64{topicId1, topicId2, topicId3},
	}
	err := k.SetBlockToActiveTopics(ctx, block, activeTopics)
	s.Require().NoError(err, "Setting active topics should not fail")

	// Verify initial state - should have 3 topics
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3, "Should have 3 active topics initially")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1)
	s.Require().Contains(retrievedTopics.TopicIds, topicId2)
	s.Require().Contains(retrievedTopics.TopicIds, topicId3)

	// Create a topic that exists and is marked as active in the system
	// but is NOT in the block's active topics list
	// This simulates a data inconsistency scenario
	topic := s.MockTopic()
	topic.Id = nonExistentInBlockTopicId
	err = k.SetTopic(ctx, nonExistentInBlockTopicId, topic)
	s.Require().NoError(err, "Setting topic should not fail")

	// Make it "active" in the system (so GetNextPossibleChurningBlockByTopicId returns true)
	// This makes the function proceed past the early return check
	err = k.SetTopicToNextPossibleChurningBlock(ctx, nonExistentInBlockTopicId, block)
	s.Require().NoError(err, "Setting topic to next possible churning block should not fail")
	err = k.SetActiveTopics(ctx, nonExistentInBlockTopicId)
	s.Require().NoError(err, "Setting active topics should not fail")

	// Set a topic weight so RemoveTopicFromPreviousTopicWeights doesn't fail
	err = k.SetPreviousTopicWeight(ctx, nonExistentInBlockTopicId, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Verify the topic is marked as active in the system
	_, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, nonExistentInBlockTopicId)
	s.Require().NoError(err)
	s.Require().True(topicIsActive, "Topic should be marked as active in the system")

	// Verify the topic is NOT in the block's active topics list
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().NotContains(retrievedTopics.TopicIds, nonExistentInBlockTopicId,
		"Topic should NOT be in the block's active topics list")

	// Call InactivateTopic - this internally calls inactivateTopicWithoutMinWeightReset
	// The bug: when the topic is not found in the block's active topics list,
	// newActiveTopicIds remains empty and all topics are removed
	// Expected behavior: topics should remain unchanged (no-op) when topic not found
	err = k.InactivateTopic(ctx, nonExistentInBlockTopicId)
	s.Require().NoError(err, "InactivateTopic should not error even when topic not found in block")

	// EXPECTED BEHAVIOR: All topics should remain unchanged when topic is not found in block
	// This test will FAIL when the bug exists (all topics removed) and PASS when bug is fixed
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3,
		"All 3 topics should remain unchanged when topic not found in block (no-op behavior)")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1, "topicId1 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId2, "topicId2 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId3, "topicId3 should remain")
}

// TestInactivateTopicWithoutMinWeightReset_TopicFoundInBlockRemovesOnlyThatTopic
// This test verifies the correct behavior when the topic IS found in the block's active topics list.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_TopicFoundInBlockRemovesOnlyThatTopic() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	block := int64(200)
	topicId1 := uint64(10)
	topicId2 := uint64(20)
	topicId3 := uint64(30)

	// Set up 3 active topics at the block
	activeTopics := types.TopicIds{
		TopicIds: []uint64{topicId1, topicId2, topicId3},
	}
	err := k.SetBlockToActiveTopics(ctx, block, activeTopics)
	s.Require().NoError(err)

	// Create and set up topicId2 to be active
	topic := s.MockTopic()
	topic.Id = topicId2
	err = k.SetTopic(ctx, topicId2, topic)
	s.Require().NoError(err)

	// Make it active in the system
	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId2, block)
	s.Require().NoError(err)
	err = k.SetActiveTopics(ctx, topicId2)
	s.Require().NoError(err)

	// Set a topic weight
	err = k.SetPreviousTopicWeight(ctx, topicId2, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err)

	// Verify initial state
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3)
	s.Require().Contains(retrievedTopics.TopicIds, topicId2)

	// Inactivate topicId2 - it IS in the block's active topics list
	err = k.InactivateTopic(ctx, topicId2)
	s.Require().NoError(err)

	// Verify only topicId2 was removed, others remain
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 2, "Should have 2 topics remaining")
	s.Require().NotContains(retrievedTopics.TopicIds, topicId2, "topicId2 should be removed")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1, "topicId1 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId3, "topicId3 should remain")
}

// TestInactivateTopicWithoutMinWeightReset_EmptyBlockListNoOp
// This test verifies behavior when the block has no active topics.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_EmptyBlockListNoOp() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	block := int64(300)
	topicId := uint64(100)

	// Create a topic that exists and is marked as active
	topic := s.MockTopic()
	topic.Id = topicId
	err := k.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	// Make it active in the system
	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId, block)
	s.Require().NoError(err)
	err = k.SetActiveTopics(ctx, topicId)
	s.Require().NoError(err)

	// Set a topic weight
	err = k.SetPreviousTopicWeight(ctx, topicId, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err)

	// Verify block has no active topics (empty list)
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Empty(retrievedTopics.TopicIds, "Block should have no active topics")

	// Call InactivateTopic - topic is not in the (empty) block's active topics list
	err = k.InactivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Verify block still has no active topics (should remain empty, not cause issues)
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Empty(retrievedTopics.TopicIds, "Block should still have no active topics")
}

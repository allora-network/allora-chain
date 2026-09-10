package keeper_test

import (
	"fmt"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

// TestTopicInvariantActiveTopicsScheduledAtChurningBlock checks that a normally activated
// topic satisfies the invariant and that a topic placed in the active set without a
// schedule is reported.
func (s *KeeperTestSuite) TestTopicInvariantActiveTopicsScheduledAtChurningBlock() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	invariant := keeper.TopicInvariantActiveTopicsScheduledAtChurningBlock(*s.EmissionsKeeper())

	msg, broken := invariant(ctx)
	s.Require().False(broken, msg)

	scheduledTopicId := s.CreateTopic()
	s.Require().NoError(k.ActivateTopic(ctx, scheduledTopicId))
	msg, broken = invariant(ctx)
	s.Require().False(broken, msg)

	// A topic in the set without a churning block or block listing must be reported.
	unscheduledTopicId := s.CreateTopic()
	s.Require().NoError(k.SetActiveTopics(ctx, unscheduledTopicId))
	msg, broken = invariant(ctx)
	s.Require().True(broken, "an active topic without a schedule must break the invariant")
	s.Require().Contains(msg, fmt.Sprintf("topic %d:", unscheduledTopicId))
}

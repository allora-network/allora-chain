package module_test

import (
	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module"
)

func (s *ModuleTestSuite) TestInactivateLowDemandTopicsRemoveTwoTopics() {
	_, err := mockCreateTopics(s, 2)
	s.Require().NoError(err, "mockCreateTopics should not throw an error")
	listTopics, err := module.InactivateLowDemandTopics(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err, "InactivateLowDemandTopics should not throw an error")
	s.Require().Len(listTopics, 0, "InactivateLowDemandTopics should return 0 topics")
	s.Require().Equal([]*state.Topic{}, listTopics, "InactivateLowDemandTopics should return an empty list of topics")
}

func (s *ModuleTestSuite) TestInactivateLowDemandTopicsRemoveOneTopicLeaveOne() {
	createdTopicIds, err := mockCreateTopics(s, 2)
	s.Require().NoError(err, "mockCreateTopics should not throw an error")
	err = s.emissionsKeeper.SetTopicUnmetDemand(s.ctx, createdTopicIds[0], cosmosMath.NewUint(keeper.MIN_TOPIC_DEMAND+1))
	s.Require().NoError(err, "SetTopicUnmetDemand should not throw an error")
	listTopics, err := module.InactivateLowDemandTopics(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err, "InactivateLowDemandTopics should not throw an error")
	s.Require().Len(listTopics, 1, "InactivateLowDemandTopics should return 0 topics")
	s.Require().Equal(createdTopicIds[0], (*listTopics[0]).Id, "InactivateLowDemandTopics should match expected")
}

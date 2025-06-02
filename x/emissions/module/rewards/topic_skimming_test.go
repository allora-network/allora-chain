package rewards_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
)

func (s *RewardsTestSuite) TestSkimTopTopicsByWeightDescSimple() {
	var unsortedTopicIds = []uint64{1, 2, 3, 4, 5}
	var weightsPerTopic = []int64{100, 300, 400, 400, 200}
	// Tiebreaker is topic id. Intended order of topics => 3 > 4 > 2 > 5 > 1
	var weights = map[uint64]*alloraMath.Dec{}
	for i, topicId := range unsortedTopicIds {
		weight := alloraMath.NewDecFromInt64(weightsPerTopic[i])
		weights[topicId] = &weight
	}
	N := uint64(3)
	mapOfTopN, listOfTopN, err := rewards.SkimTopTopicsByWeightDesc(s.Ctx(), weights, N)
	s.Require().NoError(err)

	// Check that mapOfTopN has the expected keys
	s.Require().Equal(N, uint64(len(mapOfTopN)), "SkimTopTopicsByWeightDesc should return a map with N keys")
	s.Require().Equal("400", mapOfTopN[3].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal("400", mapOfTopN[4].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal("300", mapOfTopN[2].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	// Check that mapOfTopN does not have any other keys
	_, ok := mapOfTopN[1]
	s.Require().Equal(false, ok, "SkimTopTopicsByWeightDesc should not have any other keys")
	_, ok = mapOfTopN[5]
	s.Require().Equal(false, ok, "SkimTopTopicsByWeightDesc should not have any other keys")

	// Check that listOfTopN has the expected values and size
	s.Require().Equal(N, uint64(len(listOfTopN)), "SkimTopTopicsByWeightDesc should return a list with N elements")
	s.Require().Equal(uint64(3), listOfTopN[0], "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal(uint64(4), listOfTopN[1], "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal(uint64(2), listOfTopN[2], "SkimTopTopicsByWeightDesc should return the expected sorted list")
}

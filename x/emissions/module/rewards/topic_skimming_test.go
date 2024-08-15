package rewards_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
)

func (s *RewardsTestSuite) TestSortTopicsByWeightDescWithRandomTiebreakerSimple() {
	var unsortedTopicIds []uint64 = []uint64{1, 2, 3, 4, 5}
	var weightsPerTopic []int64 = []int64{100, 300, 700, 400, 200}
	var weights map[uint64]*alloraMath.Dec = map[uint64]*alloraMath.Dec{}
	for i, topicId := range unsortedTopicIds {
		weight := alloraMath.NewDecFromInt64(weightsPerTopic[i])
		weights[topicId] = &weight
	}
	sortedList, err := rewards.SortTopicsByWeightDescWithRandomTiebreaker(unsortedTopicIds, weights, 0)
	s.Require().NoError(err)

	s.Require().Equal(len(unsortedTopicIds), len(sortedList), "SortTopicsByWeightDescWithRandomTiebreaker should return the same length list")
	s.Require().Equal(uint64(3), sortedList[0], "SortTopicsByWeightDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(4), sortedList[1], "SortTopicsByWeightDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(2), sortedList[2], "SortTopicsByWeightDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(5), sortedList[3], "SortTopicsByWeightDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(1), sortedList[4], "SortTopicsByWeightDescWithRandomTiebreaker should return the expected sorted list")
}

func (s *RewardsTestSuite) TestSkimTopTopicsByWeightDescSimple() {
	var unsortedTopicIds []uint64 = []uint64{1, 2, 3, 4, 5}
	var weightsPerTopic []int64 = []int64{100, 300, 700, 400, 200}
	var weights map[uint64]*alloraMath.Dec = map[uint64]*alloraMath.Dec{}
	for i, topicId := range unsortedTopicIds {
		weight := alloraMath.NewDecFromInt64(weightsPerTopic[i])
		weights[topicId] = &weight
	}
	N := uint64(3)
	mapOfTopN, listOfTopN, err := rewards.SkimTopTopicsByWeightDesc(s.ctx, weights, N, 0)
	s.Require().NoError(err)

	// Check that mapOfTopN has the expected keys
	s.Require().Equal(N, uint64(len(mapOfTopN)), "SkimTopTopicsByWeightDesc should return a map with N keys")
	s.Require().Equal("700", mapOfTopN[3].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
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

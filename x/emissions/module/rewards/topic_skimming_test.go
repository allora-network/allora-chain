package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
)

func (s *RewardsTestSuite) UtilSetParams() {
	s.emissionsKeeper.SetParams(s.ctx, types.Params{
		Version:                    "0.0.3",
		RewardCadence:              int64(5),
		MinTopicWeight:             alloraMath.NewDecFromInt64(100),
		MaxTopicsPerBlock:          uint64(1000),
		MaxMissingInferencePercent: alloraMath.MustNewDecFromString("0.1"),
		RequiredMinimumStake:       cosmosMath.NewUint(1),
		RemoveStakeDelayWindow:     int64(172800),
		MinEpochLength:             int64(60),
		MaxTopReputersToReward:     uint64(10),
		Sharpness:                  alloraMath.MustNewDecFromString("0.0"),
		BetaEntropy:                alloraMath.MustNewDecFromString("0.0"),
		LearningRate:               alloraMath.MustNewDecFromString("0.0"),
		MaxGradientThreshold:       alloraMath.MustNewDecFromString("0.0"),
		MinStakeFraction:           alloraMath.MustNewDecFromString("0.0"),
		Epsilon:                    alloraMath.MustNewDecFromString("0.1"),
		PInferenceSynthesis:        alloraMath.MustNewDecFromString("0.1"),
	})
}

func (s *RewardsTestSuite) TestSortTopicsByWeightDescWithRandomTiebreakerSimple() {
	var unsortedTopicIds []uint64 = []uint64{1, 2, 3, 4, 5}
	var weightsPerTopic []int64 = []int64{100, 300, 700, 400, 200}
	var weights map[uint64]*alloraMath.Dec = map[uint64]*alloraMath.Dec{}
	for i, topicId := range unsortedTopicIds {
		weight := alloraMath.NewDecFromInt64(weightsPerTopic[i])
		weights[topicId] = &weight
	}
	sortedList := rewards.SortTopicsByWeightDescWithRandomTiebreaker(unsortedTopicIds, weights, 0)

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
	mapOfTopN := rewards.SkimTopTopicsByWeightDesc(weights, N, 0)

	s.Require().Equal(N, uint64(len(mapOfTopN)), "SkimTopTopicsByWeightDesc should return a map with N keys")
	s.Require().Equal("700", mapOfTopN[3].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal("400", mapOfTopN[4].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	s.Require().Equal("300", mapOfTopN[2].String(), "SkimTopTopicsByWeightDesc should return the expected sorted list")
	// Check that mapOfTopN does not have any other keys
	_, ok := mapOfTopN[1]
	s.Require().Equal(false, ok, "SkimTopTopicsByWeightDesc should not have any other keys")
	_, ok = mapOfTopN[5]
	s.Require().Equal(false, ok, "SkimTopTopicsByWeightDesc should not have any other keys")
}

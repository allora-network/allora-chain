package rewards_test

import (
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
)

func (s *RewardsTestSuite) TestGetReputersScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate reputers data for tests
	reportedLosses, err := mockReputersData(s, topidId, block)
	s.Require().NoError(err)

	// Generate new reputer scores
	_, err = rewards.GenerateReputerScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	// TODO: Wait to merge the new losses types from kenny's PR before applying these tests (same for the other tests)
	// expectedScores := []float64{456.49, 172.71, 211.93, 52.31, 124.10}
	// for i, reputerScore := range scores {
	// 	if math.Abs(reputerScore.Score-expectedScores[i]) > 1e-2 {
	// 		s.Fail("Expected reward is not equal to the actual reward")
	// 	}
	// }
}

func (s *RewardsTestSuite) TestGetInferenceScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topidId, block)
	s.Require().NoError(err)

	// Get inference scores
	_, err = rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) TestGetForecastScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topidId, block)
	s.Require().NoError(err)

	// Get inference scores
	_, err = rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)
}

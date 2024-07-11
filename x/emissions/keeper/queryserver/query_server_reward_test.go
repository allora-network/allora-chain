package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetPreviousReputerRewardFraction() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := "reputerAddressExample"

	// Attempt to fetch a reward fraction before setting it
	req := &types.QueryPreviousReputerRewardFractionRequest{
		TopicId: topicId,
		Reputer: reputer,
	}
	response, err := s.queryServer.GetPreviousReputerRewardFraction(ctx, req)
	s.Require().NoError(err)
	defaultReward := response.RewardFraction
	notFound := response.NotFound

	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")
	s.Require().True(notFound)

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(50) // Assuming 0.50 as a fraction example
	_ = keeper.SetPreviousReputerRewardFraction(ctx, topicId, reputer, setReward)

	// Fetch and verify the reward fraction after setting
	response, err = s.queryServer.GetPreviousReputerRewardFraction(ctx, req)
	s.Require().NoError(err)
	fetchedReward := response.RewardFraction
	notFound = response.NotFound
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(notFound, "Should not return no prior value after setting")
}

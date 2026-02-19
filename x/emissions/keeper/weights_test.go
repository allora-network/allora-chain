package keeper_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func (s *KeeperTestSuite) TestLatestForecasterWeightFunctions() {
	ctx := s.Ctx()
	k := s.WeightstsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(0)
	weight := alloraMath.NewDecFromInt64(100)

	// Test initial state (should be zero)
	initialWeight, err := k.GetLatestForecasterWeight(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(initialWeight.IsZero(), "Initial weight should be zero")

	// Set weight
	err = k.SetLatestForecasterWeight(ctx, topicId, forecaster, weight)
	s.Require().NoError(err, "Setting latest forecaster weight should not fail")

	// Get and verify weight
	retrievedWeight, err := k.GetLatestForecasterWeight(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(weight, retrievedWeight, "Retrieved weight should match set weight")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentWeight, err := k.GetLatestForecasterWeight(ctx, differentTopicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(differentWeight.IsZero(), "Weight for different topic should be zero")
}

func (s *KeeperTestSuite) TestLatestInfererWeightFunctions() {
	ctx := s.Ctx()
	k := s.WeightstsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)
	weight := alloraMath.NewDecFromInt64(75)

	// Test initial state (should be zero)
	initialWeight, err := k.GetLatestInfererWeight(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().True(initialWeight.IsZero(), "Initial weight should be zero")

	// Set weight
	err = k.SetLatestInfererWeight(ctx, topicId, inferer, weight)
	s.Require().NoError(err, "Setting latest inferer weight should not fail")

	// Get and verify weight
	retrievedWeight, err := k.GetLatestInfererWeight(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(weight, retrievedWeight, "Retrieved weight should match set weight")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentWeight, err := k.GetLatestInfererWeight(ctx, differentTopicId, inferer)
	s.Require().NoError(err)
	s.Require().True(differentWeight.IsZero(), "Weight for different topic should be zero")
}

func (s *KeeperTestSuite) TestLatestRegretStdNormFunctions() {
	ctx := s.Ctx()
	k := s.WeightstsKeeper()
	topicId := uint64(1)
	stdNorm := alloraMath.NewDecFromInt64(50)

	// Test initial state (should be zero)
	initialStdNorm, err := k.GetLatestRegretStdNorm(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(initialStdNorm.IsZero(), "Initial stdNorm should be zero")

	// Set stdNorm
	err = k.SetLatestRegretStdNorm(ctx, topicId, stdNorm)
	s.Require().NoError(err, "Setting latest regret stdNorm should not fail")

	// Get and verify stdNorm
	retrievedStdNorm, err := k.GetLatestRegretStdNorm(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stdNorm, retrievedStdNorm, "Retrieved stdNorm should match set stdNorm")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentStdNorm, err := k.GetLatestRegretStdNorm(ctx, differentTopicId)
	s.Require().NoError(err)
	s.Require().True(differentStdNorm.IsZero(), "StdNorm for different topic should be zero")

	// Test setting zero value (should fail)
	err = k.SetLatestRegretStdNorm(ctx, topicId, alloraMath.ZeroDec())
	s.Require().Error(err, "Setting zero regret stdNorm should fail")
}

func (s *KeeperTestSuite) TestMonthlyRewards() {
	ctx := s.Ctx()
	k := s.WeightstsKeeper()

	// Initial state should be zero
	reputerRewards, err := k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.IsZero(), "Initial monthly reputer rewards should be zero")

	topicRewards, err := k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.IsZero(), "Initial monthly topic rewards should be zero")

	// Add some rewards
	addReputerAmount := cosmosMath.NewInt(1000)
	addTopicAmount := cosmosMath.NewInt(5000)
	err = k.AddMonthlyRewards(ctx, addReputerAmount, addTopicAmount)
	s.Require().NoError(err)

	// Check if rewards were added
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(addReputerAmount), "Monthly reputer rewards should be updated after adding")

	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(addTopicAmount), "Monthly topic rewards should be updated after adding")

	// Add more rewards
	addMoreReputerAmount := cosmosMath.NewInt(500)
	addMoreTopicAmount := cosmosMath.NewInt(2500)
	err = k.AddMonthlyRewards(ctx, addMoreReputerAmount, addMoreTopicAmount)
	s.Require().NoError(err)

	// Check total rewards
	totalExpectedReputer := addReputerAmount.Add(addMoreReputerAmount)
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(totalExpectedReputer), "Monthly reputer rewards should accumulate")

	totalExpectedTopic := addTopicAmount.Add(addMoreTopicAmount)
	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(totalExpectedTopic), "Monthly topic rewards should accumulate")

	// Reset rewards
	err = k.ResetMonthlyRewards(ctx)
	s.Require().NoError(err)

	// Check if rewards are reset to zero
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.IsZero(), "Monthly reputer rewards should be zero after reset")

	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.IsZero(), "Monthly topic rewards should be zero after reset")
}

package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestGetReputersRewards() {
	topidId := uint64(1)
	block := int64(1003)

	reputerAddrs := []string{
		s.addrs[0].String(),
		s.addrs[1].String(),
		s.addrs[2].String(),
		s.addrs[3].String(),
		s.addrs[4].String(),
	}

	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block, reputerAddrs)
	s.Require().NoError(err)

	// Reputers fractions of total reward
	reputerFractions := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.4486159141384113784159014720544400"),
		alloraMath.MustNewDecFromString("0.1697311778065889569890976063020783"),
		alloraMath.MustNewDecFromString("0.2082807308270030931355055033750957"),
		alloraMath.MustNewDecFromString("0.05141234465321950512077936581311225"),
		alloraMath.MustNewDecFromString("0.1219598325747770663387160524552737"),
	}

	// Get reputer rewards
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputerAddrs,
		reputerFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(reputerRewards))

	expectedRewards := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("456.49"),
		alloraMath.MustNewDecFromString("172.71"),
		alloraMath.MustNewDecFromString("211.93"),
		alloraMath.MustNewDecFromString("52.31"),
		alloraMath.MustNewDecFromString("124.10"),
	}

	for i, reputerReward := range reputerRewards {
		s.Require().True(
			alloraMath.InDelta(expectedRewards[i], reputerReward.Reward, alloraMath.MustNewDecFromString("0.01")),
			"expected: %s, got: %s",
			expectedRewards[i].String(),
			reputerReward.Reward.String(),
		)
	}
}

func (s *RewardsTestSuite) TestGetReputersRewardsShouldGenerateRewardsForDelegators() {
	topidId := uint64(1)
	block := int64(1003)

	reputerAddrs := []string{
		s.addrs[0].String(),
		s.addrs[1].String(),
		s.addrs[2].String(),
		s.addrs[3].String(),
		s.addrs[4].String(),
	}

	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block, reputerAddrs)
	s.Require().NoError(err)

	// Add balance to the reward account
	rewardToDistribute := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(100000000)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraRewardsAccountName, rewardToDistribute)

	// Check delegator rewards account balance
	moduleAccAddr := s.accountKeeper.GetModuleAddress(types.AlloraPendingRewardForDelegatorAccountName)
	inicialBalance := s.bankKeeper.GetBalance(s.ctx, moduleAccAddr, params.DefaultBondDenom)

	// Add delegator for the reputer 1
	err = s.emissionsKeeper.AddDelegateStake(s.ctx, topidId, s.addrs[5].String(), reputerAddrs[0], cosmosMath.NewUint(10000000000))
	s.Require().NoError(err)

	// Reputers fractions of total reward
	reputerFractions := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.4486159141384113784159014720544400"),
		alloraMath.MustNewDecFromString("0.1697311778065889569890976063020783"),
		alloraMath.MustNewDecFromString("0.2082807308270030931355055033750957"),
		alloraMath.MustNewDecFromString("0.05141234465321950512077936581311225"),
		alloraMath.MustNewDecFromString("0.1219598325747770663387160524552737"),
	}

	// Get reputer rewards
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputerAddrs,
		reputerFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(reputerRewards))

	finalBalance := s.bankKeeper.GetBalance(s.ctx, moduleAccAddr, params.DefaultBondDenom)

	// Check that the delegator has received rewards
	s.Require().True(
		finalBalance.Amount.GT(inicialBalance.Amount),
	)
}

// After removing the number of reputers, the rewards should increase for the remaining reputers
func (s *RewardsTestSuite) TestGetReputersRewardsShouldIncreaseRewardsAfterRemovingReputer() {
	topicId := uint64(1)
	block := int64(1003)

	reputerAddrs := []string{
		s.addrs[0].String(),
		s.addrs[1].String(),
		s.addrs[2].String(),
		s.addrs[3].String(),
		s.addrs[4].String(),
	}

	// Generate reputers data for tests
	_, err := mockReputersData(s, topicId, block, reputerAddrs)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err := s.emissionsKeeper.GetReputersScoresAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err)

	var reward_score []types.Score
	for _, score := range scores.Scores {
		reward_score = append(reward_score, types.Score{
			TopicId:     score.TopicId,
			BlockNumber: score.BlockNumber,
			Address:     score.Address,
			Score:       score.Score,
		})
	}
	// Get reputer rewards
	reputers, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		reward_score,
	)
	s.Require().NoError(err)
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(reputerRewards))

	expectedRewards := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("456.49"),
		alloraMath.MustNewDecFromString("172.71"),
		alloraMath.MustNewDecFromString("211.93"),
		alloraMath.MustNewDecFromString("52.31"),
		alloraMath.MustNewDecFromString("124.10"),
	}

	for i, reputerReward := range reputerRewards {
		s.Require().True(
			alloraMath.InDelta(expectedRewards[i], reputerReward.Reward, alloraMath.MustNewDecFromString("0.01")),
			"expected: %s, got: %s",
			expectedRewards[i].String(),
			reputerReward.Reward.String(),
		)
	}

	// New Topic, block and reputer addresses
	topicId = uint64(2)
	block = int64(1004)
	// Reduce number of reputer addresses
	reputerAddrs = []string{
		s.addrs[5].String(),
		s.addrs[6].String(),
		s.addrs[7].String(),
		s.addrs[8].String(),
	}

	// Generate reputers same loss data for less reputers
	_, err = mockReputersData(s, topicId, block, reputerAddrs)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err = s.emissionsKeeper.GetReputersScoresAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err)

	reward_score = make([]types.Score, 0)
	for _, score := range scores.Scores {
		reward_score = append(reward_score, types.Score{
			TopicId:     score.TopicId,
			BlockNumber: score.BlockNumber,
			Address:     score.Address,
			Score:       score.Score,
		})
	}

	// Get reputer rewards
	reputers, reputersRewardFractions, err = rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		reward_score,
	)
	s.Require().NoError(err)
	newReputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(4, len(newReputerRewards))

	for i, newReputerReward := range newReputerRewards {
		newReputerReward.Reward.Gt(reputerRewards[i].Reward)
		s.Require().True(
			newReputerReward.Reward.Gt(reputerRewards[i].Reward),
			"expected: %s, got: %s",
			newReputerReward.Reward.String(),
			reputerRewards[i].Reward.String(),
		)
	}
}

func (s *RewardsTestSuite) TestGetReputersRewardFractionsShouldIncreaseFractionOfRewardsForHigherStake() {
	topicId := uint64(1)
	block := int64(1003)

	reputerAddrs := []string{
		s.addrs[0].String(),
		s.addrs[1].String(),
		s.addrs[2].String(),
		s.addrs[3].String(),
		s.addrs[4].String(),
	}

	// Generate reputers data for tests
	reputerValueBundles, err := mockReputersData(s, topicId, block, reputerAddrs)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.ctx, s.emissionsKeeper, topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Get reputer rewards
	_, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)

	// Increase stake for the first reputer
	err = s.emissionsKeeper.AddStake(s.ctx, topicId, reputerAddrs[0], cosmosMath.NewUint(1000000))
	s.Require().NoError(err)

	// Get new reputer rewards
	_, newReputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)

	// Check that the first reputer has a higher fraction of the rewards
	s.Require().True(
		newReputersRewardFractions[0].Gt(reputersRewardFractions[0]),
	)
}

func (s *RewardsTestSuite) TestGetReputersRewardFractionsShouldOutputZeroForReputerWithZeroStake() {
	topicId := uint64(1)
	block := int64(1003)

	reputerAddrs := []string{
		s.addrs[0].String(),
		s.addrs[1].String(),
		s.addrs[2].String(),
		s.addrs[3].String(),
		s.addrs[4].String(),
	}

	// Generate reputers data for tests
	reputerValueBundles, err := mockReputersData(s, topicId, block, reputerAddrs)
	s.Require().NoError(err)

	// Remove stake for the first reputer
	err = s.emissionsKeeper.RemoveStake(s.ctx, topicId, reputerAddrs[0], cosmosMath.NewUint(1176644))
	s.Require().NoError(err)

	// Check if stake is zero
	stake, err := s.emissionsKeeper.GetStakeOnReputerInTopic(s.ctx, topicId, reputerAddrs[0])
	s.Require().NoError(err)
	s.Require().True(
		stake.IsZero(),
	)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.ctx, s.emissionsKeeper, topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Get reputer rewards
	_, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)

	// Check that the first reputer has zero rewards
	s.Require().True(
		reputersRewardFractions[0].IsZero(),
	)
}

// mockReputersData generates reputer scores, stakes and losses
func mockReputersData(s *RewardsTestSuite, topicId uint64, block int64, reputerAddrs []string) (types.ReputerValueBundles, error) {
	var scores = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("17.53436"),
		alloraMath.MustNewDecFromString("20.29489"),
		alloraMath.MustNewDecFromString("24.26994"),
		alloraMath.MustNewDecFromString("11.36754"),
		alloraMath.MustNewDecFromString("15.21749"),
	}
	var stakes = []cosmosMath.Uint{
		cosmosMath.NewUint(1176644),
		cosmosMath.NewUint(384623),
		cosmosMath.NewUint(394676),
		cosmosMath.NewUint(207999),
		cosmosMath.NewUint(368582),
	}

	var reputerValueBundles types.ReputerValueBundles
	for i, reputerAddr := range reputerAddrs {
		err := s.emissionsKeeper.AddStake(s.ctx, topicId, reputerAddr, stakes[i])
		if err != nil {
			return types.ReputerValueBundles{}, err
		}

		scoreToAdd := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     reputerAddr,
			Score:       scores[i],
		}
		err = s.emissionsKeeper.InsertReputerScore(s.ctx, topicId, block, scoreToAdd)
		if err != nil {
			return types.ReputerValueBundles{}, err
		}

		reputerValueBundle := &types.ReputerValueBundle{
			ValueBundle: &types.ValueBundle{
				TopicId:       topicId,
				Reputer:       reputerAddr,
				CombinedValue: alloraMath.MustNewDecFromString("1500.0"),
				NaiveValue:    alloraMath.MustNewDecFromString("1500.0"),
			},
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, reputerValueBundle)
	}

	err := s.emissionsKeeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, block, reputerValueBundles)
	if err != nil {
		return types.ReputerValueBundles{}, err
	}

	return reputerValueBundles, nil
}

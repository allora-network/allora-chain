package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestGetReputersRewards() {
	topidId := uint64(1)
	block := int64(1003)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block, reputerAddrs)
	s.Require().NoError(err)

	// Get reputer rewards
	reputerRewards, err := rewards.GetReputerRewards(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		alloraMath.OneDec(),
		alloraMath.MustNewDecFromString("1017.5559072418691"),
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

// After removing the number of reputers, the rewards should increase for the remaining reputers
func (s *RewardsTestSuite) TestGetReputersRewardsShouldIncreaseRewardsAfterRemovingReputer() {
	topidId := uint64(1)
	block := int64(1003)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block, reputerAddrs)
	s.Require().NoError(err)

	// Get reputer rewards
	reputerRewards, err := rewards.GetReputerRewards(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		alloraMath.OneDec(),
		alloraMath.MustNewDecFromString("1017.5559072418691"),
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
	topidId = uint64(2)
	block = int64(1004)
	// Reduce number of reputer addresses
	reputerAddrs = []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
	}

	// Generate reputers same loss data for less reputers
	_, err = mockReputersData(s, topidId, block, reputerAddrs)
	s.Require().NoError(err)

	// Get reputer rewards
	newReputerRewards, err := rewards.GetReputerRewards(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		alloraMath.OneDec(),
		alloraMath.MustNewDecFromString("1017.5559072418691"),
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

// mockReputersData generates reputer scores, stakes and losses
func mockReputersData(s *RewardsTestSuite, topicId uint64, block int64, reputerAddrs []sdk.AccAddress) (types.ReputerValueBundles, error) {
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
			Address:     reputerAddr.String(),
			Score:       scores[i],
		}
		err = s.emissionsKeeper.InsertReputerScore(s.ctx, topicId, block, scoreToAdd)
		if err != nil {
			return types.ReputerValueBundles{}, err
		}

		reputerValueBundle := &types.ReputerValueBundle{
			ValueBundle: &types.ValueBundle{
				TopicId:       topicId,
				Reputer:       reputerAddr.String(),
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

func (s *RewardsTestSuite) TestGetReputersRewardsFromTotalRewards() {
	topidId := uint64(1)
	block := int64(1003)

	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	var delegatorStake = cosmosMath.NewUint(5000)
	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block, reputerAddrs)
	_ = s.emissionsKeeper.SetDelegateStakeUponReputer(s.ctx, topidId, s.addrs[0], delegatorStake)
	s.Require().NoError(err)

	newReputerRewards, err := rewards.GetReputerRewards(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		alloraMath.OneDec(),
		alloraMath.MustNewDecFromString("1017.5559072418691"),
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(newReputerRewards))
}

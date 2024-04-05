package rewards_test

import (
	"math"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestGetReputersRewards() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate reputers data for tests
	_, err := mockReputersData(s, topidId, block)
	s.Require().NoError(err)

	// Get reputer rewards
	reputerRewards, err := rewards.GetReputerRewards(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		1,
		1017.5559072418691,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(reputerRewards))

	expectedRewards := []float64{456.49, 172.71, 211.93, 52.31, 124.10}

	for i, reputerReward := range reputerRewards {
		if math.Abs(reputerReward.Reward-expectedRewards[i]) > 1e-2 {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

// mockReputersData generates reputer scores, stakes and losses
func mockReputersData(s *RewardsTestSuite, topicId uint64, block int64) (types.ReputerValueBundles, error) {
	reputerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	var scores = []float64{17.53436, 20.29489, 24.26994, 11.36754, 15.21749}
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
			Reputer: reputerAddr.String(),
			ValueBundle: &types.ValueBundle{
				TopicId:       topicId,
				CombinedValue: 1500.0,
				NaiveValue:    1500.0,
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

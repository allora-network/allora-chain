package rewards

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The reputer rewards are calculated based on the reputer stake and the reputer score.
// The reputer score is defined right after the network loss is generated.
func GetReputerRewards(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	preward alloraMath.Dec,
	totalReputerRewards alloraMath.Dec,
) ([]TaskRewards, error) {
	// Get All reported losses from last block
	reportedLosses, err := keeper.GetReputerLossBundlesAtBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get reputer scores at block
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get reputers informations
	var reputerAddresses []sdk.AccAddress
	var reputerStakes []alloraMath.Dec
	var scoresDec []alloraMath.Dec
	for _, reportedLoss := range reportedLosses.ReputerValueBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(reportedLoss.Reputer)
		if err != nil {
			return nil, err
		}
		reputerAddresses = append(reputerAddresses, reputerAddr)

		// Get reputer topic stake
		reputerStake, err := keeper.GetDelegatedStakeUponReputer(ctx, topicId, reputerAddr)
		if err != nil {
			return nil, err
		}
		reputerStakeDec, err := alloraMath.NewDecFromSdkUint(reputerStake)
		if err != nil {
			return nil, err
		}
		reputerStakes = append(reputerStakes, reputerStakeDec)

		// Get reputer score
		for _, score := range scores {
			if score.Address == reputerAddr.String() {
				scoresDec = append(scoresDec, score.Score)
			}
		}
	}

	// Get reputer rewards fractions
	reputersFractions, err := GetReputerRewardFractions(reputerStakes, scoresDec, preward)
	if err != nil {
		return nil, err
	}

	// Calculate reputer rewards
	var reputerRewards []TaskRewards
	for i, reputerFraction := range reputersFractions {
		reward, err := reputerFraction.Mul(totalReputerRewards)
		if err != nil {
			return nil, err
		}
		reputerRewards = append(reputerRewards, TaskRewards{
			Address: reputerAddresses[i],
			Reward:  reward,
		})
	}

	return reputerRewards, nil
}

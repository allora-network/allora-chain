package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetReputerRewards(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	preward float64,
	totalReputerRewards float64,
) ([]TaskRewards, error) {
	// Get All reported losses from last block
	reportedLosses, err := keeper.GetValueBundles(ctx, topicId, block)
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
	var reputerStakes []float64
	var reputerListeningCoefficients []float64
	var scoresFloat []float64
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
		reputerStakes = append(reputerStakes, float64(reputerStake.BigInt().Int64()))

		// Get reputer listening coefficient
		res, err := keeper.GetListeningCoefficient(ctx, topicId, reputerAddr)
		if err != nil {
			return nil, err
		}
		reputerListeningCoefficients = append(reputerListeningCoefficients, res.Coefficient)

		// Get reputer score
		for _, score := range scores {
			if score.Address == reputerAddr.String() {
				scoresFloat = append(scoresFloat, score.Score)
			}
		}
	}

	// Get reputer rewards fractions
	reputersFractions, err := GetReputerRewardFractions(reputerStakes, scoresFloat, preward)
	if err != nil {
		return nil, err
	}

	// Calculate reputer rewards
	var reputerRewards []TaskRewards
	for i, reputerFraction := range reputersFractions {
		reputerRewards = append(reputerRewards, TaskRewards{
			Address: reputerAddresses[i],
			Reward:  reputerFraction * totalReputerRewards,
		})
	}

	return reputerRewards, nil
}

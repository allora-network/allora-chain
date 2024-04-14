package rewards

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetReputerTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	pRewardSpread alloraMath.Dec,
	betaEntropy alloraMath.Dec,
) (
	entropy alloraMath.Dec,
	modifiedRewardFractions []alloraMath.Dec,
	reputers []sdk.AccAddress,
	err error,
) {
	scoresAtBlock, err := k.GetReputersScoresAtBlock(ctx, topicId, ctx.BlockHeight())
	if err != nil {
		return alloraMath.Dec{}, nil, nil, err
	}
	numReputers := len(scoresAtBlock.Scores)
	stakes := make([]alloraMath.Dec, numReputers)
	scores := make([]alloraMath.Dec, numReputers)
	reputers = make([]sdk.AccAddress, numReputers)
	for i, scorePtr := range scoresAtBlock.Scores {
		scores[i] = scorePtr.Score
		addrStr := scorePtr.Address
		reputerAddr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			return alloraMath.Dec{}, nil, nil, err
		}
		reputers[i] = reputerAddr
		stake, err := k.GetStakeOnTopicFromReputer(ctx, topicId, reputerAddr)
		if err != nil {
			return alloraMath.Dec{}, nil, nil, err
		}
		stakes[i], err = alloraMath.NewDecFromSdkUint(stake)
		if err != nil {
			return alloraMath.Dec{}, nil, nil, err
		}
	}

	reputerRewardFractions, err := GetReputerRewardFractions(stakes, scores, pRewardSpread)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, err
	}
	emaReputerRewards := make([]alloraMath.Dec, numReputers)
	for i, fraction := range reputerRewardFractions {
		previousReputerRewardFraction, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputers[i])
		if err != nil {
			return alloraMath.Dec{}, nil, nil, err
		}
		emaReputerRewards[i], err = alloraMath.ExponentialMovingAverage(
			emaAlpha,
			fraction,
			previousReputerRewardFraction,
		)
		if err != nil {
			return alloraMath.Dec{}, nil, nil, err
		}
	}
	reputerNumberRatio, err := NumberRatio(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, err
	}
	modifiedRewardFractions, err = ModifiedRewardFractions(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, err
	}
	entropy, err = Entropy(
		modifiedRewardFractions,
		reputerNumberRatio,
		alloraMath.NewDecFromInt64(int64(numReputers)),
		betaEntropy,
	)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, err
	}
	return entropy, modifiedRewardFractions, reputers, nil
}

// Get the reward allocated to the reputing task in this topic, W_i
// W_i = (H_i * E_i) / (F_i + G_i + H_i)
func GetRewardForReputerTaskInTopic(
	entropyInference alloraMath.Dec, // F_i
	entropyForecasting alloraMath.Dec, // G_i
	entropyReputer alloraMath.Dec, // H_i
	topicReward alloraMath.Dec, // E_{t,i}
) (alloraMath.Dec, error) {
	numerator, err := entropyReputer.Mul(topicReward)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	denominator, err := entropyInference.Add(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	denominator, err = denominator.Add(entropyReputer)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := numerator.Quo(denominator)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

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
		reputerAddr, err := sdk.AccAddressFromBech32(reportedLoss.ValueBundle.Reputer)
		if err != nil {
			return nil, err
		}
		reputerAddresses = append(reputerAddresses, reputerAddr)

		// Get reputer topic stake
		reputerStake, err := keeper.GetStakeOnTopicFromReputer(ctx, topicId, reputerAddr)
		if err != nil {
			return nil, err
		}
		reputerStakeDec, err := alloraMath.NewDecFromSdkUint(reputerStake)
		if err != nil {
			return nil, err
		}
		reputerStakes = append(reputerStakes, reputerStakeDec)

		// Get reputer score
		for _, score := range scores.Scores {
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

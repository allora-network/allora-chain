package rewards

import (
	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetReputersRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	pRewardSpread alloraMath.Dec,
	scoresAtBlock []types.Score,
) ([]string, []alloraMath.Dec, error) {

	numReputers := len(scoresAtBlock)
	stakes := make([]alloraMath.Dec, numReputers)
	scores := make([]alloraMath.Dec, numReputers)
	reputers := make([]string, numReputers)
	for i, scorePtr := range scoresAtBlock {
		scores[i] = scorePtr.Score
		reputers[i] = scorePtr.Address
		stake, err := k.GetStakeOnReputerInTopic(ctx, topicId, scorePtr.Address)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer stake on topic %d", topicId)
		}
		stakes[i], err = alloraMath.NewDecFromSdkInt(stake)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to convert reputer stake %d", stake)
		}
	}

	rewardFractions, err := CalculateReputerRewardFractions(stakes, scores, pRewardSpread)
	if err != nil {
		return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer reward fractions")
	}

	return reputers, rewardFractions, nil
}

func GetReputerTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	betaEntropy alloraMath.Dec,
	reputers []string,
	reputerFractions []alloraMath.Dec,
) (
	entropy alloraMath.Dec,
	err error,
) {
	numReputers := len(reputers)
	emaReputerRewards := make([]alloraMath.Dec, numReputers)
	for i, reputer := range reputers {
		previousReputerRewardFraction, noPriorRegret, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to get previous reputer reward fraction")
		}
		emaReputerRewards[i], err = alloraMath.CalcEma(
			emaAlpha,
			reputerFractions[i],
			previousReputerRewardFraction,
			noPriorRegret,
		)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate EMA reputer rewards")
		}
	}

	// Calculate modified reward fractions and persist for next round
	reputerNumberRatio, err := NumberRatio(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate reputer number ratio")
	}
	modifiedRewardFractions, err := ModifiedRewardFractions(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate modified reward fractions")
	}
	for i, reputer := range reputers {
		err := k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, modifiedRewardFractions[i])
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to set previous reputer reward fraction")
		}
	}

	if numReputers > 1 {
		entropy, err = Entropy(
			modifiedRewardFractions,
			reputerNumberRatio,
			alloraMath.NewDecFromInt64(int64(numReputers)),
			betaEntropy,
		)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate entropy")
		}
	} else {
		entropy, err = EntropyForSingleParticipant()
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate entropy for single participant")
		}
	}

	return entropy, nil
}

// Get the reward allocated to the reputing task in this topic, W_i
// W_i = (H_i * E_i) / (F_i + G_i + H_i)
func GetRewardForReputerTaskInTopic(
	entropyInference alloraMath.Dec, // F_i
	entropyForecasting alloraMath.Dec, // G_i
	entropyReputer alloraMath.Dec, // H_i
	topicReward *alloraMath.Dec, // E_{t,i}
) (alloraMath.Dec, error) {
	numerator, err := entropyReputer.Mul(*topicReward)
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

// Send total reward for delegator to PENDING_ACCOUNT
// and return remain reward for reputer
func GetRewardForReputerFromTotalReward(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	reputerDelegatorRewards []TaskRewards,
) ([]TaskRewards, error) {

	var reputerRewards []TaskRewards
	for _, reputerReward := range reputerDelegatorRewards {
		reputer := reputerReward.Address
		reward := reputerReward.Reward
		totalStakeAmount, err := keeper.GetStakeOnReputerInTopic(ctx, topicId, reputer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reputer stake")
		}
		// calculate reward for delegator total staked amount and send it to AlloraPendingRewardForDelegatorAccountName
		totalDelegatorStakeAmount, err := keeper.GetDelegateStakeUponReputer(ctx, topicId, reputer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reputer upon stake")
		}

		fraction := totalDelegatorStakeAmount.Mul(synth.CosmosIntOneE18()).Quo(totalStakeAmount)
		fractionUint, err := alloraMath.NewDecFromSdkInt(fraction)
		if err != nil {
			return nil, err
		}
		delegatorReward, err := reward.Mul(fractionUint)
		if err != nil {
			return nil, err
		}
		e18, err := alloraMath.NewDecFromSdkInt(synth.CosmosIntOneE18())
		if err != nil {
			return nil, err
		}
		delegatorReward, err = delegatorReward.Quo(e18)
		if err != nil {
			return nil, err
		}
		if delegatorReward.Gt(alloraMath.NewDecFromInt64(0)) {
			// update reward share
			// new_share = current_share + (reward / total_stake)
			totalDelegatorStakeAmountDec, err := alloraMath.NewDecFromSdkInt(totalDelegatorStakeAmount)
			if err != nil {
				return nil, err
			}
			addShare, err := delegatorReward.Quo(totalDelegatorStakeAmountDec)
			if err != nil {
				return nil, err
			}
			currentShare, err := keeper.GetDelegateRewardPerShare(ctx, topicId, reputer)
			if err != nil {
				return nil, err
			}
			newShare, err := currentShare.Add(addShare)
			if err != nil {
				return nil, err
			}
			err = keeper.SetDelegateRewardPerShare(ctx, topicId, reputer, newShare)
			if err != nil {
				return nil, err
			}
			err = keeper.SendCoinsFromModuleToModule(
				ctx,
				types.AlloraRewardsAccountName,
				types.AlloraPendingRewardForDelegatorAccountName,
				sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, delegatorReward.SdkIntTrim())),
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to send coins to allora pend reward account")
			}
		}
		// Send remain rewards to reputer
		reputerRw, err := reward.Sub(delegatorReward)
		if err != nil {
			return nil, err
		}
		reputerRewards = append(reputerRewards, TaskRewards{
			Address: reputerReward.Address,
			Reward:  reputerRw,
			TopicId: reputerReward.TopicId,
			Type:    ReputerRewardType,
		})
	}

	return reputerRewards, nil
}

// Get reward per reputer based on total reputer rewards and reputer fractions
// W_im = w_ij * W_i
func GetRewardPerReputer(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	totalReputerRewards alloraMath.Dec,
	reputerAddresses []string,
	reputersFractions []alloraMath.Dec,
) ([]TaskRewards, error) {
	var reputerDelegatorTotalRewards []TaskRewards
	for i, reputerFraction := range reputersFractions {
		reward, err := reputerFraction.Mul(totalReputerRewards)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to calculate reputer rewards")
		}
		reputerDelegatorTotalRewards = append(reputerDelegatorTotalRewards, TaskRewards{
			Address: reputerAddresses[i],
			Reward:  reward,
			TopicId: topicId,
			Type:    ReputerRewardType,
		})
	}

	reputerRewards, err := GetRewardForReputerFromTotalReward(
		ctx,
		keeper,
		topicId,
		reputerDelegatorTotalRewards,
	)
	return reputerRewards, err
}

package rewards

import (
	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetReputersRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	pReward alloraMath.Dec,
	scoresAtBlock []types.Score,
) ([]string, []alloraMath.Dec, error) {

	numReputers := len(scoresAtBlock)
	stakes := make([]alloraMath.Dec, numReputers)
	scores := make([]alloraMath.Dec, numReputers)
	reputers := make([]string, numReputers)
	for i, scorePtr := range scoresAtBlock {
		scores[i] = scorePtr.Score
		reputers[i] = scorePtr.Address
		stake, err := k.GetStakeReputerAuthority(ctx, topicId, scorePtr.Address)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer stake on topic %d", topicId)
		}
		stakes[i], err = alloraMath.NewDecFromSdkInt(stake)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to convert reputer stake %d", stake)
		}
	}

	rewardFractions, err := CalculateReputerRewardFractions(stakes, scores, pReward)
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
	emaRewardFractions := make([]alloraMath.Dec, numReputers)
	for i, reputer := range reputers {
		previousReputerRewardFraction, noPriorFraction, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to get previous reputer reward fraction")
		}
		emaRewardFractions[i], err = alloraMath.CalcEma(
			emaAlpha,
			reputerFractions[i],
			previousReputerRewardFraction,
			noPriorFraction,
		)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate EMA reputer rewards")
		}
	}

	// Calculate modified reward fractions and persist for next round
	numberRatio, err := NumberRatio(emaRewardFractions)
	if err != nil {
		return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate reputer number ratio")
	}
	modifiedRewardFractions, err := ModifiedRewardFractions(emaRewardFractions)
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
			numberRatio,
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
	if topicReward == nil {
		return alloraMath.Dec{}, types.ErrInvalidReward
	}
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
	reputerDelegatorRewards []types.TaskReward,
) ([]types.TaskReward, error) {

	var reputerRewards []types.TaskReward
	for _, reputerReward := range reputerDelegatorRewards {
		reputer := reputerReward.Address
		reward := reputerReward.Reward
		totalStakeAmountInt, err := keeper.GetStakeReputerAuthority(ctx, topicId, reputer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reputer stake")
		}
		totalStakeAmountDec, err := alloraMath.NewDecFromSdkInt(totalStakeAmountInt)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert reputer total stake to dec")
		}
		// calculate reward for delegator total staked amount and send it to AlloraPendingRewardForDelegatorAccountName
		totalDelegatorStakeAmountInt, err := keeper.GetDelegateStakeUponReputer(ctx, topicId, reputer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reputer upon stake")
		}
		totalDelegatorStakeAmountDec, err := alloraMath.NewDecFromSdkInt(totalDelegatorStakeAmountInt)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert delegator stake upon reputer to dec")
		}

		fractionDec, err := totalDelegatorStakeAmountDec.Quo(totalStakeAmountDec)
		if err != nil {
			return nil, err
		}
		delegatorRewardDec, err := reward.Mul(fractionDec)
		if err != nil {
			return nil, err
		}
		delegatorRewardInt := delegatorRewardDec.SdkIntTrim()
		delegatorRewardDec, err = alloraMath.NewDecFromSdkInt(delegatorRewardInt)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to reconvert delegator reward from int to dec")
		}
		if delegatorRewardInt.GT(cosmosMath.ZeroInt()) {
			// update reward share
			// new_share = current_share + (reward / total_stake)
			addShareDec, err := delegatorRewardDec.Quo(totalDelegatorStakeAmountDec)
			if err != nil {
				return nil, err
			}
			currentShareDec, err := keeper.GetDelegateRewardPerShare(ctx, topicId, reputer)
			if err != nil {
				return nil, err
			}
			newShare, err := currentShareDec.Add(addShareDec)
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
				sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, delegatorRewardInt)),
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to send coins to allora pend reward account")
			}
		}
		// Send remain rewards to reputer
		// delegatorRewardDec has already been trimmed.
		// Any decimals are left in reputerRw to be trimmed elsewhere
		reputerRw, err := reward.Sub(delegatorRewardDec)
		if err != nil {
			return nil, err
		}
		reputerRewards = append(reputerRewards, types.TaskReward{
			Address: reputerReward.Address,
			Reward:  reputerRw,
			TopicId: reputerReward.TopicId,
			Type:    types.ReputerAndDelegatorRewardType,
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
) ([]types.TaskReward, error) {
	var reputerDelegatorTotalRewards []types.TaskReward
	for i, reputerFraction := range reputersFractions {
		reward, err := reputerFraction.Mul(totalReputerRewards)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to calculate reputer rewards")
		}
		reputerDelegatorTotalRewards = append(reputerDelegatorTotalRewards, types.TaskReward{
			Address: reputerAddresses[i],
			Reward:  reward,
			TopicId: topicId,
			Type:    types.ReputerAndDelegatorRewardType,
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

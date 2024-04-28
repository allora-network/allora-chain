package rewards

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetReputerTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	pRewardSpread alloraMath.Dec,
	betaEntropy alloraMath.Dec,
	blockHeight int64,
) (
	entropy alloraMath.Dec,
	modifiedRewardFractions []alloraMath.Dec,
	reputers []sdk.AccAddress,
	err error,
) {
	scoresAtBlock, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return alloraMath.Dec{},
			nil,
			nil,
			errors.Wrapf(err, "failed to get reputers scores at block %d", blockHeight)
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
			return alloraMath.Dec{},
				nil,
				nil,
				errors.Wrapf(err, "failed to convert reputer address %s", addrStr)
		}
		reputers[i] = reputerAddr
		stake, err := k.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr)
		if err != nil {
			return alloraMath.Dec{},
				nil,
				nil,
				errors.Wrapf(err, "failed to get reputer stake on topic %d", topicId)
		}
		stakes[i], err = alloraMath.NewDecFromSdkUint(stake)
		if err != nil {
			return alloraMath.Dec{},
				nil,
				nil,
				errors.Wrapf(err, "failed to convert reputer stake %d", stake)
		}
	}

	reputerRewardFractions, err := GetReputerRewardFractions(stakes, scores, pRewardSpread)
	if err != nil {
		return alloraMath.Dec{},
			nil,
			nil,
			errors.Wrapf(err, "failed to get reputer reward fractions")
	}
	emaReputerRewards := make([]alloraMath.Dec, numReputers)
	for i, fraction := range reputerRewardFractions {
		previousReputerRewardFraction, noPriorRegret, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputers[i])
		if err != nil {
			return alloraMath.Dec{},
				nil,
				nil,
				errors.Wrapf(err, "failed to get previous reputer reward fraction")
		}
		emaReputerRewards[i], err = alloraMath.CalcEma(
			emaAlpha,
			fraction,
			previousReputerRewardFraction,
			noPriorRegret,
		)
		if err != nil {
			return alloraMath.Dec{},
				nil,
				nil,
				errors.Wrapf(err, "failed to calculate EMA reputer rewards")
		}
	}
	reputerNumberRatio, err := NumberRatio(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{},
			nil,
			nil,
			errors.Wrapf(err, "failed to calculate reputer number ratio")
	}
	modifiedRewardFractions, err = ModifiedRewardFractions(emaReputerRewards)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, errors.Wrapf(err, "failed to calculate modified reward fractions")
	}
	entropy, err = Entropy(
		modifiedRewardFractions,
		reputerNumberRatio,
		alloraMath.NewDecFromInt64(int64(numReputers)),
		betaEntropy,
	)
	if err != nil {
		return alloraMath.Dec{}, nil, nil, errors.Wrapf(err, "failed to calculate entropy")
	}
	return entropy, modifiedRewardFractions, reputers, nil
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

		fraction := totalDelegatorStakeAmount.Mul(keeper.GetAlloraExponent()).Quo(totalStakeAmount)
		fractionUint, err := alloraMath.NewDecFromSdkUint(fraction)
		if err != nil {
			return nil, err
		}
		delegatorReward, err := reward.Mul(fractionUint)
		if err != nil {
			return nil, err
		}
		e18, err := alloraMath.NewDecFromSdkUint(keeper.GetAlloraExponent())
		if err != nil {
			return nil, err
		}
		delegatorReward, err = delegatorReward.Quo(e18)
		if delegatorReward.Gt(alloraMath.NewDecFromInt64(0)) {
			// update reward share
			// new_share = current_share + (reward / total_stake)
			totalDelegatorStakeAmountDec, err := alloraMath.NewDecFromSdkUint(totalDelegatorStakeAmount)
			if err != nil {
				return nil, err
			}
			addShare, err := delegatorReward.Mul(e18)
			if err != nil {
				return nil, err
			}
			addShare, err = addShare.Quo(totalDelegatorStakeAmountDec)
			if err != nil {
				return nil, err
			}
			currentShare, err := keeper.GetDelegateRewardPerShare(ctx, topicId, reputer)
			if err != nil {
				return nil, err
			}
			newShare := currentShare.Add(math.NewUint(addShare.SdkIntTrim().Uint64()))
			err = keeper.SetDelegateRewardPerShare(ctx, topicId, reputer, newShare)
			if err != nil {
				return nil, err
			}

			err = keeper.BankKeeper().SendCoinsFromModuleToModule(
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
		reputerRewards = append(reputerRewards, TaskRewards{
			Address: reputerReward.Address,
			Reward:  reputerRw,
		})
	}

	return reputerRewards, nil
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
		return nil, errors.Wrapf(err, "failed to get reputer loss bundles at block %d", block)
	}

	// Get reputer scores at block
	scores, err := keeper.GetReputersScoresAtBlock(ctx, topicId, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get reputers scores at block %d", block)
	}

	// Get reputers informations
	var reputerAddresses []sdk.AccAddress
	var reputerStakes []alloraMath.Dec
	var scoresDec []alloraMath.Dec
	for _, reportedLoss := range reportedLosses.ReputerValueBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(reportedLoss.ValueBundle.Reputer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert reputer address %s", reportedLoss.ValueBundle.Reputer)
		}
		reputerAddresses = append(reputerAddresses, reputerAddr)

		// Get reputer topic stake
		reputerStake, err := keeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reputer stake on topic %d", topicId)
		}
		reputerStakeDec, err := alloraMath.NewDecFromSdkUint(reputerStake)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert reputer stake %d", reputerStake)
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
		return nil, errors.Wrapf(err, "failed to get reputer reward fractions")
	}

	// Calculate reputer rewards
	var reputerDelegatorTotalRewards []TaskRewards
	for i, reputerFraction := range reputersFractions {
		reward, err := reputerFraction.Mul(totalReputerRewards)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to calculate reputer rewards")
		}
		reputerDelegatorTotalRewards = append(reputerDelegatorTotalRewards, TaskRewards{
			Address: reputerAddresses[i],
			Reward:  reward,
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

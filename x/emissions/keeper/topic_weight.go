package keeper

import (
	"context"

	"cosmossdk.io/errors"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Return the target weight of a topic
// ^w_{t,i} = S^{μ}_{t,i} * (P/C)^{ν}_{t,i}
// where S_{t,i} is the stake of of topic t in the last reward epoch i
// and (P/C)_{t,i} is the fee revenue collected for performing inference per topic epoch
// requests for topic t in the last reward epoch i
// μ, ν are global constants with fiduciary values of 0.5 and 0.5
func (k *Keeper) GetTargetWeight(
	topicStake alloraMath.Dec,
	topicEpochLength int64,
	topicFeeRevenue alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
) (alloraMath.Dec, error) {
	s, err := alloraMath.Pow(topicStake, stakeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	c := alloraMath.NewDecFromInt64(topicEpochLength)
	feePerEpoch, err := topicFeeRevenue.Quo(c)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	p, err := alloraMath.Pow(feePerEpoch, feeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return s.Mul(p)
}

func (k *Keeper) GetCurrentTopicWeight(
	ctx context.Context,
	topicId TopicId,
	topicEpochLength BlockHeight,
	topicRewardAlpha alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
	additionalRevenue cosmosMath.Int,
	additionalStake cosmosMath.Int,
) (weight alloraMath.Dec, topicRevenue cosmosMath.Int, err error) {
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get topic stake")
	}

	newTopicStake := topicStake.Add(additionalStake)
	topicStakeDec, err := alloraMath.NewDecFromSdkInt(newTopicStake)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to convert topic stake to dec")
	}

	// Get and total topic fee revenue
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get topic fee revenue")
	}

	// Calc target weight using fees, epoch length, stake, and params
	newFeeRevenue := additionalRevenue.Add(topicFeeRevenue)
	feeRevenue, err := alloraMath.NewDecFromSdkInt(newFeeRevenue)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to convert topic fee revenue to dec")
	}

	if !feeRevenue.Equal(alloraMath.ZeroDec()) {
		targetWeight, err := k.GetTargetWeight(
			topicStakeDec,
			topicEpochLength,
			feeRevenue,
			stakeImportance,
			feeImportance,
		)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get target weight")
		}

		// Take EMA of target weight with previous weight
		previousTopicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get previous topic weight")
		}
		weight, err = alloraMath.CalcEma(topicRewardAlpha, targetWeight, previousTopicWeight, noPrior)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to calculate EMA")
		}

		return weight, topicFeeRevenue, nil
	}

	return alloraMath.ZeroDec(), topicFeeRevenue, nil
}

package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"

	"cosmossdk.io/errors"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Return the target weight of a topic
// ^w_{t,i} = S^{μ}_{t,i} * (P/C)^{ν}_{t,i}
// where S_{t,i} is the stake of topic t in the last reward epoch i
// and (P/C)_{t,i} is the fee revenue collected for performing inference per topic epoch
// requests for topic t in the last reward epoch i
// μ, ν are global constants with fiduciary values of 0.5 and 0.5
func (k *Keeper) GetTargetWeight(
	topicStake alloraMath.Dec,
	topicFeeRevenue alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
) (alloraMath.Dec, error) {
	s, err := alloraMath.Pow(topicStake, stakeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	p, err := alloraMath.Pow(topicFeeRevenue, feeImportance)
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
	blocksPerMonth uint64,
) (weight alloraMath.Dec, topicRevenue cosmosMath.Int, topicStake cosmosMath.Int, err error) {
	topicStake, err = k.GetTopicStake(ctx, topicId)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get topic stake")
	}

	topicStakeDec, err := alloraMath.NewDecFromSdkInt(topicStake)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to convert topic stake to dec")
	}

	// Get and total topic fee revenue
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get topic fee revenue")
	}

	// Calc target weight using fees, epoch length, stake, and params
	topicFeeRevenueDec, err := alloraMath.NewDecFromSdkInt(topicFeeRevenue)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to convert topic fee revenue to dec")
	}

	if !topicFeeRevenueDec.Equal(alloraMath.ZeroDec()) {
		targetWeight, err := k.GetTargetWeight(
			topicStakeDec,
			topicFeeRevenueDec,
			stakeImportance,
			feeImportance,
		)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get target weight")
		}

		// Take EMA of target weight with previous weight
		previousTopicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get previous topic weight")
		}

		blocksPerWeek, err := alloraMath.CalculateBlocksPerWeek(blocksPerMonth)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to calculate blocks per week")
		}

		// Calculate alpha
		topicSmoothedAlpha, err := alloraMath.GetSmoothedAlpha(topicEpochLength, blocksPerWeek, topicRewardAlpha)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get smoothed alpha")
		}
		weightNew, err := alloraMath.CalcEma(topicSmoothedAlpha, targetWeight, previousTopicWeight, noPrior)
		if err != nil {
			return alloraMath.Dec{}, cosmosMath.Int{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to calculate EMA")
		}
		return weightNew, topicFeeRevenue, topicStake, nil
	}

	return alloraMath.ZeroDec(), topicFeeRevenue, topicStake, nil
}

func (k *Keeper) GetTopicWeightFromTopicId(ctx context.Context, topicId types.TopicId) (alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	newTopicWeight, _, _, err := k.GetCurrentTopicWeight(
		ctx,
		topicId,
		topic.EpochLength,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)

	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	return newTopicWeight, nil
}

package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *ScoresKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// InfererScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.InfererScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := k.SetInfererScoresByBlock(ctx,
				topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight,
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
		}
	}
	// ForecasterScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.ForecasterScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := k.SetForecasterScoresByBlock(
				ctx, topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight,
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
		}
	}

	// ReputerScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.ReputerScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := types.ValidateTopicId(topicIdBlockHeightScores.TopicId); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := types.ValidateBlockHeight(topicIdBlockHeightScores.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := topicIdBlockHeightScores.Scores.Validate(); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := k.reputerScoresByBlock.Set(
				ctx,
				collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
		}
	}

	// InfererScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.InfererScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetInfererScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestInfererScoresByWorker")
			}
		}
	}
	// ForecasterScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ForecasterScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetForecasterScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestForecasterScoresByWorker")
			}
		}
	}
	// ReputerScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ReputerScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetReputerScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestReputerScoresByReputer")
			}
		}
	}
	// ReputerListeningCoefficient []*TopicIdActorIdListeningCoefficient
	for _, topicIdActorIdListeningCoefficient := range data.ReputerListeningCoefficient {
		if topicIdActorIdListeningCoefficient != nil {
			if err := k.SetListeningCoefficient(ctx,
				topicIdActorIdListeningCoefficient.TopicId, topicIdActorIdListeningCoefficient.ActorId,
				*topicIdActorIdListeningCoefficient.ListeningCoefficient); err != nil {
				return errors.Wrap(err, "error setting reputerListeningCoefficient")
			}
		}
	}
	// PreviousReputerRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousReputerRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousReputerRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousReputerRewardFraction")
			}
		}
	}
	// PreviousInferenceRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousInferenceRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousInferenceRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousInferenceRewardFraction")
			}
		}
	}
	// PreviousForecastRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousForecastRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousForecastRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousForecastRewardFraction")
			}
		}
	}

	// PreviousPercentageRewardToStakedReputers
	if data.PreviousPercentageRewardToStakedReputers != alloraMath.ZeroDec() {
		if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, data.PreviousPercentageRewardToStakedReputers); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers")
		}
	} else {
		// For mint module inflation rate calculation set the initial
		// "previous percentage of rewards that went to staked reputers" to 30%
		if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.3")); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers to 0.3")
		}
	}

	// PreviousForecasterScoreRatio
	for _, topicIdDec := range data.PreviousForecasterScoreRatio {
		if topicIdDec != nil {
			if err := k.SetPreviousForecasterScoreRatio(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousForecasterScoreRatio")
			}
		}
	}

	// PreviousTopicQuantileInfererScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileInfererScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileInfererScoreEma")
			}
		}
	}

	// PreviousTopicQuantileForecasterScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileForecasterScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileForecasterScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileForecasterScoreEma")
			}
		}
	}

	// PreviousTopicQuantileReputerScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileReputerScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileReputerScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileReputerScoreEma")
			}
		}
	}

	// InitialInfererEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialInfererEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialInfererEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialInfererEmaScore")
			}
		}
	}

	// InitialForecasterEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialForecasterEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialForecasterEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialForecasterEmaScore")
			}
		}
	}

	// InitialReputerEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialReputerEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialReputerEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialReputerEmaScore")
			}
		}
	}

	// LowestInfererScoreEma []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestInfererScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestInfererScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestInfererScoreEma")
			}
		}
	}

	// LowestForecasterScoreEma []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestForecasterScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestForecasterScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestForecasterScoreEma")
			}
		}
	}

	// LowestReputerScoreEma []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestReputerScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestReputerScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestReputerScoreEma")
			}
		}
	}

	return nil
}

func (k *ScoresKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// initialInfererEmaScore
	var initialInfererEmaScore []*types.TopicIdAndDec
	if err := k.initialInfererEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialInfererEmaScore = append(initialInfererEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to walk inferer initial EMA score per topic")
	}
	data.InitialInfererEmaScore = initialInfererEmaScore

	// initialForecasterEmaScore
	var initialForecasterEmaScore []*types.TopicIdAndDec
	if err := k.initialForecasterEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialForecasterEmaScore = append(initialForecasterEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to walk forecaster initial EMA score per topic")
	}
	data.InitialForecasterEmaScore = initialForecasterEmaScore

	// initialReputerEmaScore
	var initialReputerEmaScore []*types.TopicIdAndDec
	if err := k.initialReputerEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialReputerEmaScore = append(initialReputerEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to walk reputer initial EMA score per topic")
	}
	data.InitialReputerEmaScore = initialReputerEmaScore

	// infererScoresByBlock
	infererScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	infererScoresByBlockIter, err := k.infererScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate inferer scores by block")
	}
	for ; infererScoresByBlockIter.Valid(); infererScoresByBlockIter.Next() {
		keyValue, err := infererScoresByBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: infererScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		infererScoresByBlock = append(infererScoresByBlock, &topicIdBlockHeightScores)
	}
	data.InfererScoresByBlock = infererScoresByBlock

	// forecasterScoresByBlock
	forecasterScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	forecasterScoresByBlockIter, err := k.forecasterScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate forecaster scores by block")
	}
	for ; forecasterScoresByBlockIter.Valid(); forecasterScoresByBlockIter.Next() {
		keyValue, err := forecasterScoresByBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: forecasterScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		forecasterScoresByBlock = append(forecasterScoresByBlock, &topicIdBlockHeightScores)
	}
	data.ForecasterScoresByBlock = forecasterScoresByBlock

	// reputerScoresByBlock
	reputerScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	reputerScoresByBlockIter, err := k.reputerScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate reputer scores by block")
	}
	for ; reputerScoresByBlockIter.Valid(); reputerScoresByBlockIter.Next() {
		keyValue, err := reputerScoresByBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: reputerScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		reputerScoresByBlock = append(reputerScoresByBlock, &topicIdBlockHeightScores)
	}
	data.ReputerScoresByBlock = reputerScoresByBlock

	// infererScoreEmas
	infererScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	infererScoreEmasIter, err := k.infererScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest inferer scores by worker")
	}
	for ; infererScoreEmasIter.Valid(); infererScoreEmasIter.Next() {
		keyValue, err := infererScoreEmasIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestInfererScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		infererScoreEmas = append(infererScoreEmas, &topicIdActorIdScore)
	}
	data.InfererScoreEmas = infererScoreEmas

	// forecasterScoreEmas
	forecasterScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	forecasterScoreEmaIter, err := k.forecasterScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest forecaster scores by worker")
	}
	for ; forecasterScoreEmaIter.Valid(); forecasterScoreEmaIter.Next() {
		keyValue, err := forecasterScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestForecasterScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		forecasterScoreEmas = append(forecasterScoreEmas, &topicIdActorIdScore)
	}
	data.ForecasterScoreEmas = forecasterScoreEmas

	// reputerScoreEmas
	reputerScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	reputerScoreEmasIter, err := k.reputerScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest reputer scores by reputer")
	}
	for ; reputerScoreEmasIter.Valid(); reputerScoreEmasIter.Next() {
		keyValue, err := reputerScoreEmasIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestReputerScoresByReputerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		reputerScoreEmas = append(reputerScoreEmas, &topicIdActorIdScore)
	}
	data.ReputerScoreEmas = reputerScoreEmas

	// reputerListeningCoefficient
	reputerListeningCoefficient := make([]*types.TopicIdActorIdListeningCoefficient, 0)
	reputerListeningCoefficientIter, err := k.reputerListeningCoefficient.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate reputer listening coefficient")
	}
	for ; reputerListeningCoefficientIter.Valid(); reputerListeningCoefficientIter.Next() {
		keyValue, err := reputerListeningCoefficientIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: reputerListeningCoefficientIter")
		}
		value := keyValue.Value
		topicIdActorIdListeningCoefficient := types.TopicIdActorIdListeningCoefficient{
			TopicId:              keyValue.Key.K1(),
			ActorId:              keyValue.Key.K2(),
			ListeningCoefficient: &value,
		}
		reputerListeningCoefficient = append(reputerListeningCoefficient, &topicIdActorIdListeningCoefficient)
	}
	data.ReputerListeningCoefficient = reputerListeningCoefficient

	// previousReputerRewardFraction
	previousReputerRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousReputerRewardFractionIter, err := k.previousReputerRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous reputer reward fraction")
	}
	for ; previousReputerRewardFractionIter.Valid(); previousReputerRewardFractionIter.Next() {
		keyValue, err := previousReputerRewardFractionIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousReputerRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousReputerRewardFraction = append(previousReputerRewardFraction, &topicIdActorIdDec)
	}
	data.PreviousReputerRewardFraction = previousReputerRewardFraction

	// previousInferenceRewardFraction
	previousInferenceRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousInferenceRewardFractionIter, err := k.previousInferenceRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous inference reward fraction")
	}
	for ; previousInferenceRewardFractionIter.Valid(); previousInferenceRewardFractionIter.Next() {
		keyValue, err := previousInferenceRewardFractionIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousInferenceRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousInferenceRewardFraction = append(previousInferenceRewardFraction, &topicIdActorIdDec)
	}
	data.PreviousInferenceRewardFraction = previousInferenceRewardFraction

	// previousForecastRewardFraction
	previousForecastRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousForecastRewardFractionIter, err := k.previousForecastRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous forecast reward fraction")
	}
	for ; previousForecastRewardFractionIter.Valid(); previousForecastRewardFractionIter.Next() {
		keyValue, err := previousForecastRewardFractionIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousForecastRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousForecastRewardFraction = append(previousForecastRewardFraction, &topicIdActorIdDec)
	}
	data.PreviousForecastRewardFraction = previousForecastRewardFraction

	// previousPercentageRewardToStakedReputers
	previousPercentageRewardToStakedReputers, err := k.previousPercentageRewardToStakedReputers.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get previous percentage reward to staked reputers")
	}
	data.PreviousPercentageRewardToStakedReputers = previousPercentageRewardToStakedReputers

	// previousForecasterScoreRatio
	previousForecasterScoreRatio := make([]*types.TopicIdAndDec, 0)
	previousForecasterScoreRatioIter, err := k.previousForecasterScoreRatio.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous forecaster score ratio")
	}
	for ; previousForecasterScoreRatioIter.Valid(); previousForecasterScoreRatioIter.Next() {
		keyValue, err := previousForecasterScoreRatioIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousForecasterScoreRatioIter")
		}
		previousForecasterScoreRatio = append(previousForecasterScoreRatio, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}
	data.PreviousForecasterScoreRatio = previousForecasterScoreRatio

	// previousTopicQuantileInfererScoreEma
	previousTopicQuantileInfererScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileInfererScoreEmaIter, err := k.previousTopicQuantileInfererScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous topic quantile inferer score ema")
	}
	for ; previousTopicQuantileInfererScoreEmaIter.Valid(); previousTopicQuantileInfererScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileInfererScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousTopicQuantileInfererScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileInfererScoreEma = append(previousTopicQuantileInfererScoreEma, &topicIdAndDec)
	}
	data.PreviousTopicQuantileInfererScoreEma = previousTopicQuantileInfererScoreEma

	// previousTopicQuantileForecasterScoreEma
	previousTopicQuantileForecasterScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileForecasterScoreEmaIter, err := k.previousTopicQuantileForecasterScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous topic quantile forecaster score ema")
	}
	for ; previousTopicQuantileForecasterScoreEmaIter.Valid(); previousTopicQuantileForecasterScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileForecasterScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousTopicQuantileForecasterScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileForecasterScoreEma = append(previousTopicQuantileForecasterScoreEma, &topicIdAndDec)
	}
	data.PreviousTopicQuantileForecasterScoreEma = previousTopicQuantileForecasterScoreEma

	// previousTopicQuantileReputerScoreEma
	previousTopicQuantileReputerScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileReputerScoreEmaIter, err := k.previousTopicQuantileReputerScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous topic quantile reputer score ema")
	}
	for ; previousTopicQuantileReputerScoreEmaIter.Valid(); previousTopicQuantileReputerScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileReputerScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousTopicQuantileReputerScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileReputerScoreEma = append(previousTopicQuantileReputerScoreEma, &topicIdAndDec)
	}
	data.PreviousTopicQuantileReputerScoreEma = previousTopicQuantileReputerScoreEma

	// lowestInfererScoreEma
	lowestInfererScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestInfererScoreEmaIter, err := k.lowestInfererScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate lowest inferer score emas")
	}
	for ; lowestInfererScoreEmaIter.Valid(); lowestInfererScoreEmaIter.Next() {
		keyValue, err := lowestInfererScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: lowestInfererScoreEmaIter")
		}
		lowestInfererScoreEma = append(lowestInfererScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}
	data.LowestInfererScoreEma = lowestInfererScoreEma

	// lowestForecasterScoreEma
	lowestForecasterScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestForecasterScoreEmaIter, err := k.lowestForecasterScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate lowest forecaster score emas")
	}
	for ; lowestForecasterScoreEmaIter.Valid(); lowestForecasterScoreEmaIter.Next() {
		keyValue, err := lowestForecasterScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: lowestForecasterScoreEmaIter")
		}
		lowestForecasterScoreEma = append(lowestForecasterScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}
	data.LowestForecasterScoreEma = lowestForecasterScoreEma

	// lowestReputerScoreEma
	lowestReputerScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestReputerScoreEmaIter, err := k.lowestReputerScoreEma.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate lowest reputer score emas")
	}
	for ; lowestReputerScoreEmaIter.Valid(); lowestReputerScoreEmaIter.Next() {
		keyValue, err := lowestReputerScoreEmaIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: lowestReputerScoreEmaIter")
		}
		lowestReputerScoreEma = append(lowestReputerScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}
	data.LowestReputerScoreEma = lowestReputerScoreEma

	return nil
}

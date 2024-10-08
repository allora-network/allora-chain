package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Get lowest score from all reputers
func GetLowestScoreFromAllReputers(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	reputerAddresses []string,
) (lowScore types.Score, err error) {
	for i, address := range reputerAddresses {
		score, err := k.GetReputerScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return lowScore, nil
}

// Update lowest score from new reputer addresses set
func UpdateLowestScoreFromReputerAddresses(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	reputerAddresses []string,
	addedReputer string,
	removedReputerAddress string,
) error {
	reputerAddresses = append(reputerAddresses, addedReputer)
	lowScore := types.Score{}
	for i, address := range reputerAddresses {
		if address == removedReputerAddress {
			continue
		}
		score, err := k.GetReputerScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return k.SetLowestReputerScoreEma(ctx, topicId, lowScore)
}

// Update lowest score from new inferer addresses set
func UpdateLowestScoreFromInfererAddresses(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	infererAddresses []string,
	addedInferer string,
	removedInfererAddress string,
) error {
	infererAddresses = append(infererAddresses, addedInferer)
	lowScore := types.Score{}
	for i, address := range infererAddresses {
		if address == removedInfererAddress {
			continue
		}
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return k.SetLowestInfererScoreEma(ctx, topicId, lowScore)
}

// Get lowest score from all inferers
func GetLowestScoreFromAllInferers(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	infererAddresses []string,
) (lowScore types.Score, err error) {
	for i, address := range infererAddresses {
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return lowScore, nil
}

// Update lowest score from new forecaster addresses set
func UpdateLowestScoreFromForecasterAddresses(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	forecasterAddresses []string,
	addedForecaster string,
	removedForecasterAddress string,
) error {
	forecasterAddresses = append(forecasterAddresses, addedForecaster)
	lowScore := types.Score{}
	for i, address := range forecasterAddresses {
		if address == removedForecasterAddress {
			continue
		}
		score, err := k.GetForecasterScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return k.SetLowestForecasterScoreEma(ctx, topicId, lowScore)
}

// Get lowest score from all forecasters
func GetLowestScoreFromAllForecasters(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	forecasterAddresses []string,
) (lowScore types.Score, err error) {
	for i, address := range forecasterAddresses {
		score, err := k.GetForecasterScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return lowScore, nil
}

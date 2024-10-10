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
	// remove reputer from the list
	for i, address := range reputerAddresses {
		if address == removedReputerAddress {
			reputerAddresses = append(reputerAddresses[:i], reputerAddresses[i+1:]...)
			break
		}
	}
	// add new reputer to the list
	reputerAddresses = append(reputerAddresses, addedReputer)
	lowScore, err := k.GetReputerScoreEma(ctx, topicId, reputerAddresses[0])
	if err != nil {
		return err
	}
	for _, address := range reputerAddresses {
		score, err := k.GetReputerScoreEma(ctx, topicId, address)
		if err != nil {
			return err
		}
		if lowScore.Score.Gt(score.Score) {
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
	// remove inferer from the list
	for i, address := range infererAddresses {
		if address == removedInfererAddress {
			infererAddresses = append(infererAddresses[:i], infererAddresses[i+1:]...)
			break
		}
	}
	// add new inferer to the list
	infererAddresses = append(infererAddresses, addedInferer)
	lowScore, err := k.GetInfererScoreEma(ctx, topicId, infererAddresses[0])
	if err != nil {
		return err
	}
	for _, address := range infererAddresses {
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			return err
		}
		if lowScore.Score.Gt(score.Score) {
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
	// remove forecaster from the list
	for i, address := range forecasterAddresses {
		if address == removedForecasterAddress {
			forecasterAddresses = append(forecasterAddresses[:i], forecasterAddresses[i+1:]...)
			break
		}
	}
	// add new forecaster to the list
	forecasterAddresses = append(forecasterAddresses, addedForecaster)
	lowScore, err := k.GetForecasterScoreEma(ctx, topicId, forecasterAddresses[0])
	if err != nil {
		return err
	}
	for _, address := range forecasterAddresses {
		score, err := k.GetForecasterScoreEma(ctx, topicId, address)
		if err != nil {
			return err
		}
		if lowScore.Score.Gt(score.Score) {
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

package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Get lowest score from all reputers
func (k *ScoresKeeper) GetLowestScoreFromAllReputers(
	ctx context.Context,
	topicId TopicId,
	reputerAddresses []string,
) (lowScore types.Score, err error) {
	foundValidScore := false
	for _, address := range reputerAddresses {
		score, err := k.GetReputerScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if !foundValidScore || lowScore.Score.Gt(score.Score) {
			lowScore = score
			foundValidScore = true
		}
	}
	if !foundValidScore {
		return types.Score{}, types.ErrNotFound
	}
	return lowScore, nil
}

// Update lowest score from new reputer addresses set
func (k *ScoresKeeper) UpdateLowestScoreFromReputerAddresses(
	ctx context.Context,
	topicId TopicId,
	reputerAddresses []string,
	addedReputer string,
	removedReputerAddress string,
) error {
	// Remove reputer from the list
	for i, address := range reputerAddresses {
		if address == removedReputerAddress {
			reputerAddresses = append(reputerAddresses[:i], reputerAddresses[i+1:]...)
			break
		}
	}
	// Add new reputer to the list
	reputerAddresses = append(reputerAddresses, addedReputer)

	// Get lowest score from all reputers
	lowScore, err := k.GetLowestScoreFromAllReputers(ctx, topicId, reputerAddresses)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil
		}
		return err
	}

	return k.SetLowestReputerScoreEma(ctx, topicId, lowScore)
}

// Update lowest score from new inferer addresses set
func (k *ScoresKeeper) UpdateLowestScoreFromInfererAddresses(
	ctx context.Context,
	topicId TopicId,
	infererAddresses []string,
	addedInferer string,
	removedInfererAddress string,
) error {
	// Remove inferer from the list
	for i, address := range infererAddresses {
		if address == removedInfererAddress {
			infererAddresses = append(infererAddresses[:i], infererAddresses[i+1:]...)
			break
		}
	}
	// Add new inferer to the list
	infererAddresses = append(infererAddresses, addedInferer)

	// Get lowest score from all inferers
	lowScore, err := k.GetLowestScoreFromAllInferers(ctx, topicId, infererAddresses)
	if err != nil {
		return err
	}

	return k.SetLowestInfererScoreEma(ctx, topicId, lowScore)
}

// Get lowest score from all inferers
func (k *ScoresKeeper) GetLowestScoreFromAllInferers(
	ctx context.Context,
	topicId TopicId,
	infererAddresses []string,
) (lowScore types.Score, err error) {
	foundValidScore := false
	for _, address := range infererAddresses {
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if !foundValidScore || lowScore.Score.Gt(score.Score) {
			lowScore = score
			foundValidScore = true
		}
	}
	if !foundValidScore {
		return types.Score{}, types.ErrNotFound
	}
	return lowScore, nil
}

// Update lowest score from new forecaster addresses set
func (k *ScoresKeeper) UpdateLowestScoreFromForecasterAddresses(
	ctx context.Context,
	topicId TopicId,
	forecasterAddresses []string,
	addedForecaster string,
	removedForecasterAddress string,
) error {
	// Remove forecaster from the list
	for i, address := range forecasterAddresses {
		if address == removedForecasterAddress {
			forecasterAddresses = append(forecasterAddresses[:i], forecasterAddresses[i+1:]...)
			break
		}
	}
	// Add new forecaster to the list
	forecasterAddresses = append(forecasterAddresses, addedForecaster)

	// Get lowest score from all forecasters
	lowScore, err := k.GetLowestScoreFromAllForecasters(ctx, topicId, forecasterAddresses)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil
		}
		return err
	}

	return k.SetLowestForecasterScoreEma(ctx, topicId, lowScore)
}

// Get lowest score from all forecasters
func (k *ScoresKeeper) GetLowestScoreFromAllForecasters(
	ctx context.Context,
	topicId TopicId,
	forecasterAddresses []string,
) (lowScore types.Score, err error) {
	foundValidScore := false
	for _, address := range forecasterAddresses {
		score, err := k.GetForecasterScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if !foundValidScore || lowScore.Score.Gt(score.Score) {
			lowScore = score
			foundValidScore = true
		}
	}
	if !foundValidScore {
		return types.Score{}, types.ErrNotFound
	}
	return lowScore, nil
}

// SetLowestReputerScoreEma sets the lowest reputer score EMA for a topic
func (k *ScoresKeeper) SetLowestReputerScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestReputerScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestReputerScoreEma gets the lowest reputer score EMA for a topic
func (k *ScoresKeeper) GetLowestReputerScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestReputerScoreEma.Get(ctx, topicId)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, nil
	} else if err != nil {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, errorsmod.Wrap(err, "error getting lowest reputer score EMA")
	}
	return lowestScore, true, nil
}

// SetLowestInfererScoreEma sets the lowest inferer score EMA for a topic
func (k *ScoresKeeper) SetLowestInfererScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestInfererScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestInfererScoreEma gets the lowest inferer score EMA for a topic
func (k *ScoresKeeper) GetLowestInfererScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestInfererScoreEma.Get(ctx, topicId)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, nil
	} else if err != nil {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, errorsmod.Wrap(err, "error getting lowest inferer score EMA")
	}
	return lowestScore, true, nil
}

// SetLowestForecasterScoreEma sets the lowest forecaster score EMA for a topic
func (k *ScoresKeeper) SetLowestForecasterScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestForecasterScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestForecasterScoreEma gets the lowest forecaster score EMA for a topic
func (k *ScoresKeeper) GetLowestForecasterScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestForecasterScoreEma.Get(ctx, topicId)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, nil
	} else if err != nil {
		return types.Score{
			BlockHeight: 0,
			Address:     "",
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, false, errorsmod.Wrap(err, "error getting lowest forecaster score EMA")
	}
	return lowestScore, true, nil
}

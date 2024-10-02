package inferencesynthesis

import (
	"context"
	"sort"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type networkLossesByWorker struct {
	CombinedLoss                  Loss
	InfererLosses                 map[Worker]Loss
	ForecasterLosses              map[Worker]Loss
	NaiveLoss                     Loss
	OneOutInfererForecasterLosses map[Worker]map[Worker]Loss
	OneOutInfererLosses           map[Worker]Loss
	OneOutForecasterLosses        map[Worker]Loss
	OneInForecasterLosses         map[Worker]Loss
}

// Convert a ValueBundle to a networkLossesByWorker
func ConvertValueBundleToNetworkLossesByWorker(
	valueBundle emissions.ValueBundle,
) networkLossesByWorker {
	infererLosses := make(map[Worker]Loss)
	for _, inferer := range valueBundle.InfererValues {
		infererLosses[inferer.Worker] = inferer.Value
	}

	forecasterLosses := make(map[Worker]Loss)
	for _, forecaster := range valueBundle.ForecasterValues {
		forecasterLosses[forecaster.Worker] = forecaster.Value
	}

	oneOutInfererForecasterLosses := make(map[Worker]map[Worker]Loss)
	for _, oneOutInfererForecaster := range valueBundle.OneOutInfererForecasterValues {
		if _, ok := oneOutInfererForecasterLosses[oneOutInfererForecaster.Forecaster]; !ok {
			oneOutInfererForecasterLosses[oneOutInfererForecaster.Forecaster] = make(map[Worker]Loss)
		}
		for _, infererLoss := range oneOutInfererForecaster.OneOutInfererValues {
			oneOutInfererForecasterLosses[oneOutInfererForecaster.Forecaster][infererLoss.Worker] = infererLoss.Value
		}
	}

	oneOutInfererLosses := make(map[Worker]Loss)
	for _, oneOutInferer := range valueBundle.OneOutInfererValues {
		oneOutInfererLosses[oneOutInferer.Worker] = oneOutInferer.Value
	}

	oneOutForecasterLosses := make(map[Worker]Loss)
	for _, oneOutForecaster := range valueBundle.OneOutForecasterValues {
		oneOutForecasterLosses[oneOutForecaster.Worker] = oneOutForecaster.Value
	}

	oneInForecasterLosses := make(map[Worker]Loss)
	for _, oneInForecaster := range valueBundle.OneInForecasterValues {
		oneInForecasterLosses[oneInForecaster.Worker] = oneInForecaster.Value
	}

	return networkLossesByWorker{
		CombinedLoss:                  valueBundle.CombinedValue,
		InfererLosses:                 infererLosses,
		ForecasterLosses:              forecasterLosses,
		NaiveLoss:                     valueBundle.NaiveValue,
		OneOutInfererForecasterLosses: oneOutInfererForecasterLosses,
		OneOutInfererLosses:           oneOutInfererLosses,
		OneOutForecasterLosses:        oneOutForecasterLosses,
		OneInForecasterLosses:         oneInForecasterLosses,
	}
}

func ComputeAndBuildEMRegret(
	lossA Loss,
	lossB Loss,
	previousRegret Regret,
	alpha alloraMath.Dec,
	blockHeight BlockHeight,
) (emissions.TimestampedValue, error) {
	lossDiff, err := lossA.Sub(lossB)
	if err != nil {
		return emissions.TimestampedValue{}, err
	}

	newRegret, err := alloraMath.CalcEma(alpha, lossDiff, previousRegret, false)
	if err != nil {
		return emissions.TimestampedValue{}, err
	}
	return emissions.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       newRegret,
	}, nil
}

// Calculate the new network regrets by taking EMAs between the previous network regrets
// and the new regrets admitted by the inputted network losses
// It is assumed the workers are uniquely represented in the network losses
func GetCalcSetNetworkRegrets(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	networkLosses emissions.ValueBundle,
	nonce emissions.Nonce,
	alpha alloraMath.Dec,
	cNorm alloraMath.Dec,
	pNorm alloraMath.Dec,
	epsilon alloraMath.Dec,
) error {
	// Convert the network losses to a networkLossesByWorker
	networkLossesByWorker := ConvertValueBundleToNetworkLossesByWorker(networkLosses)
	blockHeight := nonce.BlockHeight

	workersRegrets := make([]alloraMath.Dec, 0)

	sort.Slice(networkLosses.InfererValues, func(i, j int) bool {
		return networkLosses.InfererValues[i].Worker < networkLosses.InfererValues[j].Worker
	})

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to get topic")
	}

	// R_ij - Inferer Regrets
	for _, infererLoss := range networkLosses.InfererValues {
		lastRegret, newParticipant, err := k.GetInfererNetworkRegret(ctx, topicId, infererLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get inferer regret")
		}
		newInfererRegret, err := ComputeAndBuildEMRegret(
			networkLosses.CombinedValue,                             // L_i
			networkLossesByWorker.InfererLosses[infererLoss.Worker], // L_ij
			lastRegret.Value,
			alpha,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building inferer regret")
		}
		err = k.SetInfererNetworkRegret(ctx, topicId, infererLoss.Worker, newInfererRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting inferer regret")
		}

		shouldAddWorkerRegret, err := isExperiencedInferer(
			ctx,
			k,
			topic,
			infererLoss.Worker,
			newParticipant,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error checking if should add worker regret")
		}
		if shouldAddWorkerRegret {
			workersRegrets = append(workersRegrets, newInfererRegret.Value)
		}
	}

	sort.Slice(networkLosses.ForecasterValues, func(i, j int) bool {
		return networkLosses.ForecasterValues[i].Worker < networkLosses.ForecasterValues[j].Worker
	})

	// R_ik - Forecaster Regrets
	for _, forecasterLoss := range networkLosses.ForecasterValues {
		lastRegret, newParticipant, err := k.GetForecasterNetworkRegret(ctx, topicId, forecasterLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting forecaster regret")
		}
		newForecasterRegret, err := ComputeAndBuildEMRegret(
			networkLosses.CombinedValue,                                   // L_i
			networkLossesByWorker.ForecasterLosses[forecasterLoss.Worker], // L_ik
			lastRegret.Value,
			alpha,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building forecaster regret")
		}
		err = k.SetForecasterNetworkRegret(ctx, topicId, forecasterLoss.Worker, newForecasterRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting forecaster regret")
		}

		shouldAddWorkerRegret, err := isExperiencedForecaster(
			ctx,
			k,
			topic,
			forecasterLoss.Worker,
			newParticipant,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error checking if should add worker regret")
		}
		if shouldAddWorkerRegret {
			workersRegrets = append(workersRegrets, newForecasterRegret.Value)
		}
	}

	// R^-_ij - Naive Regrets
	for _, infererLoss := range networkLosses.InfererValues {
		lastRegret, _, err := k.GetNaiveInfererNetworkRegret(ctx, topicId, infererLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get inferer regret")
		}
		newInfererRegret, err := ComputeAndBuildEMRegret(
			networkLosses.NaiveValue,                                // L^-_i
			networkLossesByWorker.InfererLosses[infererLoss.Worker], // L_ij
			lastRegret.Value,
			alpha,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building inferer regret")
		}
		err = k.SetNaiveInfererNetworkRegret(ctx, topicId, infererLoss.Worker, newInfererRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting inferer regret")
		}
	}

	sort.Slice(networkLosses.OneOutInfererValues, func(i, j int) bool {
		return networkLosses.OneOutInfererValues[i].Worker < networkLosses.OneOutInfererValues[j].Worker
	})

	// R^-j′ij - One-out inferer inferer regrets
	for _, oneOutInfererLoss := range networkLosses.OneOutInfererValues {
		for _, infererLoss := range networkLosses.InfererValues {
			lastRegret, _, err := k.GetOneOutInfererInfererNetworkRegret(ctx, topicId, oneOutInfererLoss.Worker, infererLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out inferer inferer regret")
			}
			newOneOutInfererInfererRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutInfererLosses[oneOutInfererLoss.Worker], // L^-_j'i
				networkLossesByWorker.InfererLosses[infererLoss.Worker],             // L_ij
				lastRegret.Value,
				alpha,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out inferer regret")
			}
			err = k.SetOneOutInfererInfererNetworkRegret(ctx, topicId, oneOutInfererLoss.Worker, infererLoss.Worker, newOneOutInfererInfererRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out inferer inferer regret")
			}
		}
	}

	// R^-j′ik - One-out inferer forecaster regrets
	for _, oneOutInfererLoss := range networkLosses.OneOutInfererValues {
		for _, oneOutInfererForecasterLoss := range networkLosses.OneOutInfererForecasterValues {
			lastRegret, _, err := k.GetOneOutInfererForecasterNetworkRegret(ctx, topicId, oneOutInfererLoss.Worker, oneOutInfererForecasterLoss.Forecaster)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out inferer forecaster regret")
			}
			newOneOutInfererForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutInfererLosses[oneOutInfererLoss.Worker],                                                   // L^-_j'i
				networkLossesByWorker.OneOutInfererForecasterLosses[oneOutInfererForecasterLoss.Forecaster][oneOutInfererLoss.Worker], // L^-_j'ik
				lastRegret.Value,
				alpha,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out inferer forecaster regret")
			}
			err = k.SetOneOutInfererForecasterNetworkRegret(ctx, topicId, oneOutInfererLoss.Worker, oneOutInfererForecasterLoss.Forecaster, newOneOutInfererForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out inferer forecaster regret")
			}
		}
	}

	sort.Slice(networkLosses.OneOutForecasterValues, func(i, j int) bool {
		return networkLosses.OneOutForecasterValues[i].Worker < networkLosses.OneOutForecasterValues[j].Worker
	})

	// R^-k′ij - One-out forecaster inferer regrets
	for _, oneOutForecasterLoss := range networkLosses.OneOutForecasterValues {
		for _, infererloss := range networkLosses.InfererValues {
			lastRegret, _, err := k.GetOneOutForecasterInfererNetworkRegret(ctx, topicId, oneOutForecasterLoss.Worker, infererloss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out forecaster inferer regret")
			}
			newOneOutForecasterInfererRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutForecasterLosses[oneOutForecasterLoss.Worker], // L^-_k'i
				networkLossesByWorker.InfererLosses[infererloss.Worker],                   // L_ij
				lastRegret.Value,
				alpha,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out forecaster inferer regret")
			}
			err = k.SetOneOutForecasterInfererNetworkRegret(ctx, topicId, oneOutForecasterLoss.Worker, infererloss.Worker, newOneOutForecasterInfererRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out forecaster inferer regret")
			}
		}
	}

	// R^-k′ik - One-out forecaster forecaster regrets
	for _, oneOutForecasterLoss := range networkLosses.OneOutForecasterValues {
		for _, forecasterLoss := range networkLosses.ForecasterValues {
			lastRegret, _, err := k.GetOneOutForecasterForecasterNetworkRegret(ctx, topicId, oneOutForecasterLoss.Worker, forecasterLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out forecaster forecaster regret")
			}
			newOneOutForecasterForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutForecasterLosses[oneOutForecasterLoss.Worker], // L^-_k'i
				networkLossesByWorker.ForecasterLosses[forecasterLoss.Worker],             // L_ik
				lastRegret.Value,
				alpha,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out forecaster forecaster regret")
			}
			err = k.SetOneOutForecasterForecasterNetworkRegret(ctx, topicId, oneOutForecasterLoss.Worker, forecasterLoss.Worker, newOneOutForecasterForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out forecaster forecaster regret")
			}
		}
	}

	sort.Slice(networkLosses.OneInForecasterValues, func(i, j int) bool {
		return networkLosses.OneInForecasterValues[i].Worker < networkLosses.OneInForecasterValues[j].Worker
	})

	// R^+_k'ij - One-in forecaster regrets
	for _, oneInForecasterLoss := range networkLosses.OneInForecasterValues {
		// Loop over the inferer losses so that their losses may be compared against the one-in forecaster's loss, for each forecaster
		for _, infererLoss := range networkLosses.InfererValues {
			lastRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, oneInForecasterLoss.Worker, infererLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
			}
			newOneInForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], // L^+_k'i
				networkLossesByWorker.InfererLosses[infererLoss.Worker],                 // L_ij
				lastRegret.Value,
				alpha,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-in forecaster regret")
			}
			err = k.SetOneInForecasterNetworkRegret(ctx, topicId, oneInForecasterLoss.Worker, infererLoss.Worker, newOneInForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-in forecaster regret")
			}
		}

		lastRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, oneInForecasterLoss.Worker, oneInForecasterLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}
		newOneInForecasterRegret, err := ComputeAndBuildEMRegret(
			networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], // L^+_k'i
			networkLossesByWorker.ForecasterLosses[oneInForecasterLoss.Worker],      // L_ik'
			lastRegret.Value,
			alpha,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building one-in forecaster regret")
		}
		err = k.SetOneInForecasterNetworkRegret(ctx, topicId, oneInForecasterLoss.Worker, oneInForecasterLoss.Worker, newOneInForecasterRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting one-in forecaster regret")
		}
	}

	// Recalculate topic initial regret
	if len(workersRegrets) > 0 {
		params, err := k.GetParams(ctx)
		if err != nil {
			return errorsmod.Wrapf(err, "GetCalcSetNetworkRegrets error getting params")
		}
		updatedTopicInitialRegret, err := CalcTopicInitialRegret(workersRegrets, epsilon, pNorm, cNorm, params.InitialRegretQuantile, params.PnormSafeDiv)
		if err != nil {
			return errorsmod.Wrapf(err, "Error calculating topic initial regret")
		}
		err = k.UpdateTopicInitialRegret(ctx, topicId, updatedTopicInitialRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error updating topic initial regret")
		}
	}

	return nil
}

// Calculate the initial regret for all new workers in the topic
// denominator = std(regrets[i-1, :]) + epsilon
// offset = cnorm - 8.25 / pnorm
// dummy_regret[i] = np.percentile(regrets[i-1, :], 25.) + offset * denominator
// It is assumed that the regrets are filtered by experience for each actor
// i.e. if they have not been included in the topic for enough epochs, their regret is ignored.
func CalcTopicInitialRegret(
	regrets []alloraMath.Dec,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	percentileRegret alloraMath.Dec,
	pNormDiv alloraMath.Dec,
) (alloraMath.Dec, error) {
	// Calculate the Denominator
	stdDevRegrets, err := alloraMath.StdDev(regrets)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	denominator, err := stdDevRegrets.Add(epsilon)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	// calculate the offset
	eightPointTwoFiveDividedByPnorm, err := pNormDiv.Quo(pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	offset, err := cNorm.Sub(eightPointTwoFiveDividedByPnorm)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	// calculate the dummy regret
	offSetTimesDenominator, err := offset.Mul(denominator)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	// Calculate percentile
	percentile, err := alloraMath.GetQuantileOfDecs(regrets, percentileRegret)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	initialRegret, err := percentile.Add(offSetTimesDenominator)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	return initialRegret, nil
}

// Determine if a inferer is an experienced worker by checking
// the inclusions count is greater than 1/alpha_regret
func isExperiencedInferer(
	ctx context.Context,
	k keeper.Keeper,
	topic emissions.Topic,
	inferer Worker,
	newParticipant bool,
) (bool, error) {
	numInclusions := uint64(0)
	numInclusions, err := k.GetCountInfererInclusionsInTopic(ctx, topic.Id, inferer)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get inferer inclusions")
	}
	return isExperiencedActor(newParticipant, numInclusions, topic.AlphaRegret)
}

// Determine if a forecaster is an experienced worker by checking
// the inclusions count is greater than 1/alpha_regret
func isExperiencedForecaster(
	ctx context.Context,
	k keeper.Keeper,
	topic emissions.Topic,
	forecaster Worker,
	newParticipant bool,
) (bool, error) {
	numInclusions := uint64(0)
	numInclusions, err := k.GetCountForecasterInclusionsInTopic(ctx, topic.Id, forecaster)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get forecaster inclusions")
	}
	return isExperiencedActor(newParticipant, numInclusions, topic.AlphaRegret)
}

// helper function for isExperiencedForecaster and isExperiencedInferer
// check if inclusions count is greater than 1/alpha_regret
func isExperiencedActor(newParticipant bool, numInclusions uint64, alpha alloraMath.Dec) (bool, error) {
	numInclusionsDec, err := alloraMath.NewDecFromUint64(numInclusions)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get num inclusions dec")
	}
	oneOverAlpha, err := alloraMath.OneDec().Quo(alpha)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get one over alpha")
	}
	return !newParticipant && numInclusionsDec.Gte(oneOverAlpha), nil
}

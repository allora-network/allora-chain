package inferencesynthesis

import (
	"context"

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

// args for GetCalcSetNetworkRegrets
type GetCalcSetNetworkRegretsArgs struct {
	Ctx                   sdk.Context
	K                     keeper.Keeper
	TopicId               TopicId
	NetworkLosses         emissions.ValueBundle
	Nonce                 emissions.Nonce
	AlphaRegret           alloraMath.Dec
	CNorm                 alloraMath.Dec
	PNorm                 alloraMath.Dec
	EpsilonTopic          alloraMath.Dec
	InitialRegretQuantile alloraMath.Dec
	PnormSafeDiv          alloraMath.Dec
}

// Calculate the new network regrets by taking EMAs between the previous network regrets
// and the new regrets admitted by the inputted network losses
// NOTE: It is assumed the workers are uniquely represented in the network losses
// NOTE: It is assumed network losses are sorted (done in synth.CalcNetworkLosses())
func GetCalcSetNetworkRegrets(args GetCalcSetNetworkRegretsArgs) error {
	// Convert the network losses to a networkLossesByWorker
	networkLossesByWorker := ConvertValueBundleToNetworkLossesByWorker(args.NetworkLosses)
	blockHeight := args.Nonce.BlockHeight

	workersRegrets := make([]alloraMath.Dec, 0)

	// R_ij - Inferer Regrets
	for _, infererLoss := range args.NetworkLosses.InfererValues {
		lastRegret, newParticipant, err := args.K.GetInfererNetworkRegret(args.Ctx, args.TopicId, infererLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get inferer regret")
		}
		newInfererRegret, err := ComputeAndBuildEMRegret(
			args.NetworkLosses.CombinedValue,                        // L_i
			networkLossesByWorker.InfererLosses[infererLoss.Worker], // L_ij
			lastRegret.Value,
			args.AlphaRegret,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building inferer regret")
		}
		err = args.K.SetInfererNetworkRegret(args.Ctx, args.TopicId, infererLoss.Worker, newInfererRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting inferer regret")
		}

		shouldAddWorkerRegret, err := isExperiencedInferer(
			args.Ctx,
			args.K,
			args.TopicId,
			args.AlphaRegret,
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

	// R_ik - Forecaster Regrets
	for _, forecasterLoss := range args.NetworkLosses.ForecasterValues {
		lastRegret, newParticipant, err := args.K.GetForecasterNetworkRegret(args.Ctx, args.TopicId, forecasterLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting forecaster regret")
		}
		newForecasterRegret, err := ComputeAndBuildEMRegret(
			args.NetworkLosses.CombinedValue,                              // L_i
			networkLossesByWorker.ForecasterLosses[forecasterLoss.Worker], // L_ik
			lastRegret.Value,
			args.AlphaRegret,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building forecaster regret")
		}
		err = args.K.SetForecasterNetworkRegret(args.Ctx, args.TopicId, forecasterLoss.Worker, newForecasterRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting forecaster regret")
		}

		shouldAddWorkerRegret, err := isExperiencedForecaster(
			args.Ctx,
			args.K,
			args.TopicId,
			args.AlphaRegret,
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
	for _, infererLoss := range args.NetworkLosses.InfererValues {
		lastRegret, _, err := args.K.GetNaiveInfererNetworkRegret(args.Ctx, args.TopicId, infererLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get inferer regret")
		}
		newInfererRegret, err := ComputeAndBuildEMRegret(
			args.NetworkLosses.NaiveValue,                           // L^-_i
			networkLossesByWorker.InfererLosses[infererLoss.Worker], // L_ij
			lastRegret.Value,
			args.AlphaRegret,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building inferer regret")
		}
		err = args.K.SetNaiveInfererNetworkRegret(args.Ctx, args.TopicId, infererLoss.Worker, newInfererRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting inferer regret")
		}
	}

	// R^-j′ij - One-out inferer inferer regrets
	for _, oneOutInfererLoss := range args.NetworkLosses.OneOutInfererValues {
		for _, infererLoss := range args.NetworkLosses.InfererValues {
			lastRegret, _, err := args.K.GetOneOutInfererInfererNetworkRegret(args.Ctx, args.TopicId, oneOutInfererLoss.Worker, infererLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out inferer inferer regret")
			}
			newOneOutInfererInfererRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutInfererLosses[oneOutInfererLoss.Worker], // L^-_j'i
				networkLossesByWorker.InfererLosses[infererLoss.Worker],             // L_ij
				lastRegret.Value,
				args.AlphaRegret,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out inferer regret")
			}
			err = args.K.SetOneOutInfererInfererNetworkRegret(args.Ctx, args.TopicId, oneOutInfererLoss.Worker, infererLoss.Worker, newOneOutInfererInfererRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out inferer inferer regret")
			}
		}
	}

	// R^-j′ik - One-out inferer forecaster regrets
	for _, oneOutInfererLoss := range args.NetworkLosses.OneOutInfererValues {
		for _, oneOutInfererForecasterLoss := range args.NetworkLosses.OneOutInfererForecasterValues {
			lastRegret, _, err := args.K.GetOneOutInfererForecasterNetworkRegret(args.Ctx, args.TopicId, oneOutInfererLoss.Worker, oneOutInfererForecasterLoss.Forecaster)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out inferer forecaster regret")
			}
			newOneOutInfererForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutInfererLosses[oneOutInfererLoss.Worker],                                                   // L^-_j'i
				networkLossesByWorker.OneOutInfererForecasterLosses[oneOutInfererForecasterLoss.Forecaster][oneOutInfererLoss.Worker], // L^-_j'ik
				lastRegret.Value,
				args.AlphaRegret,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out inferer forecaster regret")
			}
			err = args.K.SetOneOutInfererForecasterNetworkRegret(args.Ctx, args.TopicId, oneOutInfererLoss.Worker, oneOutInfererForecasterLoss.Forecaster, newOneOutInfererForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out inferer forecaster regret")
			}
		}
	}

	// R^-k′ij - One-out forecaster inferer regrets
	for _, oneOutForecasterLoss := range args.NetworkLosses.OneOutForecasterValues {
		for _, infererloss := range args.NetworkLosses.InfererValues {
			lastRegret, _, err := args.K.GetOneOutForecasterInfererNetworkRegret(args.Ctx, args.TopicId, oneOutForecasterLoss.Worker, infererloss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out forecaster inferer regret")
			}
			newOneOutForecasterInfererRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutForecasterLosses[oneOutForecasterLoss.Worker], // L^-_k'i
				networkLossesByWorker.InfererLosses[infererloss.Worker],                   // L_ij
				lastRegret.Value,
				args.AlphaRegret,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out forecaster inferer regret")
			}
			err = args.K.SetOneOutForecasterInfererNetworkRegret(args.Ctx, args.TopicId, oneOutForecasterLoss.Worker, infererloss.Worker, newOneOutForecasterInfererRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out forecaster inferer regret")
			}
		}
	}

	// R^-k′ik - One-out forecaster forecaster regrets
	for _, oneOutForecasterLoss := range args.NetworkLosses.OneOutForecasterValues {
		for _, forecasterLoss := range args.NetworkLosses.ForecasterValues {
			lastRegret, _, err := args.K.GetOneOutForecasterForecasterNetworkRegret(args.Ctx, args.TopicId, oneOutForecasterLoss.Worker, forecasterLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-out forecaster forecaster regret")
			}
			newOneOutForecasterForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneOutForecasterLosses[oneOutForecasterLoss.Worker], // L^-_k'i
				networkLossesByWorker.ForecasterLosses[forecasterLoss.Worker],             // L_ik
				lastRegret.Value,
				args.AlphaRegret,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-out forecaster forecaster regret")
			}
			err = args.K.SetOneOutForecasterForecasterNetworkRegret(args.Ctx, args.TopicId, oneOutForecasterLoss.Worker, forecasterLoss.Worker, newOneOutForecasterForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-out forecaster forecaster regret")
			}
		}
	}

	// R^+_k'ij - One-in forecaster regrets
	for _, oneInForecasterLoss := range args.NetworkLosses.OneInForecasterValues {
		// Loop over the inferer losses so that their losses may be compared against the one-in forecaster's loss, for each forecaster
		for _, infererLoss := range args.NetworkLosses.InfererValues {
			lastRegret, _, err := args.K.GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, oneInForecasterLoss.Worker, infererLoss.Worker)
			if err != nil {
				return errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
			}
			newOneInForecasterRegret, err := ComputeAndBuildEMRegret(
				networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], // L^+_k'i
				networkLossesByWorker.InfererLosses[infererLoss.Worker],                 // L_ij
				lastRegret.Value,
				args.AlphaRegret,
				blockHeight,
			)
			if err != nil {
				return errorsmod.Wrapf(err, "Error computing and building one-in forecaster regret")
			}
			err = args.K.SetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, oneInForecasterLoss.Worker, infererLoss.Worker, newOneInForecasterRegret)
			if err != nil {
				return errorsmod.Wrapf(err, "Error setting one-in forecaster regret")
			}
		}

		lastRegret, _, err := args.K.GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, oneInForecasterLoss.Worker, oneInForecasterLoss.Worker)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}
		newOneInForecasterRegret, err := ComputeAndBuildEMRegret(
			networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], // L^+_k'i
			networkLossesByWorker.ForecasterLosses[oneInForecasterLoss.Worker],      // L_ik'
			lastRegret.Value,
			args.AlphaRegret,
			blockHeight,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error computing and building one-in forecaster regret")
		}
		err = args.K.SetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, oneInForecasterLoss.Worker, oneInForecasterLoss.Worker, newOneInForecasterRegret)
		if err != nil {
			return errorsmod.Wrapf(err, "Error setting one-in forecaster regret")
		}
	}

	// Recalculate topic initial regret
	if len(workersRegrets) > 0 {
		updatedTopicInitialRegret, err := CalcTopicInitialRegret(
			workersRegrets,
			args.EpsilonTopic,
			args.PNorm,
			args.CNorm,
			args.InitialRegretQuantile,
			args.PnormSafeDiv,
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Error calculating topic initial regret")
		}
		err = args.K.UpdateTopicInitialRegret(args.Ctx, args.TopicId, updatedTopicInitialRegret)
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
	quantileRegret alloraMath.Dec,
	pNormDiv alloraMath.Dec,
) (initialRegret alloraMath.Dec, err error) {
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

	// Calculate quantile
	quantile, err := alloraMath.GetQuantileOfDecs(regrets, quantileRegret)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	initialRegret, err = quantile.Add(offSetTimesDenominator)
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
	topicId TopicId,
	alphaRegret alloraMath.Dec,
	inferer Worker,
	newParticipant bool,
) (bool, error) {
	numInclusions := uint64(0)
	numInclusions, err := k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get inferer inclusions")
	}
	return isExperiencedActor(newParticipant, numInclusions, alphaRegret)
}

// Determine if a forecaster is an experienced worker by checking
// the inclusions count is greater than 1/alpha_regret
func isExperiencedForecaster(
	ctx context.Context,
	k keeper.Keeper,
	topicId TopicId,
	alphaRegret alloraMath.Dec,
	forecaster Worker,
	newParticipant bool,
) (bool, error) {
	numInclusions := uint64(0)
	numInclusions, err := k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to get forecaster inclusions")
	}
	return isExperiencedActor(newParticipant, numInclusions, alphaRegret)
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

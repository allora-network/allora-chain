package inferencesynthesis

import (
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

// args for GetCalcSetNetworkRegrets
type GetCalcSetNetworkRegretsArgs struct {
	Ctx           sdk.Context
	K             keeper.Keeper
	TopicId       TopicId
	NetworkLosses emissions.ValueBundle
	Nonce         emissions.Nonce
	AlphaRegret   alloraMath.Dec
	CNorm         alloraMath.Dec
	PNorm         alloraMath.Dec
	EpsilonTopic  alloraMath.Dec
}

// Calculate the new network regrets by taking EMAs between the previous network regrets
// and the new regrets admitted by the inputted network losses
// It is assumed the workers are uniquely represented in the network losses
func GetCalcSetNetworkRegrets(args GetCalcSetNetworkRegretsArgs) error {
	// Convert the network losses to a networkLossesByWorker
	networkLossesByWorker := ConvertValueBundleToNetworkLossesByWorker(args.NetworkLosses)
	blockHeight := args.Nonce.BlockHeight

	workersRegrets := make([]alloraMath.Dec, 0)

	sortedInfererValues := args.NetworkLosses.InfererValues
	sort.Slice(sortedInfererValues, func(i, j int) bool {
		return sortedInfererValues[i].Worker < sortedInfererValues[j].Worker
	})

	// R_ij - Inferer Regrets
	for _, infererLoss := range sortedInfererValues {
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
		if !newParticipant {
			workersRegrets = append(workersRegrets, newInfererRegret.Value)
		}
	}

	sortedForecasterValues := args.NetworkLosses.ForecasterValues
	sort.Slice(sortedForecasterValues, func(i, j int) bool {
		return sortedForecasterValues[i].Worker < sortedForecasterValues[j].Worker
	})

	// R_ik - Forecaster Regrets
	for _, forecasterLoss := range sortedForecasterValues {
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
		if !newParticipant {
			workersRegrets = append(workersRegrets, newForecasterRegret.Value)
		}
	}

	// R^-_ij - Naive Regrets
	for _, infererLoss := range sortedInfererValues {
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

	sortedOneOutInfererValues := args.NetworkLosses.OneOutInfererValues
	sort.Slice(sortedOneOutInfererValues, func(i, j int) bool {
		return sortedOneOutInfererValues[i].Worker < sortedOneOutInfererValues[j].Worker
	})

	// R^-j′ij - One-out inferer inferer regrets
	for _, oneOutInfererLoss := range sortedOneOutInfererValues {
		for _, infererLoss := range sortedInfererValues {
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

	sortedOneOutInfererForecasterValues := args.NetworkLosses.OneOutInfererForecasterValues
	sort.Slice(sortedOneOutInfererForecasterValues, func(i, j int) bool {
		return sortedOneOutInfererForecasterValues[i].Forecaster < sortedOneOutInfererForecasterValues[j].Forecaster
	})

	// R^-j′ik - One-out inferer forecaster regrets
	for _, oneOutInfererLoss := range sortedOneOutInfererValues {
		for _, oneOutInfererForecasterLoss := range sortedOneOutInfererForecasterValues {
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

	sortedOneOutForecasterValues := args.NetworkLosses.OneOutForecasterValues
	sort.Slice(sortedOneOutForecasterValues, func(i, j int) bool {
		return sortedOneOutForecasterValues[i].Worker < sortedOneOutForecasterValues[j].Worker
	})

	// R^-k′ij - One-out forecaster inferer regrets
	for _, oneOutForecasterLoss := range sortedOneOutForecasterValues {
		for _, infererloss := range sortedInfererValues {
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

	sortedOneOutForecasterForecasterValues := args.NetworkLosses.OneOutInfererForecasterValues
	sort.Slice(sortedOneOutForecasterForecasterValues, func(i, j int) bool {
		return sortedOneOutForecasterForecasterValues[i].Forecaster < sortedOneOutForecasterForecasterValues[j].Forecaster
	})

	// R^-k′ik - One-out forecaster forecaster regrets
	for _, oneOutForecasterLoss := range sortedOneOutForecasterValues {
		for _, forecasterLoss := range sortedForecasterValues {
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

	sortedOneInForecasterValues := args.NetworkLosses.OneInForecasterValues
	sort.Slice(sortedOneInForecasterValues, func(i, j int) bool {
		return sortedOneInForecasterValues[i].Worker < sortedOneInForecasterValues[j].Worker
	})

	// R^+_k'ij - One-in forecaster regrets
	for _, oneInForecasterLoss := range sortedOneInForecasterValues {
		// Loop over the inferer losses so that their losses may be compared against the one-in forecaster's loss, for each forecaster
		for _, infererLoss := range sortedInfererValues {
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
		updatedTopicInitialRegret, err := CalcTopicInitialRegret(workersRegrets, args.EpsilonTopic, args.PNorm, args.CNorm)
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

func CalcTopicInitialRegret(
	regrets []alloraMath.Dec,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
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
	eightPointTwoFive := alloraMath.MustNewDecFromString("8.25")

	eightPointTwoFiveDividedByPnorm, err := eightPointTwoFive.Quo(pNorm)
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

	minimumRegret := alloraMath.ZeroDec()
	for i, regret := range regrets {
		if i == 0 || regret.Lt(minimumRegret) {
			minimumRegret = regret
		}
	}

	initialRegret, err = minimumRegret.Add(offSetTimesDenominator)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	return initialRegret, nil
}

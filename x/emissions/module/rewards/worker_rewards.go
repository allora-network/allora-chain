package rewards

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var TASK_FORECAST = true
var TASK_INFERENCE = false

func GetInferenceTaskRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	blockHeight int64,
	pReward alloraMath.Dec,
	cReward alloraMath.Dec,
	latestScores []types.Score,
) ([]string, []alloraMath.Dec, error) {
	return GetWorkersRewardFractions(ctx, k, topicId, blockHeight, TASK_INFERENCE, pReward, cReward, latestScores)
}

func GetForecastingTaskRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	blockHeight int64,
	pReward alloraMath.Dec,
	cReward alloraMath.Dec,
	latestScores []types.Score,
) ([]string, []alloraMath.Dec, error) {
	return GetWorkersRewardFractions(ctx, k, topicId, blockHeight, TASK_FORECAST, pReward, cReward, latestScores)
}

func GetWorkersRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	blockHeight int64,
	which bool,
	pReward alloraMath.Dec,
	cReward alloraMath.Dec,
	latestScores []types.Score,
) ([]string, []alloraMath.Dec, error) {
	// Get all latest score for each worker and the scores from the latest time steps
	// to be used in the standard deviantion
	scores := make([][]alloraMath.Dec, 0)
	latestWorkerScores := make([]alloraMath.Dec, 0)
	workers := make([]string, 0)
	if which == TASK_INFERENCE {
		// Get latest score for each worker
		for _, latestScore := range latestScores {
			workers = append(workers, latestScore.Address)
			latestWorkerScores = append(latestWorkerScores, latestScore.Score)
		}

		// Get worker scores from the latest time steps
		latestScoresFromLastestTimeSteps, err := k.GetInferenceScoresUntilBlock(ctx, topicId, blockHeight)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get worker inference scores from the latest time steps")
		}
		var workerLastScoresDec []alloraMath.Dec
		for _, score := range latestScoresFromLastestTimeSteps {
			workerLastScoresDec = append(workerLastScoresDec, score.Score)
		}
		scores = append(scores, workerLastScoresDec)

	} else { // TASK_FORECAST
		// Get latest score for each worker
		for _, latestScore := range latestScores {
			workers = append(workers, latestScore.Address)
			latestWorkerScores = append(latestWorkerScores, latestScore.Score)
		}

		// Get worker scores from the latest time steps
		latestScoresFromLastestTimeSteps, err := k.GetForecastScoresUntilBlock(ctx, topicId, blockHeight)
		if err != nil {
			return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get worker forecast scores from the latest time steps")
		}
		var workerLastScoresDec []alloraMath.Dec
		for _, score := range latestScoresFromLastestTimeSteps {
			workerLastScoresDec = append(workerLastScoresDec, score.Score)
		}
		scores = append(scores, workerLastScoresDec)
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get topic %v", topicId)
	}
	rewardFractions, err := GetScoreFractions(latestWorkerScores, flatten(scores), pReward, cReward, topic.Epsilon)
	if err != nil {
		return []string{}, []alloraMath.Dec{}, errors.Wrapf(err, "failed to get score fractions")
	}

	return workers, rewardFractions, nil
}

func GetInferenceTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	betaEntropy alloraMath.Dec,
	workers []string,
	workersFractions []alloraMath.Dec,
) (
	entropy alloraMath.Dec,
	err error,
) {
	return getInferenceOrForecastTaskEntropy(ctx, k, topicId, emaAlpha, betaEntropy, TASK_INFERENCE, workers, workersFractions)
}

func GetForecastTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	betaEntropy alloraMath.Dec,
	workers []string,
	workersFractions []alloraMath.Dec,
) (
	entropy alloraMath.Dec,
	err error,
) {
	return getInferenceOrForecastTaskEntropy(ctx, k, topicId, emaAlpha, betaEntropy, TASK_FORECAST, workers, workersFractions)
}

func getInferenceOrForecastTaskEntropy(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	emaAlpha alloraMath.Dec,
	betaEntropy alloraMath.Dec,
	which bool,
	workers []string,
	workersFractions []alloraMath.Dec,
) (
	entropy alloraMath.Dec,
	err error,
) {
	numWorkers := len(workers)
	emaRewardFractions := make([]alloraMath.Dec, numWorkers)
	var previousRewardFraction alloraMath.Dec
	for i, worker := range workers {
		noPriorFraction := false
		if which == TASK_INFERENCE {
			previousRewardFraction, noPriorFraction, err = k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
			if err != nil {
				return alloraMath.Dec{}, errors.Wrapf(err, "failed to get previous inference reward fraction")
			}
		} else { // TASK_FORECAST
			previousRewardFraction, noPriorFraction, err = k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
			if err != nil {
				return alloraMath.Dec{}, errors.Wrapf(err, "failed to get previous forecast reward fraction")
			}
		}
		emaRewardFractions[i], err = alloraMath.CalcEma(
			emaAlpha,
			workersFractions[i],
			previousRewardFraction,
			noPriorFraction,
		)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate EMA")
		}
	}

	// Calculate modified reward fractions and persist for next round
	numberRatio, err := NumberRatio(emaRewardFractions)
	if err != nil {
		return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate number ratio")
	}
	modifiedRewardFractions, err := ModifiedRewardFractions(emaRewardFractions)
	if err != nil {
		return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate modified reward fractions")
	}
	if which == TASK_INFERENCE {
		for i, worker := range workers {
			err := k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, modifiedRewardFractions[i])
			if err != nil {
				return alloraMath.Dec{}, errors.Wrapf(err, "failed to set previous inference reward fraction")
			}
		}
	} else { // TASK_FORECAST
		for i, worker := range workers {
			err := k.SetPreviousForecastRewardFraction(ctx, topicId, worker, modifiedRewardFractions[i])
			if err != nil {
				return alloraMath.Dec{}, errors.Wrapf(err, "failed to set previous forecast reward fraction")
			}
		}
	}

	if numWorkers > 1 {
		entropy, err = Entropy(
			modifiedRewardFractions,
			numberRatio,
			alloraMath.NewDecFromInt64(int64(numWorkers)),
			betaEntropy,
		)
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate entropy")
		}
	} else {
		entropy, err = EntropyForSingleParticipant()
		if err != nil {
			return alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate entropy for single participant")
		}
	}

	return entropy, nil
}

// The performance score of the entire forecasting task T_i
// is positive if the removal of the forecasting task would
// increase the network loss, and is negative if its removal
// would decrease the network loss
// We subtract the log-loss of the complete network inference
// (L_i) from that of the naive network (L_i^-), which is
// obtained by omitting all forecast-implied inferences
// T_i = log L_i^- - log L_i
// However we store the log based forms in the keeper
// so we do not need to take the logs again
func ForecastingPerformanceScore(
	naiveNetworkInferenceLoss,
	networkInferenceLoss alloraMath.Dec,
) (alloraMath.Dec, error) {
	return naiveNetworkInferenceLoss.Sub(networkInferenceLoss)
}

// Implements the utility function for forecasting performance score
// with the new specification:
// χ = 0.1 for score < 0,
// χ = 0.5 for score > 1,
// χ = 0.4 * score + 0.1 in between
func ForecastingUtility(
	forecastingTaskUtilityScore alloraMath.Dec,
	infererScores []types.Score,
	previousForecasterScoreRatio alloraMath.Dec,
	alpha alloraMath.Dec,
) (alloraMath.Dec, error) {
	zeroPointOne := alloraMath.MustNewDecFromString("0.1")
	zeroPointFour := alloraMath.MustNewDecFromString("0.4")
	zeroPointFive := alloraMath.MustNewDecFromString("0.5")

	// Calculate the maximum of infererScores
	var maxInfererScore alloraMath.Dec
	for _, score := range infererScores {
		if score.Score.Gt(maxInfererScore) {
			maxInfererScore = score.Score
		}
	}
	scoreDenominator := maxInfererScore.Abs()
	if maxInfererScore.IsZero() {
		if forecastingTaskUtilityScore.IsZero() {
			return zeroPointFive, nil
		} else {
			scoreDenominator = forecastingTaskUtilityScore.Abs()
		}
	}

	scoreNumerator, err := forecastingTaskUtilityScore.Sub(alloraMath.Min(alloraMath.ZeroDec(), maxInfererScore))
	if err != nil {
		return alloraMath.Dec{}, err
	}
	scoreRatio, err := scoreNumerator.Quo(scoreDenominator)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	// Calculate alpha * (scoreRatio)
	alphaTimesScoreRatio, err := alpha.Mul(scoreRatio)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	// Calculate (1 - alpha) * previousForecasterScoreRatio
	oneMinusAlpha, err := alloraMath.OneDec().Sub(alpha)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusAlphaTimesPreviousForecasterScoreRatio, err := oneMinusAlpha.Mul(previousForecasterScoreRatio)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	// Calculate the final tau value
	forecasterScoreRatio, err := alphaTimesScoreRatio.Add(oneMinusAlphaTimesPreviousForecasterScoreRatio)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	// Apply the final tau value based on conditions
	if forecasterScoreRatio.Lt(alloraMath.ZeroDec()) {
		return zeroPointOne, nil
	}

	if forecasterScoreRatio.Gte(alloraMath.OneDec()) {
		return zeroPointFive, nil
	}

	// For 0 <= finalTau < 1, return 0.4 * finalTau + 0.1
	finalTauTimesZeroPointFour, err := forecasterScoreRatio.Mul(zeroPointFour)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	chiReturnValue, err := finalTauTimesZeroPointFour.Add(zeroPointOne)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	return chiReturnValue, nil
}

// renormalize with a factor γ to ensure that the
// total reward allocated to workers (Ui + Vi)
// remains constant (otherwise, this would go at the expense of reputers)
// γ = (F_i + G_i) / ( (1 − χ)*F_i + χ*G_i)
func NormalizationFactor(
	entropyInference alloraMath.Dec,
	entropyForecasting alloraMath.Dec,
	forecastingUtility alloraMath.Dec,
) (alloraMath.Dec, error) {
	numerator, err := entropyInference.Add(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusForecastingUtility, err := alloraMath.OneDec().Sub(forecastingUtility)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusForecastingUtilityTimesEntropyInference, err := oneMinusForecastingUtility.Mul(entropyInference)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	forecastingUtilityTimesEntropyForecasting, err := forecastingUtility.Mul(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	denominator, err := oneMinusForecastingUtilityTimesEntropyInference.Add(forecastingUtilityTimesEntropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := numerator.Quo(denominator)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// helper function to get chi and gamma
func GetChiAndGamma(
	naiveNetworkInferenceLoss,
	networkInferenceLoss,
	entropyInference,
	entropyForecasting alloraMath.Dec,
	infererScores []types.Score,
	previousForecasterScoreRatio alloraMath.Dec,
	alpha alloraMath.Dec,
) (chi alloraMath.Dec, gamma alloraMath.Dec, err error) {
	forecastingTaskUtilityScore, err := ForecastingPerformanceScore(
		naiveNetworkInferenceLoss,
		networkInferenceLoss,
	)
	if err != nil {
		return alloraMath.Dec{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate forecasting performance score")
	}
	chi, err = ForecastingUtility(
		forecastingTaskUtilityScore,
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	if err != nil {
		return alloraMath.Dec{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate forecasting utility")
	}
	gamma, err = NormalizationFactor(
		entropyInference,
		entropyForecasting,
		chi,
	)
	if err != nil {
		return alloraMath.Dec{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to calculate normalization factor")
	}
	return chi, gamma, nil
}

// inference rewards calculation
// U_i = ((1 - χ) * γ * F_i * E_i ) / (F_i + G_i + H_i)
func GetRewardForInferenceTaskInTopic(
	entropyInference alloraMath.Dec, // F_i
	entropyForecasting alloraMath.Dec, // G_i
	entropyReputer alloraMath.Dec, // H_i
	totalReward *alloraMath.Dec, // E_i
	chi alloraMath.Dec, // χ
	gamma alloraMath.Dec, // γ
) (alloraMath.Dec, error) {
	oneMinusChi, err := alloraMath.OneDec().Sub(chi)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusChiGamma, err := oneMinusChi.Mul(gamma)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusChiGammaEntropyInference, err := oneMinusChiGamma.Mul(entropyInference)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	numerator, err := oneMinusChiGammaEntropyInference.Mul(*totalReward)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	entropyInferencePlusForecasting, err := entropyInference.Add(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	denominator, err := entropyInferencePlusForecasting.Add(entropyReputer)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := numerator.Quo(denominator)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// forecaster rewards calculation
// V_i = (χ * γ * G_i * E_i) / (F_i + G_i + H_i)
func GetRewardForForecastingTaskInTopic(
	entropyInference alloraMath.Dec, // F_i
	entropyForecasting alloraMath.Dec, // G_i
	entropyReputer alloraMath.Dec, // H_i
	totalReward *alloraMath.Dec, // E_i
	chi alloraMath.Dec, // χ
	gamma alloraMath.Dec, // γ
) (alloraMath.Dec, error) {
	chiGamma, err := chi.Mul(gamma)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	chiGammaEntropyForecasting, err := chiGamma.Mul(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	numerator, err := chiGammaEntropyForecasting.Mul(*totalReward)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	entropyInferencePlusForecasting, err := entropyInference.Add(entropyForecasting)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	denominator, err := entropyInferencePlusForecasting.Add(entropyReputer)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := numerator.Quo(denominator)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// GetRewardPerWorker calculates the reward for workers for forecast and inference tasks
// U_ij = u_ij * Ui,
// V_ik = v_ik * Vi
func GetRewardPerWorker(
	topicId uint64,
	taskRewardType types.TaskRewardType,
	totalRewards alloraMath.Dec,
	workerAddresses []string,
	workerFractions []alloraMath.Dec,
) ([]types.TaskReward, error) {
	var rewards []types.TaskReward
	for i, fraction := range workerFractions {
		reward, err := fraction.Mul(totalRewards)
		if err != nil {
			return nil, err
		}
		rewards = append(rewards, types.TaskReward{
			Address: workerAddresses[i],
			Reward:  reward,
			TopicId: topicId,
			Type:    taskRewardType,
		})
	}

	return rewards, nil
}

package rewards

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// flatten converts a double slice of alloraMath.Dec to a single slice of alloraMath.Dec
func flatten(arr [][]alloraMath.Dec) []alloraMath.Dec {
	var flat []alloraMath.Dec
	for _, row := range arr {
		flat = append(flat, row...)
	}
	return flat
}

// RewardFractions without multiplication against total rewards are used to calculate entropy
// note the use of lowercase u as opposed to capital
// u_ij = M(Tij) / ∑_j M(T_ij)
// v_ik = M(Tik) / ∑_k M(T_ik)
func GetScoreFractions(
	latestWorkerScores []alloraMath.Dec,
	latestTimeStepsScores []alloraMath.Dec,
	pReward alloraMath.Dec,
	cReward alloraMath.Dec,
	epsilon alloraMath.Dec,
) ([]alloraMath.Dec, error) {
	mappedValues, err := GetMappingFunctionValues(latestWorkerScores, latestTimeStepsScores, pReward, cReward, epsilon)
	if err != nil {
		return nil, errors.Wrapf(err, "error in GetMappingFunctionValue")
	}
	ret := make([]alloraMath.Dec, len(mappedValues))
	mappedSum, err := alloraMath.SumDecSlice(mappedValues)
	if err != nil {
		return nil, errors.Wrapf(err, "error in SumDecSlice:")
	}
	for i, mappedValue := range mappedValues {
		ret[i], err = mappedValue.Quo(mappedSum)
		if err != nil {
			return nil, errors.Wrapf(err, "error doing division of mappedValues")
		}
	}
	return ret, nil
}

// Mapping function used by score fraction calculation
// M(T) = φ_p (T / (σ(T) + ɛ))
// phi is the phi function
// sigma is NOT the sigma function but rather represents standard deviation
func GetMappingFunctionValues(
	latestWorkerScores []alloraMath.Dec, // T - latest scores from workers
	latestTimeStepsScores []alloraMath.Dec, // σ(T) - scores for stdDev (from multiple workers/time steps)
	pReward alloraMath.Dec, // p
	cReward alloraMath.Dec, // c
	epsilon alloraMath.Dec, // ɛ
) ([]alloraMath.Dec, error) {
	stdDev := alloraMath.ZeroDec()

	var err error
	if len(latestTimeStepsScores) > 1 {
		stdDev, err = alloraMath.StdDev(latestTimeStepsScores)
		if err != nil {
			return nil, errors.Wrapf(err, "err getting stdDev")
		}
		stdDev = stdDev.Abs()
	}

	ret := make([]alloraMath.Dec, len(latestWorkerScores))
	for i, score := range latestWorkerScores {
		if stdDev.Lt(epsilon) {
			// if standard deviation is smaller than epsilon, or zero,
			// then all scores are the very close to the same and losses are the same
			// therefore everyone should be paid the same, so we
			// return the plain value 1 for everybody
			ret[i] = alloraMath.OneDec()
		} else {
			stdDevPlusEpsilon, err := stdDev.Add(epsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err adding epsilon to stdDev")
			}
			scoreDividedByStdDevPlusEpsilon, err := score.Quo(stdDevPlusEpsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err dividing score by stdDevPlusEpsilon")
			}
			ret[i], err = alloraMath.Phi(pReward, cReward, scoreDividedByStdDevPlusEpsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err calculating phi")
			}
		}
	}
	return ret, nil
}

// CalculateReputerRewardFractions calculates the reward fractions for each reputer based on their stakes, scores, and preward parameter.
// W_im
func CalculateReputerRewardFractions(
	stakes []alloraMath.Dec,
	scores []alloraMath.Dec,
	preward alloraMath.Dec,
) ([]alloraMath.Dec, error) {
	if len(stakes) != len(scores) {
		return nil, types.ErrInvalidSliceLength
	}

	var err error
	// Calculate (stakes * scores)^preward and sum of all fractions
	var totalFraction alloraMath.Dec
	fractions := make([]alloraMath.Dec, len(stakes))
	for i, stake := range stakes {
		stakeTimesScores, err := stake.Mul(scores[i])
		if err != nil {
			return []alloraMath.Dec{}, err
		}
		fractions[i], err = alloraMath.Pow(stakeTimesScores, preward)
		if err != nil {
			return []alloraMath.Dec{}, err
		}
		totalFraction, err = totalFraction.Add(fractions[i])
		if err != nil {
			return []alloraMath.Dec{}, err
		}
	}

	// Normalize fractions
	for i := range fractions {
		if fractions[i].IsZero() {
			continue
		}
		fractions[i], err = fractions[i].Quo(totalFraction)
		if err != nil {
			return []alloraMath.Dec{}, err
		}
	}

	return fractions, nil
}

// f_ij, f_ik, and f_im are all reward fractions
// that require computing the ratio of one participant to all participants
// yes this is extremely simple math
// yes we write a separate function for it anyway. The compiler can inline it if necessary
// normalizeToArray = value / sum(allValues)
// this covers equations
// f_ij =  (̃U_ij) / ∑_j(̃Uij)
// f_ik = (̃Vik) / ∑_k(̃Vik)
// f_im =  (̃Wim) / ∑_m(̃Wim)
func ModifiedRewardFractions(rewardFractions []alloraMath.Dec) ([]alloraMath.Dec, error) {
	sumValues, err := alloraMath.SumDecSlice(rewardFractions)
	if err != nil {
		return nil, err
	}
	ret := make([]alloraMath.Dec, len(rewardFractions))
	for i, value := range rewardFractions {
		ret[i], err = value.Quo(sumValues)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// We define a modified entropy for each class
// ({F_i, G_i, H_i} for the inference, forecasting, and reputer tasks, respectively
// Fi = - ∑_j( f_ij * ln(f_ij) * (N_{i,eff} / N_i)^β )
// Gi = - ∑_k( f_ik * ln(f_ik) * (N_{f,eff} / N_f)^β )
// Hi = - ∑_m( f_im * ln(f_im) * (N_{r,eff} / N_r)^β )
// we use beta = 0.25 as a fiducial value
func Entropy(
	rewardFractionsPerActor []alloraMath.Dec, // an array of every f_{ij}, f_{ik}, or f_{im}
	numberRatio alloraMath.Dec, // N_{i, eff}, N_{f,eff} or N_{r,eff}
	numParticipants alloraMath.Dec, // N_i
	beta alloraMath.Dec, // β
) (alloraMath.Dec, error) {
	multiplier, err := numberRatio.Quo(numParticipants)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	multiplier, err = alloraMath.Pow(multiplier, beta)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	sum := alloraMath.ZeroDec()
	for _, f := range rewardFractionsPerActor {
		lnF, err := alloraMath.Ln(f)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		fLnF, err := f.Mul(lnF)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		sum, err = sum.Add(fLnF)
		if err != nil {
			return alloraMath.Dec{}, err
		}
	}

	inverseSum, err := sum.Mul(alloraMath.NewDecFromInt64(-1))
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := inverseSum.Mul(multiplier)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// If there's only one worker, entropy should be the default number of 0.173286795139986
func EntropyForSingleParticipant() (alloraMath.Dec, error) {
	return alloraMath.MustNewDecFromString("0.173286795139986"), nil
}

// The number ratio term captures the number of participants in the network
// to prevent sybil attacks in the rewards distribution
// This function captures
// N_{i,eff} = 1 / ∑_j( f_ij^2 )
// N_{f,eff} = 1 / ∑_k( f_ik^2 )
// N_{r,eff} = 1 / ∑_m( f_im^2 )
func NumberRatio(rewardFractions []alloraMath.Dec) (alloraMath.Dec, error) {
	if len(rewardFractions) == 0 {
		return alloraMath.Dec{}, types.ErrNumberRatioInvalidSliceLength
	}
	sum := alloraMath.ZeroDec()
	for _, f := range rewardFractions {
		fSquared, err := f.Mul(f)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		sum, err = sum.Add(fSquared)
		if err != nil {
			return alloraMath.Dec{}, err
		}
	}
	if sum.Equal(alloraMath.ZeroDec()) {
		return alloraMath.Dec{}, types.ErrNumberRatioDivideByZero
	}
	ret, err := alloraMath.OneDec().Quo(sum)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

package module

import (
	"errors"
	"fmt"
	"math/big"

	cosmosMath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/upshot-tech/protocol-state-machine-module/keeper"
)

type Uint = cosmosMath.Uint
type Float = big.Float

// constants
const EPOCH_LENGTH = 5

// errors defined in this file
var ErrInvalidLastUpdate = errors.New("invalid last update")
var ErrEpochNotReached = errors.New("not enough blocks have passed to hit an epoch")
var ErrScalarMultiplyNegative = errors.New("token rewards multiplication output should always be positive")

func globalEmissionPerTopic(ctx sdk.Context, am AppModule, blocksSinceLastUpdate uint64) error {
	fmt.Println("Calculating global emission per topic")
	// number of epochs that have passed (if more than 1)
	epochsPassed := cosmosMath.NewUint(blocksSinceLastUpdate / uint64(EPOCH_LENGTH))

	// get emission amount
	perEpochEmission := cosmosMath.NewUintFromBigInt(am.keeper.GetAccumulatedEpochRewards(ctx).Amount.BigInt())
	//perEpochEmission := am.keeper.GetAccumulatedEpochRewards(ctx).Amount.BigInt()
	fmt.Println("Rewards to be distributed this epoch: ", perEpochEmission)
	cumulativeEmission := epochsPassed.Mul(perEpochEmission)

	// get total stake in network
	totalStake, err := am.keeper.GetTotalStake(ctx)
	if err != nil {
		fmt.Println("Error getting total stake: ", err)
		return err
	}
	// if no stake, no rewards to give away, do nothing
	if totalStake.Equal(cosmosMath.ZeroUint()) {
		err = am.keeper.SetLastRewardsUpdate(ctx, ctx.BlockHeight())
		if err != nil {
			fmt.Println("Error setting last rewards update: ", err)
			return err
		}
		return nil
	}

	// get each topic's stake
	topicStakeMap, err := am.keeper.GetAllTopicStake(ctx)
	if err != nil {
		fmt.Println("Error getting all topic stake: ", err)
		return err
	}

	for topicId, topicStake := range topicStakeMap {
		// for each topic get percentage of total emissions
		// then get each participant's percentage of that percentage
		reputerEmissions, workerEmissions, err := getTopicParticipantEmissions(
			ctx,
			am,
			topicId,
			&topicStake,
			&cumulativeEmission,
			&totalStake)
		if err != nil {
			fmt.Println("Error getting topic participant emissions: ", err)
			return err
		}

		// Save/set the above emissions to actually pay participants.
		// Do this by increasing the stake of each worker by their due ServerEmission + ValidatorEmission
		err = am.keeper.SetLastRewardsUpdate(ctx, ctx.BlockHeight())
		if err != nil {
			fmt.Println("Error setting last rewards update: ", err)
			return err
		}
		setEmissions(ctx, am, topicId, reputerEmissions, workerEmissions)
	}

	return nil
}

func getTopicParticipantEmissions(
	ctx sdk.Context,
	am AppModule,
	topicId keeper.TOPIC_ID,
	topicStake *Uint,
	cumulativeEmission *Uint,
	totalStake *Uint) (reputerEmissions map[keeper.REPUTERS]*Uint, workerEmissions map[keeper.WORKERS]*Uint, err error) {

	// get total emission for topic
	topicEmissionXStake := cumulativeEmission.Mul(*topicStake)
	topicEmissions := topicEmissionXStake.Quo(*totalStake)

	// get all participants in that topic
	reputers, err := am.keeper.GetTopicReputers(ctx, topicId)
	if err != nil {
		fmt.Println("Error getting reputers")
		return nil, nil, err
	}

	// Normalize reputer stakes to the sum
	reputerStakeNorm, err := getReputerNormalizedStake(ctx, reputers, topicStake, am)
	if err != nil {
		fmt.Println("Error getting reputer normalized stake")
		return nil, nil, err
	}

	// Get Weights between nodes in topic
	//    Weight_ij = reputer i -> worker j -> weight val
	topicWeights, err := am.keeper.GetWeightsFromTopic(ctx, topicId)
	if err != nil {
		fmt.Println("Error getting weights from topic")
		return nil, nil, err
	}

	// Ranks = matmul Weights * NormalizedStake
	// for i rows and j columns
	// i.e. rank[j] = sum(j) + weight_ij * normalizedStake_i
	ranks := matmul(topicWeights, reputerStakeNorm)

	// Incentive = normalize(Ranks)
	incentive := normalize(ranks)

	// BondDeltas using elementwise multiplication of the same vector for all rows of Weight matrix.
	// i.e. For each row i: BondDelta_ij = Weights_ij x Stake_j
	bondDeltas := elementWiseProduct(topicWeights, reputerStakeNorm)

	// Row-wise normalize BondDeltas
	bondDeltasNorm := normalizeBondDeltas(bondDeltas)

	// Dividends = normalize(BondDeltas matmul Incentive)
	dividends := matmul(bondDeltasNorm, incentive)
	dividendsNorm := normalize(dividends)

	// EmissionSum = sum(Dividends) + sum(Incentives)
	dividendsSum := sum(dividendsNorm)
	incentivesSum := sum(incentive)
	emissionSum := new(Float).Add(&dividendsSum, &incentivesSum)

	topicEmissionsFloat := new(Float).SetInt(topicEmissions.BigInt())
	if new(Float).SetInt64(0).Cmp(emissionSum) == 0 {
		// If EmissionSum == 0 then set NormalizedReputerEmissions to normalized stake
		reputerEmissionsNorm := reputerStakeNorm

		// ValidatorEmissions = scalar multiply topicEmissionsTotal x NormalizedReputerEmissions
		reputerEmissions, err = scalarMultiply(reputerEmissionsNorm, topicEmissionsFloat)
		if err != nil {
			return nil, nil, err
		}

	} else {
		// NormalizedServerEmissions = Incentives scalar divide EmmissionSum
		normalizedWorkerEmissions := getNormalizedEmissions(incentive, emissionSum)

		// NormalizedValidatorEmissions = Dividends scalar divide EmmissionSum
		normalizedReputerEmissions := getNormalizedEmissions(dividends, emissionSum)

		// ServerEmissions = scalar multiply topicEmissionsTotal x NormalizedServerEmissions
		workerEmissions, err = scalarMultiply(normalizedWorkerEmissions, topicEmissionsFloat)
		if err != nil {
			return nil, nil, err
		}

		// ValidatorEmissions = scalar multiply topicEmissionsTotal x NormalizedValidatorEmissions
		reputerEmissions, err = scalarMultiply(normalizedReputerEmissions, topicEmissionsFloat)
		if err != nil {
			return nil, nil, err
		}
	}
	return reputerEmissions, workerEmissions, nil
}

func setEmissions(
	ctx sdk.Context,
	am AppModule,
	topic keeper.TOPIC_ID,
	reputerEmissions map[keeper.REPUTERS]*Uint,
	workerEmissions map[keeper.WORKERS]*Uint) {
	// by default emissions are restaked, upon the person themselves.
	for reputer, reputerEmission := range reputerEmissions {
		fmt.Println("Setting reputer emission: ", reputer, " : ", reputerEmission)
		am.keeper.AddStake(ctx, topic, reputer, reputer, *reputerEmission)
	}
	for worker, workerEmission := range workerEmissions {
		fmt.Println("Setting worker emission: ", worker, " : ", workerEmission)
		am.keeper.AddStake(ctx, topic, worker, worker, *workerEmission)
	}
}

func getReputerNormalizedStake(
	ctx sdk.Context,
	reputers []sdk.AccAddress,
	topicStake *cosmosMath.Uint,
	am AppModule) (reputerNormalizedStakeMap map[keeper.ACC_ADDRESS]*Float, retErr error) {
	reputerNormalizedStakeMap = make(map[keeper.ACC_ADDRESS]*Float)
	for _, reputer := range reputers {
		// Get Stake in each reputer
		reputerTargetStake, err := am.keeper.GetTargetStake(ctx, reputer)
		if err != nil {
			return nil, err
		}
		reputerTotalStake := new(Float).SetInt(reputerTargetStake.BigInt())

		// How much stake does each reputer have as a percentage of the total stake in the topic?
		topicStakeFloat := new(Float).SetInt(topicStake.BigInt())
		reputerNormalizedStake := new(Float).Quo(reputerTotalStake, topicStakeFloat)
		reputerNormalizedStakeMap[reputer.String()] = reputerNormalizedStake
	}
	return reputerNormalizedStakeMap, nil
}

// Number is either a Float or a uint64
type Number interface {
	*Float | *cosmosMath.Uint
}

// matmul multiplies a matrix by a vector where both are stored in golang maps
// the index to the map is considered the row or column
// 0 values are taken to be not found in the map and so skipped during addition
// for matrix * vector, iterating through rows i and columns j,
// result_j = result_j + matrix_ij * vector_i
func matmul[N Number](
	matrix map[string]map[string]N,
	vector map[string]*Float) (result map[string]*Float) {
	result = make(map[string]*Float)
	for i, rowMap := range matrix {
		vec_i := vector[i]
		if vec_i == nil {
			continue
		}
		for j, matrix_ij := range rowMap {
			priorResult := new(Float).SetInt64(0)
			if result[j] != nil {
				priorResult = result[j]
			}
			deltaResult := new(Float)
			switch m_ij := any(matrix_ij).(type) {
			case uint64:
				deltaResult.Mul(new(Float).SetUint64(m_ij), vec_i)
			case *Float:
				deltaResult.Mul(m_ij, vec_i)
			}
			deltaResult.Add(deltaResult, priorResult)
			result[j] = deltaResult
		}
	}
	return result
}

// normalize divides every value in a map by the sum of all values in the map
func normalize(a map[string]*Float) (result map[string]*Float) {
	result = make(map[string]*Float)
	sum := new(Float).SetInt64(0)
	for _, val := range a {
		sum.Add(sum, val)
	}
	for key, val := range a {
		result[key] = new(Float).Quo(val, sum)
	}
	return result
}

// Element Wise Product takes a matrix and a vector and multiplies
// each element of the matrix by the corresponding element of the vector
// this can sometimes be called the Hadamard product
// note that we use maps to represent the matrix and vector
// so values of zero are simply not stored in the map.
// for matrix * vector, iterating through rows i and columns j,
// result_ij = matrix_ij * vector_i
func elementWiseProduct(
	matrix map[string]map[string]*Uint,
	vector map[string]*Float) (result map[string]map[string]*Float) {
	result = make(map[string]map[string]*Float)
	for i, rowMap := range matrix {
		vec_i := vector[i]
		if vec_i == nil {
			continue
		}
		result[i] = make(map[string]*Float)
		for j, matrix_ij := range rowMap {
			matrix_ijFloat := new(Float).SetInt(matrix_ij.BigInt())
			result[i][j] = new(Float).Mul(matrix_ijFloat, vec_i)
		}
	}
	return result
}

// Row-wise normalizes BondDeltas. For each row, normalizes the values in that row relative to the row
func normalizeBondDeltas(bondDeltas map[keeper.REPUTERS]map[keeper.WORKERS]*Float) (result map[keeper.REPUTERS]map[keeper.WORKERS]*Float) {
	result = make(map[keeper.REPUTERS]map[keeper.WORKERS]*Float)
	for reputer, workerWeights := range bondDeltas {
		result[reputer] = normalize(workerWeights)
	}
	return result
}

func getDividends(
	normalizedBondDeltas map[keeper.REPUTERS]map[keeper.WORKERS]*Float,
	incentive map[keeper.ACC_ADDRESS]*Float) map[keeper.ACC_ADDRESS]*Float {
	ret := make(map[keeper.ACC_ADDRESS]*Float)
	for reputer, reputerWeightsMap := range normalizedBondDeltas {
		reputerIncentive := incentive[reputer]
		if reputerIncentive == nil {
			continue
		}
		for workerOrReputer, weight := range reputerWeightsMap {
			priorDividend := new(Float).SetInt64(0)
			if ret[workerOrReputer] != nil {
				priorDividend = ret[workerOrReputer]
			}
			marginalDividend := new(Float).SetUint64(0)
			marginalDividend.Mul(weight, reputerIncentive)
			marginalDividend.Add(marginalDividend, priorDividend)
			ret[workerOrReputer] = marginalDividend
		}
	}

	return ret
}

func sum(a map[keeper.ACC_ADDRESS]*Float) Float {
	ret := new(Float).SetInt64(0)
	for _, val := range a {
		ret.Add(ret, val)
	}
	return *ret
}

func getNormalizedEmissions(
	emissionsVector map[keeper.ACC_ADDRESS]*Float,
	emissionSum *Float) map[keeper.ACC_ADDRESS]*Float {
	ret := make(map[keeper.ACC_ADDRESS]*Float)
	for actor, emissionsValue := range emissionsVector {
		ret[actor] = new(Float).Quo(emissionsValue, emissionSum)
	}
	return ret
}

// scalarMultiply multiplies a matrix by a scalar
// every value in the matrix individually is multiplied by the scalar
// for this case we then cast the Float back to a Uint
func scalarMultiply(
	matrix map[string]*Float,
	scalar *Float) (result map[string]*Uint, err error) {
	result = make(map[string]*Uint)
	err = nil
	for key, val := range matrix {
		val := new(Float).Mul(val, scalar)
		valBigInt, accuracy := val.Int(nil)
		if accuracy == big.Above {
			return nil, ErrScalarMultiplyNegative
		}
		valUint := cosmosMath.NewUintFromBigInt(valBigInt)
		result[key] = &valUint
	}
	return result, err
}

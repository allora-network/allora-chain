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
type Number interface {
	*Float | *cosmosMath.Uint
}

// constants
const EPOCH_LENGTH = 5

// errors defined in this file
var ErrInvalidLastUpdate = errors.New("invalid last update")
var ErrEpochNotReached = errors.New("not enough blocks have passed to hit an epoch")
var ErrScalarMultiplyNegative = errors.New("token rewards multiplication output should always be positive")
var ErrDivideMapValuesByZero = errors.New("cannot divide map values by zero")

// ********************************************************
// *        PUBLIC EXPORTED READ-ONLY FUNCTIONS           *
// ********************************************************

// Given the current block number, how many new tokens are going to be emitted globally?
func GetCumulativeEmission(ctx sdk.Context, am AppModule, blocksSinceLastUpdate uint64) Uint {
	// number of epochs that have passed (if more than 1)
	epochsPassed := cosmosMath.NewUint(blocksSinceLastUpdate / uint64(EPOCH_LENGTH))

	// get emission amount
	perEpochEmission := cosmosMath.NewUintFromBigInt(am.keeper.GetAccumulatedEpochRewards(ctx).Amount.BigInt())
	cumulativeEmission := epochsPassed.Mul(perEpochEmission)
	return cumulativeEmission
}

// For a given topic:
// given the sum total of all stake in that topic,
// given the amount of new tokens scheduled to be emitted this epoch,
// given the total amount of stake in the network,
// return the amount of new tokens to be emitted to each partipicant in that topic
func GetParticipantEmissionsForTopic(
	ctx sdk.Context,
	am AppModule,
	topicId keeper.TOPIC_ID,
	topicStake *Uint,
	cumulativeEmission *Uint,
	totalStake *Uint) (reputerEmissions map[keeper.REPUTERS]*Uint, workerEmissions map[keeper.WORKERS]*Uint, err error) {

	// get total emission for topic
	topicEmissionXStake := cumulativeEmission.Mul(*topicStake)
	topicEmissions := topicEmissionXStake.Quo(*totalStake)

	// get all reputers in that topic
	// get all normalized stakes of those reputers
	topicStakeFloat := big.NewFloat(0).SetInt(topicStake.BigInt())
	reputerStakeNorm, err := am.keeper.GetReputerNormalizedStake(ctx, topicId, topicStakeFloat)
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
	incentive, err := normalize(ranks)
	if err != nil {
		return nil, nil, err
	}

	// BondDeltas using elementwise multiplication of the same vector for all rows of Weight matrix.
	// i.e. For each row i: BondDelta_ij = Weights_ij x Stake_j
	bondDeltas := elementWiseProduct(topicWeights, reputerStakeNorm)

	// Row-wise normalize BondDeltas
	bondDeltasNorm, err := normalizeBondDeltas(bondDeltas)
	if err != nil {
		return nil, nil, err
	}

	// Dividends = normalize(BondDeltas matmul Incentive)
	dividends := matmul(bondDeltasNorm, incentive)
	dividendsNorm, err := normalize(dividends)
	if err != nil {
		return nil, nil, err
	}

	// EmissionSum = sum(Dividends) + sum(Incentives)
	dividendsSum := sumMapValues(dividendsNorm)
	incentivesSum := sumMapValues(incentive)
	emissionSum := big.NewFloat(0).Add(&dividendsSum, &incentivesSum)

	topicEmissionsFloat := big.NewFloat(0).SetInt(topicEmissions.BigInt())
	if big.NewFloat(0).SetInt64(0).Cmp(emissionSum) == 0 {
		// If EmissionSum == 0 then set NormalizedReputerEmissions to normalized stake
		reputerEmissionsNorm := reputerStakeNorm

		// ValidatorEmissions = scalar multiply topicEmissionsTotal x NormalizedReputerEmissions
		reputerEmissions, err = scalarMultiply(reputerEmissionsNorm, topicEmissionsFloat)
		if err != nil {
			return nil, nil, err
		}

	} else {
		// NormalizedServerEmissions = Incentives scalar divide EmmissionSum
		normalizedWorkerEmissions, err := divideMapValues(incentive, emissionSum)
		if err != nil {
			return nil, nil, err
		}

		// NormalizedValidatorEmissions = Dividends scalar divide EmmissionSum
		normalizedReputerEmissions, err := divideMapValues(dividends, emissionSum)
		if err != nil {
			return nil, nil, err
		}

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

// ********************************************************
// *            PRIVATE STATE CHANGING FUNCTIONS          *
// ********************************************************

// The function that performs the emission of new tokens
func emitRewards(ctx sdk.Context, am AppModule, blocksSinceLastUpdate uint64) error {
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

	// how many new tokens are going to be emitted globally?
	cumulativeEmission := GetCumulativeEmission(ctx, am, blocksSinceLastUpdate)

	// Save/set the above emissions to actually pay participants.
	// Do this by increasing the stake of each worker by their due ServerEmission + ValidatorEmission
	err = am.keeper.SetLastRewardsUpdate(ctx, ctx.BlockHeight())
	if err != nil {
		fmt.Println("Error setting last rewards update: ", err)
		return err
	}

	// use anonymous function to iterate through each (topic, sumStakeForTopic)
	funcEachTopic := func(topicId keeper.TOPIC_ID, topicStake Uint) (bool, error) {
		// for each topic get percentage of total emissions
		// then get each participant's percentage of that percentage
		reputerEmissions, workerEmissions, err := GetParticipantEmissionsForTopic(
			ctx,
			am,
			topicId,
			&topicStake,
			&cumulativeEmission,
			&totalStake)
		if err != nil {
			fmt.Println("Error getting topic participant emissions: ", err)
			return true, err
		}

		// Mint new tokens to the participants of that topic
		emitRewardsToTopicParticipants(ctx, am, topicId, reputerEmissions, workerEmissions)
		return false, nil
	}

	// Iterate through each (topic, sumStakeForTopic) and run funcEachTopic for each topic
	err = am.keeper.WalkAllTopicStake(ctx, funcEachTopic)
	if err != nil {
		fmt.Println("Error getting all topic stake: ", err)
		return err
	}

	return nil
}

// this function addStake to each participant of a topic according
// to how much stake the reputer/workerEmissions maps say to add
func emitRewardsToTopicParticipants(
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

// ********************************************************
// *              PRIVATE HELPER FUNCTIONS                *
// ********************************************************

// matmul multiplies a matrix by a vector where both are stored in golang maps
// the index to the map is considered the row or column
// 0 values are taken to be not found in the map and so skipped during addition
// for matrix * vector, iterating through rows i and columns j,
// result_j = result_j + matrix_ij * vector_i
//
// EXAMPLE:
// vector = { 1, 2 }
// matrix = { { 1, 2, 3 }, { 4, 5, 6 } }
// output = { 1*1 + 2*4, 1*2 + 2*5, 1*3 + 6*2}
// output = { 9, 12, 15 }
// or represented as a map:
// vector = { "a": 1, "b": 2 }
// matrix = { "a": { "c": 1, "d": 2, "e": 3 }, "b": { "c": 4, "d": 5, "e": 6 } }
// output = { "c": 1*1 + 2*4, "d": 1*2 + 2*5, "e": 1*3 + 6*2}
// output = { "c": 9, "d": 12, "e": 15 }
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
			priorResult := big.NewFloat(0)
			if result[j] != nil {
				priorResult = result[j]
			}
			deltaResult := big.NewFloat(0)
			switch m_ij := any(matrix_ij).(type) {
			case *cosmosMath.Uint:
				f := big.NewFloat(0)
				f.SetInt(m_ij.BigInt())
				deltaResult.Mul(f, vec_i)
			case *Float:
				deltaResult.Mul(m_ij, vec_i)
			default:
				panic("matmul: unknown input type")
			}
			deltaResult.Add(deltaResult, priorResult)
			result[j] = deltaResult
		}
	}
	return result
}

// normalize divides every value in a map by the sum of all values in the map
func normalize(a map[string]*Float) (map[string]*Float, error) {
	if len(a) == 0 {
		return a, nil
	}
	sum := big.NewFloat(0)
	for _, val := range a {
		sum.Add(sum, val)
	}
	return divideMapValues(a, sum)
}

// divideMapValues divides every value in a map by the divisor provided
func divideMapValues(
	a map[string]*Float,
	divisor *Float) (map[string]*Float, error) {
	if divisor.Cmp(big.NewFloat(0)) == 0 {
		return nil, ErrDivideMapValuesByZero
	}
	ret := make(map[keeper.ACC_ADDRESS]*Float)
	for key, val := range a {
		ret[key] = big.NewFloat(0).Quo(val, divisor)
	}
	return ret, nil
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
		result[i] = make(map[string]*Float)
		vec_i := vector[i]
		if vec_i == nil {
			continue
		}
		for j, matrix_ij := range rowMap {
			matrix_ijFloat := big.NewFloat(0).SetInt(matrix_ij.BigInt())
			result[i][j] = big.NewFloat(0).Mul(matrix_ijFloat, vec_i)
		}
	}
	return result
}

// Row-wise normalizes BondDeltas. For each row, normalizes the values in that row relative to the row
func normalizeBondDeltas(bondDeltas map[keeper.REPUTERS]map[keeper.WORKERS]*Float) (result map[keeper.REPUTERS]map[keeper.WORKERS]*Float, err error) {
	result = make(map[keeper.REPUTERS]map[keeper.WORKERS]*Float)
	for reputer, workerWeights := range bondDeltas {
		result[reputer], err = normalize(workerWeights)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// sumMapValues adds all values in a map together and returns the result
func sumMapValues(a map[string]*Float) Float {
	ret := big.NewFloat(0)
	for _, val := range a {
		ret.Add(ret, val)
	}
	return *ret
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
		val := big.NewFloat(0).Mul(val, scalar)
		if val.Sign() == -1 {
			return nil, ErrScalarMultiplyNegative
		}
		valBigInt, _ := val.Int(nil)
		valUint := cosmosMath.NewUintFromBigInt(valBigInt)
		result[key] = &valUint
	}
	return result, err
}

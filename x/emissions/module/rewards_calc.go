package module

import (
	"fmt"
	"math/big"

	cosmosMath "cosmossdk.io/math"
	params "github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Uint = cosmosMath.Uint
type Float = big.Float
type Number interface {
	*Float | *cosmosMath.Uint
}

// ********************************************************
// *        PUBLIC EXPORTED READ-ONLY FUNCTIONS           *
// ********************************************************
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
	accumulatedMetDemand *Uint,
	totalStake *Uint) (rewards map[string]*Uint, err error) {
	// get total emission for topic
	topicEmissionXStake := cumulativeEmission.Mul(*topicStake)
	topicEmissions := topicEmissionXStake.Quo(*totalStake).Add(*accumulatedMetDemand)

	// get all reputers in that topic
	// get all normalized stakes of those reputers
	topicStakeFloat := big.NewFloat(0).SetInt(topicStake.BigInt())
	reputerStakeNorm, err := am.keeper.GetReputerNormalizedStake(ctx, topicId, topicStakeFloat)
	if err != nil {
		fmt.Println("Error getting reputer normalized stake: ", err)
		return nil, err
	}

	// Get Weights between nodes in topic
	//    Weight_ij = reputer i -> worker j -> weight val
	topicWeights, err := am.keeper.GetWeightsFromTopic(ctx, topicId)
	if err != nil {
		fmt.Println("Error getting weights from topic: ", err)
		return nil, err
	}
	// Mask inferences if workers admit insufficient liveness
	maskedTopicWeights, err := MaskWeightsIfInsufficientLiveness(ctx, am, topicId, topicWeights)
	if err != nil {
		fmt.Println("Error masking weights if insufficient liveness: ", err)
		return nil, err
	}

	// Ranks = matmul Weights * NormalizedStake
	// for i rows and j columns
	// i.e. rank[j] = sum(j) + weight_ij * normalizedStake_i
	ranks := matmul(maskedTopicWeights, reputerStakeNorm)

	// Incentive = normalize(Ranks)
	incentive, err := normalize(ranks)
	if err != nil {
		// if error is ErrDivideMapValuesByZero (e.g. because no workers were live => all weights masked) then return empty rewards
		if err == state.ErrDivideMapValuesByZero {
			return make(map[string]*Uint), nil
		}
		fmt.Println("Error normalizing ranks: ", err)
		return nil, err
	}

	// BondDeltas using elementwise multiplication of the same vector for all rows of Weight matrix.
	// i.e. For each row i: BondDelta_ij = Weights_ij x Stake_j
	bondDeltas := elementWiseProduct(maskedTopicWeights, reputerStakeNorm)

	// Row-wise normalize BondDeltas
	bondDeltasNorm, err := normalizeBondDeltas(bondDeltas)
	if err != nil {
		fmt.Println("Error normalizing bond deltas: ", err)
		return nil, err
	}

	// Dividends = normalize(BondDeltas matmul Incentive)
	dividends := matmul(bondDeltasNorm, incentive)
	dividendsNorm, err := normalize(dividends)
	if err != nil {
		fmt.Println("Error normalizing dividends: ", err)
		return nil, err
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
		rewards, err = scalarMultiply(reputerEmissionsNorm, topicEmissionsFloat)
		if err != nil {
			fmt.Println("Error scalar multiplying reputer emissions: ", err)
			return nil, err
		}

	} else {
		// NormalizedServerEmissions = Incentives scalar divide EmmissionSum
		normalizedWorkerEmissions, err := divideMapValues(incentive, emissionSum)
		if err != nil {
			fmt.Println("Error dividing incentives by emission sum: ", err)
			return nil, err
		}

		// NormalizedValidatorEmissions = Dividends scalar divide EmmissionSum
		normalizedReputerEmissions, err := divideMapValues(dividends, emissionSum)
		if err != nil {
			fmt.Println("Error dividing dividends by emission sum: ", err)
			return nil, err
		}

		// ServerEmissions = scalar multiply topicEmissionsTotal x NormalizedServerEmissions
		workerEmissions, err := scalarMultiply(normalizedWorkerEmissions, topicEmissionsFloat)
		if err != nil {
			fmt.Println("Error scalar multiplying worker emissions: ", err)
			return nil, err
		}

		// ValidatorEmissions = scalar multiply topicEmissionsTotal x NormalizedValidatorEmissions
		reputerEmissions, err := scalarMultiply(normalizedReputerEmissions, topicEmissionsFloat)
		if err != nil {
			fmt.Println("Error scalar multiplying reputer emissions: ", err)
			return nil, err
		}
		rewards = mapAdd(reputerEmissions, workerEmissions)
	}
	return rewards, nil
}

// This function checks topic weights and then masks them if not enough inferences were collected in that timestep
// It should:
//
//	mask the inputted weights above with GetNumInferencesInRewardEpoch(),
//	checking inference cadence by reward epoch length / topic inference cadence
//	simple forgiveness check (only if many inferences are missing, don't reward)
func MaskWeightsIfInsufficientLiveness(
	ctx sdk.Context,
	am AppModule,
	topicId keeper.TOPIC_ID,
	weights map[string]map[string]*Uint) (map[string]map[string]*Uint, error) {
	maskedWeights := make(map[string]map[string]*Uint)
	for reputer, workerWeights := range weights {

		if maskedWeights[reputer] == nil {
			maskedWeights[reputer] = make(map[string]*Uint)
		}

		for worker := range workerWeights {
			// Get the topic => its inference cadence
			topic, err := am.keeper.GetTopic(ctx, topicId)
			if err != nil {
				return nil, err
			}
			// Get the number of inferences in the reward epoch
			workerAddress, err := sdk.AccAddressFromBech32(worker)
			if err != nil {
				return nil, err
			}
			numInferencesInRewardEpoch, err := am.keeper.GetNumInferencesInRewardEpoch(ctx, topicId, workerAddress)
			if err != nil {
				return nil, err
			}
			// If number of inferences in the reward epoch < amount that should be there by too much, then mask the weight
			epochLength, err := am.keeper.GetParamsEpochLength(ctx)
			if err != nil {
				return nil, err
			}
			maxPossibleInferencesInRewardEpoch := uint64(epochLength) / topic.InferenceCadence
			// Allow for for 10% of inferences to be missing. Percent directly encoded as cosmosMath.LegacyDec
			maxAllowableMissingInferencePercent, err := am.keeper.GetParamsMaxMissingInferencePercent(ctx)
			if err != nil {
				return nil, err
			}
			expectedNumInferencesInRewardEpoch := cosmosMath.LegacyOneDec().Sub(maxAllowableMissingInferencePercent).MulInt(
				cosmosMath.NewIntFromUint64(maxPossibleInferencesInRewardEpoch)).TruncateInt()
			if numInferencesInRewardEpoch.LT(cosmosMath.NewUintFromBigInt(expectedNumInferencesInRewardEpoch.BigInt())) {
				maskedVal := cosmosMath.ZeroUint()
				maskedWeights[reputer][worker] = &maskedVal
			} else {
				maskedWeights[reputer][worker] = weights[reputer][worker]
			}
		}
	}
	return maskedWeights, nil
}

// ********************************************************
// *            PRIVATE STATE CHANGING FUNCTIONS          *
// ********************************************************

// The function that performs the emission of new tokens
func emitRewards(ctx sdk.Context, am AppModule) error {
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
	emissionsAddress := am.keeper.AccountKeeper().GetModuleAddress(state.AlloraRewardsAccountName)
	emissionsBalance := am.keeper.BankKeeper().GetBalance(ctx, emissionsAddress, params.DefaultBondDenom)
	cumulativeEmission := cosmosMath.NewUintFromBigInt(emissionsBalance.Amount.BigInt())

	// Save/set the above emissions to actually pay participants.
	// Do this by increasing the stake of each worker by their due ServerEmission + ValidatorEmission
	err = am.keeper.SetLastRewardsUpdate(ctx, ctx.BlockHeight())
	if err != nil {
		fmt.Println("Error setting last rewards update: ", err)
		return err
	}

	// use anonymous function to iterate through each (topic, sumStakeForTopic)
	funcEachTopic := func(topicId keeper.TOPIC_ID, topicStake Uint) (bool, error) {
		accumulatedMetDemand, err := am.keeper.GetTopicAccumulatedMetDemand(ctx, topicId)
		if err != nil {
			fmt.Println("Error getting accumulated met demand: ", err)
			return true, err
		}
		// for each topic get percentage of total emissions
		// then get each participant's percentage of that percentage
		rewards, err := GetParticipantEmissionsForTopic(
			ctx,
			am,
			topicId,
			&topicStake,
			&cumulativeEmission,
			&accumulatedMetDemand,
			&totalStake)
		if err != nil {
			fmt.Println("Error getting participant emissions for topic: ", err)
			return true, err
		}
		// if no rewards to give, do nothing
		if len(rewards) == 0 {
			fmt.Printf(" No rewards to emit for Topic %v \n", topicId)
			return false, nil
		}

		// Mint new tokens to the participants of that topic
		emitRewardsToTopicParticipants(ctx, am, topicId, rewards)
		am.keeper.SetTopicAccumulatedMetDemand(ctx, topicId, cosmosMath.ZeroUint())
		return false, nil
	}

	// Iterate through each (topic, sumStakeForTopic) and run funcEachTopic for each topic
	err = am.keeper.WalkAllTopicStake(ctx, funcEachTopic)
	if err != nil {
		fmt.Println("Error walking all topic stake: ", err)
		return err
	}

	return am.keeper.ResetNumInferencesInRewardEpoch(ctx)
}

// this function addStake to each participant of a topic according
// to how much stake the reputer/workerEmissions maps say to add
func emitRewardsToTopicParticipants(
	ctx sdk.Context,
	am AppModule,
	topic keeper.TOPIC_ID,
	rewards map[string]*Uint) {
	// by default emissions are restaked, upon the person themselves.
	fmt.Printf("\n---------------- Rewards for Topic %v ----------------\n", topic)
	for participant, reward := range rewards {
		fmt.Printf(" Emitting %suallo to %s \n", reward.String(), participant)
		am.keeper.AddStake(ctx, []uint64{topic}, participant, participant, *reward)
		rewardCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(reward.BigInt())))
		am.keeper.BankKeeper().SendCoinsFromModuleToModule(ctx, state.AlloraRewardsAccountName, state.AlloraStakingAccountName, rewardCoins)
	}
	if len(rewards) == 0 {
		fmt.Printf(" No rewards to emit for Topic %v \n", topic)
	}
	fmt.Println("\n-----------------------------------------")
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
		return nil, state.ErrDivideMapValuesByZero
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
			return nil, state.ErrScalarMultiplyNegative
		}
		valBigInt, _ := val.Int(nil)
		valUint := cosmosMath.NewUintFromBigInt(valBigInt)
		result[key] = &valUint
	}
	return result, err
}

// mapAdd adds two maps together, summing the values of the same keys
func mapAdd(a map[string]*Uint, b map[string]*Uint) (result map[string]*Uint) {
	result = make(map[string]*Uint)
	for key, val := range a {
		val2, ok := b[key]
		if ok {
			sum := val.Add(*val2)
			result[key] = &sum
		} else {
			result[key] = val
		}
	}
	for key, val := range b {
		_, ok := a[key]
		if !ok {
			result[key] = val
		}
	}
	return result
}

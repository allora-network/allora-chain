package invariant_test

import (
	"context"
	"errors"
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testcommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

// set up the common state for the simulator
// prior to either doing random simulation
// or manual simulation
func simulateSetUp(
	m *testcommon.TestConfig,
	numActors int,
	epochLength int,
	mode SimulationMode,
) (
	faucet Actor,
	simulationData *SimulationData,
) {
	// fund all actors from the faucet with some amount
	// give everybody the same amount of money to start with
	actorsList := createActors(m, numActors)
	faucet = Actor{
		name: getFaucetName(m.Seed),
		addr: m.FaucetAddr,
		acc:  m.FaucetAcc,
	}
	preFundAmount, err := getPreFundAmount(m, faucet, numActors)
	if err != nil {
		m.T.Fatal(err)
	}
	err = fundActors(
		m,
		faucet,
		actorsList,
		preFundAmount,
	)
	if err != nil {
		m.T.Fatal(err)
	}
	data := SimulationData{
		epochLength: int64(epochLength),
		actors:      actorsList,
		counts: StateTransitionCounts{
			createTopic:                0,
			fundTopic:                  0,
			registerWorker:             0,
			registerReputer:            0,
			unregisterWorker:           0,
			unregisterReputer:          0,
			stakeAsReputer:             0,
			delegateStake:              0,
			unstakeAsReputer:           0,
			undelegateStake:            0,
			cancelStakeRemoval:         0,
			cancelDelegateStakeRemoval: 0,
			collectDelegatorRewards:    0,
			doInferenceAndReputation:   0,
		},
		registeredWorkers:  testcommon.NewRandomKeyMap[Registration, struct{}](m.Client.Rand),
		registeredReputers: testcommon.NewRandomKeyMap[Registration, struct{}](m.Client.Rand),
		reputerStakes: testcommon.NewRandomKeyMap[Registration, cosmossdk_io_math.Int](
			m.Client.Rand,
		),
		delegatorStakes: testcommon.NewRandomKeyMap[Delegation, cosmossdk_io_math.Int](
			m.Client.Rand,
		),
		mode:      mode,
		failOnErr: false,
	}
	// if we're in manual mode or behaving mode we want to fail on errors
	if mode == Manual || mode == Behave {
		data.failOnErr = true
	}
	return faucet, &data
}

// creates a new actor and registers them in the nodes account registry
func createNewActor(m *testcommon.TestConfig, numActors int) Actor {
	actorName := getActorName(m.Seed, numActors)
	actorAccount, _, err := m.Client.AccountRegistryCreate(actorName)
	if err != nil {
		if errors.Is(err, cosmosaccount.ErrAccountExists) {
			m.T.Log("WARNING WARNING WARNING\nACTOR ACCOUNTS ALREADY EXIST, YOU ARE REUSING YOUR SEED VALUE\nNON-DETERMINISM-DRAGONS AHEAD\nWARNING WARNING WARNING")
			actorAccount, err := m.Client.AccountRegistryGetByName(actorName)
			if err != nil {
				m.T.Log("Error getting actor account: ", actorName, " - ", err)
				return UnusedActor
			}
			actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
			if err != nil {
				m.T.Log("Error creating actor address: ", actorName, " - ", err)
				return UnusedActor
			}
			return Actor{
				name: actorName,
				addr: actorAddress,
				acc:  actorAccount,
			}
		} else {
			m.T.Log("Error creating actor address: ", actorName, " - ", err)
			return UnusedActor
		}
	}
	actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
	if err != nil {
		m.T.Log("Error creating actor address: ", actorName, " - ", err)
		return UnusedActor
	}
	return Actor{
		name: actorName,
		addr: actorAddress,
		acc:  actorAccount,
	}
}

// creates a list of actors both as a map and a slice, returns both
func createActors(m *testcommon.TestConfig, numToCreate int) []Actor {
	actorsList := make([]Actor, numToCreate)
	for i := 0; i < numToCreate; i++ {
		actorsList[i] = createNewActor(m, i)
	}
	return actorsList
}

// fund every target address from the sender in amount coins
func fundActors(
	m *testcommon.TestConfig,
	sender Actor,
	targets []Actor,
	amount cosmossdk_io_math.Int,
) error {
	inputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(
			params.BaseCoinUnit,
			amount.MulRaw(int64(len(targets))),
		),
	)
	outputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(params.BaseCoinUnit, amount),
	)

	outputs := make([]banktypes.Output, len(targets))
	names := make([]string, len(targets))
	i := 0
	for _, actor := range targets {
		names[i] = actor.name
		outputs[i] = banktypes.Output{
			Address: actor.addr,
			Coins:   outputCoins,
		}
		i++
	}

	// Fund the accounts from faucet account in a single transaction
	sendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: sender.addr,
				Coins:   inputCoins,
			},
		},
		Outputs: outputs,
	}
	ctx := context.Background()
	_, err := m.Client.BroadcastTx(ctx, sender.acc, sendMsg)
	if err != nil {
		m.T.Log("Error worker address: ", err)
		return err
	}
	m.T.Log(
		"Funded ",
		len(targets),
		" accounts from ",
		sender.name,
		" with ",
		amount,
		" coins:",
		" ",
		names,
	)
	return nil
}

// get the amount of money to give each actor in the simulation
// based on how much money the faucet currently has
func getPreFundAmount(
	m *testcommon.TestConfig,
	faucet Actor,
	numActors int,
) (cosmossdk_io_math.Int, error) {
	faucetBal, err := faucet.GetBalance(m)
	if err != nil {
		return cosmossdk_io_math.ZeroInt(), err
	}
	// divide by 10 so you can at least run 10 runs
	amountForThisRun := faucetBal.QuoRaw(int64(10))
	ret := amountForThisRun.QuoRaw(int64(numActors))
	if ret.Equal(cosmossdk_io_math.ZeroInt()) || ret.IsNegative() {
		return cosmossdk_io_math.ZeroInt(), fmt.Errorf(
			"Not enough funds in faucet account to fund actors",
		)
	}
	return ret, nil
}

// how much money an actor has
func (a *Actor) GetBalance(m *testcommon.TestConfig) (cosmossdk_io_math.Int, error) {
	ctx := context.Background()
	bal, err := m.Client.QueryBank().
		Balance(ctx, banktypes.NewQueryBalanceRequest(sdktypes.MustAccAddressFromBech32(a.addr), params.DefaultBondDenom))
	if err != nil {
		m.T.Logf("Error getting balance of actor %s: %v\n", a.String(), err)
		return cosmossdk_io_math.ZeroInt(), err
	}
	return bal.Balance.Amount, nil
}

// for initial state for the automatic test
// 5 workers, 4 reputers, and 2 delegators
// each set unique actors, no actor repeated anywhere
func pickAutoSetupActors(m *testcommon.TestConfig, data *SimulationData) (reputers []Actor, workers []Actor, delegators []Actor) {
	numReputers := 4
	numWorkers := 5
	numDelegators := 2
	totalActorsForSetup := numReputers + numWorkers + numDelegators

	reputers = make([]Actor, numReputers)
	workers = make([]Actor, numWorkers)
	delegators = make([]Actor, numDelegators)
	require.GreaterOrEqual(
		m.T,
		len(data.actors),
		totalActorsForSetup,
		"not enough actors to do the setup, must have at least %d actors: have %d",
		totalActorsForSetup,
		len(data.actors),
	)

	for i := 0; i < numReputers; i++ {
		newActor := data.actors[i]
		reputers[i] = newActor
	}

	for i := 0; i < numWorkers; i++ {
		newActor := data.actors[numReputers+i]
		workers[i] = newActor
	}

	for i := 0; i < numDelegators; i++ {
		newActor := data.actors[numReputers+numWorkers+i]
		delegators[i] = newActor
	}

	return reputers, workers, delegators
}

// startRegisterReputers registers and then stakes a list of reputers to a list of topics.
func startRegisterReputers(
	m *testcommon.TestConfig,
	data *SimulationData,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, reputer := range startReputers {
		for _, topicId := range listTopics {
			// register reputer on the topic
			registerReputer(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			iterationCount++
			// stake reputer on the topic
			bal, err := pickRandomBalanceLessThanHalf(m, reputer)
			requireNoError(m.T, true, err)
			stakeAsReputer(m, reputer, UnusedActor, &bal, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startRegisterWorkers registers and then stakes a list of workers to a list of topics.
func startRegisterWorkers(
	m *testcommon.TestConfig,
	data *SimulationData,
	startWorkers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, worker := range startWorkers {
		for _, topicId := range listTopics {
			registerWorker(m, worker, UnusedActor, nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startDelegateDelegators delegates a list of delegators to a list of reputers on a list of topics.
func startDelegateDelegators(
	m *testcommon.TestConfig,
	data *SimulationData,
	startDelegators []Actor,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for i, delegator := range startDelegators {
		for _, topicId := range listTopics {
			bal, err := pickRandomBalanceLessThanHalf(m, delegator)
			requireNoError(m.T, true, err)
			delegateStake(m, delegator, startReputers[i], &bal, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startFundTopics funds the topics with random amounts of money
func startFundTopics(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range listTopics {
		fundAmount, err := pickRandomBalanceLessThanHalf(m, faucet)
		requireNoError(m.T, true, err)
		fundTopic(m, faucet, UnusedActor, &fundAmount, topicId, data, iterationCount)
		iterationCount++
	}
	return iterationCount
}

// startDoInferenceAndReputation does inference and reputation for both topics
func startDoInferenceAndReputation(
	m *testcommon.TestConfig,
	data *SimulationData,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range listTopics {
		doInferenceAndReputation(m, UnusedActor, UnusedActor, nil, topicId, data, iterationCount)
		iterationCount++
	}
	return iterationCount
}

// collect delegator rewards for delegators on reputers on topics
func startCollectDelegatorRewards(
	m *testcommon.TestConfig,
	data *SimulationData,
	startDelegators []Actor,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for i, delegator := range startDelegators {
		for _, topicId := range listTopics {
			collectDelegatorRewards(m, delegator, startReputers[i], nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startUnregisterWorkers unregisters a list of workers from a list of topics.
func startUnregisterWorkers(
	m *testcommon.TestConfig,
	data *SimulationData,
	startWorkers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, worker := range startWorkers {
		for _, topicId := range listTopics {
			unregisterWorker(m, worker, UnusedActor, nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startUnregisterReputers unregisters a list of reputers from a list of topics.
func startUnregisterReputers(
	m *testcommon.TestConfig,
	data *SimulationData,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, reputer := range startReputers {
		for _, topicId := range listTopics {
			unregisterReputer(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startUndelegateStake undelegates a list of delegators from a list of reputers on a list of topics.
func startUndelegateStake(
	m *testcommon.TestConfig,
	data *SimulationData,
	startDelegators []Actor,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for i, delegator := range startDelegators {
		for _, topicId := range listTopics {
			amount := data.pickPercentOfStakeByDelegator(m.Client.Rand, topicId, delegator, startReputers[i])
			undelegateStake(m, delegator, startReputers[i], &amount, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startUnstakeAsReputer unstakes a list of reputers from a list of topics.
func startUnstakeAsReputer(
	m *testcommon.TestConfig,
	data *SimulationData,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, reputer := range startReputers {
		for _, topicId := range listTopics {
			amount := data.pickPercentOfStakeByReputer(m.Client.Rand, topicId, reputer)
			unstakeAsReputer(m, reputer, UnusedActor, &amount, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startCancelStakeRemoval cancels the removal of stake from a list of reputers on a list of topics.
func startCancelStakeRemoval(
	m *testcommon.TestConfig,
	data *SimulationData,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, reputer := range startReputers {
		for _, topicId := range listTopics {
			cancelStakeRemoval(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// startCancelDelegateStakeRemoval cancels the removal of delegated stake from a list of delegators on a list of topics.
// startReputers must correspond to the reputers startDelegators are staked upon
func startCancelDelegateStakeRemoval(
	m *testcommon.TestConfig,
	data *SimulationData,
	startDelegators []Actor,
	startReputers []Actor,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for i, delegator := range startDelegators {
		for _, topicId := range listTopics {
			cancelDelegateStakeRemoval(m, delegator, startReputers[i], nil, topicId, data, iterationCount)
			iterationCount++
		}
	}
	return iterationCount
}

// run every state transition, at least once.
func simulateAutomaticInitialState(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
) (iterationCountAfter int) {
	iterationCount := 0
	listTopics := []uint64{1, 2}

	// make sure that the setup always fails on error
	failOnErrWanted := data.failOnErr
	data.failOnErr = true

	// additive actions

	// create two topics
	createTopic(m, faucet, UnusedActor, nil, 0, data, iterationCount)
	iterationCount++
	createTopic(m, faucet, UnusedActor, nil, 0, data, iterationCount)
	iterationCount++
	// pick 4 reputers, 4 workers, and 2 delegators
	startReputers, startWorkers, startDelegators := pickAutoSetupActors(m, data)
	// register all 4 reputers on both topics
	iterationCount = startRegisterReputers(m, data, startReputers, listTopics, iterationCount)
	// register all 5 workers on both topics
	iterationCount = startRegisterWorkers(m, data, startWorkers, listTopics, iterationCount)
	// delegate stake to both topics from the delegators
	iterationCount = startDelegateDelegators(m, data, startDelegators, startReputers, listTopics, iterationCount)
	// fund the topics
	iterationCount = startFundTopics(m, faucet, data, listTopics, iterationCount)
	// do inference and reputation for both topics
	iterationCount = startDoInferenceAndReputation(m, data, listTopics, iterationCount)
	// collect delegator rewards for both topics
	iterationCount = startCollectDelegatorRewards(m, data, startDelegators, startReputers, listTopics, iterationCount)

	// subtractive actions

	unregisterWorkers := []Actor{startWorkers[0], startWorkers[1]}
	unregisterReputer := []Actor{startReputers[1]}

	unStakeReputer := []Actor{startReputers[0]}
	unStakeDelegator := []Actor{startDelegators[0]}
	unStakeDelegatorReputer := []Actor{startReputers[0]}

	justFirstTopic := []uint64{1}

	// unregister 2 workers from topic 1
	iterationCount = startUnregisterWorkers(m, data, unregisterWorkers, justFirstTopic, iterationCount)
	// unregister 1 reputer from topic 1
	iterationCount = startUnregisterReputers(m, data, unregisterReputer, justFirstTopic, iterationCount)
	// undelegate 1 delegator from 1 reputers on topic 1
	iterationCount = startUndelegateStake(m, data, unStakeDelegator, unStakeReputer, justFirstTopic, iterationCount)
	// unstake 1 reputer on topic 1
	iterationCount = startUnstakeAsReputer(m, data, unStakeReputer, justFirstTopic, iterationCount)

	// cancel the removal of stake from 1 reputer on topic 1
	iterationCount = startCancelStakeRemoval(m, data, unStakeReputer, justFirstTopic, iterationCount)
	// cancel the removal of delegated stake from 1 delegator on reputer 1 on topic 1
	iterationCount = startCancelDelegateStakeRemoval(m, data, unStakeDelegator, unStakeDelegatorReputer, justFirstTopic, iterationCount)

	// set back the failOnErr status the user requested for the fuzz run
	data.failOnErr = failOnErrWanted

	return iterationCount
}

package fuzz_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testcommon "github.com/allora-network/allora-chain/test/common"
	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
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
	mode fuzzcommon.SimulationMode,
	seed int,
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

	// ensure each random key map has a different random number generator
	// so that map accesses don't step on each other
	registeredWorkersMapRand := rand.New(rand.NewSource(int64(seed)))
	registeredReputersMapRand := rand.New(rand.NewSource(int64(seed)))
	reputerStakesMapRand := rand.New(rand.NewSource(int64(seed)))
	delegatorStakesMapRand := rand.New(rand.NewSource(int64(seed)))
	topicCreatorsMapRand := rand.New(rand.NewSource(int64(seed)))
	adminWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	globalWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	topicCreatorsWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	topicWorkersWhitelistEnabledMapRand := rand.New(rand.NewSource(int64(seed)))
	topicReputersWhitelistEnabledMapRand := rand.New(rand.NewSource(int64(seed)))
	topicWorkersWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	topicReputersWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))

	registeredWorkers := testcommon.NewRandomKeyMap[Registration, struct{}](registeredWorkersMapRand)
	registeredReputers := testcommon.NewRandomKeyMap[Registration, struct{}](registeredReputersMapRand)
	reputerStakes := testcommon.NewRandomKeyMap[Registration, struct{}](reputerStakesMapRand)
	delegatorStakes := testcommon.NewRandomKeyMap[Delegation, struct{}](delegatorStakesMapRand)
	topicCreators := testcommon.NewRandomKeyMap[uint64, Actor](topicCreatorsMapRand)
	adminWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](adminWhitelistMapRand)
	globalWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](globalWhitelistMapRand)
	topicCreatorsWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](topicCreatorsWhitelistMapRand)
	topicWorkersWhitelistEnabled := testcommon.NewRandomKeyMap[uint64, struct{}](topicWorkersWhitelistEnabledMapRand)
	topicReputersWhitelistEnabled := testcommon.NewRandomKeyMap[uint64, struct{}](topicReputersWhitelistEnabledMapRand)
	topicWorkersWhitelist := testcommon.NewRandomKeyMap[TopicWhitelistEntry, struct{}](topicWorkersWhitelistMapRand)
	topicReputersWhitelist := testcommon.NewRandomKeyMap[TopicWhitelistEntry, struct{}](topicReputersWhitelistMapRand)

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
		registeredWorkers:             registeredWorkers,
		registeredReputers:            registeredReputers,
		reputerStakes:                 reputerStakes,
		delegatorStakes:               delegatorStakes,
		topicCreators:                 topicCreators,
		adminWhitelist:                adminWhitelist,
		globalWhitelist:               globalWhitelist,
		topicCreatorsWhitelist:        topicCreatorsWhitelist,
		topicWorkersWhitelistEnabled:  topicWorkersWhitelistEnabled,
		topicReputersWhitelistEnabled: topicReputersWhitelistEnabled,
		topicWorkersWhitelist:         topicWorkersWhitelist,
		topicReputersWhitelist:        topicReputersWhitelist,

		mode:      mode,
		failOnErr: false,
	}
	// if we're in manual mode or behaving mode we want to fail on errors
	if mode == fuzzcommon.Manual || mode == fuzzcommon.Behave {
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
			panic(fmt.Errorf("cannot re-use seed values across test runs, please restart the node from a clean configuration or use a different seed value"))
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

func startAddToAdminWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := addToAdminWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startAddToGlobalWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := addToGlobalWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startAddToTopicCreatorWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := addToTopicCreatorWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startAddToTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	workers []Actor,
	topics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range topics {
		for _, worker := range workers {
			success := addToTopicWorkerWhitelist(m, sender, worker, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

func startAddToTopicReputerWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	reputers []Actor,
	topics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range topics {
		for _, reputer := range reputers {
			success := addToTopicReputerWhitelist(m, sender, reputer, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

func startRemoveFromAdminWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := removeFromAdminWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startRemoveFromGlobalWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := removeFromGlobalWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startRemoveFromTopicCreatorWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	startActors []Actor,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, actor := range startActors {
		success := removeFromTopicCreatorWhitelist(m, sender, actor, nil, 0, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	}
	return iterationCount
}

func startRemoveFromTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	workers []Actor,
	topics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range topics {
		for _, worker := range workers {
			success := removeFromTopicWorkerWhitelist(m, sender, worker, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

func startRemoveFromTopicReputerWhitelist(
	m *testcommon.TestConfig,
	data *SimulationData,
	sender Actor,
	reputers []Actor,
	topics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, topicId := range topics {
		for _, reputer := range reputers {
			success := removeFromTopicReputerWhitelist(m, sender, reputer, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
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
			success := registerReputer(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
			// stake reputer on the topic
			bal, err := pickRandomBalanceLessThanHalf(m, reputer)
			failIfOnErr(m.T, true, err)
			success = stakeAsReputer(m, reputer, UnusedActor, &bal, topicId, data, iterationCount)
			require.True(m.T, success)
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
			success := registerWorker(m, worker, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
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
			failIfOnErr(m.T, true, err)
			success := delegateStake(m, delegator, startReputers[i], &bal, topicId, data, iterationCount)
			require.True(m.T, success)
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
		failIfOnErr(m.T, true, err)
		success := fundTopic(m, faucet, UnusedActor, &fundAmount, topicId, data, iterationCount)
		require.True(m.T, success)
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
		success := doInferenceAndReputation(m, UnusedActor, UnusedActor, nil, topicId, data, iterationCount)
		require.True(m.T, success)
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
			success := collectDelegatorRewards(m, delegator, startReputers[i], nil, topicId, data, iterationCount)
			require.True(m.T, success)
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
			success := unregisterWorker(m, worker, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
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
			success := unregisterReputer(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
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
			amount := pickPercentOfStakeByDelegator(m, topicId, delegator, startReputers[i], data, iterationCount)
			success := undelegateStake(m, delegator, startReputers[i], &amount, topicId, data, iterationCount)
			require.True(m.T, success)
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
			amount := pickPercentOfStakeByReputer(m, topicId, reputer, data, iterationCount)
			success := unstakeAsReputer(m, reputer, UnusedActor, &amount, topicId, data, iterationCount)
			require.True(m.T, success)
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
			success := cancelStakeRemoval(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
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
			success := cancelDelegateStakeRemoval(m, delegator, startReputers[i], nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

// run every state transition, at least once.
func simulateAutomaticInitialState(
	f *fuzzcommon.FuzzConfig,
	faucet Actor,
	data *SimulationData,
) (iterationCountAfter int) {
	m := f.TestConfig
	iterationCount := 0

	// make sure that the setup always fails on error
	failOnErrWanted := data.failOnErr
	data.failOnErr = true

	// additive actions

	// pick 4 reputers, 5 workers, and 2 delegators
	startReputers, startWorkers, startDelegators := pickAutoSetupActors(m, data)

	// add 1 delegator to all global whitelists
	iterationCount = startAddToAdminWhitelist(m, data, faucet, startDelegators[:1], iterationCount)
	iterationCount = startAddToGlobalWhitelist(m, data, faucet, startDelegators[:1], iterationCount)
	iterationCount = startAddToTopicCreatorWhitelist(m, data, faucet, startDelegators[:1], iterationCount)

	// create two topics with both reputers and workers whitelist enabled
	success := createTopic(m, startDelegators[0], UnusedActor, nil, 0, data, iterationCount)
	require.True(m.T, success)
	iterationCount++
	success = createTopic(m, startDelegators[0], UnusedActor, nil, 0, data, iterationCount)
	require.True(m.T, success)
	iterationCount++

	listTopics := data.getTopics()
	require.Len(m.T, listTopics, 2)

	// disable whitelists for topic 1
	require.True(m.T,
		disableTopicWorkerWhitelist(m, startDelegators[0], UnusedActor, nil, listTopics[0], data, iterationCount),
	)
	iterationCount++
	require.True(m.T,
		disableTopicReputerWhitelist(m, startDelegators[0], UnusedActor, nil, listTopics[0], data, iterationCount),
	)
	iterationCount++

	// put reputers & workers in topic whitelists
	iterationCount = startAddToTopicWorkerWhitelist(m, data, faucet, startWorkers, listTopics, iterationCount)
	iterationCount = startAddToTopicReputerWhitelist(m, data, faucet, startReputers, listTopics, iterationCount)

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

	justFirstTopic := listTopics[:1]

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

	// enable whitelists for topic 1
	require.True(m.T,
		enableTopicWorkerWhitelist(m, startDelegators[0], UnusedActor, nil, listTopics[0], data, iterationCount),
	)
	iterationCount++
	require.True(m.T,
		enableTopicReputerWhitelist(m, startDelegators[0], UnusedActor, nil, listTopics[0], data, iterationCount),
	)
	iterationCount++

	// remove 2 reputers & 3 workers from topic 1 topic whitelists
	iterationCount = startRemoveFromTopicWorkerWhitelist(m, data, faucet, startWorkers[:3], justFirstTopic, iterationCount)
	iterationCount = startRemoveFromTopicReputerWhitelist(m, data, faucet, startReputers[:2], justFirstTopic, iterationCount)

	// remove 1 delegator from all global whitelists
	iterationCount = startRemoveFromAdminWhitelist(m, data, faucet, startDelegators[:1], iterationCount)
	iterationCount = startRemoveFromGlobalWhitelist(m, data, faucet, startDelegators[:1], iterationCount)
	iterationCount = startRemoveFromTopicCreatorWhitelist(m, data, faucet, startDelegators[:1], iterationCount)

	// add configured actors to the global whitelists
	iterationCount = startAddToAdminWhitelist(m, data, faucet, data.pickNRandomActors(m, f.InitialSetup.NumAdminWhitelist), iterationCount)
	iterationCount = startAddToGlobalWhitelist(m, data, faucet, data.pickNRandomActors(m, f.InitialSetup.NumGlobalWhitelist), iterationCount)
	iterationCount = startAddToTopicCreatorWhitelist(m, data, faucet, data.pickNRandomActors(m, f.InitialSetup.NumTopicCreatorWhitelist), iterationCount)

	// set back the failOnErr status the user requested for the fuzz run
	data.failOnErr = failOnErrWanted

	return iterationCount
}

package fuzz_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testcommon "github.com/allora-network/allora-chain/test/common"
	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
	globalWorkerWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	globalReputerWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
	globalAdminWhitelistMapRand := rand.New(rand.NewSource(int64(seed)))
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
	globalWorkerWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](globalWorkerWhitelistMapRand)
	globalReputerWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](globalReputerWhitelistMapRand)
	globalAdminWhitelist := testcommon.NewRandomKeyMap[Actor, struct{}](globalAdminWhitelistMapRand)
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
		globalWorkerWhitelist:         globalWorkerWhitelist,
		globalReputerWhitelist:        globalReputerWhitelist,
		globalAdminWhitelist:          globalAdminWhitelist,
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
	weights setupWeights,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for _, reputer := range startReputers {
		base, err := pickRandomBalanceLessThanHalf(m, reputer)
		failIfOnErr(m.T, true, err)
		for _, topicId := range listTopics {
			// register reputer on the topic
			success := registerReputer(m, reputer, UnusedActor, nil, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
			// stake reputer on the topic
			bal := weights.amountFor(topicId, base)
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
	weights setupWeights,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	for i, delegator := range startDelegators {
		base, err := pickRandomBalanceLessThanHalf(m, delegator)
		failIfOnErr(m.T, true, err)
		for _, topicId := range listTopics {
			bal := weights.amountFor(topicId, base)
			success := delegateStake(m, delegator, startReputers[i], &bal, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

// setupWeights decides how much stake, delegation and funding each setup topic receives.
// With uneven weights the heavy topic gets the base amount and every other topic a
// millionth of it, floored at the chain's minimum stake, so the heavy topic wins any block
// that is over the per-block limit.
type setupWeights struct {
	heavyTopicId uint64
	uneven       bool
	minAmount    cosmossdk_io_math.Int
}

const lightTopicDivisor = int64(1_000_000)

func newSetupWeights(m *testcommon.TestConfig, heavyTopicId uint64, uneven bool) setupWeights {
	minAmount := cosmossdk_io_math.OneInt()
	if uneven {
		ctx := context.Background()
		paramsResp, err := m.Client.QueryEmissions().GetParams(ctx, &emissionstypes.GetParamsRequest{})
		require.NoError(m.T, err)
		minAmount = paramsResp.Params.RequiredMinimumStake
	}
	return setupWeights{heavyTopicId: heavyTopicId, uneven: uneven, minAmount: minAmount}
}

func (w setupWeights) amountFor(topicId uint64, base cosmossdk_io_math.Int) cosmossdk_io_math.Int {
	if !w.uneven || topicId == w.heavyTopicId {
		return base
	}
	light := base.QuoRaw(lightTopicDivisor)
	if light.LT(w.minAmount) {
		return w.minAmount
	}
	return light
}

// startActivateTopicsInOneBlock activates every setup topic in the same block: the reputer
// registers on, stakes on and funds each topic in one transaction. A block admits at most
// MaxActiveTopicsPerBlock topics, so the limit is raised to the number of setup topics for
// that transaction and restored right after; the topics then compete for the restored limit
// at their shared epoch end. Returns the epoch-end block the topics now share.
func startActivateTopicsInOneBlock(
	m *testcommon.TestConfig,
	data *SimulationData,
	admin Actor,
	reputer Actor,
	listTopics []uint64,
	weights setupWeights,
	iterationCountStart int,
) (iterationCountAfter int, sharedChurningBlock int64) {
	iterationCount := iterationCountStart
	previousLimit, ok := setMaxActiveTopicsPerBlock(m, admin, uint64(len(listTopics)), data, iterationCount)
	require.True(m.T, ok)
	iterationCount++
	stakeBase, err := pickRandomBalanceLessThanHalf(m, reputer)
	failIfOnErr(m.T, true, err)
	// stakes and funding are paid from the same balance, so the two bases must fit together
	fundBase := stakeBase.QuoRaw(2)
	stakes := make([]cosmossdk_io_math.Int, 0, len(listTopics))
	funds := make([]cosmossdk_io_math.Int, 0, len(listTopics))
	for _, topicId := range listTopics {
		stakes = append(stakes, weights.amountFor(topicId, stakeBase))
		funds = append(funds, weights.amountFor(topicId, fundBase))
	}
	success := activateTopicsInOneTx(m, reputer, listTopics, stakes, funds, data, iterationCount)
	require.True(m.T, success)
	iterationCount++
	sharedChurningBlock = requireSameChurningBlock(m, listTopics)
	_, ok = setMaxActiveTopicsPerBlock(m, admin, previousLimit, data, iterationCount)
	require.True(m.T, ok)
	iterationCount++
	return iterationCount, sharedChurningBlock
}

// startCreateTopics creates the setup topics, in one transaction when the setup asks for
// topics in the same block, and returns their ids in ascending order.
func startCreateTopics(
	m *testcommon.TestConfig,
	data *SimulationData,
	creator Actor,
	setup fuzzcommon.InitialSetup,
	iterationCountStart int,
) (listTopics []uint64, iterationCountAfter int) {
	iterationCount := iterationCountStart
	if setup.TopicsInSameBlock {
		topicIds, success := createTopicsInOneTx(m, creator, setup.NumTopics, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
		listTopics = topicIds
	} else {
		for i := 0; i < setup.NumTopics; i++ {
			success := createTopic(m, creator, UnusedActor, nil, 0, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
		listTopics = data.getTopics()
	}
	sort.Slice(listTopics, func(i, j int) bool { return listTopics[i] < listTopics[j] })
	return listTopics, iterationCount
}

// startFundTopics funds the topics with random amounts of money, in one transaction when the
// setup asks for topics in the same block, else one per topic. A topic that is inactive at
// this point (refused at its epoch end) is activated again by this funding.
func startFundTopics(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
	listTopics []uint64,
	setup fuzzcommon.InitialSetup,
	weights setupWeights,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	base, err := pickRandomBalanceLessThanHalf(m, faucet)
	failIfOnErr(m.T, true, err)
	if setup.TopicsInSameBlock {
		amounts := make([]cosmossdk_io_math.Int, 0, len(listTopics))
		for _, topicId := range listTopics {
			amounts = append(amounts, weights.amountFor(topicId, base))
		}
		success := fundTopicsInOneTx(m, faucet, listTopics, amounts, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
	} else {
		for _, topicId := range listTopics {
			fundAmount := base
			if !weights.uneven {
				fundAmount, err = pickRandomBalanceLessThanHalf(m, faucet)
				failIfOnErr(m.T, true, err)
			} else {
				fundAmount = weights.amountFor(topicId, base)
			}
			success := fundTopic(m, faucet, UnusedActor, &fundAmount, topicId, data, iterationCount)
			require.True(m.T, success)
			iterationCount++
		}
	}
	return iterationCount
}

// requireSameChurningBlock asserts that every topic is active and scheduled for the same
// epoch-end block, which is what makes them compete for MaxActiveTopicsPerBlock later, and
// returns that block.
func requireSameChurningBlock(m *testcommon.TestConfig, listTopics []uint64) int64 {
	ctx := context.Background()
	churningBlocks := make(map[int64][]uint64)
	var sharedBlock int64
	for _, topicId := range listTopics {
		active := isTopicActive(m, topicId)
		require.True(m.T, active, "setup topic %d must be active right after activation", topicId)
		resp, err := m.Client.QueryEmissions().GetNextChurningBlockByTopicId(ctx, &emissionstypes.GetNextChurningBlockByTopicIdRequest{
			TopicId: topicId,
		})
		require.NoError(m.T, err)
		churningBlocks[resp.BlockHeight] = append(churningBlocks[resp.BlockHeight], topicId)
		sharedBlock = resp.BlockHeight
	}
	require.Len(m.T, churningBlocks, 1, "setup topics must share one epoch-end block, got %v", churningBlocks)
	m.T.Log("setup topics", listTopics, "share the epoch-end block", sharedBlock)
	return sharedBlock
}

// verifyEpochEndRefusal waits for the shared epoch end of the setup topics and asserts the
// block limit was enforced: the heavy topic stays active, no more topics than the limit are
// active, and at least one setup topic was inactivated.
func verifyEpochEndRefusal(
	m *testcommon.TestConfig,
	listTopics []uint64,
	heavyTopicId uint64,
	sharedChurningBlock int64,
) {
	ctx := context.Background()
	// The epoch end is processed in the EndBlock of the churning block; one more block makes
	// sure its state is queryable.
	epochEndBlock := sharedChurningBlock + 1
	m.T.Log("waiting for the first epoch end of the setup topics at block", epochEndBlock)
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, epochEndBlock))

	paramsResp, err := m.Client.QueryEmissions().GetParams(ctx, &emissionstypes.GetParamsRequest{})
	require.NoError(m.T, err)
	limit := paramsResp.Params.MaxActiveTopicsPerBlock
	require.Less(m.T, limit, uint64(len(listTopics)), "the refusal check needs more setup topics than max_active_topics_per_block")

	active := make([]uint64, 0, len(listTopics))
	inactive := make([]uint64, 0, len(listTopics))
	for _, topicId := range listTopics {
		if isTopicActive(m, topicId) {
			active = append(active, topicId)
		} else {
			inactive = append(inactive, topicId)
		}
	}
	m.T.Log("after the first epoch end: active setup topics", active, "inactive setup topics", inactive, "limit per block", limit)
	require.Contains(m.T, active, heavyTopicId, "the heaviest setup topic must survive the epoch end")
	require.LessOrEqual(m.T, uint64(len(active)), limit, "no more setup topics than the per-block limit may stay active")
	require.NotEmpty(m.T, inactive, "at least one setup topic must have been refused its next block and inactivated")
}

// isTopicActive queries the chain's own notion of an active topic.
func isTopicActive(m *testcommon.TestConfig, topicId uint64) bool {
	ctx := context.Background()
	resp, err := m.Client.QueryEmissions().IsTopicActive(ctx, &emissionstypes.IsTopicActiveRequest{TopicId: topicId})
	require.NoError(m.T, err)
	return resp.IsActive
}

// startDoInferenceAndReputation does inference and reputation for the setup topics that
// are active; a topic inactivated at its epoch end has no open worker nonce to submit to.
func startDoInferenceAndReputation(
	m *testcommon.TestConfig,
	data *SimulationData,
	listTopics []uint64,
	iterationCountStart int,
) (iterationCountAfter int) {
	iterationCount := iterationCountStart
	ranOnAnyTopic := false
	for _, topicId := range listTopics {
		if !isTopicActive(m, topicId) {
			m.T.Log("skipping inference and reputation for inactive setup topic", topicId)
			continue
		}
		success := doInferenceAndReputation(m, UnusedActor, UnusedActor, nil, topicId, data, iterationCount)
		require.True(m.T, success)
		iterationCount++
		ranOnAnyTopic = true
	}
	require.True(m.T, ranOnAnyTopic, "at least one setup topic must be active for inference and reputation")
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
	require.True(m.T, addToGlobalWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, removeFromGlobalWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, addToGlobalReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, removeFromGlobalReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, addToGlobalAdminWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, removeFromGlobalAdminWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, bulkAddToGlobalWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, bulkRemoveFromGlobalWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, bulkAddToGlobalReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++
	require.True(m.T, bulkRemoveFromGlobalReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, 0, data, iterationCount))
	iterationCount++

	// create the setup topics with both reputers and workers whitelist enabled
	var listTopics []uint64
	listTopics, iterationCount = startCreateTopics(m, data, startDelegators[0], f.InitialSetup, iterationCount)
	require.Len(m.T, listTopics, f.InitialSetup.NumTopics)
	weights := newSetupWeights(m, listTopics[0], f.InitialSetup.UnevenTopicWeights)

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

	// bulk add/remove from topic whitelists
	require.True(m.T, bulkAddToTopicWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, listTopics[0], data, iterationCount))
	iterationCount++
	require.True(m.T, bulkRemoveFromTopicWorkerWhitelist(m, faucet, pickRandomActor(m, data), nil, listTopics[0], data, iterationCount))
	iterationCount++
	require.True(m.T, bulkAddToTopicReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, listTopics[0], data, iterationCount))
	iterationCount++
	require.True(m.T, bulkRemoveFromTopicReputerWhitelist(m, faucet, pickRandomActor(m, data), nil, listTopics[0], data, iterationCount))
	iterationCount++

	// register all 4 reputers on every setup topic. With topics in the same block, the first
	// reputer registers, stakes and funds every topic in one transaction, so all of them
	// activate in that block and share their epoch-end block from then on. The refusal check
	// runs right after that shared epoch end, before any later stake can re-activate a
	// refused topic.
	remainingReputers := startReputers
	if f.InitialSetup.TopicsInSameBlock {
		var sharedChurningBlock int64
		iterationCount, sharedChurningBlock = startActivateTopicsInOneBlock(m, data, faucet, startReputers[0], listTopics, weights, iterationCount)
		remainingReputers = startReputers[1:]
		if f.InitialSetup.ExpectEpochEndRefusal {
			verifyEpochEndRefusal(m, listTopics, weights.heavyTopicId, sharedChurningBlock)
		}
	}
	iterationCount = startRegisterReputers(m, data, remainingReputers, listTopics, weights, iterationCount)
	// register all 5 workers on every setup topic
	iterationCount = startRegisterWorkers(m, data, startWorkers, listTopics, iterationCount)
	// delegate stake to every setup topic from the delegators
	iterationCount = startDelegateDelegators(m, data, startDelegators, startReputers, listTopics, weights, iterationCount)
	// fund the topics; this also re-activates any setup topic inactivated at its epoch end
	iterationCount = startFundTopics(m, faucet, data, listTopics, f.InitialSetup, weights, iterationCount)
	// do inference and reputation for the setup topics that are active
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

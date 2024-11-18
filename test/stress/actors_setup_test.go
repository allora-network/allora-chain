package stress_test

import (
	"context"
	"fmt"
	"strconv"

	"sync"
	"sync/atomic"
	"time"

	"github.com/allora-network/allora-chain/app/params"
	testcommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"

	cosmosmath "cosmossdk.io/math"
	stresscommon "github.com/allora-network/allora-chain/test/stress/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// an actor in the simulation has a
// human readable name,
// string bech32 address,
// and an account with private key etc
// add a lock to this if you need to broadcast transactions in parallel
// from actors
type StressActor struct {
	name   string
	addr   string
	params stresscommon.TransactionParams
}

type SetupActor struct {
	name string
	addr string
	acc  cosmosaccount.Account
}

// stringer for actor
func (a StressActor) String() string {
	return a.name
}

func (a SetupActor) String() string {
	return a.name
}

// get the faucet name based on the seed for this test run
func getFaucetName(seed int) string {
	return "run" + strconv.Itoa(seed) + "_faucet"
}

// generates an actors name from seed and index
func getActorName(seed int, actorIndex int) string {
	return "run" + strconv.Itoa(seed) + "_actor" + strconv.Itoa(actorIndex)
}

var UnusedActor StressActor = StressActor{} // nolint:exhaustruct

// Set up the common state for the simulator
func simulateSetUp(
	m *testcommon.TestConfig,
	numActors int,
	epochLength int,
) (
	faucet SetupActor,
	simulationData *SimulationData,
) {
	// fund all actors from the faucet with some amount
	// give everybody the same amount of money to start with
	m.T.Logf("Creating %d actors", numActors)
	actorsList := createActors(m, numActors)
	faucet = SetupActor{
		name: getFaucetName(m.Seed),
		addr: m.FaucetAddr,
		acc:  m.FaucetAcc,
	}
	preFundAmount, err := getPreFundAmount(m, faucet, numActors)
	if err != nil {
		m.T.Fatal(err)
	}
	m.T.Logf("Funding actors from faucet with amount: %s\n", preFundAmount.String())
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
		faucet:                    faucet,
		epochLength:               int64(epochLength),
		actors:                    actorsList,
		registeredWorkersByTopic:  map[uint64][]StressActor{},
		registeredReputersByTopic: map[uint64][]StressActor{},
		failOnErr:                 false,
		mu:                        sync.RWMutex{},
	}

	return faucet, &data
}

// Create a new actor and register them in the node's account registry
func createNewActor(m *testcommon.TestConfig, numActors int) StressActor {
	actorName := getActorName(m.Seed, numActors)
	privKey, pubKey, address := stresscommon.GeneratePrivKey()

	return StressActor{
		name: actorName,
		addr: address,
		params: stresscommon.TransactionParams{
			Config: stresscommon.Config{
				GasPerByte:   100,
				BaseGas:      200000,
				Denom:        params.BaseCoinUnit,
				GasLow:       25,
				GasPrecision: 3,
			},
			Sequence:    0,
			AccNum:      0,
			PrivKey:     privKey,
			PubKey:      pubKey,
			AcctAddress: address,
			NodeURL:     "http://localhost:26657",
			ChainID:     m.Client.Context().ChainID,
		},
	}
}

// Create a list of actors both as a map and a slice, returns both
func createActors(m *testcommon.TestConfig, numToCreate int) []StressActor {
	actorsList := make([]StressActor, numToCreate)
	for i := 0; i < numToCreate; i++ {
		actorsList[i] = createNewActor(m, i)
	}
	return actorsList
}

// Fund every target address from the sender in amount coins
func fundActors(
	m *testcommon.TestConfig,
	sender SetupActor,
	targets []StressActor,
	amount cosmosmath.Int,
) error {
	batchSize := 2000
	start := time.Now()
	completed := atomic.Int32{}

	m.T.Logf("Starting funding of %d actors", len(targets))

	for i := 0; i < len(targets); i += batchSize {
		end := i + batchSize
		if end > len(targets) {
			end = len(targets)
		}
		batch := targets[i:end]

		inputCoins := sdktypes.NewCoins(
			sdktypes.NewCoin(
				params.BaseCoinUnit,
				amount.MulRaw(int64(len(batch))),
			),
		)
		outputCoins := sdktypes.NewCoins(
			sdktypes.NewCoin(params.BaseCoinUnit, amount),
		)

		outputs := make([]banktypes.Output, len(batch))
		names := make([]string, len(batch))
		for j, actor := range batch {
			names[j] = actor.name
			outputs[j] = banktypes.Output{
				Address: actor.addr,
				Coins:   outputCoins,
			}
		}

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
			m.T.Log("Error funding batch: ", err)
			return err
		}

		count := completed.Add(int32(len(batch)))
		if int(count)%1000 == 0 || count == int32(len(targets)) {
			elapsed := time.Since(start)
			m.T.Logf("Processed %d/%d funding operations (%.2f%%) in %s",
				count, len(targets),
				float64(count)/float64(len(targets))*100,
				elapsed)
		}
	}

	totalTime := time.Since(start)
	m.T.Logf("Total funding time: %s", totalTime)

	return nil
}

// Get the amount of money to give each actor in the simulation
// Based on how much money the faucet currently has
func getPreFundAmount(
	m *testcommon.TestConfig,
	faucet SetupActor,
	numActors int,
) (cosmosmath.Int, error) {
	faucetBal, err := faucet.GetBalance(m)
	if err != nil {
		return cosmosmath.ZeroInt(), err
	}
	// divide by 10 so you can at least run 10 runs
	amountForThisRun := faucetBal.QuoRaw(int64(10))
	ret := amountForThisRun.QuoRaw(int64(numActors))
	if ret.Equal(cosmosmath.ZeroInt()) || ret.IsNegative() {
		return cosmosmath.ZeroInt(), fmt.Errorf(
			"Not enough funds in faucet account to fund actors",
		)
	}
	return ret, nil
}

// How much money an actor has
func (a *StressActor) GetBalance(m *testcommon.TestConfig) (cosmosmath.Int, error) {
	ctx := context.Background()
	bal, err := m.Client.QueryBank().
		Balance(ctx, banktypes.NewQueryBalanceRequest(sdktypes.MustAccAddressFromBech32(a.addr), params.DefaultBondDenom))
	if err != nil {
		m.T.Logf("Error getting balance of actor %s: %v\n", a.String(), err)
		return cosmosmath.ZeroInt(), err
	}
	return bal.Balance.Amount, nil
}

func (a *SetupActor) GetBalance(m *testcommon.TestConfig) (cosmosmath.Int, error) {
	ctx := context.Background()
	bal, err := m.Client.QueryBank().
		Balance(ctx, banktypes.NewQueryBalanceRequest(sdktypes.MustAccAddressFromBech32(a.addr), params.DefaultBondDenom))
	if err != nil {
		m.T.Logf("Error getting balance of actor %s: %v\n", a.String(), err)
		return cosmosmath.ZeroInt(), err
	}
	return bal.Balance.Amount, nil
}

// RegisterWorkers registers numWorkers as workers in topicId
func registerWorkers(
	m *testcommon.TestConfig,
	actors []StressActor,
	topicId uint64,
	data *SimulationData,
	numWorkers int,
) error {
	maxConcurrent := 1000
	sem := make(chan struct{}, maxConcurrent)

	start := time.Now()
	completed := atomic.Int32{}

	var wg sync.WaitGroup
	m.T.Logf("Starting registration of %d workers in topic: %d\n", numWorkers, topicId)

	// Process all workers without batching
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		worker := actors[i]

		go func(worker StressActor, idx int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
				count := completed.Add(1)
				if int(count)%1000 == 0 || count == int32(numWorkers) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d worker registrations (%.2f%%) for topic: %d in %s\n",
						count, numWorkers,
						float64(count)/float64(numWorkers)*100,
						topicId,
						elapsed)
				}
			}()

			request := &emissionstypes.RegisterRequest{
				Sender:    worker.addr,
				Owner:     worker.addr,
				IsReputer: false,
				TopicId:   topicId,
			}

			_, updatedSeq, err := stresscommon.SendDataWithRetry(worker.params, request)
			if err != nil {
				m.T.Logf("Error sending worker registration: %v", err.Error())
				return
			}
			worker.params.Sequence = updatedSeq
			data.addWorkerRegistration(topicId, worker)
		}(worker, i)
	}

	wg.Wait()

	totalTime := time.Since(start)
	m.T.Logf("Total worker registration time: %s\n", totalTime)

	return nil
}

// RegisterReputersAndStake registers numReputers as reputers in topicId and stakes them
func registerReputersAndStake(
	m *testcommon.TestConfig,
	actors []StressActor,
	topicId uint64,
	data *SimulationData,
	numReputers int,
) error {
	maxConcurrent := 100
	sem := make(chan struct{}, maxConcurrent)

	start := time.Now()
	completed := atomic.Int32{}

	var wg sync.WaitGroup
	m.T.Logf("Starting registration of %d reputers in topic: %d\n", numReputers, topicId)

	// Process all reputers without batching
	for i := 0; i < numReputers; i++ {
		wg.Add(1)
		reputer := actors[i]

		go func(reputer StressActor, idx int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
				count := completed.Add(1)
				if int(count)%100 == 0 || count == int32(numReputers) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d reputer registrations (%.2f%%) for topic: %d in %s\n",
						count, numReputers,
						float64(count)/float64(numReputers)*100,
						topicId,
						elapsed)
				}
			}()

			registerRequest := &emissionstypes.RegisterRequest{
				Sender:    reputer.addr,
				Owner:     reputer.addr,
				IsReputer: true,
				TopicId:   topicId,
			}
			stakeRequest := &emissionstypes.AddStakeRequest{
				Sender:  reputer.addr,
				TopicId: topicId,
				Amount:  cosmosmath.NewIntFromUint64(stakeToAdd),
			}

			_, updatedSeq, err := stresscommon.SendDataWithRetry(reputer.params, registerRequest, stakeRequest)
			if err != nil {
				m.T.Logf("Error sending reputer stake: %v", err.Error())
				return
			}
			reputer.params.Sequence = updatedSeq

			data.addReputerRegistration(topicId, reputer)
		}(reputer, i)
	}

	wg.Wait()

	totalTime := time.Since(start)
	m.T.Logf("Total reputer registration time: %s\n", totalTime)

	return nil
}

# Invariant and State Transition Testing

The invariant test in this repo is a quasi-fuzzer that will send random transactions to the chain in order to try to stimulate strange state transitions and make sure that all invariants hold in the face of those state transitions.

Example invocation:

```bash
INVARIANT_TEST=TRUE SEED=1 RPC_MODE="SingleRpc" \
    RPC_URLS="http://localhost:26657" \
    MAX_ITERATIONS=100 FAIL_ON_ERR=true MANUAL_SIMULATION=false \
    NUM_ACTORS=10 EPOCH_LENGTH=14 \
    /usr/bin/go test -timeout 15m -run ^TestInvariantTestSuite$ -v ./test/invariant
```

# Shell Environment Parameters

```bash
INVARIANT_TEST=true # required to run the invariant test, otherwise the script will not run
SEED=1 # an integer used to seed randomness and name actors during the test (e.g. run3_actor7)
RPC_MODE="SingleRpc" # Either SingleRpc, RoundRobin, or RandomBasedOnDeterministicSeed - how to interact with multiple RPC endpoints
RPC_URLS="http://localhost:26657" # RPC endpoint urls, separated by comma if multiple
MAX_ITERATIONS=100 # how many times to send transactions. Set to zero to continue forever
FAIL_ON_ERR=true # See Fail on Error section below
MANUAL_SIMULATION=false # See Automatic vs Manual mode below
NUM_ACTORS=10 # how many private keys to create to use as actors in this play
EPOCH_LENGTH=14 # when we submit inferences and reputation scores, how long to wait in between the inference and the reputation
```

# Automatic vs Manual Mode

In order to assist with testing, the simulator runs in two modes, automatic or manual. In the automatic mode it simply counts up to `MAX_ITERATIONS` and for every iteration, chooses a transaction to send to the network. If you find a bug you wish to replay, you can use manual mode to run the manual commands given in the `simulateManual` function in `invariant_test.go`.

If manual mode is set to true, then the `MAX_ITERATIONS` flag will be ignored. In manual mode, you should set the iteration counter yourself.

The simulator runs in a single threaded process, it does not attempt to do concurrency. To do concurrency, run two separate `go test` invocations at the same time (perhaps with the same seed, to mess with the same actors!)

# Fail on Error Mode

The simulator has a "fail on error" mode switch. If this is set to true, then the simulator will check the state it thinks the chain should be in and only try to do state transitions that it thinks should succeed given that state - i.e. act in expected ways. If an error occurs, it will fail the test and halt testing.

If fail on error mode is set to false, then the simulator will enter a more traditional fuzzing style approach - it will submit state transition transactions that may or may not be valid in a random order. If the RPC url returns an error, the test will not halt or complain. This is useful for trying to really spam the chain with state transitions. If you run this with error mode set to false and the chain halts, then you have found an invariant bug.

Note that when the fail on error mode is set to false, the counter for the output will only count successful state transitions, not all attempted state transitions.

# Output

The output of the simulator contains a count of every attempted state transition will look something like this:

```
    invariant_test.go:188: State Transitions Summary: {
        createTopic: 7, 
        fundTopic: 10, 
        registerWorker: 7, 
        registerReputer: 14, 
        unregisterWorker: 6, 
        unregisterReputer: 8, 
        stakeAsReputer: 10
        delegateStake: 11
        unstakeAsReputer: 9
        undelegateStake: 7
        cancelStakeRemoval: 0
        cancelDelegateStakeRemoval: 0
        collectDelegatorRewards: 4
        doInferenceAndReputation: 3
        }
```

In this example workers have _successfully_ registered 7 times, and unregistered 6 times. That means that at the time of this log, only one worker is currently registered

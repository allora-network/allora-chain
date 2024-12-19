# Fuzzing, Invariant and State Transition Testing

The fuzz tests in this repo send random transactions to the chain in order to try to stimulate strange state transitions and make sure that all invariants hold in the face of those state transitions.

Example invocation:

```bash
bash test/local_testnet_l1.sh
FUZZ_TEST=TRUE go test -timeout 15m -run ^TestFuzzTestSuite$ -v ./test/fuzz
```

# Fuzzer Parameters

The fuzzer parameters can be provided with the environment flags below, but it is more convenient to put them in `test/fuzz/.config.json` (you can see the .config.example as well in the same repo)

```bash
SEED=1 # An integer used to seed randomness and name actors during the test (e.g. run3_actor7)
RPC_MODE="SingleRpc" # Either SingleRpc, RoundRobin, or RandomBasedOnDeterministicSeed - how to interact with multiple RPC endpoints
RPC_URLS="http://localhost:26657" # RPC endpoint urls, separated by comma if multiple
MAX_ITERATIONS=100 # How many times to send transactions. Set to zero to continue forever
MODE="alternate" # See Mode section below. Valid options: "behave" "fuzz" "alternate" or "manual"
ALTERNATE_WEIGHT="20" # Every iteration, have this percentage likelihood of switching between fuzz and behave mode
NUM_ACTORS=12 # How many actors to fuzz with. Must be large enough to do the fuzz setup (at time of writing: >=11)
EPOCH_LENGTH=14 # How long to wait in between submitting inference and reputation bundles (at time of writing: >=12).
```

The fuzzer will use hardcoded values, look for values in the optional `test/fuzz/.config.json` file, and finally use environment variables. Environment variables take the highest priority, then the config.json, and finally hardcoded values. This way you can override defaults by using the shell environment.

Note: the `FUZZ_TEST` environment variable is required at all times to be set to true to run the fuzzer test.

# Simulation Modes

In order to assist with testing, the simulator supports four modes, controllable via the `MODE` environment variable or `"mode"` json field:

1. Behave mode: the simulator will check the state it thinks the chain should be in and only try to do state transitions that it thinks should succeed given that state - i.e. act in expected ways. If an error occurs, it will fail the test and halt testing.
2. Fuzz mode: the simulator will enter a more traditional fuzzing style approach - it will submit state transition transactions that may or may not be valid in a random order. If the RPC url returns an error, the test will not halt or complain. This is useful for trying to really spam the chain with state transitions.
3. Alternate mode: The fuzzer will start flip-flopping between behaving and fuzzing. This should stimulate chains of successful transactions in a row followed by chains of fuzzed transactions in a row. How often the simulator switches between the two modes is controlled by the `ALTERNATE_WEIGHT` environment variable or `"alternateWeight"` json parameter, which is a percentage between 0 and 100.
4. Manual mode: if you find a bug you wish to replay, you can use manual mode to run the manual commands given in the `simulateManual` function in `fuzz_test.go`. This is basically the same thing as an integration test.

Automatic or manual: In the automatic mode it simply counts up to `MAX_ITERATIONS` (plus the iterations for the setup) and for every iteration, chooses a transaction to send to the network. If manual mode is set to true, then the `MAX_ITERATIONS` flag will be ignored. In manual mode, you should set the iteration counter yourself.

The simulator runs in a single threaded process, it does not attempt to do concurrency. To do concurrency, run two separate `go test` invocations at the same time (perhaps with the same seed, to mess with the same actors!)

Note that in all modes, the counter for the output will only count successful state transitions, not all attempted state transitions. So if you see the state transition summary and the sum total of all counts does not equal the number of iterations ran, that is expected if iterations were allowed to fail.

# Transition Weights / Probability Distribution

The `transitionWeights` option in the config.json controls the distribution of how likely it will be to pick a specific transition during a fuzz run, based on a scale of 1-100. E.g. if you set the createTopic transitionWeight to 20, then any given transition will have a 20% chance of being a createTopic call. See the `.config.json.example` file for a recommended sane default distribution. These weights must add up to 100% in order for the fuzzer to run.

# Output

The output of the simulator contains a count of every attempted state transition will look something like this:

```
    fuzz_test.go:188: State Transitions Summary: {
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

In this example workers have _successfully_ registered 7 times, and unregistered 6 times. That means that at the time of this log, only one worker is currently registered.

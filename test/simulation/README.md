# Simulation Test

The purpose of this simulation is to verify that our logic works as researched.
To do this, the loss calculation part performed off-chain is simulated here.
As a result of simulation, we can obtain `losses.csv` and `rewards.csv` and can create a graph through them.

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run stress simulation test, set the SIMULATION_TEST variable to true

```
cd simulation
SIMULATION_TEST=true RPC_MODE="RandomBasedOnDeterministicSeed" RPC_URLS="http://localhost:26657,http://localhost:26658,http://localhost:26659" SEED=1 INFERERS_COUNT=5 FORECASTER_COUNT=3 REPUTERS_COUNT=5 MAX_ITERATIONS=20 go test -v -timeout 0 -test.run TestStressTestSuite ./test/simulation
```

options for RPC Modes include "RandomBasedOnDeterministicSeed" "RoundRobin" and "SingleRpc"

Stop the chain
```
docker compose -f devnet/compose_l1.yaml  stop
```

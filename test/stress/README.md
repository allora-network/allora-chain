# Stress Tests

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run stress tests, set the STRESS_TEST variable to true

```
cd stress
STRESS_TEST=true RPC_MODE="RandomBasedOnDeterministicSeed" RPC_URLS="http://localhost:26657,http://localhost:26658,http://localhost:26659" SEED=1 MAX_REPUTERS_PER_TOPIC=2 REPUTERS_PER_ITERATION=2 EPOCH_LENGTH=12 FINAL_REPORT=TRUE MAX_WORKERS_PER_TOPIC=2 WORKERS_PER_ITERATION=1 TOPICS_MAX=2 TOPICS_PER_ITERATION=1 MAX_ITERATIONS=2 go test -v -timeout 0 -test.run TestStressTestSuite ./test/stress
```

options for RPC Modes include "RandomBasedOnDeterministicSeed" "RoundRobin" and "SingleRpc"

Stop the chain
```
docker compose -f localnet/compose_l1.yaml  stop
```

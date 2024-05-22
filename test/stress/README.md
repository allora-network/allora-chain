# Stress Tests

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run stress tests, set the STRESS_TEST variable to true

```
STRESS_TEST=true go test -v -timeout 0 -test.run TestStressTestSuite .
```

Stop the chain
```
docker compose -f devnet/compose_l1.yaml  stop
```

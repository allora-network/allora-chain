# Stress Tests

To launch a devnet: 

```
cd stress
bash local_testnet_l1.sh
```

To run stress tests, set the STRESS_TEST variable to true

```
STRESS_TEST=true go test -v -timeout 0   -test.run TestStressTestSuite .
```


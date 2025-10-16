# Integration Tests

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run integration tests, set the INTEGRATION variable to true

```
INTEGRATION=true go test -v -timeout 15m -run TestExternalTestSuite ./test/integration
```

Stop the chain
```
docker compose -f localnet/compose_l1.yaml  stop
```

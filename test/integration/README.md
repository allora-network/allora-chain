# Integration Tests

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run integration tests, set the INTEGRATION variable to true

```
INTEGRATION=true go test -v -run TestExternalTestSuite ./test/integration
```

To run upgrade tests against the v0.17.0 scenario:

```
DO_UPGRADE=true UPGRADE_VERSION=v0.17.0 bash test/local_testnet_l1.sh
UPGRADE=true UPGRADE_TARGET=v0.17.0 go test -v -run TestUpgradeTestSuite ./test/integration
```

Stop the chain
```
docker compose -f localnet/compose_l1.yaml  stop
```

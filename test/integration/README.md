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

To run upgrade tests against the v0.18.0 scenario, from a full clone (`Dockerfile.upgrade` cannot build from a linked git worktree):

```
DO_UPGRADE=true bash test/local_testnet_l1.sh
UPGRADE=true UPGRADE_TARGET=v0.18.0 RPC_MODE=SingleRpc RPC_URLS=http://localhost:26657 go test -v -count=1 -timeout 20m -run '^TestUpgradeTestSuite$' ./test/integration
```

Stop the chain
```
docker compose -f test/localnet/compose_l1.yaml stop
```

# Integration Tests

To launch a chain: 
```
cd test
bash local_testnet_l1.sh
```

To run integration tests, set the INTEGRATION variable to true

```
INTEGRATION=true go test -count=1 -v -run TestExternalTestSuite ./test/integration
```

`TestExternalTestSuite` and `TestExternalMultiTestSuite` each need their own fresh chain. `make integration-test` runs both that way, and still runs the second suite when the first fails. To replace a running chain by hand, without a prompt and without rebuilding the image:

```
LOCALNET_RESET=true LOCALNET_REUSE_IMAGE=true bash test/local_testnet_l1.sh
INTEGRATION=true go test -count=1 -v -run TestExternalMultiTestSuite ./test/integration
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

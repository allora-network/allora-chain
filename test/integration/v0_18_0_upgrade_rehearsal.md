# v0.17.2 -> v0.18.0 Local Upgrade Rehearsal

This runbook records a local 3-validator upgrade from `v0.17.2` to `v0.18.0`, run through `test/local_testnet_l1.sh` and `TestUpgradeTestSuite`. The chain starts from a fresh genesis, so `next_topic_id` was still `1` after the upgrade: the `v16` emissions migration ran with no topics to migrate.

Run on macOS arm64 with Docker, from commit `1d692b87`.

## 1) Use a full clone

`Dockerfile.upgrade` builds the baseline from a tag in the build context's git history. A linked worktree's `.git` is a file pointing outside the context, and the build stops with `Dockerfile.upgrade needs a full clone: .git is not a directory`.

```bash
git clone . .upgrade-rehearsal-clone
cd .upgrade-rehearsal-clone
```

## 2) Build binaries and start the baseline chain

```bash
DO_UPGRADE=true bash test/local_testnet_l1.sh
```

The image holds `allorad` built from `v0.17.2` and `allorad-v0.18.0` built from the build context. The script installs the latter at `cosmovisor/upgrades/v0.18.0/bin/allorad` for each validator.

Observed before the upgrade, on all three validators from `/abci_info`: `version = v0.17.2`.

## 3) Run automated upgrade checks

```bash
UPGRADE=true UPGRADE_TARGET=v0.18.0 RPC_MODE=SingleRpc RPC_URLS=http://localhost:26657 \
  go test -v -count=1 -timeout 30m -run '^TestUpgradeTestSuite$' ./test/integration
```

The test proposes the upgrade, votes with all three validators, waits for the proposal to pass, and requires every validator to commit 5 blocks past the plan height on a different binary. `RPC_URLS` drives the proposal and queries; `UPGRADE_VALIDATOR_RPC_URLS` lists the validators to check and defaults to the three localnet ports.

Observed, plan height 75:

```text
Module scheduler pre-upgrade consensus version: 1
Module emissions pre-upgrade consensus version: 15
Module mint pre-upgrade consensus version: 6
Validator http://localhost:26657 at height 80 (plan height 75) on binary version spook/v0-18-upgrade-ci-job-1d692b87dc2b752a834e949d80e09a2d5c10e2db
Validator http://localhost:26658 at height 80 (plan height 75) on binary version spook/v0-18-upgrade-ci-job-1d692b87dc2b752a834e949d80e09a2d5c10e2db
Validator http://localhost:26659 at height 80 (plan height 75) on binary version spook/v0-18-upgrade-ci-job-1d692b87dc2b752a834e949d80e09a2d5c10e2db
--- PASS: TestUpgradeTestSuite (296.73s)
```

## 4) Verify upgrade success

Validator logs show this sequence on all three validators. `CONSENSUS FAILURE` here is the expected halt of the old binary at the plan height:

```text
UPGRADE "v0.18.0" NEEDED at height: 75: {}
CONSENSUS FAILURE!!! err="failed to apply block; error UPGRADE \"v0.18.0\" NEEDED at height: 75: {}"
RUN MIGRATIONS
MIGRATION EMISSIONS MODULE FROM VERSION 15 TO VERSION 16 COMPLETE
MIGRATIONS COMPLETED
```

Queried after the upgrade:

```bash
allorad q upgrade module-versions --node tcp://localhost:26657
allorad q upgrade plan --node tcp://localhost:26657
allorad q emissions params --node tcp://localhost:26657
```

- Module versions: `scheduler = 1`, `emissions = 16`, `mint = 6`
- Upgrade plan: empty
- Params: `max_top_inferers_to_reward = 32`, `min_top_inferers_to_reward = 5`
- `cosmovisor/current` on all three validators points to `upgrades/v0.18.0`
- All three validators reported the same `last_block_app_hash` at heights 82 and 88

## 5) Negative control

Rerunning the step 3 test against the already-upgraded chain fails before proposing anything:

```text
module emissions baseline consensus version mismatch
expected: 0xf
actual  : 0x10
--- FAIL: TestUpgradeTestSuite (0.03s)
```

## 6) Stop the chain

```bash
docker compose -f test/localnet/compose_l1.yaml down
```

## Not run

- Upgrade from a restored testnet or mainnet snapshot, so the `v16` migration has not been rehearsed here against populated topics.
- Upgrade with a release `v0.18.0` binary: the upgrade binary was built from the commit above.
- `.github/workflows/upgrade_test.yml` on GitHub Actions.
- `allorad q upgrade applied v0.18.0` output check by hand: the applied height of 75 is asserted by the test through the `AppliedPlan` query.

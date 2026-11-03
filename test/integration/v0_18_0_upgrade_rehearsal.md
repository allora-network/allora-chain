# v0.17.2 -> v0.18.0 Local Upgrade Rehearsal

This runbook records an upgrade from `v0.17.2` to `v0.18.0` on a 3-validator chain with real state, run through `test/local_testnet_l1.sh` and `TestUpgradeTestSuite`. The chain starts from a fresh genesis, and the test seeds it through transactions against the running `v0.17.2` binaries before proposing the upgrade. The upgrade binary is built with `test/fuzz/invariants.patch`, so the emissions invariants run in every end block after the upgrade.

Run from commit `27cbdfd9`: locally on macOS arm64 with Docker, and on GitHub Actions through `.github/workflows/upgrade_test.yml`.

## 1) Use a full clone

`Dockerfile.upgrade` builds the baseline from a tag in the build context's git history. A linked worktree's `.git` is a file pointing outside the context, and the build stops with `Dockerfile.upgrade needs a full clone: .git is not a directory`.

```bash
git clone . .upgrade-rehearsal-clone
cd .upgrade-rehearsal-clone
```

## 2) Apply the invariants patch, build binaries and start the baseline chain

```bash
git apply test/fuzz/invariants.patch
DO_UPGRADE=true bash test/local_testnet_l1.sh
```

The image holds `allorad` built from `v0.17.2` and `allorad-v0.18.0` built from the build context. The script installs the latter at `cosmovisor/upgrades/v0.18.0/bin/allorad` for each validator.

`local_testnet_l1.sh` tags a shared `allorad` image, so confirm the validators run this build. `Invariants broken` is a string only the patch adds:

```bash
for v in validator0 validator1 validator2; do
  grep -ac 'Invariants broken' "test/localnet/$v/cosmovisor/upgrades/v0.18.0/bin/allorad"
  docker exec "$v" "/data/$v/cosmovisor/upgrades/v0.18.0/bin/allorad" version
done
```

Observed on all three validators: one match, version `spook/v0-18-upgrade-populated-state-27cbdfd95f679a8bb4d1475397af38d9eb238c94`, and `/abci_info` reporting `v0.17.2`.

## 3) Run the populated upgrade test

```bash
UPGRADE=true UPGRADE_TARGET=v0.18.0 UPGRADE_SEED_MAX_TOP_INFERERS=0 RPC_MODE=SingleRpc RPC_URLS=http://localhost:26657 \
  go test -v -count=1 -timeout 60m -run '^TestUpgradeTestSuite$' ./test/integration
```

`UPGRADE_SEED_MAX_TOP_INFERERS` sets the global `max_top_inferers_to_reward` the baseline holds at the upgrade, and defaults to `17`. Each value selects a different migration outcome, so the workflow runs all three.

The seed creates these topics:

| Topic | State on the baseline |
| --- | --- |
| Regression | Active, with stake, delegations, registered workers and reputers, a completed epoch, a pending delegate removal and an unfulfilled reputer nonce. Its churn falls on the plan height. |
| Classification | Active, multi-label with a label whitelist, with stake, a delegation and a completed epoch. |
| Blocker | Active, with a pending reputer removal and an unfulfilled worker nonce. |
| Refused reactivation | Lost its schedule when the block limit refused its reactivation, and stays in the active set with its weight in the total. |
| Drifted | Inactivated with a stored weight. A stake removal then completed on it, which moved the total. It keeps stake. |
| Never active | No stake and no stored weight. |

## 4) Observed results

| Run | Seeded ceiling | Topic caps and ceiling after | Floor after | Result |
| --- | --- | --- | --- | --- |
| GitHub Actions run `41531178131` | 17 | 17 | 5 | `--- PASS` (1324.77s) |
| GitHub Actions run `41531178131` | 0 | 32 | 5 | `--- PASS` (1298.96s) |
| GitHub Actions run `41531178131` | 3 | 3 | 3 | `--- PASS` (1299.60s) |
| Local, fresh chain | 0 | 32 | 5 | `--- PASS` (1309.03s) |

Local run, plan height 176:

```text
Seeded drift: total 73671088.138858974058714643800900350325447, sum over scheduled topics 73643666.9204116104468327998492627508959
Topics moved by the end blocker at height 176: [4]
183 values unchanged across the upgrade block, 27 migrated or moved by the end blocker
Reconciled total: 73671088.138858974058714643800900350325447 -> 81126037.9775151023332834927781873738959, ceiling 32, floor 5
Height 186 "ACTOR_TYPE_INFERER_UNSPECIFIED": rewards=["3026248653709732832.224965374666739"]
Height 186 "ACTOR_TYPE_REPUTER": rewards=["3026248653707245228.224965374666739"]
Height 216 "ACTOR_TYPE_INFERER_UNSPECIFIED": rewards=["3293937005545760398.486900510267113"]
Height 216 "ACTOR_TYPE_REPUTER": rewards=["3293937005545352566.486900510267113"]
Topic 1 reactivated at 254 with stored weight 27422.61840160417649974348579633939
All 3 validators agree on the app hash of heights 175..254
--- PASS: TestUpgradeTestSuite (1309.03s)
```

The migration log is identical on all three validators:

```text
MIGRATION V16: repaired zero global max_top_inferers_to_reward to module default value=32
MIGRATION V16: topic max_top_inferers_to_reward backfill completed backfillValue=32 topicsUpdated=6
MIGRATION V16: params backfill completed minTopInferersToReward=5
MIGRATION V16: topic activity state reconciled and total sum recomputed addedActiveTopics=0 previous=73671088.138858974058714643800900350325447 recomputed=73643666.9204116104468327998492627508959 removedActiveTopics=1 removedSchedules=0 resetBuckets=32
```

The removed amount is the seeded drift: `previous - recomputed` is the refused topic's stored weight minus the weight change of the drifted topic. The reconciled total at the plan height also includes the end blocker's weight update for the topic that churned there.

On all three validators after the local run: no `Invariants broken` line, no panic, and one `CONSENSUS FAILURE` line, the expected halt of the old binary at the plan height. They reported the same `last_block_app_hash` at height 271.

After the upgrade the test runs, on topics that existed before it:

- a reputer payload for the nonce opened before the upgrade, and rewards for that epoch
- a full epoch on the regression and the classification topic, and rewards for the regression epoch
- an exact stake addition, and both baseline removals paying the exact amount at their recorded blocks
- a stake removal on the drifted topic, leaving the total equal to the sum over scheduled topics
- reactivation of the refused topic, whose stored weight then counts once in the total

## 5) Negative control

In a disposable clone, with `MigrateStore` changed to skip `MigrateTotalSumPreviousTopicWeights` and everything else as above, all three validators exit at the plan height:

```text
panic: Invariants broken: emissions: total sum of previous topic weights equal sum over scheduled topics invariant
```

This shows that the patched end blocker runs the invariants, and that the seeded state fails them without reconciliation. The change was never committed.

## 6) Stop the chain

```bash
docker compose -f test/localnet/compose_l1.yaml down
```

## Not run

- Upgrade from a restored testnet or mainnet snapshot. Seeded state is small and synthetic, and does not stand in for network history.
- Upgrade with a release `v0.18.0` binary: the upgrade binary was built from the commit above, plus the invariants patch.
- An upgrade binary without the invariants patch on populated state.
- Migration branches that `v0.17.2` transactions were not found to reach, covered only by `x/emissions/migrations/v16/migrate_test.go`: a topic or floor already holding a value, a schedule in the past, a schedule without its bucket listing, a bucket listing without a schedule or at the wrong block, and a valid schedule missing from the active set. `removedSchedules` and `addedActiveTopics` were `0` in every run.
- `allorad q upgrade applied v0.18.0` output check by hand: the applied height is asserted by the test through the `AppliedPlan` query.

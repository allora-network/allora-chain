# v0.16.0 -> v0.17.0 Local Upgrade Rehearsal

This runbook validates a real operator flow: restore a `v0.16.0` snapshot locally and verify an in-place upgrade to `v0.17.0`.

## 1) Build binaries

Build the local `v0.17.0` binary from your branch:

```bash
make build-local-edits
cp build/allorad /tmp/allorad-v0.17.0
```

You also need a `v0.16.0` binary for the genesis/current slot in cosmovisor.
Either download release assets or build from tag:

```bash
git checkout v0.16.0
make build
cp build/allorad /tmp/allorad-v0.16.0
git switch -
```

## 2) Restore a snapshot

Restore your testnet snapshot into `~/.allorad/data`:

```bash
SNAPSHOT_URL="https://<snapshot-url>.tar.zst" \
SNAPSHOT_SHA256="<optional-sha256>" \
APP_HOME="$HOME/.allorad" \
bash scripts/restore_snapshot_from_url.sh
```

## 3) Prepare cosmovisor layout

```bash
mkdir -p "$HOME/.allorad/cosmovisor/genesis/bin"
mkdir -p "$HOME/.allorad/cosmovisor/upgrades/v0.17.0/bin"

cp /tmp/allorad-v0.16.0 "$HOME/.allorad/cosmovisor/genesis/bin/allorad"
cp /tmp/allorad-v0.17.0 "$HOME/.allorad/cosmovisor/upgrades/v0.17.0/bin/allorad"

chmod +x "$HOME/.allorad/cosmovisor/genesis/bin/allorad"
chmod +x "$HOME/.allorad/cosmovisor/upgrades/v0.17.0/bin/allorad"
```

## 4) Ensure upgrade plan exists

If the snapshot already has a scheduled `v0.17.0` plan, verify:

```bash
allorad q upgrade current-plan --home "$HOME/.allorad"
```

If not scheduled yet, submit a governance software-upgrade proposal using `scripts/upgrade.sh` as a template and ensure `plan.name` is exactly `v0.17.0`.

## 5) Start with cosmovisor

```bash
export DAEMON_NAME=allorad
export DAEMON_HOME="$HOME/.allorad"
export UNSAFE_SKIP_BACKUP=true

cosmovisor run start --home "$HOME/.allorad" --minimum-gas-prices 0uallo
```

## 6) Verify upgrade success

After crossing upgrade height:

```bash
allorad q upgrade applied-plan v0.17.0 --home "$HOME/.allorad"
allorad q upgrade module-versions --home "$HOME/.allorad"
allorad q emissions topic 1 --home "$HOME/.allorad"
```

Expected module versions for the `v0.17.0` scenario in this branch:
- `scheduler = 1`
- `emissions = 15`
- `mint = 6`

The topic query should show:
- `active_inferer_quantile`, `active_forecaster_quantile`, and `active_reputer_quantile` preserved from the `v0.16.0` migration behavior
- classification defaults backfilled by `v15`, including `topic_type = TOPIC_TYPE_REGRESSION` and `output_arity = TOPIC_OUTPUT_ARITY_SINGLE` for migrated pre-classification topics

## 7) Run automated upgrade checks against local node

If you run the local 3-validator testnet path:

```bash
DO_UPGRADE=true UPGRADE_VERSION=v0.17.0 bash test/local_testnet_l1.sh
UPGRADE=TRUE UPGRADE_TARGET=v0.17.0 go test -timeout 10m ./test/integration/ -v
```

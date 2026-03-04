# v0.15.1 -> v0.16.0 Local Upgrade Rehearsal

This runbook validates a real operator flow: restore a `v0.15.1` snapshot locally and verify an in-place upgrade to `v0.16.0`.

## 1) Build binaries

Build the local `v0.16.0` binary from your branch:

```bash
make build-local-edits
```

You also need a `v0.15.1` binary for the genesis/current slot in cosmovisor.
Either download release assets or build from tag:

```bash
git checkout v0.15.1
make build
cp build/allorad /tmp/allorad-v0.15.1
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
mkdir -p "$HOME/.allorad/cosmovisor/upgrades/v0.16.0/bin"

cp /tmp/allorad-v0.15.1 "$HOME/.allorad/cosmovisor/genesis/bin/allorad"
cp build/allorad "$HOME/.allorad/cosmovisor/upgrades/v0.16.0/bin/allorad"

chmod +x "$HOME/.allorad/cosmovisor/genesis/bin/allorad"
chmod +x "$HOME/.allorad/cosmovisor/upgrades/v0.16.0/bin/allorad"
```

## 4) Ensure upgrade plan exists

If the snapshot already has a scheduled `v0.16.0` plan, verify:

```bash
allorad q upgrade current-plan --home "$HOME/.allorad"
```

If not scheduled yet, submit a governance software-upgrade proposal using `scripts/upgrade.sh` as a template and ensure `plan.name` is exactly `v0.16.0`.

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
allorad q upgrade applied-plan v0.16.0 --home "$HOME/.allorad"
allorad q upgrade module-versions --home "$HOME/.allorad"
```

Expected module versions for the `v0.16.0` scenario in this branch:
- `scheduler = 1`
- `emissions = 13`
- `mint = 6`

## 7) Run automated upgrade checks against local node

If you run the local 3-validator testnet path:

```bash
DO_UPGRADE=true UPGRADE_VERSION=v0.16.0 bash test/local_testnet_l1.sh
UPGRADE=TRUE UPGRADE_TARGET=v0.16.0 go test -timeout 10m ./test/integration/ -v
```


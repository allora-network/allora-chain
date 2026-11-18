# Localnet: allora-chain in Docker

A single-validator Allora chain on your machine with a fresh genesis, in two
commands. Use it to try a build, develop against the RPC/REST/gRPC endpoints
or run offchain components locally.

It is the local counterpart of the shared devnets
(`allora-network/flux-gcp-devnet`, **devnet up**). The genesis is set up the
same way: `uallo` everywhere, fast governance and unbonding, a whitelisted
topic creator and a faucet.

| | localnet (this) | `test/local_testnet_l1.sh` | devnet |
|---|---|---|---|
| Where | your Docker | your Docker | GKE, Tailscale-only |
| Validators | 1 | 1-N (default 3) | 1-5 |
| Image | prebuilt (Docker Hub / GHCR) or your own | always built from source | Docker Hub / GHCR |
| For | everyday dev, quick checks | consensus, upgrade and stress tests | shared, long-lived envs |

## Requirements

Docker with Compose v2 (`docker compose version`).

## Quick start

```sh
cd localnet
docker compose up -d
docker compose logs -f validator      # Ctrl-C to stop following
```

The first `up` runs the `init` container, which writes the genesis, keys and
config into the `allora-localnet_data` volume and prints the account
addresses (`docker compose logs init` shows them again). The validator then
starts and produces a block every few seconds.

```sh
curl -s localhost:26657/status | jq .result.sync_info.latest_block_height
```

## Choosing the image

Set `CHAIN_IMAGE` and `CHAIN_VERSION` in the environment or in a `.env` file
next to `docker-compose.yaml` (start from [`.env.example`](.env.example)).

**Release** (default): Docker Hub `alloranetwork/allora-chain`, any `v*` tag.

```sh
CHAIN_VERSION=v0.17.2 docker compose up -d
```

**Unreleased build of any branch or commit** (private GHCR):

1. Run **Actions → Build and Push dev docker image to GHCR → Run workflow**
   from `dev`, with your branch, tag or commit in `ref`.
2. Log in to GHCR once with a GitHub token that has `read:packages`
   (a classic PAT, or `gh auth token` if your gh login has that scope):

   ```sh
   echo "$GITHUB_TOKEN" | docker login ghcr.io -u <github-user> --password-stdin
   ```

3. Use the 8-character commit tag from the run summary:

   ```sh
   CHAIN_IMAGE=ghcr.io/allora-network/allora-chain CHAIN_VERSION=9d13dd26 docker compose up -d
   ```

**Your working tree**:

```sh
docker build -t allora-chain:local ..          # from localnet/, builds the repo root
CHAIN_IMAGE=allora-chain CHAIN_VERSION=local docker compose up -d
```

On Apple Silicon add `CHAIN_PLATFORM=linux/arm64` for an image you built
yourself. Published images are `linux/amd64` and run emulated there.

## Endpoints

Bound to `127.0.0.1` (set `LOCALNET_BIND=0.0.0.0` to expose them):

| | |
|---|---|
| CometBFT RPC | `http://localhost:26657` |
| REST (LCD) | `http://localhost:1317` |
| gRPC (plaintext) | `localhost:9090` |
| Chain id | `allora-localnet-1` (`CHAIN_ID` to change, before the first `up`) |

## Accounts

All keys live in the `test` keyring inside the volume.

| Name | Balance | Role |
|---|---|---|
| `validator0` | 1010 ALLO | the validator, 1000 ALLO self-stake |
| `topic-admin` | 1000 ALLO | in `topic_creator_whitelist` (creates topics) and `core_team_addresses` (whitelist admin: emissions params, whitelists) |
| `faucet` | 1M ALLO | whitelist admin too; fund your own wallets from it |
| `alice`, `bob` | 100 ALLO each | plain test wallets |

Mnemonics are in the volume as `/data/.allorad/<name>.key.json`:

```sh
docker compose exec -T validator jq -r .mnemonic /data/.allorad/faucet.key.json
```

Import one into your local `allorad` with
`allorad keys add faucet --recover --keyring-backend test`.

## Using the CLI

The container's `allorad` already points at the node, the chain id and the
`test` keyring, so commands need no extra flags:

```sh
alias allo='docker compose exec -T validator allorad'   # -T: no TTY, clean output

allo keys list
allo q bank balances "$(allo keys show alice -a)"
allo tx bank send faucet allo1... 1000000000000000000uallo \
  --gas auto --gas-adjustment 1.5 --gas-prices 10uallo -y
allo q emissions params
allo tx emissions create-topic --help     # sign with --from topic-admin
```

Every transaction needs a fee: the node runs with
`--minimum-gas-prices=10uallo`, so pass `--gas-prices 10uallo` (with
`--gas auto`) or an explicit `--fees`.

From the host, any `allorad` works against the published ports, for example
`allorad q bank balances <addr> --node tcp://localhost:26657`.

## Changing the image of a running localnet

Change `CHAIN_VERSION` and run `docker compose up -d` again. The validator
restarts on the new image and keeps its state. That only works if the new
binary can continue the existing state; a state-breaking change halts the
chain. Then either reset (below) or rehearse a real software upgrade with
`test/local_testnet_l1.sh` (`DO_UPGRADE=true`).

## Reset

```sh
docker compose down -v      # deletes the volume: chain state and keys
docker compose up -d        # fresh genesis, new keys and addresses
```

`init` never touches a volume that already has a genesis, so `down` without
`-v` (or `stop`) keeps everything.

## Tuning the genesis

`init.sh` is mounted into the `init` container, so edit it and reset. Knobs
at the top of the script: validator stake, extra accounts and balances,
voting periods and unbonding time. Any other genesis field goes into the
`jq` filter in the same script.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `denied` / `unauthorized` pulling from `ghcr.io` | Not logged in, or the token lacks `read:packages`. See [Choosing the image](#choosing-the-image). |
| `manifest unknown` | The tag does not exist: releases are `v*` on Docker Hub, GHCR has commit tags only for commits someone built. |
| `init` exits non-zero | `docker compose logs init`. Usually a genesis field the image's version does not know; check `init.sh` against that release. |
| Validator restarts, `wrong Block.Header.AppHash` or a panic after changing the image | The new binary is not state-compatible. Reset. |
| `insufficient fee` | Add `--gas-prices 10uallo` (or `--fees`). |
| Port already in use | Another node or devnet port-forward holds 26657/1317/9090; stop it or change the host side of `ports:`. |

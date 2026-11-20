#!/usr/bin/env bash
# Initialise a single-validator Allora localnet in $ALLORA_HOME
# (default $HOME/.allorad, i.e. /data/.allorad in the container).
#
# Run by the `init` service of localnet/docker-compose.yaml before the
# validator starts. Idempotent: once a run has finished (it writes
# $ALLORA_HOME/.localnet-initialized last) it does nothing, so `docker compose
# up` after `docker compose stop` keeps the chain. A home with a complete
# genesis but no marker (made before the marker existed) is kept. A home left
# by a failed run (incomplete genesis, no chain data) is wiped and initialised
# again; anything else with chain data is refused, never deleted.
# `docker compose down -v` wipes the volume and the next `up` makes a fresh
# genesis.
#
# Mirrors the devnet genesis (allora-network/flux-gcp-devnet,
# apps/devnet-template): uallo everywhere, fast governance and unbonding,
# a topic-admin account that may create topics and is a core-team address,
# and a faucet. Every key is in the `test` keyring inside the volume.
set -euo pipefail

# allorad's default home, so `allorad <cmd>` finds client.toml (chain-id,
# keyring-backend, node) without flags. An explicit --home makes allorad
# ignore client.toml's keyring-backend.
ALLORA_HOME="${ALLORA_HOME:-${HOME}/.allorad}"
CHAIN_ID="${CHAIN_ID:-allora-localnet-1}"
MONIKER="${MONIKER:-validator0}"
DENOM=uallo
KEYRING=test
ALLORAD="${ALLORAD:-allorad}"

# Allora sets sdk.DefaultPowerReduction = 1e18 (app.go): a validator needs at
# least 1e18 uallo (1 ALLO) of self-stake for any voting power, otherwise
# InitGenesis ends with an empty validator set.
VALIDATOR_STAKE="${VALIDATOR_STAKE:-1000000000000000000000}"  # 1000 ALLO
VALIDATOR_BALANCE="${VALIDATOR_BALANCE:-1010000000000000000000}"  # stake + 10 ALLO for fees

# Name and balance (uallo) of every extra account. topic-admin is whitelisted
# below; the others are plain funded wallets for testing.
ACCOUNTS=(
  "topic-admin:1000000000000000000000"   # 1000 ALLO
  "faucet:1000000000000000000000000"     # 1M ALLO
  "alice:100000000000000000000"          # 100 ALLO
  "bob:100000000000000000000"            # 100 ALLO
)

GOV_VOTING_PERIOD="${GOV_VOTING_PERIOD:-60s}"
GOV_EXPEDITED_VOTING_PERIOD="${GOV_EXPEDITED_VOTING_PERIOD:-30s}"
UNBONDING_TIME="${UNBONDING_TIME:-60s}"

a() { "${ALLORAD}" --home "${ALLORA_HOME}" "$@"; }

GENESIS="${ALLORA_HOME}/config/genesis.json"
# Written as the very last step: `allorad init` creates genesis.json first, so
# its presence alone does not mean the genesis is complete.
MARKER="${ALLORA_HOME}/.localnet-initialized"
if [ -f "${MARKER}" ]; then
  echo "localnet: ${ALLORA_HOME} is initialised, keeping the existing chain (docker compose down -v to reset)"
  exit 0
fi
if [ -e "${ALLORA_HOME}" ] && [ -n "$(ls -A "${ALLORA_HOME}")" ]; then
  # collect-gentxs is the last step that changes the genesis, so collected
  # gentxs mean a finished home: one made before the marker existed, or a
  # chain supplied through ALLORA_HOME. Keep it and record the marker.
  if [ -f "${GENESIS}" ] \
    && jq -e '(.app_state.genutil.gen_txs // []) | length > 0' "${GENESIS}" > /dev/null 2>&1; then
    echo "localnet: ${ALLORA_HOME} holds a complete genesis, keeping it (docker compose down -v to reset)"
    touch "${MARKER}"
    exit 0
  fi
  # The validator only starts after init succeeds, so a home init left
  # unfinished has never run a node. Block data means it is not ours to wipe.
  if [ -e "${ALLORA_HOME}/data/blockstore.db" ] || [ -e "${ALLORA_HOME}/data/application.db" ]; then
    echo "localnet: ${ALLORA_HOME} has chain data but no complete genesis; not touching it." >&2
    echo "localnet: run 'docker compose down -v' to reset, or point ALLORA_HOME elsewhere." >&2
    exit 1
  fi
  echo "localnet: ${ALLORA_HOME} is left over from an unfinished init, removing it and starting over" >&2
  rm -rf "${ALLORA_HOME:?}"
fi

echo "localnet: initialising ${CHAIN_ID} in ${ALLORA_HOME}"
# stdout only: on failure allorad's error must reach `docker compose logs init`.
a init "${MONIKER}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" > /dev/null

# Client defaults, so `allorad <cmd>` in the container needs no extra flags.
a config set client chain-id "${CHAIN_ID}"
a config set client keyring-backend "${KEYRING}"
a config set client node tcp://localhost:26657

add_account() {
  local name="$1" amount="$2"
  a keys add "${name}" --keyring-backend "${KEYRING}" --output json \
    > "${ALLORA_HOME}/${name}.key.json" 2>&1
  a genesis add-genesis-account "${name}" "${amount}${DENOM}" --keyring-backend "${KEYRING}"
}

add_account "${MONIKER}" "${VALIDATOR_BALANCE}"
for entry in "${ACCOUNTS[@]}"; do
  add_account "${entry%%:*}" "${entry#*:}"
done

TOPIC_ADMIN="$(a keys show topic-admin -a --keyring-backend "${KEYRING}")"
FAUCET="$(a keys show faucet -a --keyring-backend "${KEYRING}")"

# Cosmos SDK defaults that do not follow --default-denom, plus the Allora
# whitelists. Applied before gentx so the gentx is checked against them.
tmp="$(mktemp)"
jq --arg denom "${DENOM}" \
   --arg voting "${GOV_VOTING_PERIOD}" \
   --arg expedited "${GOV_EXPEDITED_VOTING_PERIOD}" \
   --arg unbonding "${UNBONDING_TIME}" \
   --arg topic_admin "${TOPIC_ADMIN}" \
   --arg faucet "${FAUCET}" '
  .app_state.staking.params.bond_denom = $denom
  | .app_state.staking.params.unbonding_time = $unbonding
  | .app_state.mint.params.mint_denom = $denom
  | (if .app_state.crisis then .app_state.crisis.constant_fee.denom = $denom else . end)
  | .app_state.gov.params.voting_period = $voting
  | .app_state.gov.params.expedited_voting_period = $expedited
  | .app_state.gov.params.min_deposit |= map(.denom = $denom)
  | (if .app_state.gov.params.expedited_min_deposit then
       .app_state.gov.params.expedited_min_deposit |= map(.denom = $denom)
     else . end)
  | (if .app_state.feemarket then .app_state.feemarket.params.fee_denom = $denom else . end)
  | .app_state.emissions.core_team_addresses =
      ((.app_state.emissions.core_team_addresses // []) + [$topic_admin, $faucet] | unique)
  | .app_state.emissions.topic_creator_whitelist =
      ((.app_state.emissions.topic_creator_whitelist // []) + [$topic_admin] | unique)
' "${GENESIS}" > "${tmp}"
mv "${tmp}" "${GENESIS}"

a genesis gentx "${MONIKER}" "${VALIDATOR_STAKE}${DENOM}" \
  --chain-id "${CHAIN_ID}" --keyring-backend "${KEYRING}" --moniker "${MONIKER}" \
  > /dev/null
a genesis collect-gentxs > /dev/null
a genesis validate-genesis
touch "${MARKER}"

echo
echo "localnet: genesis ready. Accounts (test keyring, keys in ${ALLORA_HOME}/<name>.key.json):"
for name in "${MONIKER}" $(printf '%s\n' "${ACCOUNTS[@]}" | cut -d: -f1); do
  printf '  %-12s %s\n' "${name}" "$(a keys show "${name}" -a --keyring-backend "${KEYRING}")"
done

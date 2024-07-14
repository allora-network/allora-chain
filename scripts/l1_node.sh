#!/bin/bash
set -ex

NETWORK="${NETWORK:-testnet-1}"                 #! Replace with your network name
GENESIS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/genesis.json"
SEEDS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/seeds.txt"
PEERS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/peers.txt"

export APP_HOME="${APP_HOME:-./data}"
INIT_FLAG="${APP_HOME}/.initialized"
MONIKER="${MONIKER:-$(hostname)}"
KEYRING_BACKEND=test                              #! Use test for simplicity, you should decide which backend to use !!!
GENESIS_FILE="${APP_HOME}/config/genesis.json"
DENOM="uallo"

# uncomment this block if you want to restore from a snapshot
# SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# "${SCRIPT_DIR}/restore_snapshot.sh"

echo "To re-initiate the node, remove the file: ${INIT_FLAG}"
if [ ! -f $INIT_FLAG ]; then
    rm -rf ${APP_HOME}/config

    #* Init node
    allorad --home=${APP_HOME} init ${MONIKER} --chain-id=${NETWORK} --default-denom $DENOM

    #* Download genesis
    rm -f $GENESIS_FILE
    curl -Lo $GENESIS_FILE $GENESIS_URL

    #* Import allora account, priv_validator_key.json and node_key.json from the vault here
    #* Here create a new allorad account
    allorad --home $APP_HOME keys add ${MONIKER} --keyring-backend $KEYRING_BACKEND > $APP_HOME/${MONIKER}.account_info 2>&1

    #* Adjust configs
    #* Enable prometheus metrics
    #dasel put -t bool -v true 'instrumentation.prometheus' -f ${APP_HOME}/config/config.toml

    #* Setup allorad client
    allorad --home=${APP_HOME} config set client chain-id ${NETWORK}
    allorad --home=${APP_HOME} config set client keyring-backend $KEYRING_BACKEND

    #* Create symlink for allorad config
    ln -sf . ${APP_HOME}/.allorad

    touch $INIT_FLAG
fi
echo "Node is initialized"

SEEDS=$(curl -Ls ${SEEDS_URL})
PEERS=$(curl -Ls ${PEERS_URL})

if [ "x${STATE_SYNC_RPC1}" != "x" ]; then
    echo "Enable state sync"
    TRUST_HEIGHT=$(($(curl -s $STATE_SYNC_RPC1/block | jq -r '.result.block.header.height')-1000))
    # TRUST_HEIGHT=93000

    curl -s "$STATE_SYNC_RPC1/block?height=$TRUST_HEIGHT"

    TRUST_HEIGHT_HASH=$(curl -s $STATE_SYNC_RPC1/block?height=$TRUST_HEIGHT | jq -r '.result.block_id.hash')

    echo "$TRUST_HEIGHT====$TRUST_HEIGHT_HASH"
    cat ${APP_HOME}/config/config.toml
    dasel put -t bool -v true 'statesync.enable' -f ${APP_HOME}/config/config.toml
    dasel put -t string -v "$STATE_SYNC_RPC1,$STATE_SYNC_RPC2" 'statesync.rpc_servers' -f ${APP_HOME}/config/config.toml
    dasel put -t int -v $TRUST_HEIGHT 'statesync.trust_height' -f ${APP_HOME}/config/config.toml
    dasel put -t string -v $TRUST_HEIGHT_HASH 'statesync.trust_hash' -f ${APP_HOME}/config/config.toml
    dasel put -t string -v '336h' 'statesync.trust_period' -f ${APP_HOME}/config/config.toml

fi

echo "Starting validator node"
allorad \
    --home=${APP_HOME} \
    start \
    --moniker=${MONIKER} \
    --minimum-gas-prices=0${DENOM} \
    --rpc.laddr=tcp://0.0.0.0:26657 \
    --p2p.seeds=$SEEDS \
    --p2p.persistent_peers $PEERS


#!/bin/bash
set -e

NETWORK="${NETWORK:-allora-testnet-1}"                 #! Replace with your network name
GENESIS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/genesis.json"
SEEDS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/seeds.txt"
PEERS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/peers.txt"
HEADS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/heads.txt"

export APP_HOME="${APP_HOME:-./data}"
INIT_FLAG="${APP_HOME}/.initialized"
MONIKER="${MONIKER:-$(hostname)}"
KEYRING_BACKEND=test                              #! Use test for simplicity, you should decide which backend to use !!!
GENESIS_FILE="${APP_HOME}/config/genesis.json"
DENOM="uallo"
RPC_PORT="${RPC_PORT:-26657}"

if [ "$RESTORE_S3_SNAPSHOT" == "true" ]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    "${SCRIPT_DIR}/restore_snapshot.sh"
fi

BINARY=""
if [ "$UPGRADE" == "true" ]; then
    echo "You have set upgrade to true, please make sure you are running the correct docker image (vx.x.x-upgrader)"
    BINARY=/cosmovisor/genesis/bin/allorad
else
    BINARY=allorad
fi

echo "To re-initiate the node, remove the file: ${INIT_FLAG}"
if [ ! -f $INIT_FLAG ]; then
    #* Remove if existing config
    rm -rf ${APP_HOME}/config

    #* Create symlink for allorad config - workaround
    ln -sf ${APP_HOME} ${HOME}/.allorad

    #* Init node
    $BINARY --home=${APP_HOME} init ${MONIKER} --chain-id=${NETWORK} --default-denom $DENOM

    #* Download genesis
    rm -f $GENESIS_FILE
    curl -Lo $GENESIS_FILE $GENESIS_URL

    #* Import allora account, priv_validator_key.json and node_key.json from the vault here
    #* Here create a new allorad account
    $BINARY --home $APP_HOME keys add ${MONIKER} --keyring-backend $KEYRING_BACKEND > $APP_HOME/${MONIKER}.account_info 2>&1

    #* Adjust configs
    #* Enable prometheus metrics
    #dasel put -t bool -v true 'instrumentation.prometheus' -f ${APP_HOME}/config/config.toml

    #* Setup allorad client
    $BINARY  --home=${APP_HOME} config set client chain-id ${NETWORK}
    $BINARY  --home=${APP_HOME} config set client keyring-backend $KEYRING_BACKEND

    #* Mitigate mempool spamming attacks
    dasel put mempool.max_txs_bytes -t int -v 2097152 -f ${APP_HOME}/config/config.toml
    dasel put mempool.size -t int -v 1000 -f ${APP_HOME}/config/config.toml

    #* Enable telemetry
    dasel put telemetry.enabled -t bool -v true -f ${APP_HOME}/config/app.toml

    touch $INIT_FLAG
fi
echo "Node is initialized"

SEEDS=$(curl -Ls ${SEEDS_URL})
PEERS=$(curl -Ls ${PEERS_URL})

if [ "x${STATE_SYNC_RPC1}" != "x" ]; then
    echo "Enable state sync"
    TRUST_HEIGHT=$(($(curl -s $STATE_SYNC_RPC1/block | jq -r '.result.block.header.height')))

    #* Snapshots are taken every 1000 blocks so we need to round down to the nearest 1000
    TRUST_HEIGHT=$(($TRUST_HEIGHT - ($TRUST_HEIGHT % 1000)))

    curl -s "$STATE_SYNC_RPC1/block?height=$TRUST_HEIGHT"

    TRUST_HEIGHT_HASH=$(curl -s $STATE_SYNC_RPC1/block?height=$TRUST_HEIGHT | jq -r '.result.block_id.hash')

    echo "Trust height: $TRUST_HEIGHT $TRUST_HEIGHT_HASH"

    dasel put statesync.enable -t bool -v true -f ${APP_HOME}/config/config.toml
    dasel put statesync.rpc_servers -t string -v "$STATE_SYNC_RPC1,$STATE_SYNC_RPC2" -f ${APP_HOME}/config/config.toml
    dasel put statesync.trust_height -t string -v $TRUST_HEIGHT -f ${APP_HOME}/config/config.toml
    dasel put statesync.trust_hash -t string -v $TRUST_HEIGHT_HASH -f ${APP_HOME}/config/config.toml
fi

if [ "$UPGRADE" == "true" ]; then
    if [ ! -d "/data/cosmovisor" ]; then
        echo "initialize cosmovisor"
        cp -R /cosmovisor /data/
        cosmovisor init /data/cosmovisor/genesis/bin/allorad
    fi

    echo "Starting validator node with cosmovisor"
    cosmovisor \
        run \
        --home=${APP_HOME} \
        start \
        --moniker=${MONIKER} \
        --minimum-gas-prices=0${DENOM} \
        --rpc.laddr=tcp://0.0.0.0:26657 \
        --p2p.seeds=$SEEDS \
        --p2p.persistent_peers $PEERS
else
    echo "Starting validator node without cosmovisor"
    allorad \
        --home=${APP_HOME} \
        start \
        --moniker=${MONIKER} \
        --minimum-gas-prices=0${DENOM} \
        --rpc.laddr=tcp://0.0.0.0:26657 \
        --p2p.seeds=$SEEDS \
        --p2p.persistent_peers $PEERS
fi

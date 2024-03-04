#!/bin/bash
set -exu

NETWORK="${NETWORK:-edgenet}"
GENESIS_URL="https://raw.githubusercontent.com/upshot-tech/networks/main/${NETWORK}/genesis.json"
BLOCKLESS_API_URL="https://heads.edgenet.allora.network:8443"            #! Replace with your blockless API URL


APP_HOME="${APP_HOME:-./data}"
INIT_FLAG="${APP_HOME}/.initialized"
MONIKER="${MONIKER:-$(hostname)}"
KEYRING_BACKEND=test                              #! Use test for simplicity, you should decide which backend to use !!!
GENESIS_FILE="${APP_HOME}/config/genesis.json"
DENOM="uallo"

echo "To re-initiate the node, remove the file: ${INIT_FLAG}"
if [ ! -f $INIT_FLAG ]; then
    rm -rf ${APP_HOME}

    #* Init node
    allorad --home=${APP_HOME} init ${MONIKER} --chain-id=${NETWORK} --default-denom $DENOM

    #* Download genesis
    rm -f $GENESIS_FILE
    curl -Lo $GENESIS_FILE $GENESIS_URL

    #* Import allora account, priv_validator_key.json and node_key.json from the vault here
    #* Here create a new allorad account
    allorad --home $APP_HOME keys add ${MONIKER} --keyring-backend $KEYRING_BACKEND > ${MONIKER}.account_info 2>&1

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

# PEERS=$(curl -s ${PEERS_URL})
PEERS="54f2c6967576e8287e5cea8614932581f8e80c14@peer-0.edgenet.allora.network:30003,e66473795b9893ebfd673a3c24e07a3c16c6b7e7@peer-1.edgenet.allora.network:30004,734990443d4f1225966999e316f5c36d140b9f44@peer-2.edgenet.allora.network:30005"

echo "Starting validator node"
allorad \
    --home=${APP_HOME} \
    start \
    --moniker=${MONIKER} \
    --minimum-gas-prices=0${DENOM} \
    --p2p.persistent_peers=${PEERS}


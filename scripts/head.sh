#!/bin/bash
set -eu

NETWORK="${NETWORK:-allora-testnet-1}"                 #! Replace with your network name
HEADS_URL="https://raw.githubusercontent.com/allora-network/networks/main/${NETWORK}/heads.txt"

DIALBACK_ADDRESS="${DIALBACK_ADDRESS:-my.dialback.address}"
DIALBACK_PORT="${DIALBACK_PORT:-9010}"
export APP_HOME="${APP_HOME:-./data}"

KEY_FILE="${APP_HOME}/keys/private.key"
if [ ! -f $KEY_FILE ]; then
    echo "Generate p2p keys"
    mkdir -p ${APP_HOME}/keys
    pushd ${APP_HOME}/keys
    allora-keys
    popd
fi

HEADS=$(curl -Ls ${HEADS_URL})

echo "Starting validator node"
allora-node \
    --role=head \
    --peer-db=$APP_HOME/peer-database \
    --function-db=$APP_HOME/function-database \
    --workspace=/tmp/node \
    --private-key=$APP_HOME/keys/priv.bin \
    --port=9010 \
    --rest-api=:6000 \
    --dialback-address=$DIALBACK_ADDRESS \
    --dialback-port=$DIALBACK_PORT \
    --log-level=debug \
    --boot-nodes=$HEADS

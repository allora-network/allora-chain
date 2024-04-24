#!/bin/bash
set -eu

# Ensure we're in integration folder
cd "$(dirname "$0")"

DOCKER_IMAGE=allorad
VALIDATOR_NUMBER=3
VALIDATOR_PREFIX=validator
NETWORK_PREFIX="172.20.0"
VALIDATORS_IP_START=10
HEADS_IP_START=20
CHAIN_ID="testnet"
LOCALNET_DATADIR="./$CHAIN_ID"

ACCOUNTS_TOKENS=1000000



ENV_L1="${LOCALNET_DATADIR}/.env"
L1_COMPOSE="compose_l1.yaml"

mkdir -p $LOCALNET_DATADIR
echo "UID_GID=$(id -u):$(id -g)" > ${ENV_L1}
echo "NETWORK_PREFIX=$NETWORK_PREFIX" >> ${ENV_L1}
echo "CHAIN_ID=$CHAIN_ID" >> ${ENV_L1}
echo "ALLORA_RPC=http://${NETWORK_PREFIX}.10:26657" >> ${ENV_L1}  # Take validator0

echo "Build the docker image"
pushd ..
docker build --pull -t $DOCKER_IMAGE -f ./Dockerfile.development .
popd

echo "Download generate_genesis.sh from testnet"
mkdir -p ${LOCALNET_DATADIR}
curl -so- https://raw.githubusercontent.com/allora-network/networks/main/testnet/generate_genesis.sh > ${LOCALNET_DATADIR}/generate_genesis.sh
chmod a+x ${LOCALNET_DATADIR}/generate_genesis.sh

echo "Set permissions on data folder"
docker run \
    -u 0:0 \
    -v ${LOCALNET_DATADIR}:/data \
    -e COMMON_HOME_DIR=/data \
    --entrypoint=chown \
    $DOCKER_IMAGE -R $(id -u):$(id -g) /data

echo "Generate genesis and accounts"
docker run \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e COMMON_HOME_DIR=/data \
    -e HOME=/data \
    --entrypoint=/data/generate_genesis.sh \
    $DOCKER_IMAGE

echo "Whitelist faucet account"
FAUCET_ADDRESS=$(docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    $DOCKER_IMAGE \
        --home=/data/genesis keys show faucet -a --keyring-backend=test)
FAUCET_ADDRESS="${FAUCET_ADDRESS%%[[:cntrl:]]}"

echo FAUCET_ADDRESS=$FAUCET_ADDRESS >> ${ENV_L1}
docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=dasel \
    $DOCKER_IMAGE \
        put -t string -v "$FAUCET_ADDRESS" 'app_state.emissions.core_team_addresses.append()' -f /data/genesis/config/genesis.json
echo "Faucet addr: $FAUCET_ADDRESS"

echo "Generate L1 peers"
PEERS=""
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    valName="${VALIDATOR_PREFIX}${i}"
    ipAddress="${NETWORK_PREFIX}.$((VALIDATORS_IP_START+i))"
    addr=$(docker run -t \
        -v ${LOCALNET_DATADIR}:/data \
        -u $(id -u):$(id -g) \
        -e HOME=/data/${valName} \
        $DOCKER_IMAGE \
        --home=/data/${valName} tendermint show-node-id)
    addr="${addr%%[[:cntrl:]]}"
    delim=$([ $i -lt $(($VALIDATOR_NUMBER - 1)) ] && printf "," || printf "")
    PEERS="${PEERS}${addr}@${ipAddress}:26656${delim}"
done

echo "PEERS=$PEERS" >> ${ENV_L1}

echo "Launching the network"
docker compose --env-file ${ENV_L1} -f $L1_COMPOSE up -d

echo "Waiting validator is up"
curl -o /dev/null --connect-timeout 5 \
    --retry 10 \
    --retry-delay 10 \
    --retry-all-errors \
    http://${NETWORK_PREFIX}.${VALIDATORS_IP_START}:26657/status

echo "Checking the network is up and running"
heights=()
validators=()
for ((v=0; v<$VALIDATOR_NUMBER; v++)); do
    height=$(curl -s http://${NETWORK_PREFIX}.$((VALIDATORS_IP_START+v)):26657/status|jq -r .result.sync_info.latest_block_height)
    heights+=($height)
    echo "Got height: ${heights[$v]} from validator: ${NETWORK_PREFIX}.$((VALIDATORS_IP_START+v))"
    validators+=("${NETWORK_PREFIX}.$((VALIDATORS_IP_START+v))")
    sleep 5
done
echo "Populate validators.json with validators addresses"
jq --compact-output --null-input '$ARGS.positional' --args -- "${validators[@]}" > ${LOCALNET_DATADIR}/validators.json

chain_status=0
if [ ${#heights[@]} -eq $VALIDATOR_NUMBER ]; then
    for ((v=0; v<$((VALIDATOR_NUMBER-1)); v++)); do
        if [ ${heights[$v]} -lt ${heights[$((v+1))]} ]; then
            chain_status=$((chain_status+1))
        fi
    done
fi

if [ $chain_status -eq $((VALIDATOR_NUMBER-1)) ]; then
    echo "Chain is up and running"
else
    echo "Chain is not producing blocks"
    echo "If run localy you can check the logs with: docker logs allorad_validator_0"
    echo "and connect to the validators ..."
    exit 1
fi

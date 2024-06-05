#!/bin/bash
set -eu  #e

# Ensure we're in integration folder
cd "$(dirname "$0")"

DOCKER_IMAGE=allorad
VALIDATOR_NUMBER="${VALIDATOR_NUMBER:-3}"
VALIDATOR_PREFIX=validator
NETWORK_PREFIX="192.168.250"
VALIDATORS_IP_START=10
VALIDATORS_RPC_PORT_START=26657
HEADS_IP_START=20
CHAIN_ID="${CHAIN_ID:-devnet}"
LOCALNET_DATADIR="$(pwd)/$CHAIN_ID"

ACCOUNTS_TOKENS=1000000

ENV_L1="${LOCALNET_DATADIR}/.env"
L1_COMPOSE=${LOCALNET_DATADIR}/compose_l1.yaml

if [ -d "$LOCALNET_DATADIR" ]; then
    echo "Folder $LOCALNET_DATADIR already exist, need to delete it before running the script."
    read -p "Stop validators and Delete $LOCALNET_DATADIR folder??[y/N] " -n 1 -r
    echo    # (optional) move to a new line
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Stop if containers are already up
        [ $(docker ps |wc -l) -gt 1 ]  && docker compose -f $L1_COMPOSE down
        rm -rf $LOCALNET_DATADIR
    fi
fi
mkdir -p $LOCALNET_DATADIR

UID_GID="$(id -u):$(id -g)"
# echo "NETWORK_PREFIX=$NETWORK_PREFIX" >> ${ENV_L1}
echo "CHAIN_ID=$CHAIN_ID" >> ${ENV_L1}
echo "ALLORA_RPC=http://${NETWORK_PREFIX}.${VALIDATORS_IP_START}:26657" >> ${ENV_L1}  # Take validator0

echo "Build the docker image"
pushd ..
docker build -t $DOCKER_IMAGE -f ./Dockerfile.development .
popd

echo "Download generate_genesis.sh from testnet"
mkdir -p ${LOCALNET_DATADIR}
curl -so- https://raw.githubusercontent.com/allora-network/networks/main/${CHAIN_ID}/generate_genesis.sh > ${LOCALNET_DATADIR}/generate_genesis.sh
chmod a+x ${LOCALNET_DATADIR}/generate_genesis.sh

echo "Set permissions on data folder"
docker run \
    -u 0:0 \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=chown \
    $DOCKER_IMAGE -R $(id -u):$(id -g) /data

docker run \
    -u 0:0 \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=chmod \
    $DOCKER_IMAGE -R 777 /data

echo "Generate genesis and accounts"
docker run \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e COMMON_HOME_DIR=/data \
    -e HOME=/data \
    -e VALIDATOR_NUMBER=$VALIDATOR_NUMBER \
    --entrypoint=/data/generate_genesis.sh \
    $DOCKER_IMAGE > /dev/null 2>&1

echo "Updating expedited_voting_period in genesis.json"
genesis_file="${LOCALNET_DATADIR}/genesis/config/genesis.json"
tmp_file=$(mktemp)
jq '.app_state.gov.params.expedited_voting_period = "120s" | .app_state.gov.params.voting_period = "120s"' "$genesis_file" > "$tmp_file" && mv "$tmp_file" "$genesis_file"

echo "Whitelist faucet account"
FAUCET_ADDRESS=$(docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=allorad \
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

echo "Running cosmovisor init"
docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=cosmovisor \
    --env DAEMON_HOME=/data \
    --env DAEMON_NAME=allorad \
    $DOCKER_IMAGE \
        init /usr/local/bin/allorad

echo "Generate L1 peers"
PEERS=""
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    valName="${VALIDATOR_PREFIX}${i}"
    ipAddress="${NETWORK_PREFIX}.$((VALIDATORS_IP_START+i))"
    addr=$(docker run -t \
        -v ${LOCALNET_DATADIR}:/data \
        -u $(id -u):$(id -g) \
        --entrypoint=allorad \
        -e HOME=/data/${valName} \
        $DOCKER_IMAGE \
        --home=/data/${valName} tendermint show-node-id)
    addr="${addr%%[[:cntrl:]]}"
    delim=$([ $i -lt $(($VALIDATOR_NUMBER - 1)) ] && printf "," || printf "")
    PEERS="${PEERS}${addr}@${ipAddress}:26656${delim}"
done

echo "PEERS=$PEERS" >> ${ENV_L1}
echo "Generate docker compose file"
NETWORK_PREFIX=$NETWORK_PREFIX envsubst < compose_l1_header.yaml > $L1_COMPOSE
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    ipAddress="${NETWORK_PREFIX}.$((VALIDATORS_IP_START+i))" \
    moniker="${VALIDATOR_PREFIX}${i}" \
    validatorPort=$((VALIDATORS_RPC_PORT_START+i)) \
    PEERS=$PEERS \
    NETWORK_PREFIX=$NETWORK_PREFIX \
    LOCALNET_DATADIR=$LOCALNET_DATADIR \
    UID_GID=$UID_GID \
    envsubst < validator.tmpl >> $L1_COMPOSE
done

echo "Launching the network"
DAEMON_HOME=/data DAEMON_NAME=allorad docker compose -f $L1_COMPOSE up -d

echo "Waiting validator is up"
curl -o /dev/null --connect-timeout 5 \
    --retry 10 \
    --retry-delay 10 \
    --retry-all-errors \
    http://localhost:$VALIDATORS_RPC_PORT_START/status

echo "Checking the network is up and running"
heights=()
validators=()
for ((v=0; v<$VALIDATOR_NUMBER; v++)); do
    height=$(curl -s http://localhost:$((VALIDATORS_RPC_PORT_START+v))/status|jq -r .result.sync_info.latest_block_height)
    heights+=($height)
    echo "Got height: ${heights[$v]} from validator: http://localhost:$((VALIDATORS_RPC_PORT_START+v))"
    validators+=("localhost:$((VALIDATORS_RPC_PORT_START+v))")
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

docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/${valName} \
    --entrypoint=allorad \
    $DOCKER_IMAGE \
    --home /data/genesis config set client keyring-backend test

if [ $chain_status -eq $((VALIDATOR_NUMBER-1)) ]; then
    echo "Chain is up and running"
    echo
    echo "Some useful commands:"
    echo "  - 'docker compose -f $L1_COMPOSE logs -f' -- To see logs of the containers"
    echo "  - 'docker compose -f $L1_COMPOSE logs -f validator[0-...]' -- To see logs of the specified validator"
    echo "  - 'docker compose -f $L1_COMPOSE down' -- To stop all the validators"
    echo "  - http://localhost:2665[7-...] -- Validators RPC address, port = 26657 + VALIDATOR_NUMBER"
    echo "  -   - 'curl http://localhost:26658/status|jq .' -- To get validator1 (26657+1=26658) RPC address"
    echo "To use allorad commands, you can specify \'$LOCALNET_DATADIR/genesis\' as --home, eg.:"
    echo "  - 'allorad --home $LOCALNET_DATADIR/genesis status'"
else
    echo "Chain is not producing blocks"
    echo "If run localy you can check the logs with: docker logs allorad_validator_0"
    echo "and connect to the validators ..."
    exit 1
fi

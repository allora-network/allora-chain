#!/bin/bash
set -exu

DOCKER_IMAGE=allorad
VALIDATOR_NUMBER=3
VALIDATOR_PREFIX=validator
NETWORK_PREFIX="172.20.0."
VALIDATORS_IP_START=10
HEADS_IP_START=20
CHAIN_ID="testnet"
LOCALNET_DATADIR="./$CHAIN_ID"
# VALIDATORS_IP_START=10
# PEERS=peers.txt
# ALLORA_RPC="http://localhost:26657"
ACCOUNTS_TOKENS=1000000

ALLORA_RPC="http://172.20.0.10:26657"  # Take validator0

WEIGHT_CADENCE=10800
INFERENCE_CADENCE=61

echo "Build the docker image"
pushd ..
docker build --pull -t $DOCKER_IMAGE -f ./Dockerfile.development .
popd

echo "Download generate_genesis.sh from testnet"
mkdir -p ${LOCALNET_DATADIR}
curl -so- https://raw.githubusercontent.com/allora-network/networks/main/testnet/generate_genesis.sh > ${LOCALNET_DATADIR}/generate_genesis.sh
chmod a+x ${LOCALNET_DATADIR}/generate_genesis.sh

echo "Set permissions on data folder"
docker run -it \
    -u 0:0 \
    -v ${LOCALNET_DATADIR}:/data \
    -e COMMON_HOME_DIR=/data \
    --entrypoint=chown \
    $DOCKER_IMAGE -R $(id -u):$(id -g) /data

echo "Generate genesis and accounts"
docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e COMMON_HOME_DIR=/data \
    -e HOME=/data \
    --entrypoint=/data/generate_genesis.sh \
    $DOCKER_IMAGE

echo "Whitelist faucet account"
FAUCET_ADDRESS=$(docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    $DOCKER_IMAGE \
        --home=/data/genesis keys show faucet -a --keyring-backend=test)
FAUCET_ADDRESS="${FAUCET_ADDRESS%%[[:cntrl:]]}"

docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    --entrypoint=dasel \
    $DOCKER_IMAGE \
        put -t string -v "$FAUCET_ADDRESS" 'app_state.emissions.core_team_addresses.append()' -f /data/genesis/config/genesis.json
echo "Faucet addr: $FAUCET_ADDRESS"

echo "Generate peers.txt"
PEERS=""
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    valName="${VALIDATOR_PREFIX}${i}"
    ipAddress="${NETWORK_PREFIX}$((VALIDATORS_IP_START+i))"
    addr=$(docker run -it \
        -v ${LOCALNET_DATADIR}:/data \
        -u $(id -u):$(id -g) \
        -e HOME=/data/${valName} \
        $DOCKER_IMAGE \
        --home=/data/${valName} tendermint show-node-id)
    addr="${addr%%[[:cntrl:]]}"
    delim=$([ $i -lt $(($VALIDATOR_NUMBER - 1)) ] && printf "," || printf "")
    PEERS="${PEERS}${addr}@${ipAddress}:26656${delim}"
done

echo "Launching the network"
PEERS=$PEERS docker compose up -d validator0 validator1 validator2

echo "Waiting validator is up"
curl --connect-timeout 5 \
    --retry 10 \
    --retry-delay 10 \
    --retry-all-errors \
    http://172.20.0.10:26657/status

echo "Checking the network is up and running"
heights=()
for ((v=0; v<$VALIDATOR_NUMBER; v++)); do
    height=$(curl -s http://172.20.0.$((VALIDATORS_IP_START+v)):26657/status|jq -r .result.sync_info.latest_block_height)
    heights+=($height)
    sleep 5
done

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

echo "Generating allora account keys for heads and workers and funding them"
accounts=("head0" "worker0")

for account in "${accounts[@]}"; do
    echo "Generating allora account key for $account"

    mkdir -p ${LOCALNET_DATADIR}/${account}
    ln -sfr ${LOCALNET_DATADIR}/genesis ${LOCALNET_DATADIR}/${account}/.allorad
    # ln -sfr ${LOCALNET_DATADIR}/genesis/keyring-test ${LOCALNET_DATADIR}/${account}/keyring-test

    docker run -it \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        -e HOME=/data/${account} \
        $DOCKER_IMAGE \
            --home=/data/${account}/.allorad keys add --keyring-backend=test $account > ${LOCALNET_DATADIR}/$account.account_info 2>&1

    account_address=$(docker run -it \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        -e HOME=/data/${account} \
        $DOCKER_IMAGE \
            --home=/data/${account}/.allorad keys show $account -a --keyring-backend=test)
    account_address="${account_address%%[[:cntrl:]]}"

    echo "Funding $account with $ACCOUNTS_TOKENS tokens from faucet"
    docker run -it \
        --network host \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        -e HOME=/data/genesis \
        $DOCKER_IMAGE \
            --home=/data/genesis tx bank send --keyring-backend=test \
            faucet $account_address ${ACCOUNTS_TOKENS}uallo \
            --fees=200000uallo --yes --node $ALLORA_RPC --chain-id $CHAIN_ID

    sleep 5

    echo "Initializing $account p2p keys"
    docker run -it \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        --entrypoint=bash \
        alloranetwork/allora-inference-base-head:latest \
        -c "mkdir -p /data/$account/key && cd /data/$account/key && allora-keys"
done
##########################################
echo "Register topics Linear function"
docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    --network host \
    $DOCKER_IMAGE \
        --home=/data/genesis tx emissions push-topic $FAUCET_ADDRESS "Linear 24h Prediction" \
        bafybeih6yjjjf2v7qp3wm6hodvjcdljj7galu7dufirvcekzip5gd7bthq eth-price-weights-calc.wasm $WEIGHT_CADENCE \
        bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65jiky2xqb4rdhfmikswzqm allora-inference-function.wasm $INFERENCE_CADENCE \
        "ETH" --node=$ALLORA_RPC --keyring-backend=test --keyring-dir=/data/genesis --chain-id $CHAIN_ID --yes
sleep 5

docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    --network host \
    $DOCKER_IMAGE \
        --home=/data/genesis tx emissions request-inference $FAUCET_ADDRESS \
        '{"nonce": "1","topic_id":"1","cadence":"60","max_price_per_inference":"1","bid_amount":"10000","timestamp_valid_until":"'$(date -d "$(date -d '1 day' +%Y-%m-%d)" +%s)'"}' \
        --node=$ALLORA_RPC --keyring-backend=test --keyring-dir=/data/genesis --chain-id $CHAIN_ID --yes
sleep 5

docker run -it \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    --network host \
    $DOCKER_IMAGE \
        --home=/data/genesis  tx emissions reactivate-topic $FAUCET_ADDRESS 1 \
        --node=$ALLORA_RPC --keyring-backend=test --keyring-dir=/data/genesis --chain-id $CHAIN_ID --yes
sleep 5

exit

# HEAD0_IDENTITY=$(cat ${LOCALNET_DATADIR}/head0/key/identity)

# ln -sfr ${LOCALNET_DATADIR}/genesis ${LOCALNET_DATADIR}/head0/.allorad

# PEERS=$PEERS docker compose up -d validator0 validator1 validator2 head0






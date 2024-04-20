#!/bin/bash
set -exu


DOCKER_IMAGE=allorad
VALIDATOR_NUMBER=3
VALIDATOR_PREFIX=validator
NETWORK_PREFIX="172.20.0."
VALIDATORS_IP_START=10
HEADS_IP_START=20
# VALIDATORS_IP_START=10
# PEERS=peers.txt
ALLORA_RPC="http://localhost:26657"

echo "Build the docker image"
pushd ..
docker build --pull -t $DOCKER_IMAGE -f ./Dockerfile.development .
popd

# Download generate_genesis.sh from testnet
curl -so- https://raw.githubusercontent.com/allora-network/networks/main/testnet/generate_genesis.sh > generate_genesis.sh
chmod a+x generate_genesis.sh

docker run -it \
    -u 0:0 \
    -v ./testnet:/data \
    -e COMMON_HOME_DIR=/data \
    --entrypoint=chown \
    $DOCKER_IMAGE -R $(id -u):$(id -g) /data

echo "Generate genesis and accounts"
docker run -it \
    -u $(id -u):$(id -g) \
    -v ./generate_genesis.sh:/scripts/generate_genesis.sh \
    -v ./testnet:/data \
    -e COMMON_HOME_DIR=/data \
    -e HOME=/data \
    --entrypoint=/scripts/generate_genesis.sh \
    $DOCKER_IMAGE

echo "Generate peers.txt"
PEERS=""
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    valName="${VALIDATOR_PREFIX}${i}"
    ipAddress="${NETWORK_PREFIX}$((VALIDATORS_IP_START+i))"
    addr=$(docker run -it \
        -v ./testnet:/data \
        -u $(id -u):$(id -g) \
        -e HOME=/data/${valName} \
        $DOCKER_IMAGE \
        --home=/data/${valName} tendermint show-node-id)
    addr="${addr%%[[:cntrl:]]}"
    delim=$([ $i -lt $(($VALIDATOR_NUMBER - 1)) ] && printf "," || printf "")
    PEERS="${PEERS}${addr}@${ipAddress}:26656${delim}"
done

echo "Launching the network"
PEERS=$PEERS docker compose up -d

echo "Wait node is up"
curl --connect-timeout 5 \
    --retry 10 \
    --retry-delay 10 \
    --retry-all-errors \
    http://172.20.0.10:26657/status

echo "Checking the network is up"
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
    echo "SOMETHING wrong!!!!"
fi

curl -s http://172.20.0.11:26657/status|jq .result.sync_info.latest_block_height
# echo "Check that the chain is running"
# curl -s -o /dev/null -w "%{http_code}" $ALLORA_RPC/status

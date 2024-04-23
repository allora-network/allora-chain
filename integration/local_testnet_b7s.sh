#!/bin/bash
set -exu

# Ensure we're in integration folder
cd "$(dirname "$0")"

CHAIN_ID="testnet"

LOCALNET_DATADIR="./$CHAIN_ID" # Expects to have keyring-test with all the keys in $LOCALNET_DATADIR/genesis/config/keyring-test
ENV_L1="${LOCALNET_DATADIR}/.env"

source $ENV_L1

# UID_GID=1000:1000
# NETWORK_PREFIX=172.20.0
# CHAIN_ID=testnet
# ALLORA_RPC=http://172.20.0.10:26657
# FAUCET_ADDRESS=allo1ahdn2amw62jh99eu863m0afkavunzu3dwgy8zr
# PEERS=c0901ae621f569acbdb58053d73fa9e48cfa92f4@172.20.0.10:26656,3a87503807101f5fd2d765c677be5e44b3698758@172.20.0.11:26656,a77a16894fb32065d6c11fcbacda5a7d950f8ad0@172.20.0.12:26656

HEADS_IP_START=20
DOCKER_IMAGE=allorad
VALIDATOR_PREFIX=validator

ACCOUNTS_TOKENS=1000000
WEIGHT_CADENCE=10800
INFERENCE_CADENCE=61


ENV_B7S="${LOCALNET_DATADIR}/.env_b7s"
> $ENV_B7S


echo "Generating allora account keys for heads and workers and funding them"
accounts=("head0" "worker0" "worker1")

for account in "${accounts[@]}"; do
    echo "Generating allora account key for $account"

    mkdir -p ${LOCALNET_DATADIR}/${account}
    cp -r ${LOCALNET_DATADIR}/genesis ${LOCALNET_DATADIR}/${account}/.allorad #Copy to be able to set different permissions
    # ln -sfr ${LOCALNET_DATADIR}/genesis/keyring-test ${LOCALNET_DATADIR}/${account}/keyring-test

    docker run -t \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        -e HOME=/data/${account} \
        $DOCKER_IMAGE \
            --home=/data/${account}/.allorad keys add --keyring-backend=test $account > ${LOCALNET_DATADIR}/$account.account_info 2>&1

    account_address=$(docker run -t \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        -e HOME=/data/${account} \
        $DOCKER_IMAGE \
            --home=/data/${account}/.allorad keys show $account -a --keyring-backend=test)
    account_address="${account_address%%[[:cntrl:]]}"

    echo "Funding $account with $ACCOUNTS_TOKENS tokens from faucet"
    docker run -t \
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
    docker run -t \
        -u $(id -u):$(id -g) \
        -v ${LOCALNET_DATADIR}:/data \
        --entrypoint=bash \
        alloranetwork/allora-inference-base-head:latest \
        -c "mkdir -p /data/$account/key && cd /data/$account/key && allora-keys"

    # Adjust permissions
    sudo chown -R 1001:1001 ${LOCALNET_DATADIR}/$account

done
##########################################
echo "Register topics Linear function"
docker run -t \
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

docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    --network host \
    $DOCKER_IMAGE \
        --home=/data/genesis tx emissions request-inference $FAUCET_ADDRESS \
        '{"nonce": "1","topic_id":"1","cadence":"60","max_price_per_inference":"1","bid_amount":"10000","timestamp_valid_until":"'$(date -d "$(date -d '1 day' +%Y-%m-%d)" +%s)'"}' \
        --node=$ALLORA_RPC --keyring-backend=test --keyring-dir=/data/genesis --chain-id $CHAIN_ID --yes
sleep 5

docker run -t \
    -u $(id -u):$(id -g) \
    -v ${LOCALNET_DATADIR}:/data \
    -e HOME=/data/genesis \
    --network host \
    $DOCKER_IMAGE \
        --home=/data/genesis  tx emissions reactivate-topic $FAUCET_ADDRESS 1 \
        --node=$ALLORA_RPC --keyring-backend=test --keyring-dir=/data/genesis --chain-id $CHAIN_ID --yes
sleep 5

echo "HEADS=/ip4/172.20.0.20/tcp/9010/p2p/$(cat ${LOCALNET_DATADIR}/head0/key/identity)" >> $ENV_B7S

docker compose -f compose_b7s.yaml --env-file $ENV_L1 --env-file $ENV_B7S up -d




exit
# docker compose up --env-file .env -d validator0 validator1 validator2 head0 worker0
# HEAD0_IDENTITY=$(cat ${LOCALNET_DATADIR}/head0/key/identity)

# ln -sfr ${LOCALNET_DATADIR}/genesis ${LOCALNET_DATADIR}/head0/.allorad

# PEERS=$PEERS docker compose up -d validator0 validator1 validator2 head0






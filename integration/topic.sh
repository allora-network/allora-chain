#!/usr/bin/env bash

set -e

source $(dirname $0)/common.sh

# this script expects to be ran AFTER `scripts/init.sh`
if ! test -f $GENESIS; then
  echo "Must run scripts/init.sh first."
  exit 1
fi

echo "Checking that the network is starting from topic 0"
NEXT_TOPIC_ID=$($ALLORAD_BIN query emissions next-topic-id | head -n 1 | cut -f 2 -d ":" | tr -d " " | tr -d "\"")
if [ "$NEXT_TOPIC_ID" != "1" ]; then
  echo "The network is not starting from topic 0. It is starting from topic $NEXT_TOPIC_ID"
  exit 1
fi

echo "Creating topic 1"
PT_CREATOR="$ALICE_ADDRESS"
PT_METADATA="ETH 24h Prediction"
PT_LOSS_LOGIC="bafybeih6yjjjf2v7qp3wm6hodvjcdljj7galu7dufirvcekzip5gd7bthq"
PT_LOSS_METHOD="eth-price-weights-calc.wasm"
PT_INFERENCE_LOGIC="bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65jiky2xqb4rdhfmikswzqm"
PT_INFERENCE_METHOD="allora-inference-function.wasm"
PT_EPOCH_LENGTH="10800"
PT_DEFAULT_ARG="ETH"
PT_PNORM="2"
PT_ALPHA_REGRET="3.14"
PT_PREREWARD_REPUTER="6.2"
PT_PREREWARD_INFERENCE="7.3"
PT_PREREWARD_FORECAST="8.4"
PT_F_TOLERANCE="5.5"
$ALLORAD_BIN tx emissions create-topic \
  "$PT_CREATOR" \
  "$PT_METADATA" \
  "$PT_LOSS_LOGIC" \
  "$PT_LOSS_METHOD" \
  "$PT_INFERENCE_LOGIC" \
  "$PT_INFERENCE_METHOD" \
  "$PT_EPOCH_LENGTH" \
  "$PT_DEFAULT_ARG" \
  "$PT_PNORM" \
  "$PT_ALPHA_REGRET" \
  "$PT_PREREWARD_REPUTER" \
  "$PT_PREREWARD_INFERENCE" \
  "$PT_PREREWARD_FORECAST" \
  "$PT_F_TOLERANCE" \
  --yes --keyring-backend=test --chain-id=demo \
  --gas-prices=1uallo --gas=auto --gas-adjustment=1.5;


echo "Checking that the network has incremented the topic count"
TOPIC_INCREMENTED=false
for COUNT_SLEEP in 1 2 3 4 5
do
  NEXT_TOPIC_ID=$($ALLORAD_BIN query emissions next-topic-id | head -n 1 | cut -f 2 -d ":" | tr -d " " | tr -d "\"")
  if [ "$NEXT_TOPIC_ID" != "2" ]; then
    echo "$NEXT_TOPIC_ID is not 2, transaction may not have mined yet, count sleep $COUNT_SLEEP seconds"
    COUNT_SLEEP=$((COUNT_SLEEP+1))
    sleep 1
  else
    echo "The network has incremented the topic count, topic probably created successfully"
    TOPIC_INCREMENTED=true
    break
  fi
done
if [ "$TOPIC_INCREMENTED" = false ]; then
  echo "The network has not incremented the topic count"
  exit 1
fi

echo "Topic 1 created"

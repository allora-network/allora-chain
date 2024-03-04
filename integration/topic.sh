#!/usr/bin/env bash

set -e

GENESIS=$HOME/.allorad/config/genesis.json
# this script expects to be ran AFTER `scripts/init.sh`
if ! test -f $GENESIS; then
  echo "Must run scripts/init.sh first."
  exit 1
fi

ALLORAD_BIN=$(which allorad)

ALICE_ADDRESS=$($ALLORAD_BIN keys show alice | head -n 1 | cut -f 2 -d ":" | tr -d " ")
BOB_ADDRESS=$($ALLORAD_BIN keys show bob | head -n 1 | cut -f 2 -d ":" | tr -d " ")

echo "Checking that the network is starting from topic 0"
NEXT_TOPIC_ID=$($ALLORAD_BIN query emissions next-topic-id | head -n 1 | cut -f 2 -d ":" | tr -d " " | tr -d "\"")
if [ "$NEXT_TOPIC_ID" != "1" ]; then
  echo "The network is not starting from topic 0. It is starting from topic $NEXT_TOPIC_ID"
  exit 1
fi

echo "Creating topic 1"
PT_CREATOR="$ALICE_ADDRESS"
PT_METADATA="ETH 24h Prediction"
PT_WEIGHT_LOGIC="bafybeih6yjjjf2v7qp3wm6hodvjcdljj7galu7dufirvcekzip5gd7bthq"
PT_WEIGHT_METHOD="eth-price-weights-calc.wasm"
PT_WEIGHT_CADENCE="10800"
PT_INFERENCE_LOGIC="bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65jiky2xqb4rdhfmikswzqm"
PT_INFERENCE_METHOD="allora-inference-function.wasm"
PT_INFERENCE_CADENCE="61"
PT_DEFAULT_ARG="ETH"
$ALLORAD_BIN tx emissions push-topic \
  "$PT_CREATOR" \
  "$PT_METADATA" \
  "$PT_WEIGHT_LOGIC" \
  "$PT_WEIGHT_METHOD" \
  "$PT_WEIGHT_CADENCE" \
  "$PT_INFERENCE_LOGIC" \
  "$PT_INFERENCE_METHOD" \
  "$PT_INFERENCE_CADENCE" \
  "$PT_DEFAULT_ARG" \
  --yes --keyring-backend=test --chain-id=demo;

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

echo
echo "Creating a request for inference on topic 1"

RI_CREATOR="$BOB_ADDRESS"
RI_NONCE="1"
RI_TOPIC_ID="1"
RI_CADENCE="60"
RI_MAX_PRICE_PER_INFERENCE="1"
RI_BID_AMOUNT="10000"
RI_TIMESTAMP_VALID_UNTIL=$(($(date +%s)+60*60*24))
$ALLORAD_BIN tx emissions request-inference \
  $RI_CREATOR \
  "{\"nonce\": \"$RI_NONCE\",\"topic_id\":\"$RI_TOPIC_ID\",\"cadence\":\"$RI_CADENCE\",\"max_price_per_inference\":\"$RI_MAX_PRICE_PER_INFERENCE\",\"bid_amount\":\"$RI_BID_AMOUNT\",\"timestamp_valid_until\":\"$RI_TIMESTAMP_VALID_UNTIL\"}" \
  --yes --keyring-backend=test --chain-id=demo;

echo "Checking the inference request was made correctly"

echo $MEMPOOL
MEMPOOL_INCREMENTED=false
for COUNT_SLEEP in 1 2 3 4 5
do
  MEMPOOL=$($ALLORAD_BIN query emissions all-inference-requests)
  if [ "$MEMPOOL" == "{}" ]; then
    echo "MEMPOOL is empty, transaction may not have mined yet, count sleep $COUNT_SLEEP seconds"
    COUNT_SLEEP=$((COUNT_SLEEP+1))
    sleep 1
  else
    echo "The network has appears to have something in the mempool, inference request probably created successfully"
    MEMPOOL_INCREMENTED=true
    break
  fi
done
if [ "$MEMPOOL_INCREMENTED" = false ]; then
  echo "The network failed to mine the inference request"
  exit 1
fi

echo
echo "reactivating the topic so the request will fire"

$ALLORAD_BIN tx emissions reactivate-topic $RI_CREATOR 1 --yes;
echo "not waiting for reactivate-topic"

echo "Initial state setup complete"
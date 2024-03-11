#!/usr/bin/env bash 

set -e

source $(dirname $0)/common.sh


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
  --yes --keyring-backend=test --chain-id=demo --gas-prices=1uallo --gas=auto --gas-adjustment=1.5;

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

$ALLORAD_BIN tx emissions reactivate-topic $RI_CREATOR 1 --yes --gas-prices=1uallo --gas=auto --gas-adjustment=1.5;
sleep 3

echo "Checking that the allora daemon is still running"
PSAUX=$(ps aux | grep allorad | wc -l)
if [[ "$PSAUX" != "2" ]]; then
  echo "The allora daemon is not running"
  exit 1
fi

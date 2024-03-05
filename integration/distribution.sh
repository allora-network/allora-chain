#!/usr/bin/env bash 

# This script is a quick and dirty integration test to ensure validator rewards are paid out

set -e

GENESIS=$HOME/.allorad/config/genesis.json
# this script expects to be ran AFTER `scripts/init.sh`
if ! test -f $GENESIS; then
  echo "Must run scripts/init.sh first."
  exit 1
fi

ALLORAD_BIN=$(which allorad)

VALIDATOR_ADDRESS=$(cat $GENESIS | grep "validator_address" | cut -f 2 -d ":" | tr -d " " | tr -d "\"" | tr -d ",")
if [[ ${#VALIDATOR_ADDRESS} -ne 50 ]] || [[ $VALIDATOR_ADDRESS != allovaloper* ]]; then
    echo "Validator address not found in genesis file"
    exit 1
fi

# get the current outstanding rewards
DISTRIBUTION_REWARDS_0=$($ALLORAD_BIN query distribution validator-outstanding-rewards $VALIDATOR_ADDRESS | grep "amount" | cut -f 2 -d ":" | tr -d " " | tr -d "\"")

# wait for some blocks to get mined
sleep 5

# get the current outstanding rewards
DISTRIBUTION_REWARDS_1=$($ALLORAD_BIN query distribution validator-outstanding-rewards $VALIDATOR_ADDRESS | grep "amount" | cut -f 2 -d ":" | tr -d " " | tr -d "\"")

# assert that the rewards have increased
INCREASED=$(bc <<< "$DISTRIBUTION_REWARDS_1 > $DISTRIBUTION_REWARDS_0")

if [[ $INCREASED -ne 1 ]]; then
    echo "Distribution of rewards to validators did not increase"
    exit 1
else 
    echo "Distribution of rewards to validators increased"
fi

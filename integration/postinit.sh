#!/usr/bin/env bash

set -e

GENESIS=$HOME/.allorad/config/genesis.json
# this script expects to be ran AFTER `scripts/init.sh`
if ! test -f $GENESIS; then
  echo "Must run scripts/init.sh first."
  exit 1
fi

APP_TOML=$HOME/.allorad/config/app.toml
ALLORAD_BIN=$(which allorad)

ALICE_ADDRESS=$($ALLORAD_BIN keys show alice | head -n 1 | cut -f 2 -d ":" | tr -d " ")
if [[ ${#ALICE_ADDRESS} -ne 43 ]] || [[ $ALICE_ADDRESS != allo* ]]; then
    echo "Alice address not found"
    exit 1
fi

BOB_ADDRESS=$($ALLORAD_BIN keys show bob | head -n 1 | cut -f 2 -d ":" | tr -d " ")
if [[ ${#BOB_ADDRESS} -ne 43 ]] || [[ $BOB_ADDRESS != allo* ]]; then
    echo "Bob address not found"
    exit 1
fi

VALIDATOR_ADDRESS=$(cat $GENESIS | grep "validator_address" | cut -f 2 -d ":" | tr -d " " | tr -d "\"" | tr -d ",")
if [[ ${#VALIDATOR_ADDRESS} -ne 50 ]] || [[ $VALIDATOR_ADDRESS != allovaloper* ]]; then
    echo "Validator address not found in genesis file"
    exit 1
fi

echo "Putting alice and bob in the whitelisted core team list"
GENESIS_TOTAL_LINES=$(wc -l $GENESIS | cut -f 1 -d " ")
CORE_TEAM_LINE_NUM=$(grep -n "core_team_addresses" $GENESIS | cut -f 1 -d ":")
cat $GENESIS | head -n $CORE_TEAM_LINE_NUM > $GENESIS.tmp
echo "        \"$ALICE_ADDRESS\"," >> $GENESIS.tmp
echo "        \"$BOB_ADDRESS\"," >> $GENESIS.tmp
CONTINUE_LINE_NUM=$(($GENESIS_TOTAL_LINES-$CORE_TEAM_LINE_NUM+1))
tail -n $CONTINUE_LINE_NUM $GENESIS >> $GENESIS.tmp
mv $GENESIS.tmp $GENESIS

# Not currently necessary, but keeping this code here for future reference in case ever needed
# echo "Turning on localhost GRPC-Gateway REST HTTP Server"
# APP_TOML_TOTAL_LINES=$(wc -l $APP_TOML | cut -f 1 -d " ")
# APP_TOML_GRPC_GATEWAY_LINE_NUM=$(grep -n "# Enable defines if the API server should be enabled." $APP_TOML | cut -f 1 -d ":")
# cat $APP_TOML | head -n $APP_TOML_GRPC_GATEWAY_LINE_NUM > $APP_TOML.tmp
# echo "enable = true" >> $APP_TOML.tmp
# echo "" >> $APP_TOML.tmp
# echo "# Swagger defines if swagger documentation should automatically be registered." >> $APP_TOML.tmp
# echo "swagger = true" >> $APP_TOML.tmp
# CONTINUE_LINE_NUM=$(($APP_TOML_TOTAL_LINES-$APP_TOML_GRPC_GATEWAY_LINE_NUM-4))
# tail -n $CONTINUE_LINE_NUM $APP_TOML >> $APP_TOML.tmp
# mv $APP_TOML.tmp $APP_TOML

echo "Setting Integration Test Mode to true"
INTEGRATION="TRUE"

echo "Starting allorad daemon and sleep for 3 seconds to let it start"
$ALLORAD_BIN start & disown;
sleep 3
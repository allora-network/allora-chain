#!/bin/bash

BINARY=./allorad
CHAIN_DIR=./data
CHAINID=allora_demo

echo "Starting $CHAINID in $CHAIN_DIR..."
echo "Creating log file at $CHAIN_DIR/$CHAINID.log"
$BINARY start --log_level trace --log_format json --home $CHAIN_DIR/$CHAINID --minimum-gas-prices="1uallo" --pruning=nothing > $CHAIN_DIR/$CHAINID.log 2>&1 &

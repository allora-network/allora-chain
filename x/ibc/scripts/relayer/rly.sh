#!/bin/bash

# Configure predefined mnemonic pharses
BINARY=./rly
CHAINID_1=allora_demo
CHAINID_2=axelar_demo
CHAIN_DIR=./data
RELAYER_DIR=./relayer

# Ensure rly is installed
if ! [ -x "$(command -v $BINARY)" ]; then
    echo "$BINARY is required to run this script..."
    echo "You can download at https://github.com/cosmos/relayer"
    exit 1
fi


echo "Removing previous data..."
rm -rf $CHAIN_DIR/$RELAYER_DIR &> /dev/null

echo "Initializing $BINARY..."
$BINARY config init --home $CHAIN_DIR/$RELAYER_DIR

echo "Adding configurations for both chains..."
$BINARY chains add-dir ./network/relayer/chains --home $CHAIN_DIR/$RELAYER_DIR
$BINARY paths add-dir ./network/relayer/paths --home $CHAIN_DIR/$RELAYER_DIR

echo "Restoring accounts..."
$BINARY keys restore $CHAINID_1 testkey allorarelayer --home $CHAIN_DIR/$RELAYER_DIR
$BINARY keys restore $CHAINID_2 testkey axelarrelayer --home $CHAIN_DIR/$RELAYER_DIR

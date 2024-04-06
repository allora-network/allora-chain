#!/bin/bash

BINARY=./axelard
CHAIN_DIR=./data
CHAINID=axelar_demo
VAL_MNEMONIC="clock post desk civil pottery foster expand merit dash seminar song memory figure uniform spice circle try happy obvious trash crime hybrid hood cushion"
DEMO_MNEMONIC="banner spread envelope side kite person disagree path silver will brother under couch edit food venture squirrel civil budget number acquire point work mass"
RLY_MNEMONIC="record gift you once hip style during joke field prize dust unique length more pencil transfer quit train device arrive energy sort steak upset"
P2PPORT=26656
RPCPORT=26657
RESTPORT=1317
ROSETTA=8081

# Stop if it is already running
if pgrep -x "$BINARY" >/dev/null; then
    echo "Terminating $BINARY..."
    killall $BINARY
fi

echo "Removing previous data..."
rm -rf $CHAIN_DIR/$CHAINID &> /dev/null

# Add directories for both chains, exit if an error occurs
if ! mkdir -p $CHAIN_DIR/$CHAINID 2>/dev/null; then
    echo "Failed to create chain folder. Aborting..."
    exit 1
fi

echo "Initializing $CHAINID..."
$BINARY init test --home $CHAIN_DIR/$CHAINID --chain-id=$CHAINID

echo "Adding genesis accounts..."
$BINARY keys add val --home $CHAIN_DIR/$CHAINID --recover --keyring-backend=test
$BINARY keys add demowallet --home $CHAIN_DIR/$CHAINID --recover --keyring-backend=test
$BINARY keys add axelarrelayer --home $CHAIN_DIR/$CHAINID --recover --keyring-backend=test

$BINARY genesis add-genesis-account $($BINARY --home $CHAIN_DIR/$CHAINID keys show val --keyring-backend test -a) 100000000000uaxl  --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account $($BINARY --home $CHAIN_DIR/$CHAINID keys show demowallet --keyring-backend test -a) 100000000000uaxl  --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account $($BINARY --home $CHAIN_DIR/$CHAINID keys show axelarrelayer --keyring-backend test -a) 100000000000uaxl  --home $CHAIN_DIR/$CHAINID

echo "Creating and collecting gentx..."
$BINARY gentx val2 7000000000uaxl --home $CHAIN_DIR/$CHAINID --chain-id $CHAINID --keyring-backend test
$BINARY collect-gentxs --home $CHAIN_DIR/$CHAINID

echo "Changing defaults and ports in app.toml and config.toml files..."

sed -i -e 's#"tcp://0.0.0.0:26656"#"tcp://0.0.0.0:'"$P2PPORT"'"#g' $CHAIN_DIR/$CHAINID/config/config.toml
sed -i -e 's#"tcp://127.0.0.1:26657"#"tcp://0.0.0.0:'"$RPCPORT"'"#g' $CHAIN_DIR/$CHAINID/config/config.toml
sed -i -e 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $CHAIN_DIR/$CHAINID/config/config.toml
sed -i -e 's/timeout_propose = "3s"/timeout_propose = "1s"/g' $CHAIN_DIR/$CHAINID/config/config.toml
sed -i -e 's/index_all_keys = false/index_all_keys = true/g' $CHAIN_DIR/$CHAINID/config/config.toml
sed -i -e 's/enable = false/enable = true/g' $CHAIN_DIR/$CHAINID/config/app.toml
sed -i -e 's/swagger = false/swagger = true/g' $CHAIN_DIR/$CHAINID/config/app.toml
sed -i -e 's#"tcp://0.0.0.0:1317"#"tcp://0.0.0.0:'"$RESTPORT"'"#g' $CHAIN_DIR/$CHAINID/config/app.toml
sed -i -e 's#":8080"#":'"$ROSETTA"'"#g' $CHAIN_DIR/$CHAINID/config/app.toml

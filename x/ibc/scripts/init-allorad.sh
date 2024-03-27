#!/bin/bash

BINARY=./allorad
CHAIN_DIR=./data
CHAINID=allora_demo
RLY_MNEMONIC="alley afraid soup fall idea toss can goose become valve initial strong forward bright dish figure check leopard decide warfare hub unusual join cart"
P2PPORT=16656
RPCPORT=16657
RESTPORT=1316
ROSETTA=8080

# Stop if it is already running
if pgrep -x "$BINARY" >/dev/null; then
    echo "Terminating $BINARY..."
    killall $BINARY
fi

echo "Removing previous data..."
rm -rf $CHAIN_DIR/$CHAINID &> /dev/null

# Add directories for both chains, exit if an error occurs
if ! mkdir -p $CHAIN_DIR/$CHAINID_1 2>/dev/null; then
    echo "Failed to create chain folder. Aborting..."
    exit 1
fi

echo "configure allorad"
$BINARY config set client chain-id $CHAINID
$BINARY config set client keyring-backend test
$BINARY keys add alice  --home $CHAIN_DIR/$CHAINID
$BINARY keys add bob --home $CHAIN_DIR/$CHAINID

echo "Initializing $CHAINID..."
$BINARY init test --home $CHAIN_DIR/$CHAINID --chain-id $CHAINID --default-denom uallo

echo "Adding genesis accounts..."

$BINARY genesis add-genesis-account alice 10000000allo --keyring-backend test --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account bob 1000allo --keyring-backend test --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1m4ssnux4kh5pfmjzzkpde0hvxfg0d37mla0pdf 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1m8m7u5wygh8f0m55m7aj957yts44fsqdzryjmc 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo18kq56ckavhacjjxwc7lajspfgn6zf78srfx3lk 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1asz8turchyh3f9psyc6yag4shc8ssy0v3y0kjv 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1ey0fvvpx3y99g7s8n8k7ft74dh0zq6y7l3fnke 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo14uyrh7kkg83qmjnme69dna8p07x3wugnxsxdk4 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1q4fa4tqzng2lshfhjaklx90hzfnfennxt02s0v 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1zy5akp9grwfp3x6rqd40x0g4agpzjaskxr9lnn 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1ywhj2svg67mn7ylr9mu5kz9f668z2xejnp9w9y 10000000allo --home $CHAIN_DIR/$CHAINID
$BINARY genesis add-genesis-account allo1r7hqeqdmf6jg9v9px0gh5l6n7tlr0tlxt86plc 10000000allo --home $CHAIN_DIR/$CHAINID


echo $RLY_MNEMONIC_1 | $BINARY keys add rly --home $CHAIN_DIR/$CHAINID --recover --keyring-backend=test
$BINARY genesis add-genesis-account $($BINARY --home $CHAIN_DIR/$CHAINID keys show rly --keyring-backend test -a) 10000000allo  --home $CHAIN_DIR/$CHAINID

echo "Creating and collecting gentx..."
$BINARY genesis gentx alice 50000allo --chain-id $CHAINID --home $CHAIN_DIR/$CHAINID
$BINARY genesis collect-gentxs --home $CHAIN_DIR/$CHAINID


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



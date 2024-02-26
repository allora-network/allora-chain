#!/bin/bash
set -e

# Clean up any existing data
rm -rf $HOME/.allorad/

# Create four allorad directories for validators
mkdir -p $HOME/.allorad/validator1
mkdir -p $HOME/.allorad/validator2
mkdir -p $HOME/.allorad/validator3
mkdir -p $HOME/.allorad/validator4

# Initialize all four validators
allorad init test --chain-id=demo --default-denom uallo --home=$HOME/.allorad/validator1
allorad init test --chain-id=demo --default-denom uallo --home=$HOME/.allorad/validator2
allorad init test --chain-id=demo --default-denom uallo --home=$HOME/.allorad/validator3
allorad init test --chain-id=demo --default-denom uallo --home=$HOME/.allorad/validator4

# Set up keyring-backend and chain-id for all validators
for i in 1 2 3 4; do
    allorad config --home $HOME/.allorad/validator$i set client keyring-backend test
    allorad config --home $HOME/.allorad/validator$i set client chain-id demo
done

# Create keys for all validators
allorad keys add validator1 --keyring-backend test --home $HOME/.allorad/validator1
allorad keys add validator2 --keyring-backend test --home $HOME/.allorad/validator2
allorad keys add validator3 --keyring-backend test --home $HOME/.allorad/validator3
allorad keys add validator4 --keyring-backend test --home $HOME/.allorad/validator4

# Add genesis accounts for each validator
allorad genesis add-genesis-account $(allorad keys show validator1 -a --keyring-backend test --home=$HOME/.allorad/validator1) 10000000allo --home $HOME/.allorad/validator1
allorad genesis add-genesis-account $(allorad keys show validator2 -a --keyring-backend test --home=$HOME/.allorad/validator2) 10000000allo --home $HOME/.allorad/validator2
allorad genesis add-genesis-account $(allorad keys show validator3 -a --keyring-backend test --home=$HOME/.allorad/validator3) 10000000allo --home $HOME/.allorad/validator3
allorad genesis add-genesis-account $(allorad keys show validator4 -a --keyring-backend test --home=$HOME/.allorad/validator4) 10000000allo --home $HOME/.allorad/validator4

# Create a gentx for each validator
allorad genesis gentx validator1 1000allo --chain-id demo --keyring-backend test --home=$HOME/.allorad/validator1
allorad genesis gentx validator2 1000allo --chain-id demo --keyring-backend test --home=$HOME/.allorad/validator2
allorad genesis gentx validator3 1000allo --chain-id demo --keyring-backend test --home=$HOME/.allorad/validator3
allorad genesis gentx validator4 1000allo --chain-id demo --keyring-backend test --home=$HOME/.allorad/validator4

# Collect gentxs to the first validator's genesis file
allorad genesis collect-gentxs --home=$HOME/.allorad/validator1

# Copy the genesis file from the first validator to others
cp $HOME/.allorad/validator1/config/genesis.json $HOME/.allorad/validator2/config/
cp $HOME/.allorad/validator1/config/genesis.json $HOME/.allorad/validator3/config/
cp $HOME/.allorad/validator1/config/genesis.json $HOME/.allorad/validator4/config/


# Update validator1
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $HOME/.allorad/validator1/config/config.toml
sed -i -E 's|prometheus = false|prometheus = true|g' $HOME/.allorad/validator1/config/config.toml

# Update port configurations for validator2
sed -i -E 's|tcp://127.0.0.1:26658|tcp://127.0.0.1:26655|g' $HOME/.allorad/validator2/config/config.toml # P2P
sed -i -E 's|tcp://127.0.0.1:26657|tcp://127.0.0.1:26654|g' $HOME/.allorad/validator2/config/config.toml # RPC
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26653|g' $HOME/.allorad/validator2/config/config.toml # pprof
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $HOME/.allorad/validator2/config/config.toml
sed -i -E 's|prometheus = false|prometheus = true|g' $HOME/.allorad/validator2/config/config.toml

# Update port configurations for validator3
sed -i -E 's|tcp://127.0.0.1:26658|tcp://127.0.0.1:26652|g' $HOME/.allorad/validator3/config/config.toml # P2P
sed -i -E 's|tcp://127.0.0.1:26657|tcp://127.0.0.1:26651|g' $HOME/.allorad/validator3/config/config.toml # RPC
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26650|g' $HOME/.allorad/validator3/config/config.toml # pprof
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $HOME/.allorad/validator3/config/config.toml
sed -i -E 's|prometheus = false|prometheus = true|g' $HOME/.allorad/validator3/config/config.toml

# Update port configurations for validator4
sed -i -E 's|tcp://127.0.0.1:26658|tcp://127.0.0.1:26649|g' $HOME/.allorad/validator4/config/config.toml # P2P
sed -i -E 's|tcp://127.0.0.1:26657|tcp://127.0.0.1:26648|g' $HOME/.allorad/validator4/config/config.toml # RPC
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26647|g' $HOME/.allorad/validator4/config/config.toml # pprof
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $HOME/.allorad/validator4/config/config.toml
sed -i -E 's|prometheus = false|prometheus = true|g' $HOME/.allorad/validator4/config/config.toml

# Get the node ID of validator1
VALIDATOR1_NODE_ID=$(allorad tendermint show-node-id --home $HOME/.allorad/validator1)

# Configure validator2, validator3, and validator4 to have validator1 as a persistent peer
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"${VALIDATOR1_NODE_ID}@localhost:26656\"|g" $HOME/.allorad/validator2/config/config.toml
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"${VALIDATOR1_NODE_ID}@localhost:26656\"|g" $HOME/.allorad/validator3/config/config.toml
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"${VALIDATOR1_NODE_ID}@localhost:26656\"|g" $HOME/.allorad/validator4/config/config.toml

# Start each validator in separate tmux sessions
tmux new -s validator1 -d allorad start --home=$HOME/.allorad/validator1
tmux new -s validator2 -d allorad start --home=$HOME/.allorad/validator2
tmux new -s validator3 -d allorad start --home=$HOME/.allorad/validator3
tmux new -s validator4 -d allorad start --home=$HOME/.allorad/validator4
#!/bin/bash
set -eu

DEFAULT_CHAIN_ID="localallora"
DEFAULT_VALIDATOR_MONIKER="validator"
DEFAULT_DENOM="uallo"
DEFAULT_VALIDATOR_MNEMONIC="bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort"
DEFAULT_FAUCET_MNEMONIC="increase bread alpha rigid glide amused approve oblige print asset idea enact lawn proof unfold jeans rabbit audit return chuckle valve rather cactus great"
DEFAULT_RELAYER_MNEMONIC="black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken"

# Override default values with environment variables
CHAIN_ID=${CHAIN_ID:-$DEFAULT_CHAIN_ID}
DENOM=${DENOM:-$DEFAULT_DENOM}
VALIDATOR_MONIKER=${VALIDATOR_MONIKER:-$DEFAULT_VALIDATOR_MONIKER}
VALIDATOR_MNEMONIC=${VALIDATOR_MNEMONIC:-$DEFAULT_VALIDATOR_MNEMONIC}
FAUCET_MNEMONIC=${FAUCET_MNEMONIC:-$DEFAULT_FAUCET_MNEMONIC}
RELAYER_MNEMONIC=${RELAYER_MNEMONIC:-$DEFAULT_RELAYER_MNEMONIC}

ALLORA_HOME=$HOME/.allorad
CONFIG_FOLDER=$ALLORA_HOME/config

add_genesis_accounts () {
    
    # Validator
    echo "⚖️ Add validator account"
    echo $VALIDATOR_MNEMONIC | allorad keys add $VALIDATOR_MONIKER --recover --keyring-backend=test --home $ALLORA_HOME
    VALIDATOR_ACCOUNT=$(allorad keys show -a $VALIDATOR_MONIKER --keyring-backend test --home $ALLORA_HOME)
    allorad genesis add-genesis-account $VALIDATOR_ACCOUNT 100000000allo --home $ALLORA_HOME
    
    # Faucet
    echo "🚰 Add faucet account"
    echo $FAUCET_MNEMONIC | allorad keys add faucet --recover --keyring-backend=test --home $ALLORA_HOME
    FAUCET_ACCOUNT=$(allorad keys show -a faucet --keyring-backend test --home $ALLORA_HOME)
    allorad genesis add-genesis-account $FAUCET_ACCOUNT 100000000allo --home $ALLORA_HOME

    # Relayer
    echo "🔗 Add relayer account"
    echo $RELAYER_MNEMONIC | allorad keys add relayer --recover --keyring-backend=test --home $ALLORA_HOME
    RELAYER_ACCOUNT=$(allorad keys show -a relayer --keyring-backend test --home $ALLORA_HOME)
    allorad genesis add-genesis-account $RELAYER_ACCOUNT 100000000allo --home $ALLORA_HOME
    
    allorad genesis gentx $VALIDATOR_MONIKER 1000allo --keyring-backend=test --chain-id=$CHAIN_ID --home $ALLORA_HOME
    allorad genesis collect-gentxs --home $ALLORA_HOME
}

edit_config () {
    # Remove seeds
    dasel put -t string -f $CONFIG_FOLDER/config.toml '.p2p.seeds' -v ''

    # Expose the rpc
    dasel put -t string -f $CONFIG_FOLDER/config.toml '.rpc.laddr' -v "tcp://0.0.0.0:26657"
}

if [[ ! -d $CONFIG_FOLDER ]]
then
    echo "🧪 Creating allora home for $VALIDATOR_MONIKER"
    echo $VALIDATOR_MNEMONIC | allorad init -o --chain-id=$CHAIN_ID --home $ALLORA_HOME --recover $VALIDATOR_MONIKER --default-denom $DENOM
    add_genesis_accounts
    edit_config
fi

echo "🏁 Starting $CHAIN_ID..."
allorad start --home $ALLORA_HOME

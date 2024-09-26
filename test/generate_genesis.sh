#!/usr/bin/env bash

# This script generates a genesis file for use with localnet and integration tests
# This is just a sample of how you could make a genesis file for your own allorad network.

echo "Generate Genesis"

set -eu

CHAIN_ID="localnet"
VALIDATOR_NUMBER=${VALIDATOR_NUMBER:-3}

DENOM="uallo"

# Make some tests wallets for use on the localnet
UPSHOT_WALLET_NAME="upshot"
UPSHOT_WALLET_TOKENS=$(echo '99*10^18' | bc) # 99 allo
FAUCET_WALLET_NAME="faucet"
FAUCET_WALLET_TOKENS=$(echo '10^18' | bc) # 1 allo
# These numbers should try to match the Token distribution schedule described in the whitepaper
FOUNDATION_WALLET_NAME="foundation"
FOUNDATION_WALLET_TOKENS=$(echo '10^26' | bc) # 10% of total token supply of 1e27 (1 Billion Allo) = 100M allo
INVESTORS_WALLET_NAME="investors"
INVESTORS_WALLET_TOKENS=$(echo '3.105*10^26' | bc | cut -f 1 -d '.') # 31.05% of total token supply of 1e27 = 310.5M allo
TEAM_WALLET_NAME="team"
TEAM_WALLET_TOKENS=$(echo '1.75*10^26' | bc | cut -f 1 -d '.') # 17.5% of total token supply of 1e27

VALIDATOR_TOKENS=$(echo '(10^26 - 100*10^18)/3' | bc) # 100M allo - 100 allo
COMMON_HOME_DIR="${COMMON_HOME_DIR:-$(pwd)}"

allorad=$(which allorad)
echo "Using allorad binary at $allorad"
keyringBackend=test

valPreffix="validator"
genesisHome="$COMMON_HOME_DIR/genesis"
gentxDir=${genesisHome}/gentxs

echo "Starting genesis generation for chain $CHAIN_ID with $VALIDATOR_NUMBER validators"
mkdir -p $gentxDir

$allorad --home=$genesisHome init mymoniker --chain-id $CHAIN_ID --default-denom ${DENOM}

#Create validators account
for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    valName="${valPreffix}${i}"

    echo "Generate $valName account"
    $allorad --home=$genesisHome keys add $valName \
        --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$valName.account_info 2>&1

    echo "Fund $valName account to genesis"
    $allorad --home=$genesisHome genesis add-genesis-account \
        $valName ${VALIDATOR_TOKENS}${DENOM} \
        --keyring-backend $keyringBackend
done

echo "Generate $UPSHOT_WALLET_NAME account"
$allorad --home=$genesisHome keys add $UPSHOT_WALLET_NAME \
    --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$UPSHOT_WALLET_NAME.account_info 2>&1

echo "Fund $UPSHOT_WALLET_NAME account"
$allorad --home=$genesisHome genesis add-genesis-account \
    $UPSHOT_WALLET_NAME ${UPSHOT_WALLET_TOKENS}${DENOM} \
    --keyring-backend $keyringBackend

echo "Generate $FAUCET_WALLET_NAME account"
$allorad --home=$genesisHome keys add $FAUCET_WALLET_NAME \
    --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$FAUCET_WALLET_NAME.account_info 2>&1

echo "Fund $FAUCET_WALLET_NAME account"
$allorad --home=$genesisHome genesis add-genesis-account \
    $FAUCET_WALLET_NAME ${FAUCET_WALLET_TOKENS}${DENOM} \
    --keyring-backend $keyringBackend

echo "Generate $FOUNDATION_WALLET_NAME account"
$allorad --home=$genesisHome keys add $FOUNDATION_WALLET_NAME \
    --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$FOUNDATION_WALLET_NAME.account_info 2>&1

echo "Fund $FOUNDATION_WALLET_NAME account"
$allorad --home=$genesisHome genesis add-genesis-account \
    $FOUNDATION_WALLET_NAME ${FOUNDATION_WALLET_TOKENS}${DENOM} \
    --keyring-backend $keyringBackend

echo "Generate $INVESTORS_WALLET_NAME account"
$allorad --home=$genesisHome keys add $INVESTORS_WALLET_NAME \
    --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$INVESTORS_WALLET_NAME.account_info 2>&1

echo "Fund $INVESTORS_WALLET_NAME account"
$allorad --home=$genesisHome genesis add-genesis-account \
    $INVESTORS_WALLET_NAME ${INVESTORS_WALLET_TOKENS}${DENOM} \
    --keyring-backend $keyringBackend

echo "Generate $TEAM_WALLET_NAME account"
$allorad --home=$genesisHome keys add $TEAM_WALLET_NAME \
    --keyring-backend $keyringBackend > $COMMON_HOME_DIR/$TEAM_WALLET_NAME.account_info 2>&1

echo "Fund $TEAM_WALLET_NAME account"
$allorad --home=$genesisHome genesis add-genesis-account \
    $TEAM_WALLET_NAME ${TEAM_WALLET_TOKENS}${DENOM} \
    --keyring-backend $keyringBackend

for ((i=0; i<$VALIDATOR_NUMBER; i++)); do
    echo "Initializing Validator $i"

    valName="${valPreffix}${i}"
    valHome="$COMMON_HOME_DIR/$valName"
    mkdir -p $valHome

    $allorad --home=$valHome init $valName --chain-id $CHAIN_ID --default-denom ${DENOM}

    # Symlink genesis to have the accounts
    ln -sfr $genesisHome/config/genesis.json $valHome/config/genesis.json

    # Symlink keyring-test to have keys
    ln -sfr $genesisHome/keyring-test $valHome/keyring-test

    $allorad --home=$valHome genesis gentx $valName ${VALIDATOR_TOKENS}${DENOM} \
        --chain-id $CHAIN_ID --keyring-backend $keyringBackend \
        --moniker="$valName" \
        --from=$valName \
        --output-document $gentxDir/$valName.json
done

$allorad --home=$genesisHome genesis collect-gentxs --gentx-dir $gentxDir

#Set additional genesis params
echo "Get $FAUCET_WALLET_NAME address"
FAUCET_ADDRESS=$($allorad --home=$genesisHome keys show $FAUCET_WALLET_NAME -a --keyring-backend $keyringBackend)
FAUCET_ADDRESS="${FAUCET_ADDRESS%%[[:cntrl:]]}"

# some sample default parameters for integration tests
dasel put 'app_state.emissions.core_team_addresses.append()' -t string -v $FAUCET_ADDRESS -f $genesisHome/config/genesis.json
dasel put 'app_state.gov.params.expedited_voting_period' -t string -v "300s" -f $genesisHome/config/genesis.json

cp -f $genesisHome/config/genesis.json $COMMON_HOME_DIR

echo "$CHAIN_ID genesis generated."

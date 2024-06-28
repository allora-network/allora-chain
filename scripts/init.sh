#!/bin/bash

rm -r ~/.allorad || true
ALLORAD_BIN=$(which allorad)

# configure allorad
$ALLORAD_BIN config set client chain-id demo
$ALLORAD_BIN config set client keyring-backend test
$ALLORAD_BIN keys add alice
$ALLORAD_BIN keys add bob
$ALLORAD_BIN init test --chain-id demo --default-denom uallo
$ALLORAD_BIN genesis add-genesis-account alice 10000000allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account bob 10000000allo --keyring-backend test

# create default validator
$ALLORAD_BIN genesis gentx alice 1000allo --chain-id demo
$ALLORAD_BIN genesis collect-gentxs

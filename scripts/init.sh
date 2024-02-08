#!/bin/bash

rm -r ~/.allorad || true
ALLORAD_BIN=$(which allorad)
# configure allorad
$ALLORAD_BIN config set client chain-id demo
$ALLORAD_BIN config set client keyring-backend test
$ALLORAD_BIN keys add alice
$ALLORAD_BIN keys add bob
$ALLORAD_BIN init test --chain-id demo --default-denom uallo
# update genesis
$ALLORAD_BIN genesis add-genesis-account alice 10000000allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account bob 1000allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account allo1wg289gh4tv3en5g49pex433hwnsld6yzzhppgv 6969allo
$ALLORAD_BIN genesis add-genesis-account allo182rv3s5jvn5v7lxaz3elpwzgyl7zqvqe727ee5 1337allo
$ALLORAD_BIN genesis add-genesis-account allo1acgw83473tp35wnxutpvhwadzlgull0vvf7stj 4200allo
$ALLORAD_BIN genesis add-genesis-account allo1ey0fvvpx3y99g7s8n8k7ft74dh0zq6y7j86ff8 10000000allo

# create default validator
$ALLORAD_BIN genesis gentx alice 1000allo --chain-id demo
$ALLORAD_BIN genesis collect-gentxs

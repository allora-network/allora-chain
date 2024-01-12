#!/bin/bash

rm -r ~/.uptd || true
UPTD_BIN=$(which uptd)
# configure uptd
$UPTD_BIN config set client chain-id demo
$UPTD_BIN config set client keyring-backend test
$UPTD_BIN keys add alice
$UPTD_BIN keys add bob
$UPTD_BIN init test --chain-id demo --default-denom upt
# update genesis
$UPTD_BIN genesis add-genesis-account alice 10000000upt --keyring-backend test
$UPTD_BIN genesis add-genesis-account bob 1000upt --keyring-backend test
$UPTD_BIN genesis add-genesis-account upt1wg289gh4tv3en5g49pex433hwnsld6yzzhppgv 6969upt
$UPTD_BIN genesis add-genesis-account upt182rv3s5jvn5v7lxaz3elpwzgyl7zqvqe727ee5 1337upt
$UPTD_BIN genesis add-genesis-account upt1acgw83473tp35wnxutpvhwadzlgull0vvf7stj 4200upt
# create default validator
$UPTD_BIN genesis gentx alice 1000000upt --chain-id demo
$UPTD_BIN genesis collect-gentxs

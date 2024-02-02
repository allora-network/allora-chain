#!/bin/bash

rm -r ~/.uptd || true
UPTD_BIN=$(which uptd)
# configure uptd
go run ./cmd/uptd config set client chain-id demo
go run ./cmd/uptd config set client keyring-backend test
go run ./cmd/uptd keys add alice
go run ./cmd/uptd keys add bob
go run ./cmd/uptd init test --chain-id demo --default-denom upt
# update genesis
go run ./cmd/uptd genesis add-genesis-account alice 10000000upt --keyring-backend test
go run ./cmd/uptd genesis add-genesis-account bob 1000upt --keyring-backend test
go run ./cmd/uptd genesis add-genesis-account upt1wg289gh4tv3en5g49pex433hwnsld6yzzhppgv 6969upt
go run ./cmd/uptd genesis add-genesis-account upt182rv3s5jvn5v7lxaz3elpwzgyl7zqvqe727ee5 1337upt
go run ./cmd/uptd genesis add-genesis-account upt1acgw83473tp35wnxutpvhwadzlgull0vvf7stj 4200upt
# create default validator
go run ./cmd/uptd genesis gentx alice 1000000upt --chain-id demo
go run ./cmd/uptd genesis collect-gentxs

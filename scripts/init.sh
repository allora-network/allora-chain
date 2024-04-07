#!/bin/bash

rm -r ~/.allorad || true
ALLORAD_BIN=$(which allorad)
# configure allorad
$ALLORAD_BIN config set client chain-id demo
$ALLORAD_BIN config set client keyring-backend test
$ALLORAD_BIN keys add alice
$ALLORAD_BIN keys add bob
$ALLORAD_BIN keys add head
$ALLORAD_BIN init test --chain-id demo --default-denom uallo
# update genesis
# Ecosystem non-human MODULE ACCOUNT prints tokens to itself. Starts with nothing.
# Foundation human multisig controlled account gets 10% of total supply at genesis = 100000000allo
# this is just a random test account to simulate the foundation
$ALLORAD_BIN genesis add-genesis-account allo17g6xu7z2u02f7hz0fghtqaexggrgrprhdq9j2z 100000000allo
# investors and team get nothing for the first year then vesting
# their tokens are minted at genesis,
# they're custodialized off-chain by humans bound by legal contracts
# their token lockup is enforced off-chain
# investors : 30.75% of total supply at genesis = 307500000allo
# this is just a random test account to simulate the off-chain custodian
$ALLORAD_BIN genesis add-genesis-account allo12cs03k0mlks4vea7qnhrktpxlj8tdw6zxsuqn3 307500000allo
# team : 17.5% of total supply at genesis = 175000000allo
# this is just a random test account to simulate the off-chain custodian
$ALLORAD_BIN genesis add-genesis-account allo18jkqd9dl09ejkrsfwdzvx694spqyz2azm67wr9 175000000allo

# "participants" is everybody else, they get 5% of total supply at genesis = 50000000allo
# 12 random test accounts, so each gets 4166666.
$ALLORAD_BIN genesis add-genesis-account alice 4166666allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account bob 4166666allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account head 1000allo --keyring-backend test
$ALLORAD_BIN genesis add-genesis-account allo1m4ssnux4kh5pfmjzzkpde0hvxfg0d37mla0pdf 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1m8m7u5wygh8f0m55m7aj957yts44fsqdzryjmc 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo18kq56ckavhacjjxwc7lajspfgn6zf78srfx3lk 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1asz8turchyh3f9psyc6yag4shc8ssy0v3y0kjv 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1ey0fvvpx3y99g7s8n8k7ft74dh0zq6y7l3fnke 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo14uyrh7kkg83qmjnme69dna8p07x3wugnxsxdk4 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1q4fa4tqzng2lshfhjaklx90hzfnfennxt02s0v 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1zy5akp9grwfp3x6rqd40x0g4agpzjaskxr9lnn 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1ywhj2svg67mn7ylr9mu5kz9f668z2xejnp9w9y 4166666allo
$ALLORAD_BIN genesis add-genesis-account allo1r7hqeqdmf6jg9v9px0gh5l6n7tlr0tlxt86plc 4166666allo

# create default validator
$ALLORAD_BIN genesis gentx alice 1000allo --chain-id demo
$ALLORAD_BIN genesis collect-gentxs

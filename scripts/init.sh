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
$ALLORAD_BIN genesis add-genesis-account allo1m4ssnux4kh5pfmjzzkpde0hvxfg0d37mla0pdf 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1m8m7u5wygh8f0m55m7aj957yts44fsqdzryjmc 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo18kq56ckavhacjjxwc7lajspfgn6zf78srfx3lk 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1asz8turchyh3f9psyc6yag4shc8ssy0v3y0kjv 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1ey0fvvpx3y99g7s8n8k7ft74dh0zq6y7l3fnke 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo14uyrh7kkg83qmjnme69dna8p07x3wugnxsxdk4 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1q4fa4tqzng2lshfhjaklx90hzfnfennxt02s0v 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1zy5akp9grwfp3x6rqd40x0g4agpzjaskxr9lnn 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1ywhj2svg67mn7ylr9mu5kz9f668z2xejnp9w9y 10000000allo
$ALLORAD_BIN genesis add-genesis-account allo1r7hqeqdmf6jg9v9px0gh5l6n7tlr0tlxt86plc 10000000allo
# Ecosystem account gets 36.75% of total supply
$ALLORAD_BIN genesis add-genesis-account allo12uxa8rw9hte3z2nuswjzpmfen289n30uag6agp 367500000allo

# create default validator
$ALLORAD_BIN genesis gentx alice 1000allo --chain-id demo
$ALLORAD_BIN genesis collect-gentxs

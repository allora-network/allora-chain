#!/bin/sh
rm -r ~/.relayer || true

set -a
source .env
set +a

echo "$ALLORA_RELAYER_MNEMONIC"

rly config init
rly chains add-dir ./configs
rly paths add-dir ./paths

rly keys restore allora relayer "$ALLORA_RELAYER_MNEMONIC"
rly keys restore osmosis relayer "$OSMOSIS_RELAYER_MNEMONIC"

echo "Tx Link"
rly tx link demo -d -t 3s

echo "Start Demo"
rly start demo
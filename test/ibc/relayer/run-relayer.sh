#!/bin/sh
rm -r ~/.relayer || true

rly config init
rly chains add-dir ./configs
rly chains add-dir ./configs1
rly chains add-dir ./configs2
rly paths add-dir ./paths

rly keys restore allora relayer "black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken"
rly keys restore osmosis relayer "black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken"

echo "Tx Link"
rly tx link demo -d -t 3s

echo "Start Demo"
rly start demo
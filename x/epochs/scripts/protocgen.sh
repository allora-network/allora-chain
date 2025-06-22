#!/usr/bin/env bash

set -e

echo "Generating pulsar proto code for epochs module"
cd proto
buf generate --template buf.gen.pulsar.yaml allora/epochs/module/v1/module.proto

cd ..

echo "Copying generated files..."
cp -r github.com/allora-network/allora-chain/x/epochs/* ./x/epochs/
echo "Removing old api directory..."
rm -rf x/epochs/api && mkdir -p x/epochs/api
echo "Moving epochs directory..."
mv epochs ./x/epochs/api
rm -rf epochs/
rm -rf github.com allora-network

echo "Done!"

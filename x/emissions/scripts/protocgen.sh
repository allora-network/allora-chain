#!/usr/bin/env bash

set -e

# Backup codec.go files WITH their directory structure - dynamically find all v* folders
echo "Starting backup of codec.go files..."
mkdir -p codec_backup/api/emissions
for vdir in api/emissions/v*; do
    if [ -f "$vdir/codec.go" ]; then
        version=$(basename $vdir)
        mkdir -p "codec_backup/api/emissions/$version"
        cp "$vdir/codec.go" "codec_backup/api/emissions/$version/"
        echo "Backed up: $vdir/codec.go"
    fi
done
echo "Backup complete"

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    # this regex checks if a proto file has its go_package set to github.com/allora-network/allora-chain/x/emissions/api/...
    # gogo proto files SHOULD ONLY be generated if this is false
    # we don't want gogo proto to run for proto files which are natively built for google.golang.org/protobuf
    if grep -q "option go_package" "$file" && grep -H -o -c 'option go_package.*github.com/allora-network/allora-chain/x/emissions/api' "$file" | grep -q ':0$'; then
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

echo "Generating pulsar proto code"
buf generate --template buf.gen.pulsar.yaml

cd ..

echo "Copying generated files..."
cp -r github.com/allora-network/allora-chain/x/emissions/* ./
echo "Removing old api directory..."
rm -rf api && mkdir api
echo "Moving emissions directory..."
mv emissions ./api
rm -rf emissions/
rm -rf github.com allora-network

# Restore codec.go files with their directory structure
echo "Restoring codec.go files..."
cp -r codec_backup/api/* ./api/
echo "Restored codec.go files"
echo "Cleaning up backup directory..."
rm -rf codec_backup
echo "Done!"

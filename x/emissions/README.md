Emissions Module
=============================================

## Dependencies

golang v1.21+
GNU make
docker

## Build
```bash
# get deps
go mod tidy

# rebuild the autogenerated protobuf files
make proto-gen

# build the module, making sure the source compiles
make
```

Then somewhere else you have a minimal-chain running:
```bash
cd ../minimal-chain
go mod tidy
make install
make init
minid start
```

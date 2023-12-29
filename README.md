# UpShot Appchain

This repository contains an example of a tiny, but working Cosmos SDK chain.
It uses the least modules possible and is intended to be used as a starting point for building your own chain, without all the boilerplate that other tools generate. It is a simpler version of Cosmos SDK's [simapp](https://github.com/cosmos/cosmos-sdk/tree/main/simapp).

`Uptd` uses the **latest** version of the [Cosmos-SDK](https://github.com/cosmos/cosmos-sdk).

## How to use

In addition to learn how to build a chain thanks to `uptd`, you can as well directly run `uptd`.

### Installation

Install and run `uptd`:

```sh
git clone git@github.com:cosmosregistry/chain-minimal.git
cd chain-minimal
make install # install the uptd binary
make init # initialize the chain
uptd start # start the chain
```

Note: Depending on your `go` setup you may need to add `$GOPATH/bin` to your `$PATH`.

```
export PATH=${PATH}:`go env GOPATH`/bin
```

## Contributing to Upshot State

After cloning both repos, you need to link this repository to the Cosmos Module repository.

Add this like to the replace section of `go.mod`

```
replace (
	github.com/upshot-tech/protocol-state-machine-module => ../protocol-state-machine-module
)
```

Run `go mod tidy`. (you may need to delete go.sum)


## Useful links

* [Cosmos-SDK Documentation](https://docs.cosmos.network/)

# Allora Network
<p align="center">
<img src='assets/AlloraLogo.jpeg' width='200'>
<a href="https://goreportcard.com/badge/github.com/allora-network/allora-chain">
    <img src="https://goreportcard.com/badge/github.com/allora-network/allora-chain">
</a>
</p>

![Docker!](https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white)
![Go!](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Apache License](https://img.shields.io/badge/Apache%20License-D22128?style=for-the-badge&logo=Apache&logoColor=white)

The [Allora Network](https://www.allora.network/) is a state-of-the-art protocol that uses decentralized AI and machine learning (ML) to build, extract, and deploy predictions among its participants. It offers actors who wish to use AI predictions a formalized way to obtain the output of state-of-the-art ML models on-chain and to pay the operators of AI/ML nodes who create these predictions. That way, Allora bridges the information gap between data owners, data processors, AI/ML predictors, market analysts, and the end-users or consumers who have the means to execute on these insights.

The AI/ML agents within the Allora Network use their data and algorithms to broadcast their predictions across a peer-to-peer network, and they ingest these predictions to assess the predictions from all other agents. The network consensus mechanism combines these predictions and assessments, and distributes rewards to the agents according to the quality of their predictions and assessments. This carefully designed incentive mechanism enables Allora to continually learn and improve, adjusting to the market as it evolves.

## Documentation
For the latest documentation, please go to https://docs.allora.network/

## Allorad Install

Binary can be Installed for Linux or Mac (check releases for Windows)

Specify a version to install if desired. 

```bash
curl -sSL https://raw.githubusercontent.com/allora-network/allora-chain/main/install.sh | bash -s -- v0.0.8
```

Ensure `~/.local/bin` is in your PATH.

`allorad` will be available.

```sh
git clone -b <latest-release-tag> https://github.com/allora-network/allora-chain.git
cd allora-chain && make install
```

Note: Depending on your `go` setup you may need to add `$GOPATH/bin` to your `$PATH`.

```
export PATH=$PATH:$(go env GOPATH)/bin
```

## Run a Local Network
To run a local node for testing purposes, execute the following commands:
```
make init
allorad start
```

When you run a node you have 2 options:
 - Run node and a Head, main advantage is - you can use the head for your workers and reputers
 - Run only a node, in this case you will use Allora's heads.

## Run a node with script
`scripts/l1_node.sh`, you will see the log in the output of the script.

*NOTE:* `scripts/l1_node.sh` will generate keys for the node. For production environments you need to use a proper keys storage, and follow secrets management best practices.

## Run a node

### Run
```
docker compose pull
docker compose up
```

run `docker compose up -d` to run detached.

*NOTE:* Don't forget to pull the images first, to ensure that you're using the latest images.

### See logs
`docker compose logs -f`

## Run a node with statesync enabled

To speed up nodes syncing, you can enable statesync, so the node will download state snapshot and sync only the rest blocks (last \<1000 blocks).  
Here is a [guide](https://blog.cosmos.network/cosmos-sdk-state-sync-guide-99e4cf43be2f)  from Cosmos SDK. 

To use statesync, you need:

1. Peers with state snapshots enabled. Allora [peers](https://github.com/allora-network/networks/blob/main/testnet-1/peers.txt) have enabled snapshots for every 1000 blocks.
2. 2 RPC endpoints, you can use any synced full nodes for this purpose.

**NOTE:** To enable state snapshots, you just need to pass `--state-sync.snapshot-keep-recent=X` and `--state-sync.snapshot-interval=Y` to the `allorad start` command.

### Enable statesync with docker compose

Set in the docker compose file the following environment variables
```
      - STATE_SYNC_RPC1=synced_full_node_rpc_1
      - STATE_SYNC_RPC2=synced_full_node_rpc_2
```

### Enable statesync with l1_node.sh script

Just add to the script's environment the following variables:
```
export STATE_SYNC_RPC1=synced_full_node_rpc_1
export STATE_SYNC_RPC2=synced_full_node_rpc_2
scripts/l1_node.sh
```

## Call the node
After the node is running you can exec RPC calls to it.

For instance, check its status:
`curl -so- http://localhost:26657/status | jq .`

With `curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up` you can check if the node syncing or not.

## Run a validator

You can refer to the Allora documentation for detailed instructions on [running a full node](https://docs.allora.network/devs/validators/run-full-node) and [staking a validator](https://docs.allora.network/devs/validators/stake-a-validator).

1. Run and sync a full Allora node following [the instructions](https://docs.allora.network/devs/validators/run-full-node).

2. Wait until the node is fully synced

Verify that your node has finished synching and it is caught up with the network:

`curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up`
Wait until you see the output: "false"

3. Fund account.

`l1_node.sh` script generates keys, you can find created account information in `data/*.account_info`. Get the address from the file and fund, on testnets you can use faucet `https://faucet.${NETWORK}.allora.network`.

4. Stake validator (detailed instructions [here](https://docs.allora.network/devs/validators/stake-a-validator))

Here's an example with Values which starts with a stake of 10000000uallo.

All the following command needs to be executed inside the validator container.
Run `docker compose exec validator0 bash` to get shell of the validator.

You can change `--moniker=...` with a human readable name you choose for your validator.
and `--from=` - is the account name in the keyring, you can list all available keys with `allorad --home=$APP_HOME keys --keyring-backend=test list`

Create stake info file:
```bash
cat > stake-validator.json << EOF
{
    "pubkey": $(allorad --home=$APP_HOME comet show-validator),
    "amount": "1000000uallo",
    "moniker": "validator0",
    "commission-rate": "0.1",
    "commission-max-rate": "0.2",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
}
EOF
```

Stake the validator
```bash
allorad tx staking create-validator ./stake-validator.json \
    --chain-id=testnet \
    --home="$APP_HOME" \
    --keyring-backend=test \
    --from=validator0
```
The command will output tx hash, you can check its status in the explorer: `https://explorer.testnet.allora.network:8443/allora-testnet/tx/$TX_HASH`


5. Verify validator setup

### Check that the validator node is registered and staked

```bash
VAL_PUBKEY=$(allorad --home=$APP_HOME comet show-validator | jq -r .key)
allorad --home=$APP_HOME q staking validators -o=json | \
    jq '.validators[] | select(.consensus_pubkey.value=="'$VAL_PUBKEY'")'
```

- this command should return you all the information about the validator. Similar to the following:
```
{
  "operator_address": "allovaloper1n8t4ffvwstysveuf3ccx9jqf3c6y7kte48qcxm",
  "consensus_pubkey": {
    "type": "tendermint/PubKeyEd25519",
    "value": "gOl6fwPc19BtkmiOGjjharfe6eyniaxdkfyqiko3/cQ="
  },
  "status": 3,
  "tokens": "1000000",
  "delegator_shares": "1000000000000000000000000",
  "description": {
    "moniker": "val2"
  },
  "unbonding_time": "1970-01-01T00:00:00Z",
  "commission": {
    "commission_rates": {
      "rate": "100000000000000000",
      "max_rate": "200000000000000000",
      "max_change_rate": "10000000000000000"
    },
    "update_time": "2024-02-26T22:50:31.187119394Z"
  },
  "min_self_delegation": "1"
}
```
### Check the voting power of your validator node
*NOTE:* please allow 30-60 seconds for the output to be updated

`allorad --home=$APP_HOME status | jq -r '.validator_info.voting_power'`
- Output should be > 0

## Unstaking/unbounding  a validator

If you need to delete a validator from the chain, you just need to unbound the stake.

```bash

allorad --home="$APP_HOME" \
  tx staking unbond ${VALIDATOR_OPERATOR_ADDRESS} \
  ${STAKE_AMOUNT}uallo --from ${VALIDATOR_ACCOUNT_KEY_NAME} \
   --keyring-backend=test --chain-id ${NETWORK}
```

## Run Integration Tests

To run integration tests, execute the following commands:

```bash
bash test/local_testnet_l1.sh
INTEGRATION=TRUE go test -timeout 10m ./test/integration/ -v
```

## Run Upgrade Tests

To run upgrade tests, execute the following commands:

```bash
bash test/local_testnet_upgrade_l1.sh
UPGRADE=TRUE go test -timeout 10m ./test/integration/ -v
```

## Run Stress Tests

To run stress tests, execute the following commands:

```bash
bash test/local_testnet_l1.sh
STRESS_TEST=true RPC_MODE="RandomBasedOnDeterministicSeed" RPC_URLS="http://localhost:26657,http://localhost:26658,http://localhost:26659" SEED=1 MAX_REPUTERS_PER_TOPIC=2 REPUTERS_PER_ITERATION=2 EPOCH_LENGTH=12 FINAL_REPORT=TRUE MAX_WORKERS_PER_TOPIC=2 WORKERS_PER_ITERATION=1 TOPICS_MAX=2 TOPICS_PER_ITERATION=1 MAX_ITERATIONS=2 go test -v -timeout 0 -test.run TestStressTestSuite ./test/stress
```

options for RPC Modes include "RandomBasedOnDeterministicSeed" "RoundRobin" and "SingleRpc"

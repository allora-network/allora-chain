# Allora Network
<p align="center">
<img src='assets/AlloraLogo.png' width='200'>
</p>

![Docker!](https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white)
![Go!](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Apache License](https://img.shields.io/badge/Apache%20License-D22128?style=for-the-badge&logo=Apache&logoColor=white)

The Allora Network is a state-of-the-art protocol that uses decentralized AI and machine learning (ML) to build, extract, and deploy predictions among its participants. It offers actors who wish to use AI predictions a formalized way to obtain the output of state-of-the-art ML models on-chain and to pay the operators of AI/ML nodes who create these predictions. That way, Allora bridges the information gap between data owners, data processors, AI/ML predictors, market analysts, and the end-users or consumers who have the means to execute on these insights.

The AI/ML agents within the Allora Network use their data and algorithms to broadcast their predictions across a peer-to-peer network, and they ingest these predictions to assess the predictions from all other agents. The network consensus mechanism combines these predictions and assessments, and distributes rewards to the agents according to the quality of their predictions and assessments. This carefully designed incentive mechanism enables Allora to continually learn and improve, adjusting to the market as it evolves.

## Documentation
For the latest documentation, please go to https://docs.allora.network/

## Allorad Install

Binary can be Installed for Linux or Mac (check releases for Windows)

```bash
curl -sSL https://raw.githubusercontent.com/allora-network/allora-chain/main/install.sh | bash
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

## Run a node
`scripts/l1_node.sh`, you will see the log in the output of the script.

*NOTE:* `scripts/l1_node.sh` will generate keys for the node. For production environments you need to use a proper keys storage, and follow secrets management best practices.

## Run a node with docker compose

### Run
```
docker compose pull
docker compose up
```

run `docker compose up -d` to run detached.

*NOTE:* Don't forget to pull the images first, to ensure that you're using the latest images.

### See logs
`docker compose logs -f`

## Call the node
After the node is running you can exec RPC calls to it.

For instance, check its status:
`curl -so- http://localhost:26657/status | jq .`

With `curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up` you can check if the node syncing or not.

## Run a validator

1. run and sync a full allorad node, follow [the instructions]()

2. Wait until the node is fully synced

Verify that your node has finished synching and it is caught up with the network

`curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up`
Wait until you see the output: "false"

3. Fund account.

`l1_node.sh` script generates keys, you can find created account information in `data/*.account_info`. Get the address from the file and fund, on testnets you can use faucet `https://faucet.${NETWORK}.allora.network`.

4. Stake validator

Here's an example with Values which starts with a stake of 10000000uallo.

All the following command needs to be executed inside the validator container.
Run `docker compose exec validator0 bash` to get shell of the validator.

You can change `--moniker=...` with a human readable name you choose for your validator.
and `--from=` - is the account name in the keyring, you can list all availble keys with `allorad --home=$APP_HOME keys --keyring-backend=test list`

Create stake info file:
```bash
cat > stake-validator.json << EOF
{
    "pubkey": $(allorad --home=$APP_HOME comet show-validator),
    "amount": "1000000uallo",
    "moniker": "myvalidator",
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
    --chain-id=edgenet \
    --home="$APP_HOME" \
    --keyring-backend=test \
    --from=validator0
```
The command will output tx hash, you can check its status in the explorer: `https://explorer.edgenet.allora.network:8443/allora-edgenet/tx/$TX_HASH`


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
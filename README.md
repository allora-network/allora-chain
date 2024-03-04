# Allora Network

![Banner!](assets/AlloraLogo.png)

The Allora Network is a state-of-the-art protocol that uses decentralized AI and machine learning (ML) to build, extract, and deploy predictions among its participants. It offers actors who wish to use AI predictions a formalized way to obtain the output of state-of-the-art ML models on-chain and to pay the operators of AI/ML nodes who create these predictions. That way, Allora bridges the information gap between data owners, data processors, AI/ML predictors, market analysts, and the end-users or consumers who have the means to execute on these insights.

The AI/ML agents within the Allora Network use their data and algorithms to broadcast their predictions across a peer-to-peer network, and they ingest these predictions to assess the predictions from all other agents. The network consensus mechanism combines these predictions and assessments, and distributes rewards to the agents according to the quality of their predictions and assessments. This carefully designed incentive mechanism enables Allora to continually learn and improve, adjusting to the market as it evolves.

## Documentation
For the latest documentation, please go to https://docs.allora.network/

## Allorad Install

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



*NOTE:* The following commands will generate keys for the node. For production environments you need to use a proper keys storage, and follow secrets management best practices.

## Run a node
`scripts/l1_node.sh`, you will see the log in the output of the script.

## Run a node with docker compose

### Build docker image
`docker compose build`

### Run
`docker compose up`, add `-d` to run detached.

### See logs
`docker compose logs -f`

## Call the node
After the node is running you can exec RPC calls to it.

For instance, check its status:
`curl -so- http://localhost:26657/status | jq .`

With `curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up` you can check if the node syncing or not.

## Run a validator

1. run and sync a full allorad node, follow [the instructions]()

2. Prepare an account & Fund it.
    `l1_node.sh` script generates keys you can see its address 

If you don't have an account (wallet) on Lava yet, Refer to creating new accounts and the faucet.
3. Stake & start validating

Once your account is funded, run this to stake and start validating.

    Verify that your node has finished synching and it is caught up with the network

$current_lavad_binary status | jq .SyncInfo.catching_up
# Wait until you see the output: "false"

    Verify that your account has funds in it in order to perform staking

# Make sure you can see your account name in the keys list
$current_lavad_binary keys list

# Make sure you see your account has Lava tokens in it
YOUR_ADDRESS=$($current_lavad_binary keys show -a $ACCOUNT_NAME)
$current_lavad_binary query \
    bank balances \
    $YOUR_ADDRESS \
    --denom ulava

    Back up your validator's consensus key

    A validator participates in the consensus by sending a message signed by a consensus key which is automatically generated when you first run a node. You must create a backup of this consensus key in case that you migrate your validator to another server or accidentally lose access to your validator.

    A consensus key is stored as a json file in $lavad_home_folder/config/priv_validator_key.json by default, or a custom path specified in the parameter priv_validator_key_file of config.toml.

    Stake validator

Here's an example with Values which starts with a stake of 50000000ulava. Replace <<moniker_node>> With a human readable name you choose for your validator.

$current_lavad_binary tx staking create-validator \
    --amount="50000000ulava" \
    --pubkey=$($current_lavad_binary tendermint show-validator --home "$HOME/.lava/") \
    --moniker="<<moniker_node>>" \
    --chain-id=lava-testnet-2 \
    --commission-rate="0.10" \
    --commission-max-rate="0.20" \
    --commission-max-change-rate="0.01" \
    --min-self-delegation="10000" \
    --gas="auto" \
    --gas-adjustment "1.5" \
    --gas-prices="0.05ulava" \
    --home="$HOME/.lava/" \
    --from=$ACCOUNT_NAME

Once you have finished running the command above, if you see code: 0 in the output, the command was successful

    Verify validator setup

block_time=60
# Check that the validator node is registered and staked
validator_pubkey=$($current_lavad_binary tendermint show-validator | jq .key | tr -d '"')

$current_lavad_binary q staking validators | grep $validator_pubkey

# Check the voting power of your validator node - please allow 30-60 seconds for the output to be updated
sleep $block_time
$current_lavad_binary status | jq .ValidatorInfo.VotingPower | tr -d '"'
# Output should be > 0



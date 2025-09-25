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

## System Requirements

Before installing and running an Allora Chain node, ensure your system meets the following requirements:

### Hardware Requirements

#### Minimum Requirements (Development/Testing)
- **CPU**: 2 cores, 2.0 GHz (x86_64 architecture)
- **RAM**: 4 GB
- **Storage**: 100 GB available disk space (SSD recommended)
- **Network**: Stable internet connection with at least 10 Mbps bandwidth

#### Recommended Requirements (Production/Validator)
- **CPU**: 4+ cores, 2.4 GHz (x86_64 architecture)
- **RAM**: 8 GB (16 GB+ for high-traffic validators)
- **Storage**: 500 GB+ SSD with high IOPS (NVMe preferred)
- **Network**: Dedicated internet connection with 100 Mbps+ bandwidth and low latency

#### Enterprise/High-Performance Validator
- **CPU**: 8+ cores, 3.0 GHz+ (Intel Xeon or AMD EPYC recommended)
- **RAM**: 32 GB+
- **Storage**: 1 TB+ NVMe SSD with enterprise-grade endurance
- **Network**: Multiple redundant internet connections, 1 Gbps+
- **Additional**: UPS, redundant power supplies, 24/7 monitoring

### Storage Considerations

**Disk Space Growth:**
- Initial sync: ~50-100 GB (depending on network age)
- Daily growth: ~1-5 GB (varies with network activity)
- State snapshots: Additional 10-20 GB if enabled
- Pruning: Can reduce storage by 70-80% (see [Performance Issues](#performance-issues) in Troubleshooting)

**Storage Performance:**
- **SSD Required**: HDDs are not recommended due to sync speed and I/O requirements
- **IOPS**: Minimum 1000 IOPS, 3000+ IOPS recommended for validators
- **Throughput**: Minimum 100 MB/s read/write speeds

### Network Requirements

**Ports:**
- `26656`: P2P communication (must be publicly accessible)
- `26657`: RPC endpoint (can be restricted to localhost for security)
- Additional ports may be required for monitoring and metrics

**Bandwidth:**
- **Minimum**: 10 Mbps down, 5 Mbps up
- **Recommended**: 100 Mbps down, 50 Mbps up
- **Validator**: 500+ Mbps with low latency (<100ms to major data centers)

**Connectivity:**
- Low latency connection to other validators (sub-second block times)
- Stable connection (minimal drops/interruptions)
- IPv4 support required, IPv6 optional

### Operating System Support

#### Supported Platforms
- **Linux**: Ubuntu 20.04+ LTS, Debian 11+, CentOS 8+, RHEL 8+ (Recommended)
- **macOS**: 10.15+ (development only)
- **Windows**: Windows 10+ with WSL2 (development only)

#### Container Platforms
- **Docker**: 20.10+ with Docker Compose v2
- **Kubernetes**: 1.20+ (for enterprise deployments)
- **Podman**: 3.0+ (alternative to Docker)

### Software Dependencies

#### Required
- **Go**: 1.22+ (for building from source)
- **Git**: 2.20+
- **Make**: GNU Make 4.0+
- **curl**: For installation scripts and health checks
- **jq**: JSON processing tool

#### Container Runtime (Alternative to local build)
- **Docker Engine**: 20.10+
- **Docker Compose**: v2.0+

#### Optional but Recommended
- **htop/top**: System monitoring
- **iotop**: Disk I/O monitoring
- **netstat/ss**: Network connection monitoring
- **systemd**: For service management (Linux)
- **logrotate**: Log file management
- **fail2ban**: Security (if exposing RPC publicly)

### Security Considerations

**Firewall Configuration:**
```bash
# Allow P2P port
sudo ufw allow 26656/tcp

# Restrict RPC access (localhost only recommended)
sudo ufw allow from 127.0.0.1 to any port 26657

# SSH access (change default port)
sudo ufw allow 2222/tcp
```

**Key Management:**
- Hardware Security Module (HSM) for validator keys (production)
- Encrypted storage for key files
- Regular key backup procedures
- Multi-signature setups for high-value validators

### Performance Tuning

**System Optimizations:**
```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Optimize TCP settings for blockchain workloads
echo "net.core.rmem_max = 134217728" >> /etc/sysctl.conf
echo "net.core.wmem_max = 134217728" >> /etc/sysctl.conf
echo "net.ipv4.tcp_rmem = 4096 87380 134217728" >> /etc/sysctl.conf
echo "net.ipv4.tcp_wmem = 4096 65536 134217728" >> /etc/sysctl.conf
```

**Monitoring Requirements:**
- CPU usage should remain below 80% during normal operation
- Memory usage should not exceed 80% of available RAM
- Disk I/O wait time should be less than 10%
- Network latency to peers should be under 200ms

### Cloud Provider Recommendations

**AWS:**
- **Instance Type**: m5.xlarge or c5.xlarge (minimum), m5.2xlarge+ (production)
- **Storage**: gp3 SSD with provisioned IOPS
- **Network**: Enhanced networking enabled

**Google Cloud:**
- **Instance Type**: n2-standard-4 (minimum), n2-standard-8+ (production)
- **Storage**: SSD persistent disks with high IOPS
- **Network**: Premium tier networking

**Azure:**
- **Instance Type**: Standard_D4s_v3 (minimum), Standard_D8s_v3+ (production)
- **Storage**: Premium SSD with high IOPS
- **Network**: Accelerated networking enabled

**Note**: Always use dedicated instances/VMs for production validators to avoid noisy neighbor effects.

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

## Run a Fork of Testnet/Mainnet State
To run a fork of a testnet or mainnet in order to check changes against the database state for those networks, first set up some local `$HOME/.allorad/` config genesis, etc:

```bash
allorad init devnet
allorad keys add test
```

Then copy an existing node snapshot of the `$HOME/.allorad/data/` folder to your new validator's same allorad home folder. You might use `allorad snapshots dump` to get a tar.gz snapshot.

Next get the local validator key for your new validator:
```bash
allorad comet show-address
```

Start the node with an in-place-testnet, swapping the state of the node for the snapshot state. Put the comet address from the previous step:

```bash
allorad in-place-testnet devnet allovaloper<comet address> --home $HOME/.allorad --minimum-gas-prices 0uallo --skip-confirmation
```

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

## Troubleshooting

This section covers common issues and their solutions when running Allora Chain nodes.

### System Requirements

**Minimum Requirements:**
- **CPU**: 2+ cores (4+ cores recommended for validators)
- **RAM**: 4GB (8GB+ recommended for validators)
- **Storage**: 100GB SSD (500GB+ recommended for full nodes)
- **Network**: Broadband internet connection with stable connectivity
- **OS**: Linux (Ubuntu 20.04+), macOS, or Windows with WSL2

**Software Dependencies:**
- Go 1.22+
- Docker & Docker Compose
- Git
- Make

### Docker Issues

**Problem**: `docker-compose up -d error`
**Solution**:
```bash
# Check if ports 26656 and 26657 are available
sudo netstat -tulpn | grep :26656
sudo netstat -tulpn | grep :26657

# Stop any conflicting services
sudo fuser -k 26656/tcp
sudo fuser -k 26657/tcp

# Ensure Docker daemon is running
sudo systemctl start docker
sudo systemctl enable docker
```

**Problem**: `Error exec /scripts/l1_node.sh: no such file or directory`
**Solution**:
```bash
# Check file permissions and existence
ls -la scripts/l1_node.sh
chmod +x scripts/l1_node.sh

# Verify Docker volume mounts in docker-compose.yml
docker compose config
```

**Problem**: Container startup failures or insufficient resources
**Solution**:
```bash
# Check system resources
free -h
df -h

# Increase Docker memory limits if needed
# Edit ~/.docker/daemon.json:
{
  "default-runtime": "runc",
  "runtimes": {
    "runc": {
      "path": "runc"
    }
  },
  "storage-driver": "overlay2",
  "storage-opts": ["overlay2.override_kernel_check=true"]
}
```

### Node Synchronization Issues

**Problem**: Slow block synchronization
**Solution**:
```bash
# Use state sync for faster initial sync
export STATE_SYNC_RPC1=rpc-endpoint-1
export STATE_SYNC_RPC2=rpc-endpoint-2

# Check current sync status
curl -so- http://localhost:26657/status | jq .result.sync_info.catching_up
```

**Problem**: State sync failures
**Solution**:
```bash
# Verify RPC endpoints are accessible
curl -so- http://rpc-endpoint/status

# Check network connectivity
ping 8.8.8.8

# Verify trusted state sync servers from peers.txt
wget https://github.com/allora-network/networks/raw/main/testnet-1/peers.txt
```

**Problem**: `Connection refused` errors
**Solution**:
```bash
# Check firewall settings
sudo ufw status
sudo ufw allow 26656
sudo ufw allow 26657

# Verify port binding
docker ps
netstat -tulpn | grep 26657
```

### Configuration Issues

**Problem**: `Panic: unknown field 'max_retries_to_fulfil_nonces_reputer' in types.Params`
**Solution**:
```bash
# Update to latest version
git pull origin main
make install

# Clean and rebuild
make clean
make build

# Verify configuration file syntax
allorad validate-genesis ~/.allorad/config/genesis.json
```

**Problem**: Genesis file validation errors
**Solution**:
```bash
# Download correct genesis file for your network
wget https://github.com/allora-network/networks/raw/main/testnet-1/genesis.json -O ~/.allorad/config/genesis.json

# Verify genesis file
allorad validate-genesis ~/.allorad/config/genesis.json
```

**Problem**: Key management errors
**Solution**:
```bash
# List available keys
allorad keys list --keyring-backend=test

# Recover key if needed
allorad keys add mykey --recover --keyring-backend=test

# Check key permissions
ls -la ~/.allorad/
```

### Validator-Specific Issues

**Problem**: Validator not appearing in validator set
**Solution**:
```bash
# Check validator status
VAL_PUBKEY=$(allorad comet show-validator | jq -r .key)
allorad q staking validators -o=json | jq '.validators[] | select(.consensus_pubkey.value=="'$VAL_PUBKEY'")'

# Verify minimum stake requirements
allorad q staking params

# Check voting power
allorad status | jq -r '.validator_info.voting_power'
```

**Problem**: Double signing or slashing
**Solution**:
```bash
# Check validator signing info
allorad q slashing signing-info $(allorad comet show-address)

# Unjail validator if needed (after fixing the issue)
allorad tx slashing unjail --from=mykey --chain-id=testnet
```

### Network Connectivity Issues

**Problem**: Cannot connect to peers
**Solution**:
```bash
# Add persistent peers to config
allorad config set config persistent_peers "peer1@ip1:26656,peer2@ip2:26656"

# Check peer connectivity
curl -so- http://localhost:26657/net_info | jq .result.peers
```

**Problem**: RPC endpoint not accessible
**Solution**:
```bash
# Check RPC configuration in config.toml
grep -A 5 "\[rpc\]" ~/.allorad/config/config.toml

# Enable RPC if disabled
allorad config set config rpc.laddr "tcp://0.0.0.0:26657"
```

### Performance Issues

**Problem**: High memory usage
**Solution**:
```bash
# Adjust cache settings in app.toml
grep -A 10 "cache-size" ~/.allorad/config/app.toml

# Enable pruning to reduce storage
allorad config set app pruning "default"
allorad config set app pruning-keep-recent 100
allorad config set app pruning-interval 10
```

**Problem**: Disk space issues
**Solution**:
```bash
# Check disk usage
du -h ~/.allorad/

# Clean up old snapshots if using state-sync
find ~/.allorad/data/snapshots -name "*.snap" -mtime +7 -delete

# Consider using pruning or moving to larger storage
```

### Getting Help

If you encounter issues not covered here:

1. **Check the logs**: `docker compose logs -f` or `journalctl -u allorad -f`
2. **Search existing issues**: https://github.com/allora-network/allora-chain/issues
3. **Join the community**: Visit [Allora Network Documentation](https://docs.allora.network/) for additional support
4. **Create a new issue**: Include logs, system info, and steps to reproduce

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

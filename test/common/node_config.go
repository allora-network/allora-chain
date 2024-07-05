package testcommon

import (
	"testing"

	upgrade "cosmossdk.io/x/upgrade"
	"github.com/allora-network/allora-chain/app/params"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution"
	gov "github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

// handle to various node data
type TestConfig struct {
	T              *testing.T
	Client         Client                // a testcommon.Client which holds several cosmosclient.Client instances
	AlloraHomeDir  string                // home directory for the allora keystore
	FaucetAcc      cosmosaccount.Account // account info for the faucet
	FaucetAddr     string                // faucets address, string encoded bech32
	UpshotAcc      cosmosaccount.Account // account info for the upshot account
	UpshotAddr     string                // upshot address, string encoded bech32
	AliceAcc       cosmosaccount.Account // account info for the alice test account
	AliceAddr      string                // alice address, string encoded bech32
	BobAcc         cosmosaccount.Account // account info for the bob test account
	BobAddr        string                // bob address, string encoded bech32
	Validator0Acc  cosmosaccount.Account // account info for the validator0 test account
	Validator0Addr string                // validator0 address, string encoded bech32
	Validator1Acc  cosmosaccount.Account // account info for the validator1 test account
	Validator1Addr string                // validator1 address, string encoded bech32
	Validator2Acc  cosmosaccount.Account // account info for the validator2 test account
	Validator2Addr string                // validator2 address, string encoded bech32
	Cdc            codec.Codec           // common codec for encoding/decoding
	Seed           int                   // global non-mutable seed used for naming the test run
}

// create a new config that we can use
func NewTestConfig(
	t *testing.T,
	rpcConnectionType RpcConnectionType,
	nodeRpcAddresses []string,
	alloraHomeDir string,
	seed int,
) TestConfig {
	nodeConfig := TestConfig{}
	var err error
	nodeConfig.T = t
	nodeConfig.AlloraHomeDir = alloraHomeDir
	nodeConfig.Seed = seed
	if rpcConnectionType == SingleRpc {
		require.Len(t, nodeRpcAddresses, 1, "must have exactly one rpc address")
	} else { // RoundRobin or RandomBasedOnDeterministicSeed
		require.GreaterOrEqual(t, len(nodeRpcAddresses), 1, "must have at least one rpc address")
	}
	client := NewClient(
		t,
		rpcConnectionType,
		nodeRpcAddresses,
		alloraHomeDir,
		int64(seed),
	)
	nodeConfig.Client = client
	//// restore from mnemonic
	nodeConfig.FaucetAcc, err = client.Clients[0].AccountRegistry.GetByName("faucet")
	require.NoError(t, err)
	nodeConfig.UpshotAcc, err = client.Clients[0].AccountRegistry.GetByName("upshot")
	require.NoError(t, err)
	nodeConfig.AliceAcc, err = client.Clients[0].AccountRegistry.GetByName("faucet")
	require.NoError(t, err)
	nodeConfig.BobAcc, err = client.Clients[0].AccountRegistry.GetByName("upshot")
	require.NoError(t, err)
	nodeConfig.Validator0Acc, err = client.Clients[0].AccountRegistry.GetByName("validator0")
	require.NoError(t, err)
	nodeConfig.Validator1Acc, err = client.Clients[0].AccountRegistry.GetByName("validator1")
	require.NoError(t, err)
	nodeConfig.Validator2Acc, err = client.Clients[0].AccountRegistry.GetByName("validator2")
	require.NoError(t, err)
	nodeConfig.FaucetAddr, err = nodeConfig.FaucetAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.UpshotAddr, err = nodeConfig.UpshotAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.AliceAddr, err = nodeConfig.AliceAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.BobAddr, err = nodeConfig.BobAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.Validator0Addr, err = nodeConfig.Validator0Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.Validator1Addr, err = nodeConfig.Validator1Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	nodeConfig.Validator2Addr, err = nodeConfig.Validator2Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)

	encCfg := moduletestutil.MakeTestEncodingConfig(
		mint.AppModuleBasic{},
		emissions.AppModule{},
		auth.AppModule{},
		bank.AppModule{},
		distribution.AppModule{},
		gov.AppModule{},
		upgrade.AppModule{},
	)
	nodeConfig.Cdc = codec.NewProtoCodec(encCfg.InterfaceRegistry)

	return nodeConfig
}

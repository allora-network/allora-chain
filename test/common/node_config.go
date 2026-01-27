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
	t.Helper()
	var err error
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
	//// restore from mnemonic
	faucetAcc, err := client.Clients[0].AccountRegistry.GetByName("faucet")
	require.NoError(t, err)
	upshotAcc, err := client.Clients[0].AccountRegistry.GetByName("upshot")
	require.NoError(t, err)
	aliceAcc, err := client.Clients[0].AccountRegistry.GetByName("faucet")
	require.NoError(t, err)
	bobAcc, err := client.Clients[0].AccountRegistry.GetByName("upshot")
	require.NoError(t, err)
	validator0Acc, err := client.Clients[0].AccountRegistry.GetByName("validator0")
	require.NoError(t, err)
	validator1Acc, err := client.Clients[0].AccountRegistry.GetByName("validator1")
	require.NoError(t, err)
	validator2Acc, err := client.Clients[0].AccountRegistry.GetByName("validator2")
	require.NoError(t, err)
	faucetAddr, err := faucetAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	upshotAddr, err := upshotAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	aliceAddr, err := aliceAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	bobAddr, err := bobAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	validator0Addr, err := validator0Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	validator1Addr, err := validator1Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	validator2Addr, err := validator2Acc.Address(params.HumanCoinUnit)
	require.NoError(t, err)

	encCfg := moduletestutil.MakeTestEncodingConfig(
		mint.AppModuleBasic{},    //nolint:exhaustruct
		emissions.AppModule{},    //nolint:exhaustruct
		auth.AppModule{},         //nolint:exhaustruct
		bank.AppModule{},         //nolint:exhaustruct
		distribution.AppModule{}, //nolint:exhaustruct
		gov.AppModule{},          //nolint:exhaustruct
		upgrade.AppModule{},      //nolint:exhaustruct
	)

	return TestConfig{
		T:              t,
		AlloraHomeDir:  alloraHomeDir,
		Seed:           seed,
		Client:         client,
		Cdc:            codec.NewProtoCodec(encCfg.InterfaceRegistry),
		FaucetAcc:      faucetAcc,
		FaucetAddr:     faucetAddr,
		UpshotAcc:      upshotAcc,
		UpshotAddr:     upshotAddr,
		AliceAcc:       aliceAcc,
		AliceAddr:      aliceAddr,
		BobAcc:         bobAcc,
		BobAddr:        bobAddr,
		Validator0Acc:  validator0Acc,
		Validator0Addr: validator0Addr,
		Validator1Acc:  validator1Acc,
		Validator1Addr: validator1Addr,
		Validator2Acc:  validator2Acc,
		Validator2Addr: validator2Addr,
	}
}

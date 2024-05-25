package stress_test

import (
	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testCommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// register the topic funder addresses in the account registry
func createTopicFunderAddresses(
	m testCommon.TestConfig,
	topicsMax int,
) (topicFunders NameToAccountMap) {
	topicFunders = make(map[string]AccountAndAddress)

	for topicFunderIndex := 0; topicFunderIndex < topicsMax; topicFunderIndex++ {
		topicFunderAccountName := getTopicFunderAccountName(topicFunderIndex)
		topicFunderAccount, _, err := m.Client.AccountRegistryCreate(topicFunderAccountName)
		if err != nil {
			m.T.Log("Error creating funder address: ", topicFunderAccountName, " - ", err)
			continue
		}
		topicFunderAddress, err := topicFunderAccount.Address(params.HumanCoinUnit)
		if err != nil {
			m.T.Log("Error creating funder address: ", topicFunderAccountName, " - ", err)
			continue
		}
		topicFunders[topicFunderAccountName] = AccountAndAddress{
			acc:  topicFunderAccount,
			addr: topicFunderAddress,
		}
		m.T.Log("Created funder address: ", topicFunderAccountName, " - ", topicFunderAddress)
	}
	return topicFunders
}

// fund every target address from the sender in amount coins
func fundAccounts(
	m testCommon.TestConfig,
	sender AccountAndAddress,
	targets NameToAccountMap,
	amount int64,
) error {
	inputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(
			params.BaseCoinUnit,
			cosmossdk_io_math.NewInt(amount*int64(len(targets))),
		),
	)
	outputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(params.BaseCoinUnit, cosmossdk_io_math.NewInt(amount)),
	)

	outputs := []banktypes.Output{}
	for _, accountAndAddress := range targets {
		outputs = append(outputs, banktypes.Output{
			Address: accountAndAddress.addr,
			Coins:   outputCoins,
		})
	}

	// Fund the accounts from faucet account in a single transaction
	sendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: sender.addr,
				Coins:   inputCoins,
			},
		},
		Outputs: outputs,
	}
	_, err := m.Client.BroadcastTx(m.Ctx, sender.acc, sendMsg)
	if err != nil {
		m.T.Log("Error worker address: ", err)
		return err
	}
	return nil
}

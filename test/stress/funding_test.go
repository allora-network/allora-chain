package stress_test

import (
	"context"

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
		topicFunderAccountName := getTopicFunderAccountName(m.Seed, topicFunderIndex)
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
		//m.T.Log("Created funder address: ", topicFunderAccountName, " - ", topicFunderAddress)
	}
	return topicFunders
}

// fund every target address from the sender in amount coins
func fundAccounts(
	m testCommon.TestConfig,
	topicId uint64,
	sender NameAccountAndAddress,
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

	outputs := make([]banktypes.Output, len(targets))
	names := make([]string, len(targets))
	i := 0
	for name, accountAndAddress := range targets {
		names[i] = name
		outputs[i] = banktypes.Output{
			Address: accountAndAddress.addr,
			Coins:   outputCoins,
		}
		i++
	}

	// Fund the accounts from faucet account in a single transaction
	sendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: sender.aa.addr,
				Coins:   inputCoins,
			},
		},
		Outputs: outputs,
	}
	ctx := context.Background()
	_, err := m.Client.BroadcastTx(ctx, sender.aa.acc, sendMsg)
	if err != nil {
		m.T.Log("Error worker address: ", err)
		return err
	}
	// pass zero for topic id if we're funding the funders themselves
	if topicId != 0 {
		m.T.Log(topicLog(uint64(topicId),
			"Funded ", len(targets), " accounts from ", sender.name,
			" with ", amount, " coins:", " ", names,
		))
	} else {
		m.T.Log("Funded ", len(targets), " accounts from ", sender.name,
			" with ", amount, " coins:", " ", names,
		)
	}
	return nil
}

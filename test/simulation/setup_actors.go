package simulation

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
)

const StakeToAdd uint64 = 9e3
const InitFund uint64 = 1e10

func createActorsAddresses(
	m testCommon.TestConfig,
	actorsMax int,
	actorType string,
) (actors []testCommon.AccountAndAddress) {
	actors = make([]testCommon.AccountAndAddress, 0)

	for actorIndex := 0; actorIndex < actorsMax; actorIndex++ {
		actorAccountName := getActorsAccountName(actorType, m.Seed, actorIndex)
		actorAccount, err := m.Client.AccountRegistryGetByName(actorAccountName)
		if err != nil {
			actorAccount, _, err = m.Client.AccountRegistryCreate(actorAccountName)
			if err != nil {
				m.T.Log("Error creating funder address: ", actorAccountName, " - ", err)
				continue
			}
		}
		actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
		if err != nil {
			continue
		}
		actors = append(actors, testCommon.AccountAndAddress{
			Acc:  actorAccount,
			Addr: actorAddress,
		})
	}
	return actors
}

func getActorAccountAndAddress(
	m testCommon.TestConfig,
	accountName string,
) (testCommon.AccountAndAddress, error) {
	actorAccount, err := m.Client.AccountRegistryGetByName(accountName)
	if err != nil {
		return testCommon.AccountAndAddress{
			Addr: "",
		}, err
	}
	actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
	if err != nil {
		return testCommon.AccountAndAddress{
			Addr: "",
		}, err
	}
	return testCommon.AccountAndAddress{
		Acc:  actorAccount,
		Addr: actorAddress,
	}, nil
}
func registerActorForTopic(
	m testCommon.TestConfig,
	topicId uint64,
	isReputer bool,
	actorType string,
	index int,
) error {
	actorAccountName := getActorsAccountName(actorType, m.Seed, index)
	actor, err := getActorAccountAndAddress(m, actorAccountName)
	if err != nil {
		return err
	}
	actorReputerRequest := &emissionstypes.MsgRegister{
		Sender:       actor.Addr,
		Owner:        actor.Addr,
		LibP2PKey:    actorType + "key" + strconv.Itoa(rand.Intn(10000000000)),
		MultiAddress: actorType + "multiaddress",
		TopicId:      topicId,
		IsReputer:    isReputer,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, actor.Acc, actorReputerRequest)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	if err != nil {
		return err
	}
	return nil
}
func stakeReputer(
	m testCommon.TestConfig,
	topicId uint64,
	stakeToAdd uint64,
	index int,
) error {
	actorAccountName := getActorsAccountName(REPUTER_TYPE, m.Seed, index)
	actorAccount, err := m.Client.AccountRegistryGetByName(actorAccountName)
	if err != nil {
		return err
	}
	actorAddressToFund, err := actorAccount.Address(params.HumanCoinUnit)
	if err != nil {
		return err
	}
	addStake := &emissionstypes.MsgAddStake{
		Sender:  actorAddressToFund,
		TopicId: topicId,
		Amount:  cosmosMath.NewIntFromUint64(stakeToAdd),
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, actorAccount, addStake)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	if err != nil {
		return err
	}

	return nil
}

func fundActors(
	m testCommon.TestConfig,
	sender testCommon.AccountAndAddress,
	targets []testCommon.AccountAndAddress,
) error {
	inputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(
			params.BaseCoinUnit,
			cosmosMath.NewInt(int64(InitFund)*int64(len(targets))),
		),
	)
	outputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(params.BaseCoinUnit, cosmosMath.NewInt(int64(InitFund))),
	)
	outputs := make([]banktypes.Output, len(targets))
	index := 0
	for _, accountAndAddress := range targets {
		outputs[index] = banktypes.Output{
			Address: accountAndAddress.Addr,
			Coins:   outputCoins,
		}
		index++
	}

	// Fund the accounts from faucet account in a single transaction
	sendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: sender.Addr,
				Coins:   inputCoins,
			},
		},
		Outputs: outputs,
	}
	_, err := m.Client.BroadcastTx(m.Ctx, sender.Acc, sendMsg)
	if err != nil {
		m.T.Log("Error actor address: ", err)
		return err
	}
	return nil
}
func RegisterAndStakeTopic(
	m testCommon.TestConfig,
	infererCount int,
	forecasterCount int,
	reputerCount int,
	topicId uint64,
) {
	for index := 0; index < infererCount; index++ {
		_ = registerActorForTopic(m, topicId, false, INFERER_TYPE, index)
	}
	for index := 0; index < forecasterCount; index++ {
		_ = registerActorForTopic(m, topicId, false, FORECASTER_TYPE, index)
	}
	for index := 0; index < reputerCount; index++ {
		_ = registerActorForTopic(m, topicId, true, REPUTER_TYPE, index)
		_ = stakeReputer(m, topicId, StakeToAdd, index)
	}
	m.T.Log("registered and staked actors")
}

func GenerateActors(
	m testCommon.TestConfig,
	infererCount int,
	forecasterCount int,
	reputerCount int,
) {
	inferers := createActorsAddresses(m, infererCount, INFERER_TYPE)
	forecasters := createActorsAddresses(m, forecasterCount, FORECASTER_TYPE)
	reputers := createActorsAddresses(m, reputerCount, REPUTER_TYPE)
	require.Equal(m.T, len(inferers), infererCount)
	require.Equal(m.T, len(forecasters), forecasterCount)
	require.Equal(m.T, len(reputers), reputerCount)
	sender := testCommon.AccountAndAddress{
		Acc:  m.FaucetAcc,
		Addr: m.FaucetAddr,
	}
	_ = fundActors(m, sender, inferers)
	_ = fundActors(m, sender, forecasters)
	_ = fundActors(m, sender, reputers)
	m.T.Log("generated and faucet actors")
}

package invariant_test

import (
	"context"
	"errors"
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	testcommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

// creates a new actor and registers them in the nodes account registry
func createNewActor(m *testcommon.TestConfig, numActors int) Actor {
	actorName := getActorName(m.Seed, numActors)
	actorAccount, _, err := m.Client.AccountRegistryCreate(actorName)
	if err != nil {
		if errors.Is(err, cosmosaccount.ErrAccountExists) {
			m.T.Log("WARNING WARNING WARNING\nACTOR ACCOUNTS ALREADY EXIST, YOU ARE REUSING YOUR SEED VALUE\nNON-DETERMINISM-DRAGONS AHEAD\nWARNING WARNING WARNING")
			actorAccount, err := m.Client.AccountRegistryGetByName(actorName)
			if err != nil {
				m.T.Log("Error getting actor account: ", actorName, " - ", err)
				return Actor{}
			}
			actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
			if err != nil {
				m.T.Log("Error creating actor address: ", actorName, " - ", err)
				return Actor{}
			}
			return Actor{
				name: actorName,
				addr: actorAddress,
				acc:  actorAccount,
			}
		} else {
			m.T.Log("Error creating actor address: ", actorName, " - ", err)
			return Actor{}
		}
	}
	actorAddress, err := actorAccount.Address(params.HumanCoinUnit)
	if err != nil {
		m.T.Log("Error creating actor address: ", actorName, " - ", err)
		return Actor{}
	}
	return Actor{
		name: actorName,
		addr: actorAddress,
		acc:  actorAccount,
	}
}

// creates a list of actors both as a map and a slice, returns both
func createActors(m *testcommon.TestConfig, numToCreate int) []Actor {
	actorsList := make([]Actor, numToCreate)
	for i := 0; i < numToCreate; i++ {
		actorsList[i] = createNewActor(m, i)
	}
	return actorsList
}

// fund every target address from the sender in amount coins
func fundActors(
	m *testcommon.TestConfig,
	sender Actor,
	targets []Actor,
	amount cosmossdk_io_math.Int,
) error {
	inputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(
			params.BaseCoinUnit,
			amount.MulRaw(int64(len(targets))),
		),
	)
	outputCoins := sdktypes.NewCoins(
		sdktypes.NewCoin(params.BaseCoinUnit, amount),
	)

	outputs := make([]banktypes.Output, len(targets))
	names := make([]string, len(targets))
	i := 0
	for _, actor := range targets {
		names[i] = actor.name
		outputs[i] = banktypes.Output{
			Address: actor.addr,
			Coins:   outputCoins,
		}
		i++
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
	ctx := context.Background()
	_, err := m.Client.BroadcastTx(ctx, sender.acc, sendMsg)
	if err != nil {
		m.T.Log("Error worker address: ", err)
		return err
	}
	m.T.Log(
		"Funded ",
		len(targets),
		" accounts from ",
		sender.name,
		" with ",
		amount,
		" coins:",
		" ",
		names,
	)
	return nil
}

// get the amount of money to give each actor in the simulation
// based on how much money the faucet currently has
func getPreFundAmount(
	m *testcommon.TestConfig,
	faucet Actor,
	numActors int,
) (cosmossdk_io_math.Int, error) {
	faucetBal, err := faucet.GetBalance(m)
	if err != nil {
		return cosmossdk_io_math.ZeroInt(), err
	}
	// divide by 10 so you can at least run 10 runs
	amountForThisRun := faucetBal.QuoRaw(int64(10))
	ret := amountForThisRun.QuoRaw(int64(numActors))
	if ret.Equal(cosmossdk_io_math.ZeroInt()) || ret.IsNegative() {
		return cosmossdk_io_math.ZeroInt(), fmt.Errorf(
			"Not enough funds in faucet account to fund actors",
		)
	}
	return ret, nil
}

// how much money an actor has
func (a *Actor) GetBalance(m *testcommon.TestConfig) (cosmossdk_io_math.Int, error) {
	ctx := context.Background()
	bal, err := m.Client.QueryBank().
		Balance(ctx, banktypes.NewQueryBalanceRequest(sdktypes.MustAccAddressFromBech32(a.addr), params.DefaultBondDenom))
	if err != nil {
		m.T.Logf("Error getting balance of actor %s: %v\n", a.String(), err)
		return cosmossdk_io_math.ZeroInt(), err
	}
	return bal.Balance.Amount, nil
}

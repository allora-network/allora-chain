package fuzz_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

// log wrapper for consistent logging style
func iterLog(t *testing.T, iteration int, a ...any) {
	t.Helper()
	t.Log(fmt.Sprint("[ITER ", iteration, "]: ", a))
}

// log wrapper for when iterations are complete consistent logging style
func iterSuccessLog(t *testing.T, iteration int, a ...any) {
	t.Helper()
	t.Log(fmt.Sprint("[SUCCESS ITER ", iteration, "]: ", a))
}

// log wrapper for when iterations are complete consistent logging style
func iterFailLog(t *testing.T, iteration int, a ...any) {
	t.Helper()
	t.Log(fmt.Sprint("[FAIL ITER ", iteration, "]: ", a))
}

// wrapper around require.NoError to only error if noFail is false
func failIfOnErr(t *testing.T, failOnErr bool, err error) {
	t.Helper()
	if failOnErr {
		require.NoError(t, err)
	}
}

// an actor in the simulation has a
// human readable name,
// string bech32 address,
// and an account with private key etc
// add a lock to this if you need to broadcast transactions in parallel
// from actors
type Actor struct {
	name string
	addr string
	acc  cosmosaccount.Account
}

// stringer for actor
func (a Actor) String() string {
	return a.name
}

// get the faucet name based on the seed for this test run
func getFaucetName(seed int) string {
	return "run" + strconv.Itoa(seed) + "_faucet"
}

// generates an actors name from seed and index
func getActorName(seed int, actorIndex int) string {
	return "run" + strconv.Itoa(seed) + "_actor" + strconv.Itoa(actorIndex)
}

// pick a random balance that is less than half of the actors balance
func pickRandomBalanceLessThanHalf(
	m *testcommon.TestConfig,
	actor Actor,
) (cosmossdk_io_math.Int, error) {
	balOfActor, err := actor.GetBalance(m)
	if err != nil {
		return cosmossdk_io_math.ZeroInt(), err
	}
	if balOfActor.Equal(cosmossdk_io_math.ZeroInt()) {
		return cosmossdk_io_math.ZeroInt(), nil
	}
	halfBal := balOfActor.QuoRaw(2)
	if halfBal.Equal(cosmossdk_io_math.ZeroInt()) {
		return cosmossdk_io_math.ZeroInt(), nil
	}
	divisor := m.Client.Rand.Int63()%1000 + 1
	randomBal := halfBal.QuoRaw(divisor)
	return randomBal, nil
}

func broadcastTxAndWait(
	m *testcommon.TestConfig,
	iteration int,
	failOnErr bool,
	sender Actor,
	msg types.Msg,
	resp proto.Message,
	beginMsg, failMsg, successMsg string,
) bool {
	iterLog(m.T, iteration, beginMsg)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, sender.acc, msg)
	failIfOnErr(m.T, failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, failMsg, ": tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, failMsg, ": tx wait error", err)
		return false
	}

	err = txResp.Decode(resp)
	failIfOnErr(m.T, failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, failMsg, ": tx decode error", err)
		return false
	}

	iterSuccessLog(m.T, iteration, successMsg)
	return true
}

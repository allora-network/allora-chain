package newstress_test

import (
	"fmt"
	"strconv"
	"testing"

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
func requireNoError(t *testing.T, failOnErr bool, err error) {
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

// helper function to check if an error was thrown cumulatively
func orErr(wasBeforeErr bool, err error) bool {
	return wasBeforeErr || err != nil
}

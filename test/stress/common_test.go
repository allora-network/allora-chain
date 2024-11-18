package stress_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// wrapper around require.NoError to only error if noFail is false
func requireNoError(t *testing.T, failOnErr bool, err error) {
	t.Helper()
	if failOnErr {
		require.NoError(t, err)
	}
}

// helper function to check if an error was thrown cumulatively
func orErr(wasBeforeErr bool, err error) bool {
	return wasBeforeErr || err != nil
}

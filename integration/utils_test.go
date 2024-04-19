package integration_test

import (
	"errors"
	"time"

	"github.com/stretchr/testify/require"
)

// Waits a specified number of blocks. Times out after a specified duration with an error.
func WaitNumBlocks(m TestMetadata, num int64, timeout time.Duration) (blockHeight int64, err error) {
	blockHeightStart, err := m.n.Client.LatestBlockHeight(m.ctx)
	require.NoError(m.t, err)
	var sleepCycles int64 = int64(timeout.Seconds() / (5 * time.Second).Seconds())
	blockHeight = blockHeightStart
	for blockHeight < blockHeightStart+num {
		if sleepCycles < 0 {
			return 0, errors.New("timeout waiting for blocks")
		}
		blockHeight, err = m.n.Client.LatestBlockHeight(m.ctx)
		require.NoError(m.t, err)
		time.Sleep(5 * time.Second)
		sleepCycles -= 1
	}
	return blockHeightStart, nil
}

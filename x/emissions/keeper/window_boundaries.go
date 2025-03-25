package keeper

import (
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Return true if the nonce is within the worker submission window for the topic
// Inclusive of the start block height and of the end block height.
func BlockWithinWorkerSubmissionWindowOfNonce(topic types.Topic, nonce types.Nonce, blockHeight int64) (bool, error) {
	if topic.WorkerSubmissionWindow == 0 {
		return false, errorsmod.Wrap(types.ErrInvalidValue, "worker submission window cannot be 0")
	}
	lowerBound := nonce.BlockHeight
	if lowerBound > math.MaxInt64-topic.WorkerSubmissionWindow {
		return false, errorsmod.Wrapf(types.ErrInvalidValue,
			"nonce block height %d is too high, adding window %d would overflow",
			nonce.BlockHeight,
			topic.WorkerSubmissionWindow)
	}
	upperBound := nonce.BlockHeight + topic.WorkerSubmissionWindow
	return lowerBound <= blockHeight && blockHeight <= upperBound, nil
}

// Return true if the nonce is within the reputer submission window for the topic
// Inclusive of the start block height and of the end block height
func BlockWithinReputerSubmissionWindowOfNonce(topic types.Topic, nonce types.ReputerRequestNonce, blockHeight int64) (bool, error) {
	// extraLag is the difference between the point at which the gt_lag is revealed and the end of epoch.
	extraLag := topic.GroundTruthLag % topic.EpochLength
	if extraLag != 0 {
		extraLag = topic.EpochLength - extraLag
	}
	// The block at which ground truth is revealed
	revealedGroundTruthBlock := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag

	// Allow 1 EpochLength + any extra lag as defined above.
	if revealedGroundTruthBlock > math.MaxInt64-(extraLag+topic.EpochLength) {
		return false, errorsmod.Wrapf(types.ErrInvalidValue,
			"nonce block height %d is too high, adding window %d would overflow",
			nonce.ReputerNonce.BlockHeight,
			topic.WorkerSubmissionWindow)
	}
	lowerBound := revealedGroundTruthBlock
	upperBound := revealedGroundTruthBlock + extraLag + topic.EpochLength
	return lowerBound <= blockHeight && blockHeight <= upperBound, nil
}

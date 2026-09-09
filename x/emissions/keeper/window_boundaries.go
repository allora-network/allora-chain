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
	lowerBound, upperBound, err := ReputerSubmissionWindowBounds(topic, nonce)
	if err != nil {
		return false, err
	}
	return lowerBound <= blockHeight && blockHeight <= upperBound, nil
}

// ReputerSubmissionWindowBounds returns the inclusive block range in which a
// reputer payload for this nonce is accepted. Callers that need to report the
// window (error messages, queries) must use this rather than recomputing it:
// the bounds were previously spelled out in four places, and the one that drifted
// is what allowed consecutive windows to overlap.
func ReputerSubmissionWindowBounds(topic types.Topic, nonce types.ReputerRequestNonce) (int64, int64, error) {
	extraLag := topic.GroundTruthLag % topic.EpochLength
	if extraLag != 0 {
		extraLag = topic.EpochLength - extraLag
	}
	revealedGroundTruthBlock := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag
	if revealedGroundTruthBlock > math.MaxInt64-(extraLag+topic.EpochLength) {
		return 0, 0, errorsmod.Wrapf(types.ErrInvalidValue,
			"nonce block height %d is too high, adding the submission window would overflow",
			nonce.ReputerNonce.BlockHeight)
	}
	lowerBound := revealedGroundTruthBlock + extraLag
	return lowerBound, lowerBound + topic.EpochLength, nil
}

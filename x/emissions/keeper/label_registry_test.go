package keeper_test

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// TestDecrementLabelRefCount_RemovesRowAtZero pins the core v2 invariant
// that BuildFinalEpochLabelRegistryFromActiveSet relies on: when a label's
// refcount reaches zero the entry is removed from activeInfererLabelRefCount,
// not merely zero-valued. Without this, BuildFinal would surface orphaned
// labels from evicted workers.
func (s *KeeperTestSuite) TestDecrementLabelRefCount_RemovesRowAtZero() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topicId := s.CreateTopic()
	nonce := types.BlockHeight(7)

	s.Require().NoError(tk.IncrementLabelRefCount(ctx, topicId, nonce, []string{"a"}))
	count, err := tk.GetLabelRefCount(ctx, topicId, nonce, "a")
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)

	s.Require().NoError(tk.DecrementLabelRefCount(ctx, topicId, nonce, []string{"a"}))

	count, err = tk.GetLabelRefCount(ctx, topicId, nonce, "a")
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count)

	visited := make(map[string]uint64)
	err = tk.IterateLabelsForNonce(ctx, topicId, nonce, func(label string, c uint64) (bool, error) {
		visited[label] = c
		return false, nil
	})
	s.Require().NoError(err)
	s.Require().Empty(visited, "row must be deleted at zero, not left as a zero-valued entry")
}

// TestDecrementLabelRefCount_UnderflowReturnsErrLogic pins that decrementing
// a label that was never incremented (missing row) is treated as a logic
// bug, not silently tolerated. Missing-row and already-zero are both
// wrapped by sdkerrors.ErrLogic with a "refcount underflow" message.
func (s *KeeperTestSuite) TestDecrementLabelRefCount_UnderflowReturnsErrLogic() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topicId := s.CreateTopic()
	nonce := types.BlockHeight(7)

	err := tk.DecrementLabelRefCount(ctx, topicId, nonce, []string{"a"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, sdkerrors.ErrLogic)
	s.Require().Contains(err.Error(), "refcount underflow")
}

// TestDecrementLabelRefCount_RejectsEmptyLabel mirrors the increment side's
// defensive check: the empty string is never a valid canonical label and
// callers that have not run canonicalization must fail loudly rather than
// write a row keyed on "".
func (s *KeeperTestSuite) TestDecrementLabelRefCount_RejectsEmptyLabel() {
	ctx := s.Ctx()
	tk := s.TopicKeeper()

	topicId := s.CreateTopic()
	nonce := types.BlockHeight(7)

	err := tk.DecrementLabelRefCount(ctx, topicId, nonce, []string{""})
	s.Require().Error(err)
	s.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

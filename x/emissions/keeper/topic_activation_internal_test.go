package keeper

import (
	"errors"
	"testing"

	errorsmod "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// A refusal keeps its sentinel and its weights through the wrap its callers receive it in.
func TestTopicRefusedErrorMatchesSentinelThroughWrap(t *testing.T) {
	compared := alloraMath.NewDecFromInt64(1)
	lowest := alloraMath.NewDecFromInt64(2)
	err := errorsmod.Wrap(&topicRefusedError{comparedWeight: compared, lowestWeight: lowest}, "wrapped")

	require.True(t, errorsmod.IsOf(err, types.ErrTopicCannotBeActivated))
	var refused *topicRefusedError
	require.True(t, errors.As(err, &refused))
	require.True(t, compared.Equal(refused.comparedWeight))
	require.True(t, lowest.Equal(refused.lowestWeight))
}

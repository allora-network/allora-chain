package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func TestSkimTopTopicIDsByWeightDescZeroKeepsNone(t *testing.T) {
	weight := alloraMath.MustNewDecFromString("1")
	got := skimTopTopicIDsByWeightDesc(map[TopicId]*alloraMath.Dec{
		1: &weight,
		2: &weight,
	}, 0)
	require.Empty(t, got)
}

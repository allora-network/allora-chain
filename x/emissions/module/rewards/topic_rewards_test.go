package rewards

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func TestGetAndUpdateActiveTopicWeights(t *testing.T) {
	ctx := sdk.Context{}
	block := int64(100)

	// Create a mock keeper
	k := keeper.Keeper{}

	// Set up the mock keeper's parameters
	moduleParams := types.Params{
		DefaultPageLimit:                100,
		TopicRewardAlpha:                math.NewDecFromInt64(10),
		TopicRewardStakeImportance:      math.NewDecFromInt64(0.5),
		TopicRewardFeeRevenueImportance: math.NewDecFromInt64(0.5),
		MinTopicWeight:                  math.NewDecFromInt64(1),
	}
	k.SetParams(ctx, moduleParams)

	// Set up the mock keeper's topic data
	topic := types.Topic{
		Id:          1,
		EpochLength: 100,
	}
	k.SetTopic(ctx, topic.Id, topic)

	// Call the function being tested
	weights, sumWeight, totalRevenue, err := GetAndUpdateActiveTopicWeights(ctx, k, block)

	// Check the results
	require.NoError(t, err)
	require.NotNil(t, weights)
	require.Equal(t, math.NewDecFromInt64(1), sumWeight)
	require.Equal(t, sdk.ZeroInt(), totalRevenue)
}

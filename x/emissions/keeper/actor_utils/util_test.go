package msgserver

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFindTopNByScoreDesc(t *testing.T) {
	topicId := uint64(0)

	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()
	worker5PrivateKey := secp256k1.GenPrivKey()

	worker1Addr := sdk.AccAddress(worker1PrivateKey.PubKey().Address())
	worker2Addr := sdk.AccAddress(worker2PrivateKey.PubKey().Address())
	worker3Addr := sdk.AccAddress(worker3PrivateKey.PubKey().Address())
	worker4Addr := sdk.AccAddress(worker4PrivateKey.PubKey().Address())
	worker5Addr := sdk.AccAddress(worker5PrivateKey.PubKey().Address())

	ReputerScoreEmas := make(map[string]types.Score)
	ReputerScoreEmas[worker1Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker1Addr.String(), Score: alloraMath.NewDecFromInt64(90)}
	ReputerScoreEmas[worker2Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker2Addr.String(), Score: alloraMath.NewDecFromInt64(40)}
	ReputerScoreEmas[worker3Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)}
	ReputerScoreEmas[worker4Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker4Addr.String(), Score: alloraMath.NewDecFromInt64(20)}
	ReputerScoreEmas[worker5Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)}

	topActors := FindTopNByScoreDesc(3, ReputerScoreEmas, 1)
	require.Equal(t, worker5Addr.String(), topActors[0])
	require.Equal(t, worker1Addr.String(), topActors[1])
	require.Equal(t, worker3Addr.String(), topActors[2])
}

func TestFindTopNByScoreDescWithNils(t *testing.T) {
	topicId := uint64(0)

	worker1PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker5PrivateKey := secp256k1.GenPrivKey()

	worker1Addr := sdk.AccAddress(worker1PrivateKey.PubKey().Address())
	worker3Addr := sdk.AccAddress(worker3PrivateKey.PubKey().Address())
	worker5Addr := sdk.AccAddress(worker5PrivateKey.PubKey().Address())

	ReputerScoreEmas := make(map[string]types.Score)
	ReputerScoreEmas[worker1Addr.String()] = types.Score{}
	ReputerScoreEmas[worker3Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)}
	ReputerScoreEmas[worker5Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)}

	// Actors with nil scores sent to the end
	topActors := FindTopNByScoreDesc(3, ReputerScoreEmas, 1)
	require.Equal(t, worker5Addr.String(), topActors[0])
	require.Equal(t, worker3Addr.String(), topActors[1])
	require.Equal(t, worker1Addr.String(), topActors[2])
}

func TestGetQuantileOfDescendingSliceAsAscending(t *testing.T) {
	scoresByActor := map[Actor]Score{
		"w1": types.Score{Score: alloraMath.NewDecFromInt64(90)},
		"w2": types.Score{Score: alloraMath.NewDecFromInt64(80)},
		"w3": types.Score{Score: alloraMath.NewDecFromInt64(70)},
		"w4": types.Score{Score: alloraMath.NewDecFromInt64(60)},
		"w5": types.Score{Score: alloraMath.NewDecFromInt64(50)},
	}

	sortedSlice := []Actor{"w1", "w2", "w3", "w4", "w5"}

	quantile := alloraMath.MustNewDecFromString("0.5")
	expectedResult := alloraMath.NewDecFromInt64(70)

	result, err := GetQuantileOfDescendingSliceAsAscending(scoresByActor, sortedSlice, quantile)
	require.NoError(t, err)
	require.Equal(t, expectedResult, result)

	quantile = alloraMath.MustNewDecFromString("0.2")
	expectedResult = alloraMath.NewDecFromInt64(58)

	result, err = GetQuantileOfDescendingSliceAsAscending(scoresByActor, sortedSlice, quantile)
	require.NoError(t, err)
	expectedInt, err := expectedResult.Int64()
	require.NoError(t, err)
	actualInt, err := result.Int64()
	require.NoError(t, err)
	require.Equal(t, expectedInt, actualInt)
}

package actorutils

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFindTopNByScoreDesc(t *testing.T) {
	key := storetypes.NewKVStoreKey("test")
	topicId := uint64(0)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx

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

	latestReputerScores := []types.Score{
		{TopicId: topicId, BlockHeight: 1, Address: worker1Addr.String(), Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: topicId, BlockHeight: 1, Address: worker2Addr.String(), Score: alloraMath.NewDecFromInt64(40)},
		{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: worker4Addr.String(), Score: alloraMath.NewDecFromInt64(20)},
		{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)},
	}

	topActors, _, topActorsBool := FindTopNByScoreDesc(testCtx, 3, latestReputerScores, 1)
	require.Equal(t, worker5Addr.String(), topActors[0].Address)
	require.Equal(t, worker1Addr.String(), topActors[1].Address)
	require.Equal(t, worker3Addr.String(), topActors[2].Address)

	_, isTop := topActorsBool[worker1Addr.String()]
	require.True(t, isTop)
	_, isTop = topActorsBool[worker2Addr.String()]
	require.False(t, isTop)
	_, isTop = topActorsBool[worker3Addr.String()]
	require.True(t, isTop)
	_, isTop = topActorsBool[worker4Addr.String()]
	require.False(t, isTop)
	_, isTop = topActorsBool[worker5Addr.String()]
	require.True(t, isTop)
}

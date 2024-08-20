package actorutils

import (
	"context"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFindTopNByScoreDesc(t *testing.T) {

	topicId := uint64(0)
	ctx := context.Background()
	sdkCtx := sdk.UnwrapSDKContext(ctx)

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

	latestReputerScores := make(map[string]types.Score)
	latestReputerScores[worker1Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker1Addr.String(), Score: alloraMath.NewDecFromInt64(90)}
	latestReputerScores[worker2Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker2Addr.String(), Score: alloraMath.NewDecFromInt64(40)}
	latestReputerScores[worker3Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)}
	latestReputerScores[worker4Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker4Addr.String(), Score: alloraMath.NewDecFromInt64(20)}
	latestReputerScores[worker5Addr.String()] = types.Score{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)}

	topActors, topActorsBool := FindTopNByScoreDesc(sdkCtx, 3, latestReputerScores, 1)
	require.Equal(t, worker5Addr.String(), topActors[0])
	require.Equal(t, worker1Addr.String(), topActors[1])
	require.Equal(t, worker3Addr.String(), topActors[2])

	require.Equal(t, topActorsBool[worker1Addr.String()], true)
	require.Equal(t, topActorsBool[worker2Addr.String()], false)
	require.Equal(t, topActorsBool[worker3Addr.String()], true)
	require.Equal(t, topActorsBool[worker4Addr.String()], false)
	require.Equal(t, topActorsBool[worker5Addr.String()], true)
}

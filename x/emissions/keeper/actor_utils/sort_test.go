package actorutils_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// test helper function reduce code duplication
func setUpSomeWorkers(t *testing.T) (
	worker1Addr sdk.AccAddress,
	worker2Addr sdk.AccAddress,
	worker3Addr sdk.AccAddress,
	worker4Addr sdk.AccAddress,
	worker5Addr sdk.AccAddress,
) {
	t.Helper()
	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()
	worker5PrivateKey := secp256k1.GenPrivKey()

	worker1Addr = sdk.AccAddress(worker1PrivateKey.PubKey().Address())
	worker2Addr = sdk.AccAddress(worker2PrivateKey.PubKey().Address())
	worker3Addr = sdk.AccAddress(worker3PrivateKey.PubKey().Address())
	worker4Addr = sdk.AccAddress(worker4PrivateKey.PubKey().Address())
	worker5Addr = sdk.AccAddress(worker5PrivateKey.PubKey().Address())

	return worker1Addr, worker2Addr, worker3Addr, worker4Addr, worker5Addr
}

func TestFindTopNByScoreDesc(t *testing.T) {
	topicId := uint64(0)
	worker1Addr,
		worker2Addr,
		worker3Addr,
		worker4Addr,
		worker5Addr := setUpSomeWorkers(t)

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: worker1Addr.String(), Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: topicId, BlockHeight: 1, Address: worker2Addr.String(), Score: alloraMath.NewDecFromInt64(40)},
		{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: worker4Addr.String(), Score: alloraMath.NewDecFromInt64(20)},
		{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)},
	}

	topActors, _, topActorsBool := actorutils.FindTopNByScoreDesc(3, reputerScoreEmas, 1)
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

func tiedScores(addresses []string, score int64) []emissionstypes.Score {
	scores := make([]emissionstypes.Score, 0, len(addresses))
	for _, a := range addresses {
		scores = append(scores, emissionstypes.Score{
			TopicId:     1,
			BlockHeight: 1,
			Address:     a,
			Score:       alloraMath.NewDecFromInt64(score),
		})
	}
	return scores
}

func sortedAddresses(scores []emissionstypes.Score) []string {
	addresses := make([]string, 0, len(scores))
	for _, s := range scores {
		addresses = append(addresses, s.Address)
	}
	return addresses
}

// permutations returns every ordering of items (Heap's algorithm).
func permutations(items []string) [][]string {
	var out [][]string
	var generate func(k int, a []string)
	generate = func(k int, a []string) {
		if k == 1 {
			out = append(out, append([]string(nil), a...))
			return
		}
		generate(k-1, a)
		for i := 0; i < k-1; i++ {
			if k%2 == 0 {
				a[i], a[k-1] = a[k-1], a[i]
			} else {
				a[0], a[k-1] = a[k-1], a[0]
			}
			generate(k-1, a)
		}
	}
	generate(len(items), append([]string(nil), items...))
	return out
}

// The expected orders are sha256(bigEndian(seed) || address), ascending,
// computed outside the function under test. Any change to these orders
// changes which actors win ties, which is a consensus change.
func TestFindTopNByScoreDescTieBreakIsPinned(t *testing.T) {
	addresses := []string{"alice", "bob", "carol", "dave", "erin"}
	tests := []struct {
		seed int64
		want []string
	}{
		{seed: 1, want: []string{"erin", "carol", "bob", "dave", "alice"}},
		{seed: 2, want: []string{"erin", "dave", "carol", "alice", "bob"}},
		{seed: 42, want: []string{"dave", "carol", "bob", "alice", "erin"}},
		{seed: 1000, want: []string{"alice", "carol", "bob", "dave", "erin"}},
	}
	for _, tc := range tests {
		t.Run(strconv.FormatInt(tc.seed, 10), func(t *testing.T) {
			_, all, _ := actorutils.FindTopNByScoreDesc(5, tiedScores(addresses, 7), tc.seed)
			require.Equal(t, tc.want, sortedAddresses(all))
		})
	}
}

func TestFindTopNByScoreDescIgnoresInputOrder(t *testing.T) {
	addresses := []string{"alice", "bob", "carol", "dave", "erin", "frank"}
	scoreOf := map[string]int64{"alice": 5, "bob": 9, "carol": 5, "dave": 9, "erin": 5, "frank": 1}

	var want []string
	for _, perm := range permutations(addresses) {
		scores := make([]emissionstypes.Score, 0, len(perm))
		for _, a := range perm {
			scores = append(scores, tiedScores([]string{a}, scoreOf[a])...)
		}
		top, all, isTop := actorutils.FindTopNByScoreDesc(3, scores, 42)
		got := sortedAddresses(all)
		if want == nil {
			want = got
		}
		require.Equal(t, want, got, "input order %v", perm)
		require.Equal(t, want[:3], sortedAddresses(top))
		require.Len(t, isTop, 3)
		for _, a := range want[:3] {
			require.Contains(t, isTop, a)
		}
	}
	// Scores stay descending; only the order within each tie depends on the seed.
	require.ElementsMatch(t, []string{"bob", "dave"}, want[:2])
	require.ElementsMatch(t, []string{"alice", "carol", "erin"}, want[2:5])
	require.Equal(t, "frank", want[5])
}

func TestFindTopNByScoreDescSeedChoosesTieWinner(t *testing.T) {
	addresses := []string{"alice", "bob", "carol", "dave", "erin"}
	top1, _, _ := actorutils.FindTopNByScoreDesc(1, tiedScores(addresses, 7), 1)
	top1000, _, _ := actorutils.FindTopNByScoreDesc(1, tiedScores(addresses, 7), 1000)
	require.Equal(t, "erin", top1[0].Address)
	require.Equal(t, "alice", top1000[0].Address)
}

// helper function that handles the expected types of the top actors
func requireIsTop(t *testing.T, expectedTop alloraMath.Dec, isTop bool) {
	t.Helper()
	require.True(
		t, expectedTop.Equal(alloraMath.OneDec()) || expectedTop.Equal(alloraMath.ZeroDec()),
		"expectedTop must be 0 or 1, got %s", expectedTop.String(),
	)
	expectedTopBool := expectedTop.Equal(alloraMath.OneDec())
	require.Equal(t, expectedTopBool, isTop)
}

func TestFindTopNByScoreDescCsv(t *testing.T) {
	for epoch := 301; epoch < 400; epoch++ {
		epochGet := alloratestutil.GetSortitionSimulatorValuesGetterForEpochs()[epoch]
		topicId := uint64(0)

		nParticipants, err := epochGet("n_participants").UInt64()
		require.NoError(t, err)
		reputerScoreEmas := make([]emissionstypes.Score, 0)
		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			reputerScoreEmas = append(reputerScoreEmas, emissionstypes.Score{
				TopicId:     topicId,
				BlockHeight: int64(epoch),
				Address:     participantName,
				Score:       epochGet(fmt.Sprintf("%s_prev_quality_ema", participantName)),
			})
		}

		nParticipantsDrawn, err := epochGet("n_participants_drawn").UInt64()
		require.NoError(t, err)

		_, _, topActorsBool := actorutils.FindTopNByScoreDesc(
			nParticipantsDrawn,
			reputerScoreEmas,
			1,
		)

		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			expectedTop := epochGet(fmt.Sprintf("%s_active", participantName))
			_, isTop := topActorsBool[participantName]
			requireIsTop(t, expectedTop, isTop)
		}
	}
}

// Not convinced we should be not throwing errors in FindTopNbyScoreDesc
// but for now instead of throwing errors, we find top N including empty scores
// and just file empty scores at the end of the list
func TestFindTopNByScoreDescWithNils(t *testing.T) {
	topicId := uint64(0)
	_, _, worker3Addr, _, worker5Addr := setUpSomeWorkers(t)

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)},
		{}, //nolint:exhaustruct
	}

	// Actors with nil scores sent to the end
	topActors, allActorsSorted, actorIsTop := actorutils.FindTopNByScoreDesc(3, reputerScoreEmas, 1)
	require.Len(t, topActors, 3)
	require.Equal(t, worker5Addr.String(), topActors[0].Address)
	require.Equal(t, struct{}{}, actorIsTop[worker5Addr.String()])
	require.Equal(t, worker3Addr.String(), topActors[1].Address)
	require.Equal(t, struct{}{}, actorIsTop[worker3Addr.String()])
	require.Equal(t, "", topActors[2].Address)
	require.Len(t, allActorsSorted, 3)
}

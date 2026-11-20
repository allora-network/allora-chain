package actorutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"slices"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// FindTopNByScoreDesc sorts scores in place by score, descending, and returns
// the top n, all scores in sorted order, and the set of top-n addresses.
//
// Equal scores are ordered by sha256(tieBreakSeed || address), ascending.
// Entries that are still equal keep their input order. The result is fully
// determined by the inputs, independent of the sort algorithm, which is what
// keeps every validator on the same order.
func FindTopNByScoreDesc(
	n uint64,
	scores []emissionstypes.Score,
	tieBreakSeed int64,
) (topNActorsSorted []emissionstypes.Score, allActorsSorted []emissionstypes.Score, actorIsTop map[string]struct{}) {
	var seed [8]byte
	binary.BigEndian.PutUint64(seed[:], uint64(tieBreakSeed)) //nolint:gosec // the seed's bit pattern is hashed; the sign is irrelevant

	type rankedScore struct {
		score      emissionstypes.Score
		tieBreaker [sha256.Size]byte
	}
	ranked := make([]rankedScore, len(scores))
	for i, s := range scores {
		ranked[i] = rankedScore{
			score:      s,
			tieBreaker: sha256.Sum256(append(seed[:], s.Address...)),
		}
	}

	slices.SortStableFunc(ranked, func(x, y rankedScore) int {
		switch {
		case x.score.Score.Gt(y.score.Score):
			return -1
		case x.score.Score.Lt(y.score.Score):
			return 1
		default:
			return bytes.Compare(x.tieBreaker[:], y.tieBreaker[:])
		}
	})
	for i := range ranked {
		scores[i] = ranked[i].score
	}

	N := min(n, uint64(len(scores)))
	topNActorsSorted = make([]emissionstypes.Score, N)
	actorIsTop = make(map[string]struct{}, N)
	for i := uint64(0); i < N; i++ {
		topNActorsSorted[i] = scores[i]
		actorIsTop[scores[i].Address] = struct{}{}
	}

	return topNActorsSorted, scores, actorIsTop
}

package msgserver

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Source: https://pkg.go.dev/container/heap#Push

// A structure to hold the original value and a random tiebreaker
type SortableItem struct {
	Value      Actor
	Weight     Score
	Tiebreaker uint32
	index      int
}

type Actor = string
type BlockHeight = int64
type Score = types.Score

// type PriorityQueue []*SortableItem

// func (pq PriorityQueue) Len() int { return len(pq) }

// func (pq PriorityQueue) Less(i, j int) bool {
// 	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
// 	if pq[i].Weight.Score.Equal(pq[j].Weight.Score) {
// 		return pq[i].Value > pq[j].Value
// 	}

// 	return pq[i].Weight.Score.Gt(pq[j].Weight.Score)
// }

// func (pq PriorityQueue) Swap(i, j int) {
// 	pq[i], pq[j] = pq[j], pq[i]
// 	pq[i].index = i
// 	pq[j].index = j
// }

// func (pq *PriorityQueue) Push(x any) {
// 	n := len(*pq)
// 	item := x.(*SortableItem)
// 	item.index = n
// 	*pq = append(*pq, item)
// }

// func (pq *PriorityQueue) Pop() any {
// 	old := *pq
// 	n := len(old)
// 	item := old[n-1]
// 	old[n-1] = nil  // avoid memory leak
// 	item.index = -1 // for safety
// 	*pq = old[0 : n-1]
// 	return item
// }

// // Sorts the given actors by score, desc, breaking ties randomly
// // Returns the top N actors as a map with the actor as the key and a boolean (True) as the value
// func FindTopNByScoreDesc_WITH_RANDOMNESS_OLD(n uint64, scoresByActor map[Actor]Score, randSeed BlockHeight) []Actor {
// 	// r := rand.New(rand.NewSource(randSeed))
// 	queue := &PriorityQueue{}
// 	i := 0
// 	for actor, score := range scoresByActor {
// 		queue.Push(&SortableItem{actor, score, uint32(1), i})
// 		i++
// 	}

// 	heap.Init(queue)

// 	topN := make([]Actor, 0)
// 	for i := 0; i < int(n); i++ {
// 		if queue.Len() == 0 {
// 			break
// 		}
// 		item := heap.Pop(queue).(*SortableItem)
// 		topN = append(topN, item.Value)
// 	}

// 	return topN
// }

// Track N highest scores. Track actors of highest score.
// If new score is higher than lowest score among them, then find and set the lowest score.
func FindTopNByScoresDesc(n uint64, scoresByActor map[Actor]Score) []Actor {
	if len(scoresByActor) == 0 {
		return []Actor{}
	}
	topN := make([]Actor, 0)
	lowestScore := types.Score{}
	firstIter := true
	for actor, score := range scoresByActor {
		if len(topN) < int(n) {
			topN = append(topN, actor)
			if firstIter || score.Score.Gt(lowestScore.Score) {
				lowestScore = score
				firstIter = false
			}
		} else {
			if score.Score.Gt(lowestScore.Score) {
				// Find the index of the lowest score
				lowestScoreIndex := -1
				for i, a := range topN {
					if scoresByActor[a].Score.Equal(lowestScore.Score) {
						lowestScoreIndex = i
						break
					}
				}
				if lowestScoreIndex == -1 {
					panic("Lowest score not found in topN")
				}
				topN[lowestScoreIndex] = actor
				lowestScore = score
			}
		}
	}
	return topN
}

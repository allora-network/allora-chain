package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// A tx function that accepts a list of inferences and possibly returns an error
func (ms msgServer) InsertInferences(ctx context.Context, msg *types.MsgInsertInferences) (*types.MsgInsertInferencesResponse, error) {
	// Group inferences by topicId - Create a map to store the grouped inferences
	groupedInferences := make(map[uint64][]*types.Inference)

	// Iterate through the array and group by topic_id
	for _, inference := range msg.Inferences {
		groupedInferences[inference.TopicId] = append(groupedInferences[inference.TopicId], inference)
	}

	// Update all_inferences
	for topicId, inferences := range groupedInferences {
		topicInferences := &types.Inferences{
			Inferences: inferences,
		}
		err := ms.k.InsertInferences(ctx, topicId, msg.BlockHeight, *topicInferences)
		if err != nil {
			return nil, err
		}
	}

	// Return an empty response as the operation was successful
	return &types.MsgInsertInferencesResponse{}, nil
}

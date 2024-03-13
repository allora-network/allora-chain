package msgserver

import (
	"context"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a list of forecasts and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) ProcessForecasts(ctx context.Context, msg *state.MsgProcessForecasts) (*state.MsgProcessForecastsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	forecasts := msg.Forecasts
	// Group inferences by topicId - Create a map to store the grouped inferences
	groupedForecasts := make(map[uint64][]*state.Forecast)

	// Iterate through the array and group by topic_id
	for _, forecast := range forecasts {
		groupedForecasts[forecast.TopicId] = append(groupedForecasts[forecast.TopicId], forecast)
	}

	actualTimestamp := uint64(sdkCtx.BlockTime().Unix())

	// Update all_inferences
	for topicId, forecasts := range groupedForecasts {
		forecasts := &state.Forecasts{
			Forecasts: forecasts,
		}
		err := ms.k.InsertForecasts(ctx, topicId, actualTimestamp, *forecasts)
		if err != nil {
			return nil, err
		}
	}

	// Return an empty response as the operation was successful
	return &state.MsgProcessForecastsResponse{}, nil
}

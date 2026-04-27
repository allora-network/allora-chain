package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InitGenesis initializes the WorkerKeeper state from a genesis state.
func (k *WorkerKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// TopicWorkers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicWorkers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := k.SetTopicWorker(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
		}
	}

	// Inferences []*TopicIdActorIdInference
	//
	// Deprecated: replaced by WorkerLatestInputInferences in Epoch Label
	// Registry and drained by the v15 migration. Kept for backwards-
	// compatible genesis round-trip of any in-flight entries captured
	// pre-migration; post-upgrade this list is always empty.
	for _, topicIdActorIdInference := range data.Inferences {
		if topicIdActorIdInference != nil {
			if topicIdActorIdInference.Inference == nil {
				return errors.Wrap(types.ErrInvalidValue, "inference cannot be nil")
			}
			if err := topicIdActorIdInference.Inference.Validate(); err != nil {
				return errors.Wrap(err, "inference in list is invalid")
			}
			if err := k.inferences.Set(ctx,
				collections.Join(
					topicIdActorIdInference.TopicId,
					topicIdActorIdInference.ActorId),
				*topicIdActorIdInference.Inference); err != nil {
				return errors.Wrap(err, "error setting inferences")
			}
		}
	}

	// WorkerLatestInputInferences []*TopicIdActorIdInputInference
	//
	// Staged raw worker submissions inside an open worker submission window.
	// Does NOT call InputInference.ValidateWithLimits here because the
	// effective cap/whitelist depend on runtime topic/module state that is
	// itself being imported; instead we rely on the payload having been
	// validated at submission time and on migrate.go to produce
	// well-formed entries from the legacy inferences store.
	for _, row := range data.WorkerLatestInputInferences {
		if row == nil {
			continue
		}
		if row.InputInference == nil {
			return errors.Wrap(types.ErrInvalidValue, "worker_latest_input_inferences: input_inference cannot be nil")
		}
		if err := types.ValidateTopicId(row.TopicId); err != nil {
			return errors.Wrap(err, "worker_latest_input_inferences: topic id invalid")
		}
		if err := types.ValidateBech32(row.ActorId); err != nil {
			return errors.Wrap(err, "worker_latest_input_inferences: actor id invalid")
		}
		if err := k.workerLatestInputInferences.Set(ctx,
			collections.Join(row.TopicId, row.ActorId),
			*row.InputInference); err != nil {
			return errors.Wrap(err, "error setting worker_latest_input_inferences")
		}
	}

	// Forecasts []*TopicIdActorIdForecast
	for _, topicIdActorIdForecast := range data.Forecasts {
		if topicIdActorIdForecast != nil {
			if topicIdActorIdForecast.Forecast == nil {
				return errors.Wrap(types.ErrInvalidValue, "forecast cannot be nil")
			}
			if err := topicIdActorIdForecast.Forecast.Validate(); err != nil {
				return errors.Wrap(err, "forecast in list is invalid")
			}
			if err := k.forecasts.Set(ctx,
				collections.Join(
					topicIdActorIdForecast.TopicId,
					topicIdActorIdForecast.ActorId),
				*topicIdActorIdForecast.Forecast); err != nil {
				return errors.Wrap(err, "error setting forecasts")
			}
		}
	}

	// Workers []*LibP2PKeyAndOffchainNode
	for _, libP2PKeyAndOffchainNode := range data.Workers {
		if libP2PKeyAndOffchainNode != nil {
			if libP2PKeyAndOffchainNode.OffchainNode == nil {
				return errors.Wrap(types.ErrInvalidValue, "worker info cannot be nil")
			}
			if err := libP2PKeyAndOffchainNode.OffchainNode.Validate(); err != nil {
				return errors.Wrap(err, "worker info validation failed")
			}
			if err := k.workers.Set(
				ctx,
				libP2PKeyAndOffchainNode.LibP2PKey,
				*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
				return errors.Wrap(err, "error setting workers")
			}
		}
	}

	// AllInferences []*TopicIdBlockHeightInferences
	for _, topicIdBlockHeightInferences := range data.AllInferences {
		if topicIdBlockHeightInferences != nil {
			if topicIdBlockHeightInferences.Inferences == nil {
				return errors.Wrap(types.ErrInvalidValue, "all inferences cannot be nil")
			}
			for _, inference := range topicIdBlockHeightInferences.Inferences.Inferences {
				if inference != nil {
					if err := inference.Validate(); err != nil {
						return errors.Wrap(err, "inference validation failed")
					}
				}
			}
			if err := k.allInferences.Set(ctx,
				collections.Join(topicIdBlockHeightInferences.TopicId, topicIdBlockHeightInferences.BlockHeight),
				*topicIdBlockHeightInferences.Inferences); err != nil {
				return errors.Wrap(err, "error setting allInferences")
			}
		}
	}

	// AllForecasts []*TopicIdBlockHeightForecasts
	for _, topicIdBlockHeightForecasts := range data.AllForecasts {
		if topicIdBlockHeightForecasts != nil {
			if topicIdBlockHeightForecasts.Forecasts == nil {
				return errors.Wrap(types.ErrInvalidValue, "all forecasts cannot be nil")
			}
			for _, forecast := range topicIdBlockHeightForecasts.Forecasts.Forecasts {
				if forecast != nil {
					if err := forecast.Validate(); err != nil {
						return errors.Wrap(err, "forecast validation failed")
					}
				}
			}
			if err := k.allForecasts.Set(ctx,
				collections.Join(topicIdBlockHeightForecasts.TopicId, topicIdBlockHeightForecasts.BlockHeight),
				*topicIdBlockHeightForecasts.Forecasts); err != nil {
				return errors.Wrap(err, "error setting allForecasts")
			}
		}
	}

	// ActiveInferers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveInferers {
		if topicAndActorId != nil {
			if err := k.AddActiveInferer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeInferers")
			}
		}
	}

	// ActiveForecasters []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveForecasters {
		if topicAndActorId != nil {
			if err := k.AddActiveForecaster(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeForecasters")
			}
		}
	}

	return nil
}

// ExportGenesis exports the WorkerKeeper state into a genesis state.
func (k *WorkerKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// topicWorkers
	topicWorkers := make([]*types.TopicAndActorId, 0)
	topicWorkersIter, err := k.topicWorkers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic workers")
	}
	for ; topicWorkersIter.Valid(); topicWorkersIter.Next() {
		key, err := topicWorkersIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: topicWorkersIter")
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicWorkers = append(topicWorkers, &topicIdAndActorId)
	}
	data.TopicWorkers = topicWorkers

	// inferences (deprecated: see InitGenesis)
	inferences := make([]*types.TopicIdActorIdInference, 0)
	inferencesIter, err := k.inferences.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate inferences")
	}
	for ; inferencesIter.Valid(); inferencesIter.Next() {
		keyValue, err := inferencesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: inferencesIter")
		}
		value := keyValue.Value
		topicIdActorIdInference := types.TopicIdActorIdInference{
			TopicId:   keyValue.Key.K1(),
			ActorId:   keyValue.Key.K2(),
			Inference: &value,
		}
		inferences = append(inferences, &topicIdActorIdInference)
	}
	data.Inferences = inferences

	// workerLatestInputInferences (v2)
	workerInputs := make([]*types.TopicIdActorIdInputInference, 0)
	workerInputsIter, err := k.workerLatestInputInferences.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate worker_latest_input_inferences")
	}
	for ; workerInputsIter.Valid(); workerInputsIter.Next() {
		keyValue, err := workerInputsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: workerInputsIter")
		}
		value := keyValue.Value
		workerInputs = append(workerInputs, &types.TopicIdActorIdInputInference{
			TopicId:        keyValue.Key.K1(),
			ActorId:        keyValue.Key.K2(),
			InputInference: &value,
		})
	}
	data.WorkerLatestInputInferences = workerInputs

	// forecasts
	forecasts := make([]*types.TopicIdActorIdForecast, 0)
	forecastsIter, err := k.forecasts.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate forecasts")
	}
	for ; forecastsIter.Valid(); forecastsIter.Next() {
		keyValue, err := forecastsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: forecastsIter")
		}
		value := keyValue.Value
		topicIdActorIdForecast := types.TopicIdActorIdForecast{
			TopicId:  keyValue.Key.K1(),
			ActorId:  keyValue.Key.K2(),
			Forecast: &value,
		}
		forecasts = append(forecasts, &topicIdActorIdForecast)
	}
	data.Forecasts = forecasts

	// workers
	workers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	workersIter, err := k.workers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate workers")
	}
	for ; workersIter.Valid(); workersIter.Next() {
		keyValue, err := workersIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: workersIter")
		}
		value := keyValue.Value
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &value,
		}
		workers = append(workers, &libP2PKeyAndOffchainNode)
	}
	data.Workers = workers

	// allInferences
	allInferences := make([]*types.TopicIdBlockHeightInferences, 0)
	allInferencesIter, err := k.allInferences.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate all inferences")
	}
	for ; allInferencesIter.Valid(); allInferencesIter.Next() {
		keyValue, err := allInferencesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: allInferencesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightInferences := types.TopicIdBlockHeightInferences{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Inferences:  &value,
		}
		allInferences = append(allInferences, &topicIdBlockHeightInferences)
	}
	data.AllInferences = allInferences

	// allForecasts
	allForecasts := make([]*types.TopicIdBlockHeightForecasts, 0)
	allForecastsIter, err := k.allForecasts.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate all forecasts")
	}
	for ; allForecastsIter.Valid(); allForecastsIter.Next() {
		keyValue, err := allForecastsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: allForecastsIter")
		}
		value := keyValue.Value
		topicIdBlockHeightForecasts := types.TopicIdBlockHeightForecasts{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Forecasts:   &value,
		}
		allForecasts = append(allForecasts, &topicIdBlockHeightForecasts)
	}
	data.AllForecasts = allForecasts

	// activeInferers
	activeInferers := make([]*types.TopicAndActorId, 0)
	activeInferersIter, err := k.activeInferers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate active inferers")
	}
	for ; activeInferersIter.Valid(); activeInferersIter.Next() {
		key, err := activeInferersIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: activeInferersIter")
		}
		activeInferers = append(activeInferers, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}
	data.ActiveInferers = activeInferers

	// activeForecasters
	activeForecasters := make([]*types.TopicAndActorId, 0)
	activeForecasterIter, err := k.activeForecasters.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate active forecasters")
	}
	for ; activeForecasterIter.Valid(); activeForecasterIter.Next() {
		key, err := activeForecasterIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: activeForecasterIter")
		}
		activeForecasters = append(activeForecasters, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}
	data.ActiveForecasters = activeForecasters

	return nil
}

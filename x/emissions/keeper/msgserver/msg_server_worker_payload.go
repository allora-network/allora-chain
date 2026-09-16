package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InsertWorkerPayload accepts an individual inference and forecast and possibly returns an error.
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields.
// Only 1 payload per registered worker is kept, ignore the rest. In particular, take the first payload from each
// registered worker and none from any unregistered actor.
// Signatures, anti-sybil procedures, and "skimming of only the top few workers by EMA score descending" should be done here.
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.InsertWorkerPayloadRequest) (_ *types.InsertWorkerPayloadResponse, err error) {
	defer metrics.RecordMetrics("InsertWorkerPayload", time.Now(), &err)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err = types.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error validating sender address")
	}
	err = types.ValidateStringIsBech32(msg.WorkerDataBundle.Worker)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error validating worker address")
	}
	canSubmit, err := ms.wlk.CanSubmitWorkerPayload(ctx, msg.WorkerDataBundle.TopicId, msg.WorkerDataBundle.Worker)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error checking if worker can submit payload")
	} else if !canSubmit {
		return nil, errorsmod.Wrapf(types.ErrNotPermittedToSubmitWorkerPayload, "Worker is not permitted to submit payload")
	}

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err, "Error validating worker data bundle")
	}

	moduleParams, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params")
	}
	blockHeight := sdkCtx.BlockHeight()

	err = checkInputLength(moduleParams.MaxSerializedMsgLength, msg)
	if err != nil {
		return nil, err
	}

	nonce := msg.WorkerDataBundle.Nonce
	topicId := msg.WorkerDataBundle.TopicId

	topic, err := ms.tk.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}
	nonceUnfulfilled, err := ms.nk.IsWorkerNonceUnfulfilled(ctx, topicId, nonce)
	if err != nil {
		return nil, err
	} else if !nonceUnfulfilled {
		return nil, types.ErrUnfulfilledNonceNotFound
	}

	// Prefer Epoch wall-clock windows when a live scheduler epoch matches this legacy nonce.
	epochFound, err := ms.k.CheckWorkerSubmissionWindow(ctx, topicId, *nonce)
	if err != nil {
		return nil, err
	}
	if !epochFound {
		withinWindow, err := keeper.BlockWithinWorkerSubmissionWindowOfNonce(topic, *nonce, blockHeight)
		if err != nil {
			return nil, err
		} else if !withinWindow {
			return nil, errorsmod.Wrapf(
				types.ErrWorkerNonceWindowNotAvailable,
				"Worker window not open for topic: %d, current block %d, start window: %d, end window: %d",
				topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow,
			)
		}
	}

	isWorkerRegistered, err := ms.wk.IsWorkerRegisteredInTopic(ctx, topicId, msg.WorkerDataBundle.Worker)
	if err != nil {
		return nil, err
	} else if !isWorkerRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "worker is not registered in this topic")
	}

	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, err
	}

	inputBundle := msg.WorkerDataBundle
	inferenceForecastsBundle := inputBundle.GetInferenceForecastsBundle()
	// Convert (and validate) the forecast up front so a malformed forecast fails
	// the tx before any inference state is written; it is processed after inferences.
	var forecast *types.Forecast
	if rawForecast := inferenceForecastsBundle.GetForecast(); rawForecast != nil {
		forecast, err = ms.convertWorkerForecastPayload(inputBundle, rawForecast)
		if err != nil {
			return nil, err
		}
	}
	if rawInference := inferenceForecastsBundle.GetInference(); rawInference != nil {
		err = ms.processWorkerInferencePayload(ctx, sdkCtx, topic, nonce.BlockHeight, msg, rawInference, moduleParams)
		if err != nil {
			return nil, err
		}
	}
	if forecast != nil {
		err = ms.processWorkerForecastPayload(ctx, sdkCtx, topic, nonce.BlockHeight, inputBundle, forecast, moduleParams)
		if err != nil {
			return nil, err
		}
	}

	return &types.InsertWorkerPayloadResponse{}, nil
}

func (ms msgServer) processWorkerInferencePayload(
	ctx context.Context,
	sdkCtx sdk.Context,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	msg *types.InsertWorkerPayloadRequest,
	rawInference *types.InputInference,
	moduleParams types.Params,
) error {
	if err := validateWorkerInferenceLabels(topic, rawInference, moduleParams); err != nil {
		return err
	}

	// Re-resolve the stored cap into the global [min, max] on every payload: a
	// value outside the range (genesis-supplied, or left behind by a governance
	// change) is clamped rather than trusted, keeping the active set within the
	// global-sized score-retention window.
	maxTopInferersToReward := types.EffectiveMaxTopInferersToReward(
		topic.MaxTopInferersToReward,
		moduleParams.MinTopInferersToReward,
		moduleParams.MaxTopInferersToReward,
	)
	plan, err := ms.wk.PlanInferenceAdmission(
		sdkCtx,
		topic,
		nonceBlockHeight,
		rawInference.Inferer,
		maxTopInferersToReward,
	)
	if err != nil {
		return errorsmod.Wrap(err, "Error planning inference admission")
	}

	if !plan.Admitted() {
		// Preserve the existing score/liveness side effects for non-admitted
		// inferers, but do not normalize: normalization registers ELR labels.
		if err := ms.wk.CommitNonAdmittedInference(sdkCtx, topic, nonceBlockHeight, plan); err != nil {
			return errorsmod.Wrap(err, "Error committing non-admitted inference")
		}
		return nil
	}

	// This is the only worker-payload path that normalizes inference labels.
	// At this point the worker is admitted, so ELR registration is intentional.
	inference, err := ms.wk.NormalizeInputInference(ctx, topic, nonceBlockHeight, rawInference)
	if err != nil {
		return errorsmod.Wrap(err, "failed to normalize admitted inference")
	}
	if inference.TopicId != msg.WorkerDataBundle.TopicId {
		return errorsmod.Wrapf(types.ErrInvalidTopicId, "inferer not using the same topic as bundle")
	}
	if err := inference.Validate(); err != nil {
		return errorsmod.Wrap(err, "inference is invalid")
	}
	if err := ms.wk.CommitAdmittedInference(sdkCtx, topic, nonceBlockHeight, inference, plan); err != nil {
		return errorsmod.Wrap(err, "Error committing admitted inference")
	}
	// Emitted only on admission: a not-admitted inferer returns above without an event.
	types.EmitNewInsertInfererPayloadEvent(ctx, msg.WorkerDataBundle)
	return nil
}

func validateWorkerInferenceLabels(
	topic types.Topic,
	rawInference *types.InputInference,
	moduleParams types.Params,
) error {
	labelCap := topic.MaxLabelsPerSubmission
	if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
		labelCap = 1
	}
	whitelist := types.CanonicalLabelSet(topic.LabelWhitelist)
	if err := rawInference.ValidateWithLimits(
		labelCap,
		whitelist,
		moduleParams.MaxCanonicalLabelByteLength,
		topic.LabelCaseSensitive,
	); err != nil {
		return errorsmod.Wrapf(err, "input inference failed label validation")
	}
	if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE && len(rawInference.Values) == 1 {
		if err := types.ValidateSingleArityLabel(rawInference.Values[0].Label); err != nil {
			return err
		}
	}
	return nil
}

func (ms msgServer) convertWorkerForecastPayload(
	inputBundle *types.InputWorkerDataBundle,
	rawForecast *types.InputForecast,
) (*types.Forecast, error) {
	// Forecast conversion is independent of inference admission and does not
	// touch the epoch label registry.
	forecast, err := types.NewForecastFromInput(rawForecast)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to convert forecast")
	}
	if len(forecast.ForecastElements) == 0 {
		return nil, errorsmod.Wrapf(types.ErrNoValidForecastElements, "No valid forecast elements found in Forecast")
	}
	if forecast.TopicId != inputBundle.TopicId {
		return nil, errorsmod.Wrapf(types.ErrInvalidTopicId, "forecaster not using the same topic as bundle")
	}
	return forecast, nil
}

func (ms msgServer) processWorkerForecastPayload(
	ctx context.Context,
	sdkCtx sdk.Context,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	inputBundle *types.InputWorkerDataBundle,
	forecast *types.Forecast,
	moduleParams types.Params,
) error {
	// Limit forecast elements to top inferers.
	latestScoresForForecastedInferers := make([]types.Score, 0)
	for _, el := range forecast.ForecastElements {
		score, err := ms.sck.GetInfererScoreEma(ctx, forecast.TopicId, el.Inferer)
		if err != nil {
			continue
		}
		latestScoresForForecastedInferers = append(latestScoresForForecastedInferers, score)
	}

	_, _, topNInferer := actorutils.FindTopNByScoreDesc(
		sdkCtx,
		moduleParams.MaxElementsPerForecast,
		latestScoresForForecastedInferers,
		forecast.BlockHeight,
	)

	// Remove duplicate forecast elements.
	acceptedForecastElements := make([]*types.ForecastElement, 0)
	seenInferers := make(map[string]bool)
	for _, el := range forecast.ForecastElements {
		notAlreadySeen := !seenInferers[el.Inferer]
		_, isTopInferer := topNInferer[el.Inferer]
		if notAlreadySeen && isTopInferer {
			acceptedForecastElements = append(acceptedForecastElements, el)
			seenInferers[el.Inferer] = true
		}
	}

	if len(acceptedForecastElements) > 0 {
		forecast.ForecastElements = acceptedForecastElements
		if err := ms.wk.AppendForecast(sdkCtx, topic, nonceBlockHeight, forecast, moduleParams.MaxTopForecastersToReward); err != nil {
			return errorsmod.Wrapf(err, "Error appending forecast")
		}

		// Emitted only when forecast elements were actually stored: a fully
		// filtered-out forecast returns without an event, mirroring the
		// admission-gated inference path in processWorkerInferencePayload.
		eventBundle := &types.WorkerDataBundle{
			Worker:  inputBundle.Worker,
			Nonce:   inputBundle.Nonce,
			TopicId: inputBundle.TopicId,
			InferenceForecastsBundle: &types.InferenceForecastBundle{
				Inference: nil,
				Forecast:  forecast,
			},
		}
		types.EmitNewInsertForecasterPayloadEvent(ctx, eventBundle)
	}
	return nil
}

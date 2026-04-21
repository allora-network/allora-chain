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

// effectiveMaxLabelsPerSubmission returns the effective per-submission label
// cap for a topic: min(params.MaxLabelsPerSubmission, topic.cap_if_nonzero).
// A zero topic cap means "inherit the module-level cap", which matches how
// other optional topic fields work on the chain.
func effectiveMaxLabelsPerSubmission(paramsCap, topicCap uint64) uint64 {
	switch {
	case topicCap == 0:
		return paramsCap
	case paramsCap == 0:
		return topicCap
	case topicCap < paramsCap:
		return topicCap
	default:
		return paramsCap
	}
}

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

	// Pre-validate the raw input inference with the effective per-topic cap
	// and whitelist. ValidateWithLimits canonicalizes labels in place on the
	// inbound InputInference; subsequent Normalize + AppendInference +
	// SetWorkerLatestInputInference see canonical forms only.
	if rawInput := msg.WorkerDataBundle.GetInferenceForecastsBundle().GetInference(); rawInput != nil {
		effectiveCap := effectiveMaxLabelsPerSubmission(moduleParams.MaxLabelsPerSubmission, topic.MaxLabelsPerSubmission)
		whitelist := types.CanonicalLabelSet(topic.LabelWhitelist)
		if err := rawInput.ValidateWithLimits(effectiveCap, whitelist); err != nil {
			return nil, errorsmod.Wrapf(err, "input inference failed label validation")
		}
	}

	wdb, err := ms.wk.NewWorkerDataBundleFromInput(ctx, topic, nonce.BlockHeight, msg.WorkerDataBundle)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Worker bad data format for block: %d", blockHeight)
	}
	wdb, err := ms.wk.MaterializeWorkerDataBundle(ctx, nonce.BlockHeight, normalized)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to materialize worker data bundle for block: %d", blockHeight)
	}

	// Process Inferences
	if inference := wdb.InferenceForecastsBundle.GetInference(); inference != nil {
		if inference.TopicId != wdb.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId,
				"inferer not using the same topic as bundle")
		}

		// AppendInference admits/rejects based on EMA scoring using the
		// (already canonicalized) raw input inference. On success we stage
		// the raw submission + bump the per-label refcount.
		rawInput := msg.WorkerDataBundle.InferenceForecastsBundle.GetInference()
		err = ms.wk.AppendInference(sdkCtx, topic, nonce.BlockHeight, rawInput, moduleParams.MaxTopInferersToReward)
		if err != nil {
			return nil, errorsmod.Wrap(err, "Error appending inference")
		}

		// Persist raw submission AFTER AppendInference so an admission failure
		// cannot leave a ghost entry. Then bump the per-label refcount,
		// which reads the just-staged InputInference. The order matters:
		// Set-then-Increment keeps the refcount in lock-step with the store.
		if err := ms.wk.SetWorkerLatestInputInference(ctx, topic.Id, rawInput.Inferer, *rawInput); err != nil {
			return nil, errorsmod.Wrap(err, "Error staging worker input inference")
		}
		if err := ms.wk.IncrementStagedLabelRefCount(ctx, topic, nonce.BlockHeight, rawInput.Inferer); err != nil {
			return nil, errorsmod.Wrap(err, "Error incrementing active inferer label refcount")
		}

		types.EmitNewInsertInfererPayloadEvent(ctx, msg.WorkerDataBundle)
	}

	// Process Forecasts
	if forecast := wdb.InferenceForecastsBundle.GetForecast(); forecast != nil {
		if len(forecast.ForecastElements) == 0 {
			return nil, errorsmod.Wrapf(types.ErrNoValidForecastElements, "No valid forecast elements found in Forecast")
		}
		if forecast.TopicId != wdb.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId, "forecaster not using the same topic as bundle")
		}

		// Limit forecast elements to top inferers
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

		// Remove duplicate forecast elements
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
			err = ms.wk.AppendForecast(sdkCtx, topic, nonce.BlockHeight, forecast, moduleParams.MaxTopForecastersToReward)
			if err != nil {
				return nil, errorsmod.Wrapf(err,
					"Error appending forecast")
			}
		}

		types.EmitNewInsertForecasterPayloadEvent(ctx, wdb)
	}

	return &types.InsertWorkerPayloadResponse{}, nil
}

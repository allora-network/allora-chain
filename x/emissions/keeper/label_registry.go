package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// IncrementLabelRefCount bumps the refcount for each label in the
// (topicId, nonce, label) store by one. Labels must already be in canonical
// form. Called from the worker admission tracker in AppendInference after a
// successful AddActiveInferer on a MULTI topic.
//
// Any error short-circuits the whole batch; partial updates are fine because
// the enclosing Cosmos tx will roll back on error.
func (k *TopicKeeper) IncrementLabelRefCount(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	labels []string,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	for _, label := range labels {
		if label == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot increment refcount for empty label")
		}
		key := collections.Join3(topicId, nonce, label)
		curr, err := k.activeInfererLabelRefCount.Get(ctx, key)
		if err != nil && !errIsNotFound(err) {
			return errorsmod.Wrap(err, "error reading active inferer label refcount")
		}
		if err := k.activeInfererLabelRefCount.Set(ctx, key, curr+1); err != nil {
			return errorsmod.Wrap(err, "error writing active inferer label refcount")
		}
	}
	return nil
}

// DecrementLabelRefCount decreases the refcount for each label. When a
// row's count reaches zero it is deleted so that BuildFinalEpochLabelRegistry
// cannot see orphaned labels from evicted workers. Labels must already be in
// canonical form. Missing rows and underflows are turned into errors: the
// algebra must be exact (each inferer admit increments once, each eviction
// decrements once; any deviation is a logic bug).
func (k *TopicKeeper) DecrementLabelRefCount(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	labels []string,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce block height validation failed")
	}
	for _, label := range labels {
		if label == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot decrement refcount for empty label")
		}
		key := collections.Join3(topicId, nonce, label)
		curr, err := k.activeInfererLabelRefCount.Get(ctx, key)
		if err != nil {
			if errIsNotFound(err) {
				return errorsmod.Wrapf(sdkerrors.ErrLogic, "refcount underflow: label %q missing for (topic=%d, nonce=%d)", label, topicId, nonce)
			}
			return errorsmod.Wrap(err, "error reading active inferer label refcount")
		}
		if curr == 0 {
			return errorsmod.Wrapf(sdkerrors.ErrLogic, "refcount underflow: label %q already zero for (topic=%d, nonce=%d)", label, topicId, nonce)
		}
		if curr == 1 {
			if err := k.activeInfererLabelRefCount.Remove(ctx, key); err != nil {
				return errorsmod.Wrap(err, "error removing active inferer label refcount")
			}
			continue
		}
		if err := k.activeInfererLabelRefCount.Set(ctx, key, curr-1); err != nil {
			return errorsmod.Wrap(err, "error writing active inferer label refcount")
		}
	}
	return nil
}

// GetLabelRefCount returns the current refcount for (topicId, nonce, label).
// Missing rows are reported as count=0.
func (k *TopicKeeper) GetLabelRefCount(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	label string,
) (uint64, error) {
	key := collections.Join3(topicId, nonce, label)
	curr, err := k.activeInfererLabelRefCount.Get(ctx, key)
	if err != nil {
		if errIsNotFound(err) {
			return 0, nil
		}
		return 0, errorsmod.Wrap(err, "error reading active inferer label refcount")
	}
	return curr, nil
}

// ClearLabelRefCountsForTopic removes every (topicId, *, *) entry in the
// refcount store. Called from ResetWorkersIndividualSubmissionsForTopic at
// the end of CloseWorkerNonce to wipe WSW state.
func (k *TopicKeeper) ClearLabelRefCountsForTopic(ctx context.Context, topicId types.TopicId) error {
	rng := collections.NewPrefixedTripleRange[types.TopicId, types.BlockHeight, string](topicId)
	if err := k.activeInfererLabelRefCount.Clear(ctx, rng); err != nil {
		return errorsmod.Wrap(err, "error clearing active inferer label refcount for topic")
	}
	return nil
}

// IterateLabelsForNonce walks every (topicId, nonce, label) -> count row.
// Iteration order is the collections triple key order: label bytes ascending.
func (k *TopicKeeper) IterateLabelsForNonce(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	cb func(label string, count uint64) (stop bool, err error),
) error {
	rng := collections.NewSuperPrefixedTripleRange[types.TopicId, types.BlockHeight, string](topicId, nonce)
	iter, err := k.activeInfererLabelRefCount.Iterate(ctx, rng)
	if err != nil {
		return errorsmod.Wrap(err, "error iterating active inferer label refcount")
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return errorsmod.Wrap(err, "error reading active inferer label refcount entry")
		}
		stop, err := cb(kv.Key.K3(), kv.Value)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	return nil
}

// BuildFinalEpochLabelRegistryFromActiveSet materializes the EpochLabelRegistry
// for (topicId, nonce) from the activeInfererLabelRefCount store. Labels with
// count > 0 are collected, lex-sorted deterministically, assigned ids 1..L,
// and the registry is persisted.
//
// SINGLE topics short-circuit to a 1-entry registry of the canonical label
// "y" so that close-time projection is uniform across arities.
//
// Returns types.ErrEpochLabelRegistryEmpty if a MULTI topic has no labels
// with positive refcount; the caller surfaces that as no-qualified-inferers
// (no registry is written in that case).
func (k *TopicKeeper) BuildFinalEpochLabelRegistryFromActiveSet(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
) (types.EpochLabelRegistry, error) {
	if err := types.ValidateTopicId(topic.Id); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "nonce block height validation failed")
	}

	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		registry := types.EpochLabelRegistry{
			TopicId: topic.Id,
			//nolint:gosec // nonce is non-negative (validated above)
			EpochId: uint64(nonce),
			Labels: []*types.TopicLabel{
				{Id: 1, Name: "y"},
			},
		}
		if err := k.topicLabelRegistry.Set(ctx, collections.Join(topic.Id, nonce), registry); err != nil {
			return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error setting topic label registry")
		}
		return registry, nil
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		// fall through
	default:
		return types.EpochLabelRegistry{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	labels := make([]string, 0)
	err := k.IterateLabelsForNonce(ctx, topic.Id, nonce, func(label string, count uint64) (bool, error) {
		if count > 0 && label != "" {
			labels = append(labels, label)
		}
		return false, nil
	})
	if err != nil {
		return types.EpochLabelRegistry{}, err
	}
	if len(labels) == 0 {
		return types.EpochLabelRegistry{}, types.ErrEpochLabelRegistryEmpty
	}
	sort.Strings(labels)

	entries := make([]*types.TopicLabel, len(labels))
	for i, name := range labels {
		//nolint:gosec // bounded by len(labels), which is itself capped by params.MaxLabelsPerSubmission
		entries[i] = &types.TopicLabel{Id: LabelId(i + 1), Name: name}
	}
	registry := types.EpochLabelRegistry{
		TopicId: topic.Id,
		//nolint:gosec // nonce is non-negative (validated above)
		EpochId: uint64(nonce),
		Labels:  entries,
	}
	if err := k.topicLabelRegistry.Set(ctx, collections.Join(topic.Id, nonce), registry); err != nil {
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error setting topic label registry")
	}
	return registry, nil
}

// GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose reads each
// worker's raw InputInference from workerLatestInputInferences and projects
// it into a committed types.Inference whose Values slice is aligned to the
// provided (already-frozen) EpochLabelRegistry.
//
// SINGLE topics produce a single-element Values slice from InputInference.Value
// (labeled value is also accepted if present and unique).
// MULTI topics scatter each labeled value into the index derived from the
// registry (ids are 1-based). Labels submitted by the worker that are not
// present in the registry are treated as errors (should be impossible if
// the caller built the registry from activeInfererLabelRefCount keyed by
// canonical labels).
func (k *WorkerKeeper) GetWorkersLatestInferencesByTopicIdValuesMaterializedAtClose(
	ctx context.Context,
	topic types.Topic,
	nonce types.BlockHeight,
	workers []ActorId,
	registry types.EpochLabelRegistry,
) (*types.Inferences, error) {
	active := make([]*types.Inference, 0, len(workers))
	switch topic.OutputArity {
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
		for _, addr := range workers {
			in, err := k.GetWorkerLatestInputInferenceByTopicId(ctx, topic.Id, addr)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "missing worker latest input inference for %s", addr)
			}
			var dec alloraMath.Dec
			switch {
			case len(in.Values) == 1:
				dec = in.Values[0].Value.ToDec()
			case len(in.Values) == 0:
				dec = in.Value.ToDec()
			default:
				return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "single-arity input inference has %d labeled values", len(in.Values))
			}
			active = append(active, &types.Inference{
				TopicId:     in.TopicId,
				BlockHeight: in.BlockHeight,
				Inferer:     in.Inferer,
				Values:      []alloraMath.Dec{dec},
				ExtraData:   in.ExtraData,
				Proof:       in.Proof,
			})
		}
	case types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
		L := len(registry.Labels)
		if L == 0 {
			return nil, types.ErrEpochLabelRegistryEmpty
		}
		idByLabel := make(map[string]int, L)
		for _, lbl := range registry.Labels {
			if lbl == nil {
				continue
			}
			idByLabel[lbl.Name] = int(lbl.Id) - 1
		}
		zero := alloraMath.ZeroDec()
		for _, addr := range workers {
			in, err := k.GetWorkerLatestInputInferenceByTopicId(ctx, topic.Id, addr)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "missing worker latest input inference for %s", addr)
			}
			values := make([]alloraMath.Dec, L)
			for i := range values {
				values[i] = zero
			}
			for _, lv := range in.Values {
				if lv == nil {
					continue
				}
				idx, ok := idByLabel[lv.Label]
				if !ok {
					return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "label %q from worker %s not in frozen registry", lv.Label, addr)
				}
				if idx < 0 || idx >= L {
					return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "label %q id out of range", lv.Label)
				}
				values[idx] = lv.Value.ToDec()
			}
			active = append(active, &types.Inference{
				TopicId:     in.TopicId,
				BlockHeight: in.BlockHeight,
				Inferer:     in.Inferer,
				Values:      values,
				ExtraData:   in.ExtraData,
				Proof:       in.Proof,
			})
		}
	default:
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "output_arity is invalid")
	}

	sort.Slice(active, func(i, j int) bool { return active[i].Inferer < active[j].Inferer })
	return &types.Inferences{Inferences: active}, nil
}

// errIsNotFound reports whether err wraps collections.ErrNotFound. Kept local
// so other keeper files don't have to re-import "errors".
func errIsNotFound(err error) bool {
	return err != nil && (err == collections.ErrNotFound || errorsmod.IsOf(err, collections.ErrNotFound))
}

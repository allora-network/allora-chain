package integration_test

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Payload admission replay. Insert events say whether a submission was admitted
// into the topic's active set, eviction events replace a stored payload with the
// evicting one, and retire events remove a pending submission. Replaying those
// events across blocks reconstructs the live payload set, which is the only
// thing the chain exposes as queries: stored payloads per actor and per nonce.
// The audit asserts membership and the stored values an event claims whenever
// the chain still holds the payload the block ended with.

// clearPendingForTopic drops the replayed worker submissions when the topic's
// worker close resets its active sets and stored inferences and forecasts.
// Retire events clear reputer submissions one actor at a time.
func (a *eventAudit) clearPendingForTopic(topicId uint64, kinds ...string) {
	for key := range a.pendingPayloads { //nolint:maprange // entries are deleted, order does not matter
		if key.topicId != topicId {
			continue
		}
		for _, kind := range kinds {
			if key.kind == kind {
				delete(a.pendingPayloads, key)
			}
		}
	}
}

// trackInsert records the outcome one insert event reports. NOT_ADMITTED leaves
// only score/liveness side effects, so the actor must not appear among stored
// payloads; ADMITTED with a prior live submission replaces it, and
// NO_ELEMENTS_ACCEPTED stores no forecast at all.
func (a *eventAudit) trackInsert(height int64, topicId uint64, kind, actor string, nonce int64, outcome emissionstypes.PayloadAdmissionOutcome) {
	key := pendingKey{topicId: topicId, kind: kind, actor: actor}
	switch outcome {
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED:
		if _, exists := a.pendingPayloads[key]; !exists {
			a.pendingPayloads[key] = &pendingPayload{}
		}
		a.pendingPayloads[key].nonce = nonce
		a.coverage["insert/"+kind+"/admitted"]++
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_NOT_ADMITTED:
		a.pendingPayloads[key] = &pendingPayload{notAdmitted: true, nonce: nonce}
		a.coverage["insert/"+kind+"/not_admitted"]++
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_NO_ELEMENTS_ACCEPTED:
		if kind != "forecaster" {
			a.failf(height, "insert %s topic %d actor %s reports NO_ELEMENTS_ACCEPTED", kind, topicId, actor)
		}
		a.pendingPayloads[key] = &pendingPayload{notAdmitted: true, nonce: nonce}
		a.coverage["insert/"+kind+"/no_elements_accepted"]++
	default:
		a.failf(height, "insert %s topic %d actor %s has admission outcome %s", kind, topicId, actor, outcome)
	}
}

// trackEviction applies an eviction to the replayed pending set: the evicted
// actor's stored payload disappears and the evicting actor must already have an
// admitted insert in the same window. A lost payload for an actor with no
// tracked insert means an event the audit did not see, not a violated rule.
func (a *eventAudit) trackEviction(height int64, kind string, topicId uint64, evicted, evicting string) {
	a.coverage["eviction/"+kind]++
	evictedKey := pendingKey{topicId: topicId, kind: kind, actor: evicted}
	if _, tracked := a.pendingPayloads[evictedKey]; !tracked {
		a.skipped["eviction payload content: evicted insert not seen"]++
	}
	delete(a.pendingPayloads, evictedKey)
	evictingKey := pendingKey{topicId: topicId, kind: kind, actor: evicting}
	if tracked, ok := a.pendingPayloads[evictingKey]; ok {
		if tracked.notAdmitted {
			a.failf(height, "%s eviction topic %d: evicting %s had a not-admitted insert", kind, topicId, evicting)
		}
	} else {
		a.skipped["eviction payload content: evicting insert not seen"]++
	}
}

// auditBlockPayloads checks the replayed pending set against the state the block
// committed. Not-admitted actors must have no stored payload. Admitted actors
// have a stored payload whose values the insert event already carried; those are
// checked at the close against the at-block queries, so a wrong or dropped
// insert surfaces where the payload is consumed.
func (a *eventAudit) auditBlockPayloads(block chainBlock) {
	for i, event := range block.events {
		if event.phase != phaseTx {
			continue
		}
		switch typed := event.typed.(type) {
		case *emissionstypes.EventInsertInfererPayload:
			a.checkInfererStored(block.height, typed, block.events[i+1:])
		case *emissionstypes.EventInsertReputerPayload:
			a.checkReputerStored(block.height, typed, block.events[i+1:])
		case *emissionstypes.EventReputerSubmissionRetired:
			a.checkReputerRetired(block.height, typed)
		}
	}
}

// sameBlockEvicted reports whether a later event in the same block evicts the
// actor's payload; the stored value then belongs to the evicting actor.
func sameBlockEvicted(events []chainEvent, kind, actor string, topicId uint64) bool {
	for _, event := range events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventInferenceEvictedFromActiveSet:
			if kind == "inferer" && typed.TopicId == topicId && typed.EvictedInferer == actor {
				return true
			}
		case *emissionstypes.EventReputerLossEvictedFromActiveSet:
			if kind == "reputer" && typed.TopicId == topicId && typed.EvictedReputer == actor {
				return true
			}
		case *emissionstypes.EventReputerSubmissionRetired:
			if kind == "reputer" && typed.TopicId == topicId && typed.Reputer == actor {
				return true
			}
		}
	}
	return false
}

func (a *eventAudit) checkInfererStored(height int64, event *emissionstypes.EventInsertInfererPayload, rest []chainEvent) {
	response, err := a.m.Client.QueryEmissions().GetWorkerLatestInputInferenceByTopicId(atHeight(a.ctx, height), &emissionstypes.GetWorkerLatestInputInferenceByTopicIdRequest{TopicId: event.TopicId, WorkerAddress: event.Inferer})
	stored := err == nil && response.LatestInputInference != nil
	notFound := status.Code(err) == codes.NotFound || (err != nil && strings.Contains(err.Error(), "not found"))
	if !stored && !notFound && err != nil {
		a.failf(height, "GetWorkerLatestInputInferenceByTopicId topic %d inferer %s: %v", event.TopicId, event.Inferer, err)
		return
	}
	if event.AdmissionOutcome == emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
		if !stored && !sameBlockEvicted(rest, "inferer", event.Inferer, event.TopicId) {
			a.failf(height, "admitted inferer %s topic %d has no stored inference at block end", event.Inferer, event.TopicId)
		}
		return
	}
	if stored {
		a.failf(height, "%s inferer %s topic %d has a stored inference at block end", event.AdmissionOutcome, event.Inferer, event.TopicId)
	}
}

func (a *eventAudit) checkReputerStored(height int64, event *emissionstypes.EventInsertReputerPayload, rest []chainEvent) {
	if event.Nonce == 0 {
		return
	}
	if sameBlockEvicted(rest, "reputer", event.Reputer, event.TopicId) {
		return
	}
	response, err := a.m.Client.QueryEmissions().GetReputerLossBundlesAtBlock(atHeight(a.ctx, height), &emissionstypes.GetReputerLossBundlesAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		// The close moved the live bundles into their at-block store only when
		// the nonce closed in this block; an open window has nothing at-block.
		return
	}
	for _, bundle := range response.LossBundles {
		if bundle == nil {
			continue
		}
		if bundle.GetReputer() == event.Reputer && event.AdmissionOutcome != emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
			a.failf(height, "%s reputer %s topic %d nonce %d has a stored loss bundle", event.AdmissionOutcome, event.Reputer, event.TopicId, event.Nonce)
		}
	}
}

// checkReputerRetired verifies the retirement removes the pending submission:
// a subsequent insert in the same block for the same reputer would re-admit it,
// which is exactly the state the retirement exists to unblock.
func (a *eventAudit) checkReputerRetired(height int64, event *emissionstypes.EventReputerSubmissionRetired) {
	// The retire path removes both the active marker and the stored bundle;
	// GetReputerLossBundlesAtBlock exposes only closed nonces, so the replayed
	// pending set asserts the removal and the next close asserts the result.
	if event.Nonce < 0 {
		a.failf(height, "EventReputerSubmissionRetired topic %d reputer %s has negative nonce %d", event.TopicId, event.Reputer, event.Nonce)
	}
}

// clearPendingForNonce drops replayed submissions stored for one nonce, used
// when the chain prunes that nonce and everything it held.
func (a *eventAudit) clearPendingForNonce(topicId uint64, nonce int64) {
	for key, entry := range a.pendingPayloads { //nolint:maprange // entries are deleted, order does not matter
		if key.topicId == topicId && entry.nonce == nonce {
			delete(a.pendingPayloads, key)
		}
	}
}

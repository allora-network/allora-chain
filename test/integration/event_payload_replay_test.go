package integration_test

import (
	"bytes"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Payload admission replay. Insert events say whether a submission was admitted
// into the topic's active set, eviction events replace a stored payload with the
// evicting one, and retire events remove a pending submission. Replaying those
// events across blocks reconstructs the live payload set, which the chain
// exposes through the at-block and latest-input queries. The audit asserts
// membership and stored values whenever the block's events identify the
// payload the chain committed.

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
				delete(a.priorAdmittedInferer, key)
			}
		}
	}
}

// clearPendingForNonce drops replayed submissions stored for one nonce, used
// when the chain prunes that nonce and everything it held or when its close
// fails and cleanup wipes the submissions.
func (a *eventAudit) clearPendingForNonce(topicId uint64, nonce int64) {
	for key, entry := range a.pendingPayloads { //nolint:maprange // entries are deleted, order does not matter
		if key.topicId == topicId && entry.nonce == nonce {
			delete(a.pendingPayloads, key)
			delete(a.priorAdmittedInferer, key)
		}
	}
}

// trackInsert records the outcome one insert event reports. NOT_ADMITTED leaves
// only score/liveness side effects and does not erase an admitted payload from
// an earlier submission in the same window. NO_ELEMENTS_ACCEPTED is the same
// but names the reason no forecast element survived filtering.
func (a *eventAudit) trackInsert(height int64, topicId uint64, kind, actor string, nonce int64, outcome emissionstypes.PayloadAdmissionOutcome) {
	key := pendingKey{topicId: topicId, kind: kind, actor: actor}
	switch outcome {
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED:
		if _, exists := a.pendingPayloads[key]; !exists {
			a.pendingPayloads[key] = &pendingPayload{notAdmitted: false, nonce: nonce}
		}
		a.pendingPayloads[key].nonce = nonce
		if kind == "inferer" {
			a.priorAdmittedInferer[key] = true
		}
		a.coverage["insert/"+kind+"/admitted"]++
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_NOT_ADMITTED:
		if _, exists := a.pendingPayloads[key]; !exists {
			a.pendingPayloads[key] = &pendingPayload{notAdmitted: true, nonce: nonce}
		}
		a.coverage["insert/"+kind+"/not_admitted"]++
	case emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_NO_ELEMENTS_ACCEPTED:
		if kind != "forecaster" {
			a.failf(height, "insert %s topic %d actor %s reports NO_ELEMENTS_ACCEPTED", kind, topicId, actor)
		}
		if _, exists := a.pendingPayloads[key]; !exists {
			a.pendingPayloads[key] = &pendingPayload{notAdmitted: true, nonce: nonce}
		}
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
	delete(a.priorAdmittedInferer, evictedKey)
	evictingKey := pendingKey{topicId: topicId, kind: kind, actor: evicting}
	if tracked, ok := a.pendingPayloads[evictingKey]; ok {
		if tracked.notAdmitted {
			a.failf(height, "%s eviction topic %d: evicting %s had a not-admitted insert", kind, topicId, evicting)
		}
	} else {
		a.skipped["eviction payload content: evicting insert not seen"]++
	}
}

// auditBlockPayloads checks each insert and retire event against the state the
// block committed, using the rest of the block's events to know whether the
// payload was legitimately removed (a close, prune, or eviction) before the
// state was finalized.
func (a *eventAudit) auditBlockPayloads(block chainBlock) {
	for i, event := range block.events {
		if event.phase != phaseTx {
			continue
		}
		switch typed := event.typed.(type) {
		case *emissionstypes.EventInsertInfererPayload:
			a.checkInfererStored(block.height, typed, block.events[i+1:])
		case *emissionstypes.EventInsertForecasterPayload:
			a.checkForecasterStored(block.height, typed)
		case *emissionstypes.EventInsertReputerPayload:
			a.checkReputerStored(block.height, typed, block.events[i+1:])
		case *emissionstypes.EventReputerSubmissionRetired:
			a.checkReputerRetired(block.height, typed)
		}
	}
}

// sameBlockCleared reports whether a later event in the same block removes the
// actor's payload by close, prune, eviction, or retirement. The stored value at
// block end then no longer belongs to this actor.
func sameBlockCleared(events []chainEvent, kind, actor string, topicId uint64) bool {
	for _, event := range events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventWorkerSubmissionWindowClosed:
			if typed.TopicId == topicId {
				return true
			}
		case *emissionstypes.EventNonceExpired:
			if typed.TopicId == topicId && (typed.Reason == emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_PRUNED_BY_AGE || typed.Reason == emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_CLOSE_FAILED) {
				return true
			}
		case *emissionstypes.EventInferenceEvictedFromActiveSet:
			if kind == "inferer" && typed.TopicId == topicId && typed.EvictedInferer == actor {
				return true
			}
		case *emissionstypes.EventForecastEvictedFromActiveSet:
			if kind == "forecaster" && typed.TopicId == topicId && typed.EvictedForecaster == actor {
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

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound || (err != nil && strings.Contains(err.Error(), "not found"))
}

func (a *eventAudit) checkInfererStored(height int64, event *emissionstypes.EventInsertInfererPayload, rest []chainEvent) {
	if sameBlockCleared(rest, "inferer", event.Inferer, event.TopicId) {
		return
	}
	response, err := a.m.Client.QueryEmissions().GetWorkerLatestInputInferenceByTopicId(atHeight(a.ctx, height), &emissionstypes.GetWorkerLatestInputInferenceByTopicIdRequest{TopicId: event.TopicId, WorkerAddress: event.Inferer})
	stored := err == nil && response.LatestInputInference != nil
	if !stored && !isNotFound(err) {
		a.failf(height, "GetWorkerLatestInputInferenceByTopicId topic %d inferer %s: %v", event.TopicId, event.Inferer, err)
		return
	}
	if event.AdmissionOutcome == emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
		if !stored {
			a.failf(height, "admitted inferer %s topic %d has no stored inference at block end", event.Inferer, event.TopicId)
			return
		}
		got := response.LatestInputInference
		if got.Inferer != event.Inferer || got.TopicId != event.TopicId || got.BlockHeight != event.Nonce {
			a.failf(height, "stored inference for %s topic %d has inferer %s topic %d nonce %d, event nonce %d", event.Inferer, event.TopicId, got.Inferer, got.TopicId, got.BlockHeight, event.Nonce)
		}
		if !bytes.Equal(got.ExtraData, event.ExtraData) {
			a.failf(height, "stored inference for %s topic %d extra_data differs from the insert event", event.Inferer, event.TopicId)
		}
		if !got.Value.Equal(event.Value) {
			a.failf(height, "stored inference for %s topic %d value %s, event value %s", event.Inferer, event.TopicId, got.Value, event.Value)
		}
		if len(got.Values) != len(event.Values) {
			a.failf(height, "stored inference for %s topic %d has %d labeled values, event has %d", event.Inferer, event.TopicId, len(got.Values), len(event.Values))
		}
		return
	}
	// A not-admitted submission must not create a stored inference, but an
	// admitted submission from earlier in the same window stays stored.
	if stored {
		key := pendingKey{topicId: event.TopicId, kind: "inferer", actor: event.Inferer}
		if !a.priorAdmittedInferer[key] {
			a.failf(height, "%s inferer %s topic %d has a stored inference at block end with no admitted submission in the window", event.AdmissionOutcome, event.Inferer, event.TopicId)
		}
	}
}

// checkForecasterStored checks a forecaster insert against the stored at-block
// forecast set. The chain stores forecasts per nonce, so the query returns the
// forecasters whose submissions the audit already watched close for that nonce;
// for an open nonce the response is empty. The close-time rule then compares
// the two lists against the accepted payloads since the previous close.
func (a *eventAudit) checkForecasterStored(height int64, event *emissionstypes.EventInsertForecasterPayload) {
	response, err := a.m.Client.QueryEmissions().GetForecastsAtBlock(atHeight(a.ctx, height), &emissionstypes.GetForecastsAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		a.failf(height, "GetForecastsAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	var stored *emissionstypes.Forecast
	for _, forecast := range response.GetForecasts().GetForecasts() {
		if forecast.Forecaster == event.Forecaster {
			stored = forecast
			break
		}
	}
	// Only a closed nonce returns the stored set. For an open window the close
	// rule audits the forecast this submission contributes to.
	if stored == nil {
		return
	}
	if event.AdmissionOutcome != emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
		a.failf(height, "%s forecaster %s topic %d nonce %d has a stored forecast at the closed nonce", event.AdmissionOutcome, event.Forecaster, event.TopicId, event.Nonce)
		return
	}
	if len(stored.ForecastElements) == 0 && len(event.InfererAddresses) != 0 {
		a.failf(height, "stored forecast of %s topic %d nonce %d is empty, event has %d elements", event.Forecaster, event.TopicId, event.Nonce, len(event.InfererAddresses))
	}
	if !bytes.Equal(stored.ExtraData, event.ExtraData) {
		a.failf(height, "stored forecast of %s topic %d nonce %d extra_data differs from the event", event.Forecaster, event.TopicId, event.Nonce)
	}
	storedInferers := make([]string, 0, len(stored.ForecastElements))
	for _, el := range stored.ForecastElements {
		storedInferers = append(storedInferers, el.Inferer)
	}
	if !equalSortedStrings(storedInferers, event.InfererAddresses) {
		a.failf(height, "stored forecast of %s topic %d nonce %d inferers %v, event inferers %v", event.Forecaster, event.TopicId, event.Nonce, storedInferers, event.InfererAddresses)
		return
	}
	byInferer := make(map[string]alloraMath.Dec, len(event.InfererAddresses))
	for i, inferer := range event.InfererAddresses {
		if i < len(event.InfererValues) {
			byInferer[inferer] = event.InfererValues[i]
		}
	}
	for _, el := range stored.ForecastElements {
		if expected, ok := byInferer[el.Inferer]; ok && !el.Value.Equal(expected) {
			a.failf(height, "stored forecast of %s topic %d nonce %d element %s is %s, event has %s", event.Forecaster, event.TopicId, event.Nonce, el.Inferer, el.Value, expected)
		}
	}
}

func (a *eventAudit) checkReputerStored(height int64, event *emissionstypes.EventInsertReputerPayload, rest []chainEvent) {
	if event.Nonce == 0 {
		return
	}
	if sameBlockCleared(rest, "reputer", event.Reputer, event.TopicId) {
		return
	}
	response, err := a.m.Client.QueryEmissions().GetReputerLossBundlesAtBlock(atHeight(a.ctx, height), &emissionstypes.GetReputerLossBundlesAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		if isNotFound(err) {
			return
		}
		a.failf(height, "GetReputerLossBundlesAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	// An empty store means the nonce is still open, so the pending bundle has
	// not been moved to the at-block store yet; its close will assert it.
	if len(response.LossBundles) == 0 {
		return
	}
	var stored *emissionstypes.ValueBundle
	for _, bundle := range response.LossBundles {
		if bundle != nil && bundle.GetReputer() == event.Reputer {
			stored = bundle
			break
		}
	}
	if stored == nil {
		if event.AdmissionOutcome == emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
			// An admitted bundle that was evicted before the close does not
			// reach the at-block store; the close-time active-set rule covers
			// which actors it reports.
			return
		}
		return
	}
	if event.AdmissionOutcome != emissionstypes.PayloadAdmissionOutcome_PAYLOAD_ADMISSION_OUTCOME_ADMITTED {
		a.failf(height, "%s reputer %s topic %d nonce %d has a stored loss bundle at the closed nonce", event.AdmissionOutcome, event.Reputer, event.TopicId, event.Nonce)
		return
	}
	if event.Bundle == nil {
		a.failf(height, "EventInsertReputerPayload topic %d reputer %s has no bundle to compare", event.TopicId, event.Reputer)
		return
	}
	a.requireDecEqual(height, "EventInsertReputerPayload combined value of "+event.Reputer, event.Bundle.CombinedValue, stored.CombinedValue)
	a.requireDecEqual(height, "EventInsertReputerPayload naive value of "+event.Reputer, event.Bundle.NaiveValue, stored.NaiveValue)
	if !bytes.Equal(stored.ExtraData, event.Bundle.ExtraData) {
		a.failf(height, "stored loss bundle of %s topic %d nonce %d extra_data differs from the event", event.Reputer, event.TopicId, event.Nonce)
	}
	storedInferers := make([]string, 0, len(stored.InfererValues))
	for _, v := range stored.InfererValues {
		storedInferers = append(storedInferers, v.Worker)
	}
	if !equalSortedStrings(storedInferers, event.Bundle.InfererAddresses) {
		a.failf(height, "stored loss bundle of %s topic %d nonce %d inferers %v, event inferers %v", event.Reputer, event.TopicId, event.Nonce, storedInferers, event.Bundle.InfererAddresses)
	}
	storedForecasters := make([]string, 0, len(stored.ForecasterValues))
	for _, v := range stored.ForecasterValues {
		storedForecasters = append(storedForecasters, v.Worker)
	}
	if !equalSortedStrings(storedForecasters, event.Bundle.ForecasterAddresses) {
		a.failf(height, "stored loss bundle of %s topic %d nonce %d forecasters %v, event forecasters %v", event.Reputer, event.TopicId, event.Nonce, storedForecasters, event.Bundle.ForecasterAddresses)
	}
}

// checkReputerRetired verifies the retirement is consistent with the replay:
// the reputer had a tracked pending submission, and after the event none
// remains. Submissions that started before the audit began are not tracked;
// a negative nonce is always wrong.
func (a *eventAudit) checkReputerRetired(height int64, event *emissionstypes.EventReputerSubmissionRetired) {
	if event.Nonce < 0 {
		a.failf(height, "EventReputerSubmissionRetired topic %d reputer %s has negative nonce %d", event.TopicId, event.Reputer, event.Nonce)
	}
	key := pendingKey{topicId: event.TopicId, kind: "reputer", actor: event.Reputer}
	if _, still := a.pendingPayloads[key]; still {
		a.failf(height, "EventReputerSubmissionRetired topic %d reputer %s left a tracked pending submission", event.TopicId, event.Reputer)
	}
}

// equalSortedStrings compares two address lists ignoring order.
func equalSortedStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

# Topic Parameters

Per-topic configuration stored on `emissions.v10.Topic`. Set at `CreateNewTopic`, and a subset can be changed later with `UpdateTopic`.

Mutability falls into three groups:

| Group | Meaning |
|---|---|
| **Immutable** | Fixed at creation. `UpdateTopic` cannot carry them, or rejects a change. |
| **Editable** | Changeable with `UpdateTopic` at any time. |
| **Editable (WSW-guarded)** | Changeable with `UpdateTopic`, but **rejected while a worker submission window is open** for any unfulfilled nonce of the topic. |

The WSW guard exists because those fields steer active-set admission or label semantics for an epoch that workers may already be submitting against. Changing them mid-window would race payload processing. The guard is in `TopicKeeper.UpdateTopic` (`x/emissions/keeper/topic.go`) and returns `ErrWorkerNonceWindowNotAvailable`, naming the offending fields.

> `UpdateTopic` is a **full replacement**, not a patch. Every editable field is overwritten by what the message carries — there is no "leave as-is". Send the current value back to keep it. This matters most for `label_whitelist` (an empty list means *unrestricted*, not *unchanged*) and `max_top_inferers_to_reward` (see below).

---

## Table

| Field | Type | Mutability | Notes |
|---|---|---|---|
| `id` | `uint64` | Immutable | Assigned by the chain. `0` is reserved. |
| `creator` | `string` | Immutable | Only this address may call `UpdateTopic`. |
| `metadata` | `string` | Editable | Free-form; bounded by `max_string_length`. |
| `loss_method` | `string` | Editable | Non-empty; bounded by `max_string_length`. |
| `epoch_length` | `int64` | Immutable | Blocks per epoch. `>= min_epoch_length`. |
| `epoch_last_ended` | `int64` | Chain-managed | Bookkeeping, not user-set. |
| `ground_truth_lag` | `int64` | Immutable | `>= epoch_length`, and `<= max_unfulfilled_reputer_requests * epoch_length`. |
| `worker_submission_window` | `int64` | Immutable | `> 0` and `<= epoch_length`. Defines the WSW referenced throughout this page. |
| `p_norm` | `Dec` | Editable | In `[1, 10]`. |
| `alpha_regret` | `Dec` | Editable | In `(0, 1]`. |
| `c_norm` | `Dec` | Editable | In `[-100, 100]`. Per-topic; the global `c_norm` param is deprecated and ignored. |
| `epsilon` | `Dec` | Immutable | `> 0`. |
| `allow_negative` | `bool` | Immutable | Both values valid; no validation. |
| `initial_regret` | `Dec` | Chain-managed | Recomputed by the chain. |
| `merit_sortition_alpha` | `Dec` | **Editable (WSW-guarded)** | In `[0, 1)`. |
| `active_inferer_quantile` | `Dec` | Immutable | In `[0, 1]`. |
| `active_forecaster_quantile` | `Dec` | Immutable | In `[0, 1]`. |
| `active_reputer_quantile` | `Dec` | Immutable | In `[0, 1]`. |
| `topic_type` | `TopicType` | Immutable | `REGRESSION` or `CLASSIFICATION`. |
| `output_arity` | `TopicOutputArity` | Immutable | `SINGLE` or `MULTI`. |
| `require_unity` | `bool` | Immutable | Only meaningful for `MULTI`. |
| `unity_tolerance` | `Dec` | Immutable | Only meaningful when `require_unity`. |
| `label_case_sensitive` | `bool` | Immutable | Explicitly rejected on change — see below. |
| `max_labels_per_submission` | `uint64` | **Editable (WSW-guarded)** | In `[1, 1024]`. |
| `label_whitelist` | `[]string` | **Editable (WSW-guarded)** | Empty means unrestricted. Canonicalized before storage. |
| `label_default_value` | `Dec` | **Editable (WSW-guarded)** | Any finite `Dec`. |
| `max_top_inferers_to_reward` | `uint64` | **Editable (WSW-guarded)** | `0` means "use the global maximum". See below. |

---

## Details

### Identity and ownership

**`id`** — assigned sequentially by the chain. Topic id `0` is reserved and rejected by `ValidateTopicId`.

**`creator`** — bech32 address of the topic creator. `UpdateTopic` rejects any other sender with `ErrUnauthorized`.

### Freely editable

**`metadata`** — free-form description. Only length-bounded, by the global `max_string_length`.

**`loss_method`** — identifier of the loss function used for reputer scoring. Must be non-empty and within `max_string_length`.

**`p_norm`**, **`alpha_regret`**, **`c_norm`** — reward-math parameters. Editable at any time; only their own ranges are checked. `c_norm` is per-topic; the global `c_norm` param still exists in `Params` but is deprecated and ignored by chain logic.

### Scheduling (immutable)

**`epoch_length`**, **`ground_truth_lag`**, **`worker_submission_window`** — fixed at creation because they define the topic's cadence and the shape of every nonce window. Changing them would invalidate in-flight nonces.

The cross-field rules are enforced by `Topic.Validate`: `worker_submission_window <= epoch_length`, `ground_truth_lag >= epoch_length`, and `ground_truth_lag <= max_unfulfilled_reputer_requests * epoch_length`.

### Classification (immutable)

**`topic_type`**, **`output_arity`**, **`require_unity`**, **`unity_tolerance`** — absent from `UpdateTopicRequest` entirely, so they cannot be changed. Their mutual consistency is enforced by `ValidateClassificationConsistency`.

**`label_case_sensitive`** is a special case: it *is* present on the stored topic but is explicitly rejected on change by the keeper, with `ErrInvalidTopicUpdate`. Flipping it would change the canonical form of every already-persisted label — whitelist entries and epoch registry names alike — silently orphaning them.

### WSW-guarded fields

All four below are rejected while a worker submission window is open for **any** unfulfilled nonce, not just the newest one, because workers may submit against any currently-open nonce.

**`merit_sortition_alpha`** — EMA weight for previous scores when filtering the active set. In `[0, 1)`. Guarded for reward-math continuity.

**`max_labels_per_submission`** — per-topic cap on distinct canonical labels in a single payload. In `[1, 1024]` (`MinMaxLabelsPerSubmission` / `MaxMaxLabelsPerSubmission`).

**`label_whitelist`** — optional allowlist of canonical label names. **Empty means unrestricted**, so under full-replacement semantics, omitting the list clears an existing restriction rather than preserving it. Bounded by the global `max_topic_label_whitelist_size`, and canonicalized (NFC-normalized, trimmed, optionally lowercased per `label_case_sensitive`) before both the change comparison and storage.

**`label_default_value`** — value used for missing slots in dense MULTI inference vectors. Submitted labels equal to this value are treated as absent for epoch label registry inclusion.

### `max_top_inferers_to_reward`

Per-topic cap on how many inferers are admitted to the active inference set. At worker-payload insert time an inferer is admitted while the active set is below the cap; otherwise it must out-score the lowest current member.

**`0` is a sentinel meaning "use the global maximum".** It is resolved at write time, so the topic stores a concrete number, not the sentinel:

```
requested 0  ->  stored as the current global max_top_inferers_to_reward
```

This pins the topic to the global value **as it was when the topic was written**. A later governance change to the global does not retroactively move it. To re-sync a topic to a new global, send `0` again.

**Accepted range.** A non-zero request must fall within the global `[min_top_inferers_to_reward, max_top_inferers_to_reward]`:

| Request | Result |
|---|---|
| `0` | Accepted → stored as the global maximum |
| `< min_top_inferers_to_reward` | Rejected, `ErrTopicMaxTopInferersToRewardTooSmall` |
| within range | Accepted, stored verbatim |
| `> max_top_inferers_to_reward` | Rejected, `ErrTopicMaxTopInferersToRewardTooBig` |

**Consequence worth knowing.** Because the floor applies to *requests*, raising the global floor above a topic's stored cap makes that stored value un-resubmittable. Such a topic can still be updated — any value in the new `[min, max]`, or `0` — but it **cannot keep its old cap**. Two knock-on effects:

- Read-modify-write clients break: fetching the topic and resubmitting it verbatim now fails, because the fetched cap is below the new floor.
- Since the cap is forced to change, the update always trips the WSW guard, so such a topic can only be edited **between** submission windows.

Admission is separately defensive: it re-resolves the stored value through the global range on every payload, so a value below the floor (reachable only via a hand-written genesis) is raised to the floor, and the global maximum is applied last and always wins. The cap can therefore never push the active set past the global-sized score-retention window.

`Topic.Validate` deliberately does **not** bound this field against the globals. The globals bound *admission*, not storage, so a stored value outside the range stays loadable rather than making the topic unreadable after a governance change.

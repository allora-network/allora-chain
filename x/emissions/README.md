Emissions Module
=============================================

The `emissions` module is a core component of the Allora Network that manages the topic definition, actor participation, economic incentives, rewards system and more. This is the main part of the implementation of the [Allora Whitepaper](https://research.assets.allora.network/allora.0x10001.pdf). 

Key Features:
- Topic Management: Creation and management of prediction topics
- Stake Management: Handling of stakes for workers and reputers
- Reward Distribution: Calculation and distribution of rewards based on performance
- Delegation System: Support for stake delegation to reputers
- Performance Metrics: Tracking of worker, reputer, and forecaster scores
- Fee Collection: Management of network fees and revenue distribution

## Queries

### Nonce Submission Windows

The module provides queries to check which nonces are currently open for submission:

- **`GetOpenReputerSubmissionWindows`**: Returns only the reputer nonces that are currently open for submission. A nonce is considered open if the current block height is within its submission window (from when ground truth is revealed until the window closes).
  
  - Endpoint: `/emissions/v9/open_reputer_submission_windows/{topic_id}`
  - CLI: `allorad query emissions open-reputer-submission-windows [topic_id]`
  
  This query filters the results from `GetUnfulfilledReputerNonces` to only include nonces that are currently within their submission window. Typically, only one nonce will be open at a time.

- **`GetOpenWorkerSubmissionWindows`**: Returns only the worker nonces that are currently open for submission. A nonce is considered open if the current block height is within its submission window (from the nonce block height until the window closes).
  
  - Endpoint: `/emissions/v9/open_worker_submission_windows/{topic_id}`
  - CLI: `allorad query emissions open-worker-submission-windows [topic_id]`
  
  This query filters the results from `GetUnfulfilledWorkerNonces` to only include nonces that are currently within their submission window. Typically, only one nonce will be open at a time.

**Note**: These queries are more specific than `GetUnfulfilledReputerNonces` and `GetUnfulfilledWorkerNonces`, which return all unfulfilled nonces regardless of whether they are currently open for submission.

## Monitoring

Allora node emits its own `emissions` module metrics in each event, query and tx.
event: `allora_emissions_event_total`
query/tx: `allora_emissions_request_total` for occurrences, `allora_emissions_request_duration_ms` for latency measures.
Different labels are applied where appropriate (eg "topic_id", "address", "nonce", etc.)
See `x/emissions/metrics/` for details.

## Event value representation

`Dec` values placed in emissions events are magnitude-clamped to `[1e-40, 1e40]` so extremely small or large values stay compact when serialized. This shapes event output only; consensus state is stored separately and is not clamped, so an event value may differ from the stored value at the extremes. See `ClampMagnitude` and the `clamp*` helpers in `x/emissions/types/events_utils.go`.

## Parameters

- **[Global parameters](docs/global-parameters.md)** — chain-wide parameters on `Params`: their defaults, and the ones whose behavior is not obvious from the name.
- **[Topic parameters](docs/topic-parameters.md)** — per-topic configuration on `Topic`: which fields are immutable, which are editable, and which are editable only outside an open worker submission window.

## Topic configuration updates (UpdateTopic)

The `UpdateTopic` tx is intentionally limited to a subset of mutable fields. Topic creators can update:

- **`metadata`**, **`loss_method`**, **`alpha_regret`**, **`p_norm`**, **`c_norm`** — at any time.
- **`merit_sortition_alpha`**, **`max_labels_per_submission`**, **`label_whitelist`**, **`label_default_value`**, **`max_top_inferers_to_reward`** — only while no worker submission window is open.

All other topic fields are immutable because they impact state transitions, scheduling/cadence, or other invariants that should remain stable once a topic is created.

`UpdateTopic` is a **full replacement**: every editable field is overwritten by what the message carries, so send the current value back to keep it. An empty `label_whitelist` means *unrestricted*, not *unchanged*, and `max_top_inferers_to_reward` of `0` means *use the global maximum*, not *unchanged*.

### Constraints

- The five WSW-guarded fields above are rejected with `ErrWorkerNonceWindowNotAvailable` while the topic is active *and* any unfulfilled worker nonce is inside its submission window — every open nonce is checked, not just the newest.
- **`label_case_sensitive`** is stored on the topic but explicitly rejected on change: flipping it would change the canonical form of every already-persisted label.

See [Topic parameters](docs/topic-parameters.md) for the full field-by-field reference.

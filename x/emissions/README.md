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

`Dec` values placed in emissions events are magnitude-clamped to `[1e-40, 1e40]` so extremely small or large values stay compact when serialized. This shapes event and query output only; consensus state is stored separately and is not clamped, so an event value may differ from the stored value at the extremes. See `ClampMagnitude` and the `clamp*` helpers in `x/emissions/types/events_utils.go`.

## Topic configuration updates (UpdateTopic)

The `UpdateTopic` tx is intentionally limited to a small set of mutable fields. Today, topic creators can update:

- **`metadata`**
- **`loss_method`**
- **`alpha_regret`**
- **`merit_sortition_alpha`**
- **`p_norm`**

All other topic fields are treated as immutable because they impact state transitions, scheduling/cadence, or other invariants that should remain stable once a topic is created.

### Constraints

- **`merit_sortition_alpha`** cannot be updated while the topic is active *and* the worker submission window is currently open (to avoid changing selection dynamics mid-window).

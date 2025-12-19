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


## Monitoring

Allora node emits its own `emissions` module metrics in each event, query and tx.
event: `allora_emissions_event_total`
query/tx: `allora_emissions_request_total` for occurrences, `allora_emissions_request_duration_ms` for latency measures.
Different labels are applied where appropriate (eg "topic_id", "address", "nonce", etc.)
See `x/emissions/metrics/` for details.

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

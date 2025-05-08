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
event: `allora_loadtest_produce_count`
query/tx: `allora_request_counter` for occurrences, `allora_request_latency_ms` for latency measures.
Different labels are applied where appropriate (eg "topic_id", "address", "nonce", etc.)
See `x/emissions/metrics/` for details.

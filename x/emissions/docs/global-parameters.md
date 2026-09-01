# Global Emissions Parameters

Chain-wide parameters on `emissions.v9.Params`. They apply to every topic and are changed only by an `UpdateParams` tx from a whitelisted admin.

```bash
allorad tx emissions update-params [sender] [params]
```

`UpdateParams` takes an `OptionalParams` message in which every field is a **repeated** scalar used as an optional: send a one-element list to change a field, omit it (or send an empty list) to leave it alone. All supplied fields are applied first, then `Params.Validate()` runs **once** over the merged result — so a change that is only valid alongside another change must travel in the same transaction.

Defaults live in `DefaultParams()` (`x/emissions/types/params.go`); validation lives alongside it in the same file.

---

## Table

Defaults are the module defaults, not necessarily what any given chain is running.

| Parameter | Type | Default | Purpose |
|---|---|---|---|
| `version` | `string` | `v10` | Protocol version string, in lockstep with the release tag. |
| `max_serialized_msg_length` | `int64` | `1000000` | Max bytes for msg/query server payloads. |
| `min_topic_weight` | `Dec` | `100` | Below this weight a topic does not run inference solicitation or loss update. |
| `required_minimum_stake` | `Int` | `10000` | Minimum stake to act as worker or reputer. |
| `remove_stake_delay_window` | `int64` | `604800` | Blocks to wait before a stake withdrawal finalizes (~3 weeks). |
| `min_epoch_length` | `int64` | `12` | Shortest epoch cadence a topic may set. |
| `beta_entropy` | `Dec` | `0.25` | Resilience of reward payouts against copycat workers. |
| `learning_rate` | `Dec` | `0.05` | Gradient descent step size. |
| `gradient_descent_max_iters` | `uint64` | `10` | Max gradient descent iterations. |
| `max_gradient_threshold` | `Dec` | `0.001` | Gradient descent stops below this. |
| `min_stake_fraction` | `Dec` | `0.5` | Minimum stake fraction listened to when setting consensus listening coefficients. |
| `epsilon_reputer` | `Dec` | `0.01` | Tolerance capping reputer scores at close proximities. |
| `epsilon_safe_div` | `Dec` | `0.0000001` | Tolerance capping division by zero. |
| `max_unfulfilled_worker_requests` | `uint64` | `100` | Outstanding worker nonces tracked per topic. |
| `max_unfulfilled_reputer_requests` | `uint64` | `100` | Outstanding reputer nonces per topic. Also bounds a topic's `ground_truth_lag`. |
| `topic_reward_stake_importance` | `Dec` | `0.5` | Exponent μ: weight of stake in topic reward. |
| `topic_reward_fee_revenue_importance` | `Dec` | `0.5` | Exponent ν: weight of fee revenue in topic reward. |
| `topic_reward_alpha` | `Dec` | `0.5` | EMA parameter for topic reward. |
| `task_reward_alpha` | `Dec` | `0.1` | EMA parameter for task reward. |
| `validators_vs_allora_percent_reward` | `Dec` | `0.25` | Share of supply to validators; remainder to Allora actors. |
| `max_samples_to_scale_scores` | `uint64` | `10` | Scores retained for standard-deviation scaling. **See below.** |
| `min_top_inferers_to_reward` | `uint64` | `5` | Floor for the per-topic inferer admission cap. **See below.** |
| `max_top_inferers_to_reward` | `uint64` | `32` | Ceiling for the per-topic inferer admission cap. **See below.** |
| `max_top_forecasters_to_reward` | `uint64` | `6` | Top forecasters by score rewarded per topic. |
| `max_top_reputers_to_reward` | `uint64` | `6` | Top reputers by score rewarded per topic. |
| `max_elements_per_forecast` | `uint64` | `12` | Top forecast elements by score. |
| `create_topic_fee` | `Int` | `75000` | Topic registration fee. |
| `registration_fee` | `Int` | `200` | Worker/reputer registration fee per topic. |
| `data_sending_fee` | `Int` | `10` | Payload submission fee. |
| `default_page_limit` | `uint64` | `100` | Default pagination size. |
| `max_page_limit` | `uint64` | `1000` | Max pagination size. |
| `min_epoch_length_record_limit` | `int64` | `3` | Minimum epochs of records retained per topic. |
| `blocks_per_month` | `uint64` | `864000` | Assumed blocks per month for emission rate (~3s blocks). |
| `p_reward_inference` | `Dec` | `3` | Reward-curve fiducial for inference. |
| `p_reward_forecast` | `Dec` | `3` | Reward-curve fiducial for forecast. |
| `p_reward_reputer` | `Dec` | `1` | Reward-curve fiducial for reputer. |
| `c_reward_inference` | `Dec` | `0.75` | Reward-curve fiducial for inference. |
| `c_reward_forecast` | `Dec` | `0.75` | Reward-curve fiducial for forecast. |
| `c_norm` | `Dec` | `0.75` | **Deprecated** — now per-topic (`Topic.c_norm`); ignored by chain logic. |
| `half_max_process_stake_removals_end_block` | `uint64` | `40` | Stake removals per end block; applied twice, so the effective max is double. |
| `max_active_topics_per_block` | `uint64` | `1` | Active topics churned per block. |
| `max_string_length` | `uint64` | `255` | Max length of chain-uploaded strings. |
| `initial_regret_quantile` | `Dec` | `0.25` | Quantile for initial regret during network regret calculation. |
| `p_norm_safe_div` | `Dec` | `8.25` | Divisor used to derive the offset from `c_norm`. |
| `min_experienced_worker_regrets` | `uint64` | `10` | Experienced workers needed before their regrets set a topic's initial regret. |
| `inference_outlier_detection_threshold` | `Dec` | `11` | Outlier detection threshold. |
| `inference_outlier_detection_alpha` | `Dec` | `0.2` | Outlier detection EMA weight. |
| `lambda_initial_score` | `Dec` | `2` | New-participant score initialization. |
| `global_whitelist_enabled` | `bool` | `true` | Global whitelist gates topic creation and participation. |
| `topic_creator_whitelist_enabled` | `bool` | `true` | Only whitelisted actors may create topics. |
| `global_worker_whitelist_enabled` | `bool` | `true` | Global worker whitelist governs worker participation. |
| `global_reputer_whitelist_enabled` | `bool` | `true` | Global reputer whitelist governs reputer participation. |
| `global_admin_whitelist_appended` | `bool` | `true` | Global admins are appended to whitelists. |
| `max_whitelist_input_array_length` | `uint64` | `2000` | Max array length for whitelist operations. |
| `min_weight_threshold_for_stdnorm` | `Dec` | `0.000001` | Retained for compatibility; currently unused. |
| `max_canonical_label_byte_length` | `uint64` | `64` | Max bytes of a canonical label name after NFC + trim. |
| `max_topic_label_whitelist_size` | `uint64` | `256` | Max canonical labels in a topic label whitelist. |
| `max_epoch_label_registry_size` | `uint64` | `32768` | Max labels in one `(topic, nonce)` epoch label registry. |

---

## Details

Most parameters above are self-explanatory from their purpose column. The ones below have consequences that are not obvious from the name alone.

### `min_top_inferers_to_reward` and `max_top_inferers_to_reward`

Despite the naming symmetry with `max_top_forecasters_to_reward` and `max_top_reputers_to_reward`, this **pair** does not simply set how many inferers get rewarded. They are the **bounds on the per-topic admission cap** `Topic.max_top_inferers_to_reward`, and they behave asymmetrically.

**They bound admission, not storage.** `Topic.Validate` does not check the per-topic cap against them, so a stored cap may legitimately sit outside the range — the bounds are applied when the value is *used*, not when it is written. This keeps a governance change from making existing topics unloadable.

**Three different things happen depending on the entry point:**

| Path | Behavior |
|---|---|
| `CreateNewTopic` / `UpdateTopic` | **Rejects** a non-zero request outside `[min, max]`. `0` means "use `max`" and is resolved to a concrete stored value. |
| Genesis import | Stores whatever is supplied, **unvalidated** against the range. |
| Worker payload admission | **Clamps** the stored value into `[min, max]`, with `max` applied last. |

So a request below the floor is an error, but a *stored* value below the floor is silently raised at admission. That asymmetry is intentional: rejecting bad input is useful, but a governance floor raise must not brick topics that were valid when they were written.

**The ceiling is applied last and always wins**, so even an inverted range (which validation prevents, but defence in depth applies) can never push the active set above `max_top_inferers_to_reward`. That invariant matters because the score-retention window is sized as `max_samples_to_scale_scores * max_top_inferers_to_reward` — an active set larger than the ceiling would overflow it.

**Validation couples the two.** `Params.Validate` rejects `min > max`. Since all fields are applied before validation runs, lowering the ceiling below the current floor requires lowering the floor **in the same transaction** — not in a prior one.

**`max` may not be zero**; `min` may. A zero ceiling would admit nobody and would zero the score-retention window. A zero floor simply means "no floor".

**Raising the floor has a delayed cost.** Topics whose stored cap falls below the new floor keep working — admission raises them — but their owners can no longer resubmit that cap, so any `UpdateTopic` forces the cap to change, which in turn trips the worker-submission-window guard. See [Topic Parameters](topic-parameters.md#max_top_inferers_to_reward).

### `max_samples_to_scale_scores`

Sizes the score-retention window jointly with `max_top_inferers_to_reward` (`max_samples_to_scale_scores * max_top_inferers_to_reward`). Raising either widens per-topic score storage multiplicatively, so they are best considered together rather than in isolation.

### `max_unfulfilled_reputer_requests`

Beyond bounding tracked nonces, this participates in topic validation: a topic's `ground_truth_lag` may not exceed `max_unfulfilled_reputer_requests * epoch_length`. Lowering it does not retroactively invalidate stored topics, but it does constrain what new topics may set.

### `c_norm` (deprecated)

Superseded by the per-topic `Topic.c_norm` and ignored by chain logic. It remains in `Params` for wire compatibility with existing clients, and is still returned by param queries. Do not use it to reason about behavior.

### `blocks_per_month`

An *assumption*, not a measurement — it drives the emission rate. If real block time drifts from the assumed ~3s, actual monthly emissions drift with it. Changing it changes the emission schedule.

### Whitelist flags

The five whitelist booleans compose rather than override one another: `global_whitelist_enabled` gates broadly, while `topic_creator_whitelist_enabled`, `global_worker_whitelist_enabled` and `global_reputer_whitelist_enabled` gate specific roles, and `global_admin_whitelist_appended` adds global admins on top. Reason about the combination, not any single flag.

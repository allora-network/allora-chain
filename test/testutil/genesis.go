package testutil

func MintGenesis() []byte {
	return []byte(`
    {
      "params": {
        "mint_denom": "uallo",
        "max_supply": "1000000000000000000000000000",
        "f_emission": "0.025000000000000000",
        "one_month_smoothing_degree": "0.100000000000000000",
        "ecosystem_treasury_percent_of_total_supply": "0.359500000000000000",
        "foundation_treasury_percent_of_total_supply": "0.100000000000000000",
        "participants_percent_of_total_supply": "0.055000000000000000",
        "investors_percent_of_total_supply": "0.310500000000000000",
        "team_percent_of_total_supply": "0.175000000000000000",
        "maximum_monthly_percentage_yield": "0.009500000000000000"
      },
      "previous_reward_emission_per_unit_staked_token": "0.000000000000000000",
      "previous_block_emission": "0",
      "ecosystem_tokens_minted": "0"
    },
	`)
}

func BankGenesis() []byte {
	return []byte(`
	{
      "params": {
        "send_enabled": [],
        "default_send_enabled": true
      },
      "balances": [
        {
          "address": "allo1qq3asnzpw0gdzsmshufv9p28znuf6ecdyp90ry",
          "coins": [
            {
              "denom": "uallo",
              "amount": "310500000000000000000000000"
            }
          ]
        },
        {
          "address": "allo1rrzsuakwtwactszf9h23dh88hh5cw9n9whyj9r",
          "coins": [
            {
              "denom": "uallo",
              "amount": "100000000000000000000000000"
            }
          ]
        },
        {
          "address": "allo1yxd0jjc423065vcy6w6eeptyes4rlwtvle94qs",
          "coins": [
            {
              "denom": "uallo",
              "amount": "33333300000000000000000000"
            }
          ]
        },
        {
          "address": "allo1xkfcz9rmgn82vye3l39k4magghn5lj97umlt0k",
          "coins": [
            {
              "denom": "uallo",
              "amount": "99000000000000000000"
            }
          ]
        },
        {
          "address": "allo1gwc2eq50dtcde43r2fhvlt6njp0s8yaz6j4vwy",
          "coins": [
            {
              "denom": "uallo",
              "amount": "1000000000000000000"
            }
          ]
        },
        {
          "address": "allo1fszyk5spq9u2feyvftxnxekyt87fcd02jhky4j",
          "coins": [
            {
              "denom": "uallo",
              "amount": "33333300000000000000000000"
            }
          ]
        },
        {
          "address": "allo16kymud82y5hqgflnsf3el20wt5wk9e5t4kg4pn",
          "coins": [
            {
              "denom": "uallo",
              "amount": "175000000000000000000000000"
            }
          ]
        },
        {
          "address": "allo1mrg2e7zf5v8ushgyeldhcfazpyjmkvkzy3sqqf",
          "coins": [
            {
              "denom": "uallo",
              "amount": "33333300000000000000000000"
            }
          ]
        }
      ],
      "supply": [
        {
          "denom": "uallo",
          "amount": "685500000000000000000000000"
        }
      ],
      "denom_metadata": [],
      "send_enabled": []
    }`)
}

// todo put this back when PR 469 has been merged
// "half_max_process_stake_removals_end_block": "40"
func EmissionsGenesis() []byte {
	return []byte(`
	{
      "params": {
        "version": "0.0.3",
        "max_serialized_msg_length": "1000000",
        "min_topic_weight": "100",
        "max_topics_per_block": "128",
        "required_minimum_stake": "100",
        "remove_stake_delay_window": "5",
        "min_epoch_length": "12",
        "beta_entropy": "0.25",
        "learning_rate": "0.05",
        "max_gradient_threshold": "0.001",
        "min_stake_fraction": "0.5",
        "max_unfulfilled_worker_requests": "100",
        "max_unfulfilled_reputer_requests": "100",
        "topic_reward_stake_importance": "0.5",
        "topic_reward_fee_revenue_importance": "0.5",
        "topic_reward_alpha": "0.5",
        "task_reward_alpha": "0.1",
        "validators_vs_allora_percent_reward": "0.25",
        "max_samples_to_scale_scores": "10",
        "max_top_inferers_to_reward": "48",
        "max_top_forecasters_to_reward": "6",
        "max_top_reputers_to_reward": "12",
        "create_topic_fee": "10",
        "gradient_descent_max_iters": "10",
        "max_retries_to_fulfil_nonces_worker": "1",
        "max_retries_to_fulfil_nonces_reputer": "3",
        "registration_fee": "10",
        "default_page_limit": "100",
        "max_page_limit": "1000",
        "min_epoch_length_record_limit": "3",
        "blocks_per_month": "525960",
        "p_reward_inference": "1",
        "p_reward_forecast": "3",
        "p_reward_reputer": "3",
        "c_reward_inference": "0.75",
        "c_reward_forecast": "0.75",
        "c_norm": "0.75",
        "topic_fee_revenue_decay_rate": "0.0025",
        "epsilon_reputer": "0.01",
        "min_effective_topic_revenue": "0.00000001"
      },
      "nextTopicId": "0",
      "topics": [],
      "activeTopics": [],
      "churnableTopics": [],
      "rewardableTopics": [],
      "topicWorkers": [],
      "topicReputers": [],
      "topicRewardNonce": [],
      "infererScoresByBlock": [],
      "forecasterScoresByBlock": [],
      "reputerScoresByBlock": [],
      "latestInfererScoresByWorker": [],
      "latestForecasterScoresByWorker": [],
      "latestReputerScoresByReputer": [],
      "reputerListeningCoefficient": [],
      "previousReputerRewardFraction": [],
      "previousInferenceRewardFraction": [],
      "previousForecastRewardFraction": [],
      "totalStake": "0",
      "topicStake": [],
      "stakeReputerAuthority": [],
      "stakeSumFromDelegator": [],
      "delegatedStakes": [],
      "stakeFromDelegatorsUponReputer": [],
      "delegateRewardPerShare": [],
      "stakeRemovalsByBlock": [],
      "stakeRemovalsByActor": [],
      "delegateStakeRemovalsByBlock": [],
      "delegateStakeRemovalsByActor": [],
      "inferences": [],
      "forecasts": [],
      "workers": [],
      "reputers": [],
      "topicFeeRevenue": [],
      "previousTopicWeight": [],
      "allInferences": [],
      "allForecasts": [],
      "allLossBundles": [],
      "networkLossBundles": [],
      "previousPercentageRewardToStakedReputers": "0",
      "unfulfilledWorkerNonces": [],
      "unfulfilledReputerNonces": [],
      "latestInfererNetworkRegrets": [],
      "latestForecasterNetworkRegrets": [],
      "latestOneInForecasterNetworkRegrets": [],
      "latestOneInForecasterSelfNetworkRegrets": [],
      "core_team_addresses": [
        "allo16270t36amc3y6wk2wqupg6gvg26x6dc2nr5xwl",
        "allo1xm0jg40dcvccqvzqwv5skxlpc7t6eku69kfz6y",
        "allo1g4y6ra95z2zewupm7p45z4ny00rs7m24rj5hn8",
        "allo10w0jcq50ufsuy9332dkz6zf4gu00xm9zhfyn3s",
        "allo1lvymnmzndmam00uvxq8hr63jq8jfrups4ymlg2",
        "allo1d7vr2dxahkcz0snk28pets9uqvyxjdlysst3z3",
        "allo19gtttc7qg50n3hjn0qxdasdudf260cx7vevk8j",
        "allo1jc2mme2fj458kg08v2z92m8f9vsqwfzt0ju9ys",
        "allo1uff55lgqpjkw2mlsx2q0p8q8z7k7p00w9s4s0f",
        "allo136eeqhawxx66sjgsfeqk9gewq0e0msyu5tjmj3",
        "allo1gwc2eq50dtcde43r2fhvlt6njp0s8yaz6j4vwy",
        "allo1gwc2eq50dtcde43r2fhvlt6njp0s8yaz6j4vwy"
      ],
      "topicLastWorkerCommit": [],
      "topicLastReputerCommit": [],
      "topicLastWorkerPayload": [],
      "topicLastReputerPayload": []
    },
	`)
}

//     authGenesis := []byte(`
// {
//      "params": {
//        "max_memo_characters": "256",
//        "tx_sig_limit": "7",
//        "tx_size_cost_per_byte": "10",
//        "sig_verify_cost_ed25519": "590",
//        "sig_verify_cost_secp256k1": "1000"
//      },
//      "accounts": [
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1mrg2e7zf5v8ushgyeldhcfazpyjmkvkzy3sqqf",
//          "pub_key": null,
//          "account_number": "0",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1fszyk5spq9u2feyvftxnxekyt87fcd02jhky4j",
//          "pub_key": null,
//          "account_number": "1",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1yxd0jjc423065vcy6w6eeptyes4rlwtvle94qs",
//          "pub_key": null,
//          "account_number": "2",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1xkfcz9rmgn82vye3l39k4magghn5lj97umlt0k",
//          "pub_key": null,
//          "account_number": "3",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1gwc2eq50dtcde43r2fhvlt6njp0s8yaz6j4vwy",
//          "pub_key": null,
//          "account_number": "4",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1rrzsuakwtwactszf9h23dh88hh5cw9n9whyj9r",
//          "pub_key": null,
//          "account_number": "5",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo1qq3asnzpw0gdzsmshufv9p28znuf6ecdyp90ry",
//          "pub_key": null,
//          "account_number": "6",
//          "sequence": "0"
//        },
//        {
//          "@type": "/cosmos.auth.v1beta1.BaseAccount",
//          "address": "allo16kymud82y5hqgflnsf3el20wt5wk9e5t4kg4pn",
//          "pub_key": null,
//          "account_number": "7",
//          "sequence": "0"
//        }
//      ]
//    }
// `)

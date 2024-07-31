package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	statev1 "github.com/allora-network/allora-chain/x/emissions/api/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: statev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Get the current module parameters",
				},
				{
					RpcMethod: "GetNextTopicId",
					Use:       "next-topic-id",
					Short:     "Get next topic id",
				},
				{
					RpcMethod: "GetTopic",
					Use:       "topic [topic_id]",
					Short:     "Get topic by topic id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "TopicExists",
					Use:       "topic-exists [topic_id]",
					Short:     "True if topic exists at given id, else false",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "IsTopicActive",
					Use:       "is-topic-active [topic_id]",
					Short:     "True if topic is active, else false",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetActiveTopics",
					Use:       "active-topics [pagination]",
					Short:     "Get Active Topics",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "pagination"},
					},
				},
				{
					RpcMethod: "GetChurnableTopics",
					Use:       "churnable-topics",
					Short:     "Get Churnable Topics",
				},
				{
					RpcMethod: "GetRewardableTopics",
					Use:       "rewardable-topics",
					Short:     "Get Rewardable Topics",
				},
				{
					RpcMethod: "GetReputerStakeInTopic",
					Use:       "stake-in-topic-reputer [address] [topic_id]",
					Short:     "Get reputer stake in a topic, including stake delegated to them in that topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeInTopicInReputer",
					Use:       "stake-total-delegated-in-topic-reputer [reputer_address] [topic_id]",
					Short:     "Get total delegate stake in a topic and reputer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "reputer_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetDelegateRewardPerShare",
					Use:       "delegate-reward-per-share [topic_id] [reputer_address]",
					Short:     "Get total delegate reward per share stake in a reputer for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetDelegateStakePlacement",
					Use:       "delegate-reward-per-share [topic_id] [delegator] [target]",
					Short:     "Get amount of token delegated to a target by a delegator in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "delegator"},
						{ProtoField: "target"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeRemoval",
					Use:       "delegate-stake-removal [block_height] [topic_id] [delegator] [reputer]",
					Short:     "Get current state of a pending delegate stake removal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "block_height"},
						{ProtoField: "topic_id"},
						{ProtoField: "delegator"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeUponReputer",
					Use:       "delegate-stake-on-reputer [topic_id] [target]",
					Short:     "Get total amount of token delegated to a target reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "target"},
					},
				},
				{
					RpcMethod: "GetForecastScoresUntilBlock",
					Use:       "forecast-scores-until-block [topic_id] [block_height]",
					Short:     "Get all saved scores for all forecasters for a topic descending until a given past block height. Number of forecasts is limited by MaxSamplesToScaleScores",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetForecasterNetworkRegret",
					Use:       "forecaster-regret [topic_id] [worker]",
					Short:     "Get current network regret for given forecaster",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetInferenceScoresUntilBlock",
					Use:       "inference-scores-until-block [topic_id] [block_height]",
					Short:     "Get all saved scores for all inferers for a topic descending until a given past block height. Number of forecasts is limited by MaxSamplesToScaleScores",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetInfererNetworkRegret",
					Use:       "inferer-regret [topic_id] [actor_id]",
					Short:     "Get current network regret for given inferer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "actor_id"},
					},
				},
				{
					RpcMethod: "IsReputerNonceUnfulfilled",
					Use:       "reputer-nonce-unfulfilled [topic_id] [block_height]",
					Short:     "True if reputer nonce is unfulfilled (still awaiting a reputer response), else false",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "IsWorkerNonceUnfulfilled",
					Use:       "worker-nonce-unfulfilled [topic_id] [block_height]",
					Short:     "True if worker nonce is unfulfilled (still awaiting a worker response), else false",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetLatestAvailableNetworkInference",
					Use:       "latest-available-network-inference [topic_id]",
					Short:     "Returns network inference only if all available information to compute the inference is present",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetLatestForecasterScore",
					Use:       "latest-forecaster-score [topic_id] [forecaster]",
					Short:     "Returns latest score for a forecaster in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "forecaster"},
					},
				},
				{
					RpcMethod: "GetLatestInfererScore",
					Use:       "latest-inferer-score [topic_id] [inferer]",
					Short:     "Returns latest score for a inferer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "inferer"},
					},
				},
				{
					RpcMethod: "GetLatestReputerScore",
					Use:       "latest-reputer-score [topic_id] [reputer]",
					Short:     "Returns latest score for a reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetLatestTopicInferences",
					Use:       "latest-topic-raw-inferences [topic_id]",
					Short:     "Returns latest round of raw inferences from workers topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetListeningCoefficient",
					Use:       "listening-coefficient [topic_id] [reputer]",
					Short:     "Returns current listening coefficient for a given reputer. Default to 1 if does not exist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetMultiReputerStakeInTopic",
					Use:       "multi-coefficient [addresses] [topic_id]",
					Short:     "Returns stakes for each reputer in a given list. List can be up to MaxPageLimit in length. Default to 0 if does not exist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "addresses"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetOneInForecasterNetworkRegret",
					Use:       "one-in-forecaster-regret [topic_id] [forecaster] [inferer]",
					Short:     "Returns regret born from including [forecaster]'s implied inference in a batch with [inferer]. Default to topic InitialRegret if does not exist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "forecaster"},
						{ProtoField: "inferer"},
					},
				},
				{
					RpcMethod: "GetOneInForecasterSelfNetworkRegret",
					Use:       "one-in-forecaster-self-regret [topic_id] [forecaster]",
					Short:     "Returns regret born from including [forecaster]'s implied inference. Default to topic InitialRegret if does not exist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "forecaster"},
					},
				},
				{
					RpcMethod: "GetPreviousForecastRewardFraction",
					Use:       "previous-forecaster-reward-fraction [topic_id] [worker]",
					Short:     "Return previous reward fraction for actor",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetPreviousInferenceRewardFraction",
					Use:       "previous-inference-reward-fraction [topic_id] [worker]",
					Short:     "Return previous reward fraction for actor",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetPreviousPercentageRewardToStakedReputers",
					Use:       "previous-percentage-reputer-reward",
					Short:     "Return previous percent reward paid to staked reputers",
				},
				{
					RpcMethod: "GetPreviousReputerRewardFraction",
					Use:       "previous-reputer-reward-fraction [topic_id] [reputer]",
					Short:     "Return previous reward fraction for actor",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetPreviousTopicWeight",
					Use:       "previous-topic-weight [topic_id]",
					Short:     "Return previous topic weight. Useful for extrapolating future and previous topic weight and the topic's likelihood for churn",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetReputerLossBundlesAtBlock",
					Use:       "reputer-loss-bundle [topic_id] [block_height]",
					Short:     "Return reputer loss bundle at block height. May not exist if it was already pruned",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetReputersScoresAtBlock",
					Use:       "reputer-scores [topic_id] [block_height]",
					Short:     "Return reputer scores at block. Note: the chain only stores up to MaxSamplesToScaleScores many scores per actor type per topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetStakeRemovalForReputerAndTopicId",
					Use:       "reputer-scores [reputer] [topic_id]",
					Short:     "Return stake removal information for reputer in topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "reputer"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetStakeReputerAuthority",
					Use:       "reputer-authority [topic_id] [reputer]",
					Short:     "Return total stake on reputer in a topic, including delegate stake and their own",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "GetTopicFeeRevenue",
					Use:       "topic-fee-revenue [topic_id]",
					Short:     "Return effective fee revenue for a topic i.e. the total fees collected by the topic less an exponential decay of the fees over time. This is the impact of topic fees on the topic's weight",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicLastReputerPayload",
					Use:       "latest-reputer-payload [topic_id]",
					Short:     "Return latest reputer payload delivered for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicLastWorkerPayload",
					Use:       "latest-worker-payload [topic_id]",
					Short:     "Return latest worker payload delivered for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicRewardNonce",
					Use:       "topic-reward-nonce [topic_id]",
					Short:     "If a topic is rewardable, then this is the nonce that will be used to calculate topic rewards. The actors that participated in the worker/reputer rounds started at this nonce (block height) will be rewarded",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicStake",
					Use:       "topic-stake [topic_id]",
					Short:     "Return total stake in topic including delegate stake",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTotalRewardToDistribute",
					Use:       "total-rewards",
					Short:     "Return total rewards to be distributed among all rewardable topics in the block",
				},
				{
					RpcMethod: "GetUnfulfilledReputerNonces",
					Use:       "unfulfilled-reputer-nonces [topic_id]",
					Short:     "Return topic reputer nonces that have yet to be fulfilled",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetUnfulfilledWorkerNonces",
					Use:       "unfulfilled-worker-nonces [topic_id]",
					Short:     "Return topic worker nonces that have yet to be fulfilled",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetWorkerForecastScoresAtBlock",
					Use:       "forecast-scores [topic_id] [block_height]",
					Short:     "Return scores for topic worker at a block height. Default is empty. May not exist if it was already pruned",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetWorkerInferenceScoresAtBlock",
					Use:       "inference-scores [topic_id] [block_height]",
					Short:     "Return scores for topic worker at a block height. Default is empty. May not exist if it was already pruned",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetStakeFromReputerInTopicInSelf",
					Use:       "stake-reputer-in-topic-self [reputer_address] [topic_id]",
					Short:     "Get the stake of a reputer in a topic that they put on themselves",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "reputer_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetStakeFromDelegatorInTopicInReputer",
					Use:       "stake-delegator-in-topic-reputer [delegator_address] [reputer_address] [topic_id]",
					Short:     "Get amount of stake from delegator in a topic for a reputer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "delegator_address"},
						{ProtoField: "reputer_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetStakeFromDelegatorInTopic",
					Use:       "stake-delegator-in-topic [delegator_address] [topic_id]",
					Short:     "Get amount of stake in a topic for a delegator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "delegator_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetWorkerLatestInferenceByTopicId",
					Use:       "worker-latest-inference [topic_id] [worker_address]",
					Short:     "Get the latest inference for a given worker and topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "worker_address"},
					},
				},
				{
					RpcMethod: "GetInferencesAtBlock",
					Use:       "inferences-at-block [topic_id] [block_height]",
					Short:     "Get All Inferences produced for a topic in a particular timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetWorkerNodeInfo",
					Use:       "worker-info [libp2p_key]",
					Short:     "Get node info for worker node libp2p key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "libp2p_key"},
					},
				},
				{
					RpcMethod: "GetReputerNodeInfo",
					Use:       "reputer-info [libp2p_key]",
					Short:     "Get node info for reputer node libp2p key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "libp2p_key"},
					},
				},
				{
					RpcMethod: "GetWorkerAddressByP2PKey",
					Use:       "worker-address [libp2p_key]",
					Short:     "Get Worker Address by libp2p key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "libp2p_key"},
					},
				},
				{
					RpcMethod: "GetReputerAddressByP2PKey",
					Use:       "reputer-address [libp2p_key]",
					Short:     "Get Reputer Address by libp2p key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "libp2p_key"},
					},
				},
				{
					RpcMethod: "IsWorkerRegisteredInTopicId",
					Use:       "is-worker-registered [topic_id] [address]",
					Short:     "True if worker is registered in the topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "IsReputerRegisteredInTopicId",
					Use:       "is-reputer-registered [topic_id] [address]",
					Short:     "True if reputer is registered in the topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "GetTotalStake",
					Use:       "total-stake",
					Short:     "Get the total amount of staked tokens by all participants in the network",
				},
				{
					RpcMethod: "GetForecastsAtBlock",
					Use:       "forecasts-at-block [topic_id] [block]",
					Short:     "Get the Forecasts for a topic at block height ",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetNetworkInferencesAtBlock",
					Use:       "network-inferences-at-block [topic_id] [block_height_last_inference] [block_height_last_reward]",
					Short:     "Get the Network Inferences for a topic at a block height where the last inference was made and the last reward was given",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height_last_inference"},
						{ProtoField: "block_height_last_reward"},
					},
				},
				{
					RpcMethod: "GetLatestNetworkInference",
					Use:       "latest-network-inference [topic_id]",
					Short:     "Get the latest Network inferences and weights for a topic. Will return whatever information it has available.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetNetworkLossBundleAtBlock",
					Use:       "network-loss-bundle-at-block [topic_id] [block]",
					Short:     "Get the network loss bundle for a topic at given block height",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "IsWhitelistAdmin",
					Use:       "is-whitelist-admin [address]",
					Short:     "Check if an address is a whitelist admin. True if so, else false",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "GetStakeRemovalsUpUntilBlock",
					Use:       "stake-removals-up-until-block [block_height]",
					Short:     "Get all pending stake removal requests going to happen at a given block height",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeRemovalsUpUntilBlock",
					Use:       "delegate-stake-removals-up-until-block [block_height]",
					Short:     "Get all pending delegate stake removal requests going to happen at a given block height",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetStakeRemovalInfo",
					Use:       "stake-removal-info [address] [topic_id]",
					Short:     "Get a pending stake removal for a reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "reputer"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeRemovalInfo",
					Use:       "delegate-stake-removal-info [delegator] [reputer] [topic_id]",
					Short:     "Get a pending delegate stake removal for a delegator in a topic upon a reputer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "delegator"},
						{ProtoField: "reputer"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicLastWorkerCommitInfo",
					Use:       "topic-last-worker-commit [topic_id]",
					Short:     "Get topic last commit by worker",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetTopicLastReputerCommitInfo",
					Use:       "topic-last-reputer-commit [topic_id]",
					Short:     "Get topic last commit by reputer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: statev1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Use:       "update-params [sender] [params]",
					Short:     "Update params of the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "params"},
					},
				},
				{
					RpcMethod: "CreateNewTopic",
					Use:       "create-topic [creator] [metadata] [loss_logic] [loss_method] [inference_logic] [inference_method] [epoch_length] [ground_truth_lag] [default_arg] [p_norm] [alpha_regret] [allow_negative] [epsilon]",
					Short:     "Add a new topic to the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
						{ProtoField: "metadata"},
						{ProtoField: "loss_logic"},
						{ProtoField: "loss_method"},
						{ProtoField: "inference_logic"},
						{ProtoField: "inference_method"},
						{ProtoField: "epoch_length"},
						{ProtoField: "ground_truth_lag"},
						{ProtoField: "default_arg"},
						{ProtoField: "p_norm"},
						{ProtoField: "alpha_regret"},
						{ProtoField: "allow_negative"},
						{ProtoField: "epsilon"},
					},
				},
				{
					RpcMethod: "Register",
					Use:       "register [sender] [lib_p2p_key] [multi_address] [topic_ids] [initial_stake] [owner] [is_reputer]",
					Short:     "Register a new reputer or worker for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "multi_address"},
						{ProtoField: "topic_id"},
						{ProtoField: "owner"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "RemoveRegistration",
					Use:       "remove-registration [sender] [owner] [is_reputer]",
					Short:     "Remove a reputer or worker from a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "AddStake",
					Use:       "add-stake [sender] [topic_id] [amount]",
					Short:     "Add stake [amount] to ones self sender [reputer or worker] for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "RemoveStake",
					Use:       "remove-stake [sender] [topic_id] [amount]",
					Short:     "modify sender's [reputer] stake position by removing [amount] stake from a topic [topic_id]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "CancelRemoveStake",
					Use:       "cancel-remove-stake [sender] [topic_id]",
					Short:     "Cancel the removal of stake for a reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "DelegateStake",
					Use:       "delegate-stake [sender] [topic_id] [reputer] [amount]",
					Short:     "Delegate stake [amount] to a reputer for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "RemoveDelegateStake",
					Use:       "remove-delegate-stake [sender] [topic_id] [reputer] [amount]",
					Short:     "Modify sender's [reputer] delegate stake position by removing [amount] stake from a topic [topic_id] from a reputer [reputer]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "CancelRemoveDelegateStake",
					Use:       "cancel-remove-delegate-stake [sender] [topic_id] [reputer]",
					Short:     "Cancel the removal of delegated stake for a delegator staking on a reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "RewardDelegateStake",
					Use:       "reward-delegate-stake [sender] [topic_id] [reputer]",
					Short:     "Get Reward for Delegator [sender] for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
					},
				},
				{
					RpcMethod: "FundTopic",
					Use:       "fund-topic [sender] [topic_id] [amount] [extra_data]",
					Short:     "send funds to a topic to pay for inferences",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "AddToWhitelistAdmin",
					Use:       "add-to-whitelist-admin [sender] [address]",
					Short:     "add an admin address to the whitelist used for admin functions on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "RemoveFromWhitelistAdmin",
					Use:       "remove-from-whitelist-admin [sender] [address]",
					Short:     "remove a admin address from the whitelist used for admin functions on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "InsertBulkWorkerPayload",
					Use:       "insert-bulk-worker-payload [worker_data_bundles]",
					Short:     "Insert bulk worker payload",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "worker_data_bundles"},
					},
				},
				{
					RpcMethod: "InsertBulkReputerPayload",
					Use:       "insert-bulk-reputer-payload [reputer_value_bundles]",
					Short:     "Insert bulk reputer payload",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "reputer_value_bundles"},
					},
				},
			},
		},
	}
}

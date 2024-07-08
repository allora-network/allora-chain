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
					RpcMethod: "GetActiveTopics",
					Use:       "active-topics [pagination]",
					Short:     "Get Active Topics",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "pagination"},
					},
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
					Short:     "Get the latest Network inferences and weights for a topic",
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
					RpcMethod: "GetStakeRemovalsForBlock",
					Use:       "stake-removals-for-block [block_height]",
					Short:     "Get all pending stake removal requests going to happen at a given block height",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeRemovalsForBlock",
					Use:       "delegate-stake-removals-for-block [block_height]",
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

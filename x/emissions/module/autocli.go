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
					Use:       "active-topics",
					Short:     "Get Active Topics",
				},
				{
					RpcMethod: "GetReputerStakeInTopic",
					Use:       "reputer-topic-stake [address] [topic_id]",
					Short:     "Get reputer stake in a topic, including stake delegated to them in that topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetDelegateStakeInTopicInReputer",
					Use:       "reputer-topic-stake [reputer_address] [topic_id]",
					Short:     "Get total delegate stake in a reputer in a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "reputer_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetStakeFromDelegatorInTopicInReputer",
					Use:       "delegate-topic-stake-in-reputer [delegator_address] [reputer_address] [topic_id]",
					Short:     "Get amount of stake from delegator in a topic for a reputer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "delegator_address"},
						{ProtoField: "reputer_address"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetStakeFromDelegatorInTopic",
					Use:       "delegator-topic-stake [delegator_address] [topic_id]",
					Short:     "Get Account Stake in a topic",
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
					Use:       "inference [topic_id] [block_height]",
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
					RpcMethod: "GetMempoolInferenceRequest",
					Use:       "inference-request [topic_id] [request_id]",
					Short:     "Get a specific Inference Request and demand left in the mempool by topic id and request id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "request_id"},
					},
				},
				{
					RpcMethod: "GetMempoolInferenceRequestsByTopic",
					Use:       "all-inference-requests",
					Short:     "Get Inference Requests by topic and demand left for each request in mempool",
				},
				{
					RpcMethod: "GetTopicUnmetDemand",
					Use:       "topic-unmet-demand [topic_id]",
					Short:     "Get Topic Unmet Demand",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetLastRewardsUpdate",
					Use:       "last-rewards-update",
					Short:     "Get timestamp of the last rewards update",
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
					Use:       "get-forecasts-at-block [topic_id] [block]",
					Short:     "Get the Forecasts for a topic at block height ",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetNetworkInferencesAtBlock",
					Use:       "get-network-inferences-at-block [topic_id] [block]",
					Short:     "Get the Network Inferences for a topic at block height ",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
					},
				},
				{
					RpcMethod: "GetNetworkLossBundleAtBlock",
					Use:       "get-network-loss-bundle-at-block [topic_id] [block]",
					Short:     "Get the network loss bundle for a topic at block height ",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "block_height"},
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
					Use:       "create-topic [creator] [metadata] [loss_logic] [loss_method] [inference_logic] [inference_method] [epoch_length] [default_arg] [pnorm] [alpha_regret] [preward_reputer] [preward_inference] [preward_forecast] [f_tolerance]",
					Short:     "Add a new topic to the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
						{ProtoField: "metadata"},
						{ProtoField: "loss_logic"},
						{ProtoField: "loss_method"},
						{ProtoField: "inference_logic"},
						{ProtoField: "inference_method"},
						{ProtoField: "epoch_length"},
						{ProtoField: "default_arg"},
						{ProtoField: "pnorm"},
						{ProtoField: "alpha_regret"},
						{ProtoField: "preward_reputer"},
						{ProtoField: "preward_inference"},
						{ProtoField: "preward_forecast"},
						{ProtoField: "f_tolerance"},
					},
				},
				{
					RpcMethod: "Register",
					Use:       "register [creator] [lib_p2p_key] [multi_address] [topic_ids] [initial_stake] [owner] [is_reputer]",
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
					Use:       "remove-registration [creator] [owner] [is_reputer]",
					Short:     "Remove a reputer or worker from a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "AddStake",
					Use:       "add-stake [sender] [amount] [topic_id]",
					Short:     "Add stake [amount] to ones self sender [reputer or worker] for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "amount"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "StartRemoveStake",
					Use:       "start-remove-stake [sender] [placements_remove]",
					Short:     "modify sender's [reputer or worker] stake position by removing stake from [placements_remove]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "placements_remove"},
					},
				},
				{
					RpcMethod: "ConfirmRemoveStake",
					Use:       "confirm-remove-stake [sender] [target] [amount]",
					Short:     "Proceed with removing stake [amount] from a stakeTarget [reputer or worker] back to a sender [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
					},
				},
				{
					RpcMethod: "ActivateTopic",
					Use:       "activate-topic [sender] [topic_id]",
					Short:     "Activate a topic that has become inactivated",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "RequestInference",
					Use:       "request-inference [sender] [requests]",
					Short:     "Request a batch of inferences to be kicked off",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "requests"},
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
					RpcMethod: "AddToTopicCreationWhitelist",
					Use:       "add-to-topic-creation-whitelist [sender] [address]",
					Short:     "add an address to the whitelist used for creating topics on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "RemoveFromTopicCreationWhitelist",
					Use:       "remove-from-topic-creation-whitelist [sender] [address]",
					Short:     "remove an address from the whitelist used for creating topics on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "AddToReputerWhitelist",
					Use:       "add-to-weight-setting-whitelist [sender] [address]",
					Short:     "add an address to the whitelist used for setting weights on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "RemoveFromReputerWhitelist",
					Use:       "remove-from-weight-setting-whitelist [sender] [address]",
					Short:     "remove an address from the whitelist used for setting weights on-chain",
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

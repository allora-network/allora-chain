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
					RpcMethod: "GetAllTopics",
					Use:       "all-topics",
					Short:     "Get the full list of all topics created on the network",
				},
				{
					RpcMethod: "GetTopicsByCreator",
					Use:       "topics-by-creator [creator]",
					Short:     "Get Topics by Creator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
					},
				},
				{
					RpcMethod: "GetAccountStakeList",
					Use:       "account-stake-list [address]",
					Short:     "Get Account Stake List",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "GetWeight",
					Use:       "weight [topic_id] [reputer] [worker]",
					Short:     "Get Weight From a Reputer to a Worker for a Topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetAllInferences",
					Use:       "inference [topic_id] [timestamp]",
					Short:     "Get Latest Inference for a Topic in a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "timestamp"},
					},
				},
				{
					RpcMethod: "GetInferencesToScore",
					Use:       "inferences-to-score [topic_id]",
					Short:     "Get Latest Inferences for a Topic to be scored",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetWorkerNodeRegistration",
					Use:       "worker-registration [owner|libp2p-pub-key]",
					Short:     "Get registration for worker node id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "node_id"},
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
					RpcMethod: "GetExistingInferenceRequest",
					Use:       "inference-request [topic_id] [request_id]",
					Short:     "Get a specific Inference Request and demand left in the mempool by topic id and request id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "request_id"},
					},
				},
				{
					RpcMethod: "GetAllExistingInferenceRequests",
					Use:       "all-inference-requests",
					Short:     "Get All Inference Requests and demand left for each request in mempool",
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
					RpcMethod: "GetAccumulatedEpochRewards",
					Use:       "accumulated-epoch-rewards",
					Short:     "Get the accumlated rewards for the current epoch that have not yet been paid out to network participants",
				},
				{
					RpcMethod: "GetLastRewardsUpdate",
					Use:       "last-rewards-update",
					Short:     "Get timestamp of the last rewards update",
				},
				{
					RpcMethod: "GetRegisteredTopicIds",
					Use:       "registered-topic-ids [address] [bool is_reputer]",
					Short:     "Get the list of topics that a reputer or worker is registered to",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "GetTotalStake",
					Use:       "total-stake",
					Short:     "Get the total amount of staked tokens by all participants in the network",
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
					Use:       "push-topic [creator] [metadata] [weight_logic] [weight_method] [weight_cadence] [inference_logic] [inference_method] [inference_cadence] [default_arg]",
					Short:     "Add a new topic to the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
						{ProtoField: "metadata"},
						{ProtoField: "weight_logic"},
						{ProtoField: "weight_method"},
						{ProtoField: "weight_cadence"},
						{ProtoField: "inference_logic"},
						{ProtoField: "inference_method"},
						{ProtoField: "inference_cadence"},
						{ProtoField: "default_arg"},
					},
				},
				{
					RpcMethod: "Register",
					Use:       "register [creator] [lib_p2p_key] [multi_address] [topic_ids] [initial_stake] [owner] [is_reputer]",
					Short:     "Register a new reputer or worker for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "multi_address"},
						{ProtoField: "topic_ids"},
						{ProtoField: "initial_stake"},
						{ProtoField: "owner"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "AddNewRegistration",
					Use:       "add-registration [creator] [lib_p2p_key] [multi_address] [topic_id] [owner] [is_reputer]",
					Short:     "Register a reputer or worker for an additional topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
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
						{ProtoField: "creator"},
						{ProtoField: "topic_id"},
						{ProtoField: "is_reputer"},
					},
				},
				{
					RpcMethod: "AddStake",
					Use:       "add-stake [sender] [target] [amount]",
					Short:     "Add stake [amount] from a sender [reputer or worker] to a stakeTarget [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "stake_target"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "ModifyStake",
					Use:       "modify-stake [sender] [placements_remove] [placements_add]",
					Short:     "modify sender's [reputer or worker] stake position by removing stake from [placements_remove] and moving that stake to [placements_add]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "placements_remove"},
						{ProtoField: "placements_add"},
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
					RpcMethod: "StartRemoveAllStake",
					Use:       "start-remove-all-stake [sender]",
					Short:     "Start the process to remove all stake from a sender [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
					},
				},
				{
					RpcMethod: "ProcessInferences",
					Use:       "process-inferences [sender] [inferences]",
					Short:     "Process a batch of inferences",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "inferences"},
					},
				},
				{
					RpcMethod: "SetWeights",
					Use:       "set-weights [sender] [weights]",
					Short:     "Set a batch of weights",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "weights"},
					},
				},
				{
					RpcMethod: "ReactivateTopic",
					Use:       "reactivate-topic [sender] [topic_id]",
					Short:     "Reactivate a topic that has become inactivated",
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
					RpcMethod: "AddToWeightSettingWhitelist",
					Use:       "add-to-weight-setting-whitelist [sender] [address]",
					Short:     "add an address to the whitelist used for setting weights on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "RemoveFromWeightSettingWhitelist",
					Use:       "remove-from-weight-setting-whitelist [sender] [address]",
					Short:     "remove an address from the whitelist used for setting weights on-chain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "address"},
					},
				},
			},
		},
	}
}

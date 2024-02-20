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
					Use:       "get-next-topic-id",
					Short:     "Get next topic id",
				},
				{
					RpcMethod: "GetTopic",
					Use:       "get-topic topic_id",
					Short:     "Get topic by topic id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetActiveTopics",
					Use:       "get-active-topics",
					Short:     "Get Active Topics",
				},
				{
					RpcMethod: "GetTopicsByCreator",
					Use:       "get-topics-by-creator [creator]",
					Short:     "Get Topics by Creator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
					},
				},
				{
					RpcMethod: "GetAccountStakeList",
					Use:       "get-account-stake-list [address]",
					Short:     "Get Account Stake List",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "GetWeight",
					Use:       "get-weight [topic_id] [reputer] [worker]",
					Short:     "Get Weight From a Reputer to a Worker for a Topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "reputer"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetAllInferences",
					Use:       "get-inference [topic_id] [timestamp]",
					Short:     "Get Latest Inference for a Topic in a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
						{ProtoField: "timestamp"},
					},
				},
				{
					RpcMethod: "GetInferencesToScore",
					Use:       "get-inferences-to-score [topic_id]",
					Short:     "Get Latest Inferences for a Topic to be scored",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topic_id"},
					},
				},
				{
					RpcMethod: "GetWorkerNodeRegistration",
					Use:       "inference-nodes [owner|libp2p-pub-key]",
					Short:     "Get Latest Inference From Worker for a Topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "node_id"},
					},
				},
				{
					RpcMethod: "GetWorkerAddressByP2PKey",
					Use:       "get-inference-node [libp2p_key]",
					Short:     "Get Worker Address by libp2p key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "libp2p_key"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: statev1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// {
				// 	RpcMethod: "UpdateParams",
				// 	Skip:      true, // This is a authority gated tx, so we skip it.
				// },
				{
					RpcMethod: "CreateNewTopic",
					Use:       "push-topic [creator] [metadata] [weight_logic] [weight_method] [weight_cadence] [inference_logic] [inference_method] [inference_cadence] [active] [default_arg]",
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
						{ProtoField: "active"},
						{ProtoField: "default_arg"},
					},
				},
				{
					RpcMethod: "RegisterReputer",
					Use:       "register-reputer lib_p2p_key network_address topic_id initial_stake",
					Short:     "Register a new reputer for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "multi_address"},
						{ProtoField: "topic_id"},
						{ProtoField: "initial_stake"},
					},
				},
				{
					RpcMethod: "RegisterWorker",
					Use:       "register-worker lib_p2p_key network_address topic_id initial_stake",
					Short:     "Register a new worker for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "multi_address"},
						{ProtoField: "topic_id"},
						{ProtoField: "initial_stake"},
					},
				},
				{
					RpcMethod: "AddStake",
					Use:       "add-stake sender target amount",
					Short:     "Add stake [amount] from a sender [reputer or worker] to a stakeTarget [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "stake_target"},
						{ProtoField: "amount"},
					},
				},
				// {
				// 	RpcMethod: "ModifyStake",
				// 	Use:       "modify-stake sender placements_remove placements_add",
				// 	Short:     "modify sender's [reputer or worker] stake position by removing stake from [placements_remove] and moving that stake to [placements_add]",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{
				// 		{ProtoField: "sender"},
				// 		{ProtoField: "placements_remove"},
				// 		{ProtoField: "placements_add"},
				// 	},
				// },
				// {
				// 	RpcMethod: "StartRemoveStake",
				// 	Use:       "start-remove-stake sender target amount",
				// 	Short:     "Start remove stake process for [amount] from a stakeTarget [reputer or worker] back to a sender [reputer or worker]",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{
				// 		{ProtoField: "sender"},
				// 		{ProtoField: "stake_target"},
				// 		{ProtoField: "amount"},
				// 	},
				// },
				// {
				// 	RpcMethod: "ConfirmRemoveStake",
				// 	Use:       "confirm-remove-stake sender target amount",
				// 	Short:     "Proceed with removing stake [amount] from a stakeTarget [reputer or worker] back to a sender [reputer or worker]",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{
				// 		{ProtoField: "sender"},
				// 	},
				// },
				// {
				// 	RpcMethod: "StartRemoveAllStake",
				// 	Use:       "start-remove-all-stake sender",
				// 	Short:     "Start the process to remove all stake from a sender [reputer or worker]",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{
				// 		{ProtoField: "sender"},
				// 	},
				// },
				// {
				// 	RpcMethod: "ConfirmRemoveAllStake",
				// 	Use:       "confirm-remove-all-stake sender",
				// 	Short:     "Proceed with removing all stake from a sender [reputer or worker]",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{
				// 		{ProtoField: "sender"},
				// 	},
				// },
			},
		},
	}
}

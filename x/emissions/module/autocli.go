package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	statev1 "github.com/upshot-tech/protocol-state-machine-module/api/v1"
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
					Use:       "get-topic topicId",
					Short:     "Get topic by topicId",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topicId"},
					},
				},
				{
					RpcMethod: "GetWeight",
					Use:       "get-weight [topicId] [reputer] [worker]",
					Short:     "Get Weight From a Reputer to a Worker for a Topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topicId"},
						{ProtoField: "reputer"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetInference",
					Use:       "get-inference [topicId] [worker]",
					Short:     "Get Latest Inference From Worker for a Topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topicId"},
						{ProtoField: "worker"},
					},
				},
				{
					RpcMethod: "GetInferencesToScore",
					Use:       "get-inferences-to-score [topicId]",
					Short:     "Get Latest Inferences for a Topic to be scored",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "topicId"},
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
					Use:       "push-topic metadata cadence active validation_steps",
					Short:     "Add a new topic to the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "metadata"},
						{ProtoField: "weight_cadence"},
						{ProtoField: "active"},
						{ProtoField: "validation_steps"},
					},
				},
				{
					RpcMethod: "RegisterReputer",
					Use:       "register-reputer lib_p2p_key network_address topicId initial_stake",
					Short:     "Register a new reputer for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "network_address"},
						{ProtoField: "topicId"},
						{ProtoField: "initial_stake"},
					},
				},
				{
					RpcMethod: "RegisterWorker",
					Use:       "register-worker lib_p2p_key network_address topicId initial_stake",
					Short:     "Register a new worker for a topic",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "lib_p2p_key"},
						{ProtoField: "network_address"},
						{ProtoField: "topicId"},
						{ProtoField: "initial_stake"},
					},
				},
				{
					RpcMethod: "AddStake",
					Use:       "add-stake sender target amount",
					Short:     "Add stake [amount] from a sender [reputer or worker] to a stakeTarget [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "stakeTarget"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "RemoveStake",
					Use:       "remove-stake sender target amount",
					Short:     "Remove stake [amount] from a stakeTarget [reputer or worker] back to a sender [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "stakeTarget"},
						{ProtoField: "amount"},
					},
				},
				{
					RpcMethod: "RemoveAllStake",
					Use:       "remove-all-stake sender",
					Short:     "Remove all stake from a sender [reputer or worker]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
					},
				},
			},
		},
	}
}

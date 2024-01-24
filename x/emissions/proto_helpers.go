package protocol_state_machine_module

import (
	cosmosMath "cosmossdk.io/math"
)

// This helper file exists because protobuf does not make getters
// for fields that are custom types:
// https://github.com/gogo/protobuf/issues/477

func (m *MsgRegisterReputer) GetInitialStake() cosmosMath.Uint {
	if m != nil {
		return m.InitialStake
	}
	return cosmosMath.ZeroUint()
}

func (m *MsgRegisterWorker) GetInitialStake() cosmosMath.Uint {
	if m != nil {
		return m.InitialStake
	}
	return cosmosMath.ZeroUint()
}

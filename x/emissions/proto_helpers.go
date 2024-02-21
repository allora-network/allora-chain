package emissions

import (
	cosmosMath "cosmossdk.io/math"
)

// This helper file exists because protobuf does not make getters
// for fields that are custom types:
// https://github.com/gogo/protobuf/issues/477

func (m *MsgRegister) GetInitialStake() cosmosMath.Uint {
	if m != nil {
		return m.InitialStake
	}
	return cosmosMath.ZeroUint()
}

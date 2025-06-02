package types

import alloramath "github.com/allora-network/allora-chain/math"

func (m *InputWorkerAttributedValue) SetWorker(s string) {
	m.Worker = s
}

func (m *InputWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (m *InputWithheldWorkerAttributedValue) SetWorker(worker string) {
	m.Worker = worker
}

func (m *InputWithheldWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

package types

import alloramath "github.com/allora-network/allora-chain/math"

func (m *InputWorkerAttributedValue) SetWorker(s string) {
	m.Worker = s
}

//nolint:stylecheck
func (m *InputWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (m *InputWithheldWorkerAttributedValue) SetWorker(worker string) {
	m.Worker = worker
}

//nolint:stylecheck
func (m *InputWithheldWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

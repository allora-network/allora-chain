package types

import alloramath "github.com/allora-network/allora-chain/math"

func (m *InputWorkerAttributedValue) SetWorker(s string) {
	m.Worker = s
}

//nolint:stylecheck
func (m *InputWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (m *InputWorkerAttributedValue) GetIValue() any {
	return m.Value
}

func (m *InputWithheldWorkerAttributedValue) SetWorker(worker string) {
	m.Worker = worker
}

//nolint:stylecheck
func (m *InputWithheldWorkerAttributedValue) SetValue(val alloramath.Dec) {
	m.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (m *InputWithheldWorkerAttributedValue) GetIValue() any {
	return m.Value
}

func (m *InputOneOutInfererForecasterValues) GetWorker() string {
	return m.Forecaster
}

func (m *InputOneOutInfererForecasterValues) GetIValue() any {
	return m.OneOutInfererValues
}

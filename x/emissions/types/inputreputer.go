package types

import alloramath "github.com/allora-network/allora-chain/math"

func (w *InputWorkerAttributedValue) SetWorker(s string) {
	w.Worker = s
}

//nolint:stylecheck
func (w *InputWorkerAttributedValue) SetValue(val alloramath.Dec) {
	w.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (w *InputWorkerAttributedValue) GetIValue() any {
	return w.Value
}

func (w *InputWithheldWorkerAttributedValue) SetWorker(worker string) {
	w.Worker = worker
}

//nolint:stylecheck
func (w *InputWithheldWorkerAttributedValue) SetValue(val alloramath.Dec) {
	w.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (w *InputWithheldWorkerAttributedValue) GetIValue() any {
	return w.Value
}

func (w *InputOneOutInfererForecasterValues) GetWorker() string {
	return w.Forecaster
}

func (w *InputOneOutInfererForecasterValues) GetIValue() any {
	return w.OneOutInfererValues
}

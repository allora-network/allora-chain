package types

import alloramath "github.com/allora-network/allora-chain/math"

func (w *WorkerAttributedValue) GetValue() alloramath.Dec {
	return w.Value
}

func (w *WorkerAttributedValue) GetIValue() any {
	return w.Value
}

func (w *WithheldWorkerAttributedValue) GetValue() alloramath.Dec {
	return w.Value
}

func (w *WithheldWorkerAttributedValue) GetIValue() any {
	return w.Value
}

func (w *OneOutInfererForecasterValues) GetWorker() string {
	return w.Forecaster
}

func (w *OneOutInfererForecasterValues) GetIValue() any {
	return w.OneOutInfererValues
}

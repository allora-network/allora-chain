package types

import alloramath "github.com/allora-network/allora-chain/math"

func (workerValue *WorkerAttributedValue) GetValue() alloramath.Dec {
	return workerValue.Value
}

func (workerValue *WorkerAttributedValue) GetIValue() any {
	return workerValue.Value
}

func (withheldWorkerValue *WithheldWorkerAttributedValue) GetValue() alloramath.Dec {
	return withheldWorkerValue.Value
}

func (withheldWorkerValue *WithheldWorkerAttributedValue) GetIValue() any {
	return withheldWorkerValue.Value
}

func (w *OneOutInfererForecasterValues) GetWorker() string {
	return w.Forecaster
}

func (w *OneOutInfererForecasterValues) GetIValue() any {
	return w.OneOutInfererValues
}

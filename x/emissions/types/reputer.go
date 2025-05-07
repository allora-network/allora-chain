package types

func (w *WorkerAttributedValue) GetIValue() any              { return w.Value }
func (w *InputWorkerAttributedValue) GetIValue() any         { return w.Value }
func (w *WithheldWorkerAttributedValue) GetIValue() any      { return w.Value }
func (w *InputWithheldWorkerAttributedValue) GetIValue() any { return w.Value }
func (w *OneOutInfererForecasterValues) GetWorker() string   { return w.Forecaster }
func (w *OneOutInfererForecasterValues) GetIValue() any {
	return w.OneOutInfererValues
}
func (w *InputOneOutInfererForecasterValues) GetWorker() string { return w.Forecaster }
func (w *InputOneOutInfererForecasterValues) GetIValue() any {
	return w.OneOutInfererValues
}

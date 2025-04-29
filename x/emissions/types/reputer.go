package types

type WorkerValue struct {
	Worker string
	Value  any
}

type worker interface {
	GetWorker() string
	GetValue() any
}

func (w *WorkerAttributedValue) GetValue() any              { return w.Value }
func (w *InputWorkerAttributedValue) GetValue() any         { return w.Value }
func (w *WithheldWorkerAttributedValue) GetValue() any      { return w.Value }
func (w *InputWithheldWorkerAttributedValue) GetValue() any { return w.Value }
func (w *OneOutInfererForecasterValues) GetWorker() string  { return w.Forecaster }
func (w *OneOutInfererForecasterValues) GetValue() any {
	return ConvertToWorkerValues(w.OneOutInfererValues)
}
func (w *InputOneOutInfererForecasterValues) GetWorker() string { return w.Forecaster }
func (w *InputOneOutInfererForecasterValues) GetValue() any {
	return ConvertToWorkerValues(w.OneOutInfererValues)
}

func ConvertToWorkerValues[T worker](workerValues []T) []*WorkerValue {
	result := make([]*WorkerValue, len(workerValues))
	for i, wv := range workerValues {
		result[i] = &WorkerValue{
			Worker: wv.GetWorker(),
			Value:  wv.GetValue(),
		}
	}
	return result
}

package types

import alloramath "github.com/allora-network/allora-chain/math"

func (workerValue *WorkerAttributedValue) GetValue() alloramath.Dec {
	return workerValue.Value
}

func (withheldWorkerValue *WithheldWorkerAttributedValue) GetValue() alloramath.Dec {
	return withheldWorkerValue.Value
}

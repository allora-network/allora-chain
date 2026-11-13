package types

import alloramath "github.com/allora-network/allora-chain/math"

func (inputWorkerValue *InputWorkerAttributedValue) SetWorker(s string) {
	inputWorkerValue.Worker = s
}

func (inputWorkerValue *InputWorkerAttributedValue) SetValue(val alloramath.Dec) {
	inputWorkerValue.Value = alloramath.MustNewBoundedExp40Dec(val)
}

func (inputWithheldWorkerValue *InputWithheldWorkerAttributedValue) SetWorker(worker string) {
	inputWithheldWorkerValue.Worker = worker
}

func (inputWithheldWorkerValue *InputWithheldWorkerAttributedValue) SetValue(val alloramath.Dec) {
	inputWithheldWorkerValue.Value = alloramath.MustNewBoundedExp40Dec(val)
}

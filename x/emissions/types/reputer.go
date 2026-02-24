package types

import alloramath "github.com/allora-network/allora-chain/math"

func (workerValue *WorkerAttributedValue) GetValue() alloramath.Dec {
	return workerValue.Value
}

func (withheldWorkerValue *WithheldWorkerAttributedValue) GetValue() alloramath.Dec {
	return withheldWorkerValue.Value
}

func (rvb *ReputerValueBundles) Validate() error {
	if rvb == nil || rvb.ReputerValueBundles == nil {
		return nil
	}
	lbs := make(LossBundles, len(rvb.ReputerValueBundles))
	for i := range rvb.ReputerValueBundles {
		if rvb.ReputerValueBundles[i].ValueBundle == nil {
			continue
		}
		lbs[i] = rvb.ReputerValueBundles[i].ValueBundle
	}
	return lbs.Validate()
}

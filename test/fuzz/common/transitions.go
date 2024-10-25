package fuzzcommon

// full list of all possible transitions
type TransitionWeights struct {
	CreateTopic                uint8
	FundTopic                  uint8
	RegisterWorker             uint8
	RegisterReputer            uint8
	StakeAsReputer             uint8
	DelegateStake              uint8
	CollectDelegatorRewards    uint8
	DoInferenceAndReputation   uint8
	UnregisterWorker           uint8
	UnregisterReputer          uint8
	UnstakeAsReputer           uint8
	UndelegateStake            uint8
	CancelStakeRemoval         uint8
	CancelDelegateStakeRemoval uint8
}

// Return the "weight" aka, the percentage probability of each transition
// assuming a perfectly random distribution when picking transitions
// i.e. below, the probability of picking createTopic is 2%
func GetTransitionWeights() TransitionWeights {
	return TransitionWeights{
		CreateTopic:                2,
		FundTopic:                  10,
		RegisterWorker:             4,
		RegisterReputer:            4,
		StakeAsReputer:             10,
		DelegateStake:              10,
		CollectDelegatorRewards:    10,
		DoInferenceAndReputation:   30,
		UnregisterWorker:           4,
		UnregisterReputer:          4,
		UnstakeAsReputer:           6,
		UndelegateStake:            6,
		CancelStakeRemoval:         0,
		CancelDelegateStakeRemoval: 0,
	}
}

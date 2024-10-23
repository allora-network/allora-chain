package invariant_test

import "strconv"

// state transition counts, keep fields sync with allTransitions above
type StateTransitionCounts struct {
	createTopic                int
	fundTopic                  int
	registerWorker             int
	registerReputer            int
	unregisterWorker           int
	unregisterReputer          int
	stakeAsReputer             int
	delegateStake              int
	unstakeAsReputer           int
	undelegateStake            int
	cancelStakeRemoval         int
	cancelDelegateStakeRemoval int
	collectDelegatorRewards    int
	doInferenceAndReputation   int
}

// stringer for state transition counts
func (s StateTransitionCounts) String() string {
	return "{\ncreateTopic: " + strconv.Itoa(s.createTopic) + ", " +
		"\nfundTopic: " + strconv.Itoa(s.fundTopic) + ", " +
		"\nregisterWorker: " + strconv.Itoa(s.registerWorker) + ", " +
		"\nregisterReputer: " + strconv.Itoa(s.registerReputer) + ", " +
		"\nunregisterWorker: " + strconv.Itoa(s.unregisterWorker) + ", " +
		"\nunregisterReputer: " + strconv.Itoa(s.unregisterReputer) + ", " +
		"\nstakeAsReputer: " + strconv.Itoa(s.stakeAsReputer) +
		"\ndelegateStake: " + strconv.Itoa(s.delegateStake) +
		"\nunstakeAsReputer: " + strconv.Itoa(s.unstakeAsReputer) +
		"\nundelegateStake: " + strconv.Itoa(s.undelegateStake) +
		"\ncancelStakeRemoval: " + strconv.Itoa(s.cancelStakeRemoval) +
		"\ncancelDelegateStakeRemoval: " + strconv.Itoa(s.cancelDelegateStakeRemoval) +
		"\ncollectDelegatorRewards: " + strconv.Itoa(s.collectDelegatorRewards) +
		"\ndoInferenceAndReputation: " + strconv.Itoa(s.doInferenceAndReputation) +
		"\n}"
}

// how many times have we created topics?
func (s *StateTransitionCounts) incrementCreateTopicCount() {
	s.createTopic++
}

// how many times have we funded topics?
func (s *StateTransitionCounts) incrementFundTopicCount() {
	s.fundTopic++
}

// how many times have we registered workers?
func (s *StateTransitionCounts) incrementRegisterWorkerCount() {
	s.registerWorker++
}

// how many times have we registered reputers?
func (s *StateTransitionCounts) incrementRegisterReputerCount() {
	s.registerReputer++
}

// how many times have we unregistered workers?
func (s *StateTransitionCounts) incrementUnregisterWorkerCount() {
	s.unregisterWorker++
}

// how many times have we unregistered reputers?
func (s *StateTransitionCounts) incrementUnregisterReputerCount() {
	s.unregisterReputer++
}

// how many times have we staked as a reputer?
func (s *StateTransitionCounts) incrementStakeAsReputerCount() {
	s.stakeAsReputer++
}

// how many times have we delegated stake?
func (s *StateTransitionCounts) incrementDelegateStakeCount() {
	s.delegateStake++
}

// how many times have we unstaked as a reputer?
func (s *StateTransitionCounts) incrementUnstakeAsReputerCount() {
	s.unstakeAsReputer++
}

// how many times have we undelegated stake?
func (s *StateTransitionCounts) incrementUndelegateStakeCount() {
	s.undelegateStake++
}

// how many times have we cancelled stake removal?
func (s *StateTransitionCounts) incrementCancelStakeRemovalCount() {
	s.cancelStakeRemoval++
}

// how many times have we cancelled delegated stake removal?
func (s *StateTransitionCounts) incrementCancelDelegateStakeRemovalCount() {
	s.cancelDelegateStakeRemoval++
}

// how many times have we collected delegator rewards?
func (s *StateTransitionCounts) incrementCollectDelegatorRewardsCount() {
	s.collectDelegatorRewards++
}

// how many times have we produced inferences and reputations?
func (s *StateTransitionCounts) incrementDoInferenceAndReputationCount() {
	s.doInferenceAndReputation++
}

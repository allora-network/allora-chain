package emissions

import "cosmossdk.io/collections"

const ModuleName = "emissions"
const AlloraStakingModuleName = "allorastaking"
const AlloraRequestsModuleName = "allorarequests"

var (
	ParamsKey             = collections.NewPrefix(0)
	TotalStakeKey         = collections.NewPrefix(1)
	TopicStakeKey         = collections.NewPrefix(2)
	LastRewardsUpdateKey  = collections.NewPrefix(3)
	RewardsKey            = collections.NewPrefix(4)
	NextTopicIdKey        = collections.NewPrefix(5)
	TopicsKey             = collections.NewPrefix(6)
	TopicWorkersKey       = collections.NewPrefix(7)
	TopicReputersKey      = collections.NewPrefix(8)
	DelegatorStakeKey     = collections.NewPrefix(9)
	BondsKey              = collections.NewPrefix(10)
	TargetStakeKey        = collections.NewPrefix(11)
	WeightsKey            = collections.NewPrefix(12)
	InferencesKey         = collections.NewPrefix(13)
	WorkerNodesKey        = collections.NewPrefix(14)
	ReputerNodesKey       = collections.NewPrefix(15)
	LatestInferencesTsKey = collections.NewPrefix(16)
	MempoolKey            = collections.NewPrefix(17)
	FundsKey              = collections.NewPrefix(18)
	AllInferencesKey      = collections.NewPrefix(19)
	StakeRemovalQueueKey  = collections.NewPrefix(20)
	AllTopicStakeSumKey   = collections.NewPrefix(21)
	AddressTopicsKey      = collections.NewPrefix(22)
)

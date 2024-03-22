package types

import "cosmossdk.io/collections"

const ModuleName = "emissions"
const AlloraStakingAccountName = "allorastaking"
const AlloraRequestsAccountName = "allorarequests"
const AlloraRewardsAccountName = "allorarewards"

var (
	ParamsKey                     = collections.NewPrefix(0)
	TotalStakeKey                 = collections.NewPrefix(1)
	TopicStakeKey                 = collections.NewPrefix(2)
	LastRewardsUpdateKey          = collections.NewPrefix(3)
	RewardsKey                    = collections.NewPrefix(4)
	NextTopicIdKey                = collections.NewPrefix(5)
	TopicsKey                     = collections.NewPrefix(6)
	TopicWorkersKey               = collections.NewPrefix(7)
	TopicReputersKey              = collections.NewPrefix(8)
	DelegatorStakeKey             = collections.NewPrefix(9)
	BondsKey                      = collections.NewPrefix(10)
	TargetStakeKey                = collections.NewPrefix(11)
	InferencesKey                 = collections.NewPrefix(12)
	ForecastsKey                  = collections.NewPrefix(13)
	WorkerNodesKey                = collections.NewPrefix(14)
	ReputerNodesKey               = collections.NewPrefix(15)
	LatestInferencesTsKey         = collections.NewPrefix(16)
	MempoolKey                    = collections.NewPrefix(17)
	RequestUnmetDemandKey         = collections.NewPrefix(18)
	TopicUnmetDemandKey           = collections.NewPrefix(19)
	AllInferencesKey              = collections.NewPrefix(20)
	AllForecastsKey               = collections.NewPrefix(21)
	AllLossBundlesKey             = collections.NewPrefix(22)
	StakeRemovalQueueKey          = collections.NewPrefix(23)
	StakeByReputerAndTopicId      = collections.NewPrefix(24)
	DelegatedStakeRemovalQueueKey = collections.NewPrefix(25)
	AllTopicStakeSumKey           = collections.NewPrefix(26)
	AddressTopicsKey              = collections.NewPrefix(27)
	AccumulatedMetDemandKey       = collections.NewPrefix(28)
	NumInferencesInRewardEpochKey = collections.NewPrefix(29)
	WhitelistAdminsKey            = collections.NewPrefix(30)
	TopicCreationWhitelistKey     = collections.NewPrefix(31)
	ReputerWhitelistKey           = collections.NewPrefix(32)
	ChurnReadyTopicsKey           = collections.NewPrefix(33)
	FoundationWhitelistKey        = collections.NewPrefix(34)
	StakeByReputerAndTopicIdKey   = collections.NewPrefix(35)
	NetworkLossBundlesKey         = collections.NewPrefix(36)
)

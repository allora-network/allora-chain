package types

import "cosmossdk.io/errors"

var (
	ErrTopicReputerStakeDoesNotExist                 = errors.Register(ModuleName, 1, "topic reputer stake does not exist")
	ErrIntegerUnderflowTopicReputerStake             = errors.Register(ModuleName, 2, "integer underflow for topic reputer stake")
	ErrIntegerUnderflowTopicStake                    = errors.Register(ModuleName, 3, "integer underflow for topic stake")
	ErrIntegerUnderflowTotalStake                    = errors.Register(ModuleName, 4, "integer underflow for total stake")
	ErrIterationLengthDoesNotMatch                   = errors.Register(ModuleName, 5, "iteration length does not match")
	ErrInvalidTopicId                                = errors.Register(ModuleName, 6, "invalid topic ID")
	ErrReputerAlreadyRegisteredInTopic               = errors.Register(ModuleName, 7, "reputer already registered in topic")
	ErrAddressAlreadyRegisteredInATopic              = errors.Register(ModuleName, 8, "address already registered in a topic")
	ErrAddressIsNotRegisteredInAnyTopic              = errors.Register(ModuleName, 9, "address is not registered in any topic")
	ErrAddressIsNotRegisteredInThisTopic             = errors.Register(ModuleName, 10, "address is not registered in this topic")
	ErrInsufficientStakeToRegister                   = errors.Register(ModuleName, 11, "insufficient stake to register")
	ErrLibP2PKeyRequired                             = errors.Register(ModuleName, 12, "libp2p key required")
	ErrAddressNotRegistered                          = errors.Register(ModuleName, 13, "address not registered")
	ErrInsufficientStakeToRemove                     = errors.Register(ModuleName, 14, "insufficient stake to remove")
	ErrBlockHeightNegative                           = errors.Register(ModuleName, 15, "block height negative")
	ErrBlockHeightLessThanPrevious                   = errors.Register(ModuleName, 16, "block height less than previous")
	ErrConfirmRemoveStakeNoRemovalStarted            = errors.Register(ModuleName, 17, "confirm remove stake no removal started")
	ErrConfirmRemoveStakeTooEarly                    = errors.Register(ModuleName, 18, "confirm remove stake too early")
	ErrConfirmRemoveStakeTooLate                     = errors.Register(ModuleName, 19, "confirm remove stake too late")
	ErrTopicIdListValueDecodeInvalidLength           = errors.Register(ModuleName, 20, "topic ID list value decode invalid length")
	ErrTopicIdListValueDecodeJsonInvalidLength       = errors.Register(ModuleName, 21, "topic ID list value decode JSON invalid length")
	ErrTopicIdListValueDecodeJsonInvalidFormat       = errors.Register(ModuleName, 22, "topic ID list value decode JSON invalid format")
	ErrTopicDoesNotExist                             = errors.Register(ModuleName, 23, "topic does not exist")
	ErrInferenceRequestAlreadyInMempool              = errors.Register(ModuleName, 24, "inference request already in mempool")
	ErrInferenceRequestBidAmountLessThanPrice        = errors.Register(ModuleName, 25, "inference request bid amount less than price")
	ErrInferenceRequestCadenceTooFast                = errors.Register(ModuleName, 26, "inference request cadence too fast")
	ErrInferenceRequestCadenceTooSlow                = errors.Register(ModuleName, 27, "inference request cadence too slow")
	ErrInferenceRequestWillNeverBeScheduled          = errors.Register(ModuleName, 28, "inference request will never be scheduled")
	ErrOwnerCannotBeEmpty                            = errors.Register(ModuleName, 29, "owner cannot be empty")
	ErrInsufficientStakeAfterRemoval                 = errors.Register(ModuleName, 30, "insufficient stake after removal")
	ErrInferenceRequestBidAmountTooLow               = errors.Register(ModuleName, 31, "inference request bid amount too low")
	ErrIntegerUnderflowUnmetDemand                   = errors.Register(ModuleName, 32, "integer underflow for unmet demand")
	ErrNotWhitelistAdmin                             = errors.Register(ModuleName, 33, "not whitelist admin")
	ErrNotInTopicCreationWhitelist                   = errors.Register(ModuleName, 34, "not in topic creation whitelist")
	ErrNotInReputerWhitelist                         = errors.Register(ModuleName, 35, "not in reputer whitelist")
	ErrTopicNotEnoughDemand                          = errors.Register(ModuleName, 36, "topic not enough demand")
	ErrInvalidRequestId                              = errors.Register(ModuleName, 37, "invalid request ID")
	ErrInferenceRequestNotInMempool                  = errors.Register(ModuleName, 38, "inference request not in mempool")
	ErrIntegerUnderflowStakeFromDelegator            = errors.Register(ModuleName, 39, "integer underflow for stake from delegator")
	ErrIntegerUnderflowDelegateStakePlacement        = errors.Register(ModuleName, 40, "integer underflow for delegate stake placement")
	ErrIntegerUnderflowDelegateStakeUponReputer      = errors.Register(ModuleName, 41, "integer underflow for delegate stake upon reputer")
	ErrAdjustedStakeInvalidSliceLength               = errors.Register(ModuleName, 42, "adjusted stake: invalid slice length")
	ErrFractionDivideByZero                          = errors.Register(ModuleName, 43, "fraction: divide by zero")
	ErrNumberRatioDivideByZero                       = errors.Register(ModuleName, 44, "number ratio: divide by zero")
	ErrNumberRatioInvalidSliceLength                 = errors.Register(ModuleName, 45, "number ratio: invalid slice length")
	ErrInvalidSliceLength                            = errors.Register(ModuleName, 46, "invalid slice length")
	ErrTopicCadenceBelowMinimum                      = errors.Register(ModuleName, 47, "topic cadence must be at least 60 seconds (1 minute)")
	ErrPhiCannotBeZero                               = errors.Register(ModuleName, 48, "phi: cannot be zero")
	ErrInferenceRequestBlockValidUntilInPast         = errors.Register(ModuleName, 49, "inference request block valid until in past")
	ErrInferenceRequestBlockValidUntilTooFarInFuture = errors.Register(ModuleName, 50, "inference request block valid until too far in future")
	ErrSumWeightsLessThanEta                         = errors.Register(ModuleName, 51, "sum weights less than eta")
	ErrSliceLengthMismatch                           = errors.Register(ModuleName, 52, "slice length mismatch")
	ErrNonceAlreadyFulfilled                         = errors.Register(ModuleName, 53, "nonce already fulfilled")
	ErrNonceStillUnfulfilled                         = errors.Register(ModuleName, 54, "nonce still unfulfilled")
	ErrTopicCreatorNotEnoughDenom                    = errors.Register(ModuleName, 55, "topic creator does not have enough denom")
	ErrSignatureVerificationFailed                   = errors.Register(ModuleName, 56, "signature verification was failed")
	ErrTopicRegistrantNotEnoughDenom                 = errors.Register(ModuleName, 57, "topic registrant does not have enough denom")
)

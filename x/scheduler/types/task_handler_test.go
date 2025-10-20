//nolint:exhaustruct
package types

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestTaskHandlerPackArgs(t *testing.T) {
	var zeroNilArgs proto.Message
	nilArgsHandler := taskHandler[proto.Message]{
		name:     "nilArgsHandler",
		zeroArgs: zeroNilArgs,
	}
	var zeroNonNilArgs *cosmostypes.Coin
	withArgsHandler := taskHandler[*cosmostypes.Coin]{
		name:     "withArgsHandler",
		zeroArgs: zeroNonNilArgs,
	}

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	require.NoError(t, err)

	testCases := []struct {
		name             string
		handler          TaskHandler
		input            proto.Message
		expectPackedArgs *codectypes.Any
		expectErr        bool
	}{
		{
			name:             "pass nil args to no args handler",
			handler:          nilArgsHandler,
			input:            nil,
			expectPackedArgs: nil,
			expectErr:        false,
		},
		{
			name:             "pass non nil args to no args handler",
			handler:          nilArgsHandler,
			input:            args,
			expectPackedArgs: nil,
			expectErr:        true,
		},
		{
			name:             "pass non nil args to args handler",
			handler:          withArgsHandler,
			input:            args,
			expectPackedArgs: packedArgs,
			expectErr:        false,
		},
		{
			name:             "pass nil args to args handler",
			handler:          withArgsHandler,
			input:            nil,
			expectPackedArgs: nil,
			expectErr:        true,
		},
		{
			name:             "pass wrong type args to args handler",
			handler:          withArgsHandler,
			input:            &cosmostypes.DecCoin{Denom: "udenom", Amount: math.LegacyNewDec(12000)},
			expectPackedArgs: nil,
			expectErr:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packed, err := tc.handler.PackArgs(tc.input)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectPackedArgs, packed)
		})
	}
}

func TestTaskHandlerUnpackArgs(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()

	var zeroNilArgs proto.Message
	nilArgsHandler := taskHandler[proto.Message]{
		name:     "nilArgsHandler",
		zeroArgs: zeroNilArgs,
	}
	var zeroNonNilArgs *cosmostypes.Coin
	withArgsHandler := taskHandler[*cosmostypes.Coin]{
		name:     "withArgsHandler",
		zeroArgs: zeroNonNilArgs,
	}

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	require.NoError(t, err)

	wrongPackedArgs, err := codectypes.NewAnyWithValue(&cosmostypes.DecCoin{Denom: "udenom", Amount: math.LegacyNewDec(12000)})
	require.NoError(t, err)

	testCases := []struct {
		name               string
		handlerWithArgs    bool
		input              *codectypes.Any
		expectUnpackedArgs proto.Message
		expectErr          bool
	}{
		{
			name:               "pass nil args to no args handler",
			handlerWithArgs:    false,
			input:              nil,
			expectUnpackedArgs: nil,
			expectErr:          false,
		},
		{
			name:               "pass non nil args to no args handler",
			handlerWithArgs:    false,
			input:              packedArgs,
			expectUnpackedArgs: nil,
			expectErr:          true,
		},
		{
			name:               "pass non nil args to args handler",
			handlerWithArgs:    true,
			input:              packedArgs,
			expectUnpackedArgs: args,
			expectErr:          false,
		},
		{
			name:               "pass nil args to args handler",
			handlerWithArgs:    true,
			input:              nil,
			expectUnpackedArgs: nil,
			expectErr:          true,
		},
		{
			name:               "pass wrong type args to args handler",
			handlerWithArgs:    true,
			input:              wrongPackedArgs,
			expectUnpackedArgs: nil,
			expectErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var unpacked proto.Message
			var err error
			if tc.handlerWithArgs {
				unpacked, err = withArgsHandler.UnpackArgs(encCfg.Codec, tc.input)
			} else {
				unpacked, err = nilArgsHandler.UnpackArgs(encCfg.Codec, tc.input)
			}

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectUnpackedArgs == nil {
				require.Nil(t, unpacked)
			} else {
				require.IsType(t, tc.expectUnpackedArgs, unpacked)
				require.Equal(t, tc.expectUnpackedArgs, unpacked)
			}
		})
	}
}

func TestTaskHandlerArbitrate(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()

	args1 := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs1, err := codectypes.NewAnyWithValue(args1)
	require.NoError(t, err)
	args2 := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(15000)}
	packedArgs2, err := codectypes.NewAnyWithValue(args2)
	require.NoError(t, err)
	wrongPackedArgs, err := codectypes.NewAnyWithValue(&cosmostypes.DecCoin{Denom: "udenom", Amount: math.LegacyNewDec(12000)})
	require.NoError(t, err)

	testCases := []struct {
		name                              string
		arbitrateInput                    []Task
		arbitrateReturnErr                error
		arbitrateReturnDecisions          map[TaskID]ArbitrageDecision
		arbitrateFnShouldBeCalled         bool
		arbitrateShouldErr                bool
		arbitrateShouldReceiveInvocations []Invocation[*cosmostypes.Coin]
	}{
		{
			name:                              "arbitrate with no tasks called with no invocations",
			arbitrateInput:                    nil,
			arbitrateReturnErr:                nil,
			arbitrateReturnDecisions:          nil,
			arbitrateFnShouldBeCalled:         true,
			arbitrateShouldErr:                false,
			arbitrateShouldReceiveInvocations: []Invocation[*cosmostypes.Coin]{},
		},
		{
			name:                              "arbitrate properly map tasks to invocations",
			arbitrateInput:                    []Task{{Id: "task1", Args: packedArgs1}, {Id: "task2", Args: packedArgs2}},
			arbitrateReturnErr:                nil,
			arbitrateReturnDecisions:          nil,
			arbitrateFnShouldBeCalled:         true,
			arbitrateShouldErr:                false,
			arbitrateShouldReceiveInvocations: []Invocation[*cosmostypes.Coin]{{TaskID: "task1", Args: args1}, {TaskID: "task2", Args: args2}},
		},
		{
			name:                              "arbitrate with wrong args type should fail",
			arbitrateInput:                    []Task{{Id: "task1", Args: wrongPackedArgs}},
			arbitrateReturnErr:                nil,
			arbitrateReturnDecisions:          nil,
			arbitrateFnShouldBeCalled:         false,
			arbitrateShouldErr:                true,
			arbitrateShouldReceiveInvocations: nil,
		},
		{
			name:                              "arbitrate should return underlying fn decisions",
			arbitrateInput:                    []Task{{Id: "task1", Args: packedArgs1}},
			arbitrateReturnErr:                nil,
			arbitrateReturnDecisions:          map[TaskID]ArbitrageDecision{"task1": {Action: ArbitrageActionCancel}},
			arbitrateFnShouldBeCalled:         true,
			arbitrateShouldErr:                false,
			arbitrateShouldReceiveInvocations: []Invocation[*cosmostypes.Coin]{{TaskID: "task1", Args: args1}},
		},
		{
			name:                              "arbitrate should return underlying fn error",
			arbitrateInput:                    []Task{{Id: "task1", Args: packedArgs1}},
			arbitrateReturnErr:                fmt.Errorf("some error"),
			arbitrateReturnDecisions:          nil,
			arbitrateFnShouldBeCalled:         true,
			arbitrateShouldErr:                true,
			arbitrateShouldReceiveInvocations: []Invocation[*cosmostypes.Coin]{{TaskID: "task1", Args: args1}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var arbitrateFnCalled bool
			var arbitrateFnReceivedInvocations []Invocation[*cosmostypes.Coin]
			var zeroNonNilArgs *cosmostypes.Coin
			handler := taskHandler[*cosmostypes.Coin]{
				name:     "handler",
				zeroArgs: zeroNonNilArgs,
				arbitrateFn: func(ctx context.Context, tasks []Invocation[*cosmostypes.Coin]) (map[TaskID]ArbitrageDecision, error) {
					arbitrateFnCalled = true
					arbitrateFnReceivedInvocations = tasks
					return tc.arbitrateReturnDecisions, tc.arbitrateReturnErr
				},
			}

			decisions, err := handler.Arbitrate(context.Background(), encCfg.Codec, tc.arbitrateInput)
			if tc.arbitrateShouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.arbitrateFnShouldBeCalled, arbitrateFnCalled)
			require.Equal(t, tc.arbitrateShouldReceiveInvocations, arbitrateFnReceivedInvocations)
			require.Equal(t, tc.arbitrateReturnDecisions, decisions)
		})
	}
}

func TestTaskHandlerRun(t *testing.T) {
	now := time.Now().UTC()
	d10Min := time.Duration(10) * time.Minute
	nowMinus10Min := now.Add(-d10Min)

	encCfg := moduletestutil.MakeTestEncodingConfig()

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	require.NoError(t, err)
	wrongPackedArgs, err := codectypes.NewAnyWithValue(&cosmostypes.DecCoin{Denom: "udenom", Amount: math.LegacyNewDec(12000)})
	require.NoError(t, err)

	testCases := []struct {
		name                string
		inputTask           Task
		runFnReturnErr      error
		runFnShouldBeCalled bool
		runShouldErr        bool
	}{
		{
			name: "run called with proper inputs",
			inputTask: Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &now,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			runFnReturnErr:      nil,
			runFnShouldBeCalled: true,
			runShouldErr:        false,
		},
		{
			name: "run called with proper inputs (bis)",
			inputTask: Task{
				Id:                 "task15",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &now,
				Interval:           &d10Min,
				LastRunAt:          &nowMinus10Min,
				RunCount:           16,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			runFnReturnErr:      nil,
			runFnShouldBeCalled: true,
			runShouldErr:        false,
		},
		{
			name: "run should fail if underlying fn returns error",
			inputTask: Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &now,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			runFnReturnErr:      fmt.Errorf("some error"),
			runFnShouldBeCalled: true,
			runShouldErr:        true,
		},
		{
			name: "run with wrong args should fail",
			inputTask: Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               wrongPackedArgs,
				ScheduledFor:       &now,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			runFnReturnErr:      nil,
			runFnShouldBeCalled: false,
			runShouldErr:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var runFnCalled bool
			var runFnReceivedTask Task
			var runFnReceivedArgs *cosmostypes.Coin
			var zeroNonNilArgs *cosmostypes.Coin
			handler := taskHandler[*cosmostypes.Coin]{
				name:     "handler",
				zeroArgs: zeroNonNilArgs,
				runFn: func(ctx context.Context, task Task, args *cosmostypes.Coin) error {
					runFnCalled = true
					runFnReceivedTask = task
					runFnReceivedArgs = args
					return tc.runFnReturnErr
				},
			}

			err = handler.Run(context.Background(), encCfg.Codec, tc.inputTask)
			if tc.runShouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.runFnShouldBeCalled, runFnCalled)
			if tc.runFnShouldBeCalled {
				require.Equal(t, tc.inputTask, runFnReceivedTask)
				require.Equal(t, args, runFnReceivedArgs)
			}
		})
	}
}

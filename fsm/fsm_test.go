package fsm_test

import (
	"testing"

	"github.com/allora-network/allora-chain/fsm"
	"github.com/stretchr/testify/require"
)

type testState string

func (s testState) Name() string {
	return string(s)
}

type testSymbol string

func (s testSymbol) Name() string {
	return string(s)
}

type testFSM struct {
	state testState
}

func (f *testFSM) CurrentState() fsm.State {
	return f.state
}

func (f *testFSM) Advance(to fsm.State) {
	f.state = to.(testState)
}

func TestNewEngine(t *testing.T) {
	testCases := []struct {
		name        string
		initState   fsm.State
		finalStates []fsm.State
		transitions fsm.TransitionsTable[*testFSM]
		expectError bool
	}{
		{
			name:      "valid engine",
			initState: testState("init"),
			finalStates: []fsm.State{
				testState("final"),
			},
			transitions: fsm.TransitionsTable[*testFSM]{
				testState("init"): {
					testSymbol("toFinal"): {To: testState("final"), Action: nil},
				},
			},
			expectError: false,
		},
		{
			name:      "valid engine with multiple final states",
			initState: testState("init"),
			finalStates: []fsm.State{
				testState("final"),
				testState("final2"),
			},
			transitions: fsm.TransitionsTable[*testFSM]{
				testState("init"): {
					testSymbol("toFinal"):  {To: testState("final"), Action: nil},
					testSymbol("toFinal2"): {To: testState("final2"), Action: nil},
				},
			},
			expectError: false,
		},
		{
			name:      "no init state",
			initState: nil,
			finalStates: []fsm.State{
				testState("final"),
			},
			transitions: fsm.TransitionsTable[*testFSM]{
				testState("init"): {
					testSymbol("toFinal"): {To: testState("final"), Action: nil},
				},
			},
			expectError: true,
		},
		{
			name:        "no final states",
			initState:   testState("init"),
			finalStates: []fsm.State{},
			transitions: fsm.TransitionsTable[*testFSM]{
				testState("init"): {
					testSymbol("toFinal"): {To: testState("final"), Action: nil},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fsm.NewEngine[*testFSM](tc.initState, tc.finalStates, tc.transitions)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInit(t *testing.T) {
	eng, err := fsm.NewEngine[*testFSM](
		testState("init"),
		[]fsm.State{testState("final")},
		fsm.TransitionsTable[*testFSM]{
			testState("init"): {
				testSymbol("toFinal"): {To: testState("final"), Action: nil},
			},
		},
	)
	require.NoError(t, err)

	var mFSM testFSM
	eng.Init(&mFSM)
	require.Equal(t, testState("init"), mFSM.CurrentState())
}

func TestRunning(t *testing.T) {
	eng, err := fsm.NewEngine[*testFSM](
		testState("init"),
		[]fsm.State{testState("final")},
		fsm.TransitionsTable[*testFSM]{
			testState("init"): {
				testSymbol("toFinal"): {To: testState("final"), Action: nil},
			},
		},
	)
	require.NoError(t, err)

	var mFSM testFSM
	eng.Init(&mFSM)
	require.True(t, eng.Running(&mFSM))

	// Move to final state
	mFSM.Advance(testState("final"))
	require.False(t, eng.Running(&mFSM))
}

func TestTerminated(t *testing.T) {
	eng, err := fsm.NewEngine[*testFSM](
		testState("init"),
		[]fsm.State{testState("final")},
		fsm.TransitionsTable[*testFSM]{
			testState("init"): {
				testSymbol("toFinal"): {To: testState("final"), Action: nil},
			},
		},
	)
	require.NoError(t, err)

	var mFSM testFSM
	eng.Init(&mFSM)
	require.False(t, eng.Terminated(&mFSM))

	// Move to final state
	mFSM.Advance(testState("final"))
	require.True(t, eng.Terminated(&mFSM))
}

func TestConsume(t *testing.T) {
	eng, err := fsm.NewEngine[*testFSM](
		testState("init"),
		[]fsm.State{testState("final")},
		fsm.TransitionsTable[*testFSM]{
			testState("init"): {
				testSymbol("toFinal"): {To: testState("final"), Action: nil},
			},
		},
	)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		fromState   testState
		symbol      testSymbol
		expectErr   bool
		expectState fsm.State
	}{
		{
			name:        "valid transition to final",
			fromState:   testState("init"),
			symbol:      testSymbol("toFinal"),
			expectErr:   false,
			expectState: testState("final"),
		},
		{
			name:      "transition from final state",
			fromState: testState("final"),
			symbol:    testSymbol("whatever"),
			expectErr: true,
		},
		{
			name:      "unknown symbol",
			fromState: testState("init"),
			symbol:    testSymbol("whatever"),
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mFSM := testFSM{state: tc.fromState}
			err := eng.Consume(nil, &mFSM, tc.symbol)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectState, mFSM.CurrentState())
			}
		})
	}
}

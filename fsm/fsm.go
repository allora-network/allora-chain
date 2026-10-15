package fsm

import (
	"context"
	"fmt"
)

// State denotes a particular state of a state machine.
type State interface {
	Name() string
}

// Symbol denotes a particular symbol of a state machine's vocabulary.
type Symbol interface {
	Name() string
}

// Transition represents a state transition in a state machine, leading to a new state with a related action to perform.
type Transition[T FSM] struct {
	// To is the state to transition to.
	To State

	// Action is an optional action to perform during the transition.
	Action func(context.Context, T) error
}

// TransitionsTable defines the mapping of states and symbols to their corresponding transitions.
type TransitionsTable[T FSM] map[State]map[Symbol]Transition[T]

// Engine represents a finite state machine engine that manages state transitions based on input symbols.
// It is generic over the FSM type T, which must implement the FSM interface.
// An Engine instance is defined by its states and transitions, allowing to manage the lifecycle of FSM instances.
type Engine[T FSM] struct {
	initState   State
	finalStates map[State]struct{}
	transitions map[Symbol]map[State]Transition[T]
}

// FSM defines the interface that any finite state machine must implement to be managed by the Engine.
type FSM interface {
	// CurrentState returns the current state of the FSM.
	CurrentState() State

	// Advance transitions the FSM to the specified state.
	// This shall only be called by the Engine during state transitions. No invariants should be manually checked here.
	Advance(to State)
}

// NewEngine creates a new FSM engine with the specified initial state, final states, and transitions table.
// It ensures there is an init state and at least one final state defined, but does not ensure the transitions table is
// complete and finite.
func NewEngine[T FSM](initialState State, finalStates []State, transitions TransitionsTable[T]) (*Engine[T], error) {
	if initialState == nil {
		return nil, fmt.Errorf("initial state cannot be nil")
	}
	if len(finalStates) == 0 {
		return nil, fmt.Errorf("there must be at least one final state")
	}

	e := &Engine[T]{
		initState:   initialState,
		finalStates: map[State]struct{}{},
		transitions: map[Symbol]map[State]Transition[T]{},
	}

	for _, state := range finalStates {
		e.finalStates[state] = struct{}{}
	}

	for fromState, trs := range transitions {
		for symbol, tr := range trs {
			stateTransitions, ok := e.transitions[symbol]
			if !ok {
				stateTransitions = map[State]Transition[T]{}
				e.transitions[symbol] = stateTransitions
			}

			stateTransitions[fromState] = tr
		}
	}

	return e, nil
}

// Init initializes the FSM to its initial state.
func (e *Engine[T]) Init(fsm T) {
	fsm.Advance(e.initState)
}

// Running checks if the FSM is currently in a running state (i.e., not in a final state).
func (e *Engine[T]) Running(fsm T) bool {
	_, ok := e.finalStates[fsm.CurrentState()]
	return !ok
}

// Terminated checks if the FSM has reached a final state.
func (e *Engine[T]) Terminated(fsm T) bool {
	return !e.Running(fsm)
}

// Consume processes the given symbol for the FSM, performing the associated transition and action, if any.
func (e *Engine[T]) Consume(ctx context.Context, fsm T, symbol Symbol) error {
	if e.Terminated(fsm) {
		return fmt.Errorf("cannot consume a stopped fsm")
	}

	stateTransitions, ok := e.transitions[symbol]
	if !ok {
		return fmt.Errorf("unknown symbol '%s'", symbol.Name())
	}

	transition, ok := stateTransitions[fsm.CurrentState()]
	if !ok {
		return fmt.Errorf("no transition found for symbol '%s' from state '%s'", symbol.Name(), fsm.CurrentState().Name())
	}

	if transition.Action != nil {
		if err := transition.Action(ctx, fsm); err != nil {
			return err
		}
	}

	fsm.Advance(transition.To)
	return nil
}

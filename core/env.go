package core

import "context"

// A generic environment explored by RL
type Environment interface {
	// Reset the state of the environment
	Reset() (State, error)
	// Step through with a specified action and return the resulting state
	// Error if the transition is unsuccessful or disallowed
	Step(Action, *StepContext) (State, error)
}

// State of the environment
type State interface {
	// Key to the state
	Hash() string
	// Set of actions posible from the state
	Actions() []Action
}

// Generic Action taken by RL
type Action interface {
	// Key to the action
	Hash() string
}

// Context that wraps static and dynamic information of the episode
// Static: info about the episode
// Dynamic: info collected during the episode - trace, error or timeout
type EpisodeContext struct {
	// Context used when running to stop if required
	Context context.Context
	// Episode number
	Episode int
	// Horizon of the episode
	Horizon int
	// Run number
	Run int
	// Start time step of the episode
	StartTimeStep int

	// Trace including the steps taken in this episode
	Trace *Trace

	err     error
	timeout bool
	doneCh  chan struct{}
}

// Creates a new episode context
func NewEpisodeContext(ctx context.Context) *EpisodeContext {
	return &EpisodeContext{
		Context: ctx,
		doneCh:  make(chan struct{}),
		Trace:   NewTrace(),
	}
}

func (e *EpisodeContext) Error(err error) {
	e.err = err
	close(e.doneCh)
}

func (e *EpisodeContext) Timeout() {
	e.timeout = true
	close(e.doneCh)
}

func (e *EpisodeContext) Finish() {
	close(e.doneCh)
}

func (e *EpisodeContext) IsError() bool {
	return e.err != nil
}

func (e *EpisodeContext) IsTimeout() bool {
	return e.timeout
}

func (e *EpisodeContext) Done() <-chan struct{} {
	return e.doneCh
}

type StepContext struct {
	Step int
	*EpisodeContext
}

type EnvironmentConstructor interface {
	// NewEnvironment creates a new environment with the given instance number.
	NewEnvironment(int) Environment
}

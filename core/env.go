package core

import "context"

type Environment interface {
	Reset() (State, error)
	Step(Action, *StepContext) (State, error)
}

type State interface {
	Hash() string
	Actions() []Action
}

type Action interface {
	Hash() string
}

type EpisodeContext struct {
	Context       context.Context
	Episode       int
	Horizon       int
	Run           int
	StartTimeStep int

	Trace *Trace

	err     error
	timeout bool
	doneCh  chan struct{}
}

func NewEpisodeContext(ctx context.Context) *EpisodeContext {
	return &EpisodeContext{
		Context: ctx,
		doneCh:  make(chan struct{}),
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

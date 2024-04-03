package core

import "sync"

type Step struct {
	State     State
	Action    Action
	NextState State

	Misc map[string]interface{}
}

type Trace struct {
	mtx   *sync.Mutex
	steps []*Step
	err   error
}

func NewTrace() *Trace {
	return &Trace{
		steps: make([]*Step, 0),
		mtx:   new(sync.Mutex),
		err:   nil,
	}
}

func (t *Trace) AddStep(s *Step) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.steps = append(t.steps, s)
}

func (t *Trace) Step(i int) *Step {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.steps[i]
}

func (t *Trace) Len() int {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return len(t.steps)
}

func (t *Trace) Last() *Step {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.steps[len(t.steps)-1]
}

func (t *Trace) SetError(err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	t.err = err
}

func (t *Trace) Error() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.err
}

package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
)

var (
	Init Predicate = Predicate{
		Name:  "Init",
		Check: func(s core.State) bool { return true },
	}
)

// Predicate used in the predicate hierarchy
type PredicateFunc func(core.State) bool

// And connective
func (p PredicateFunc) And(other PredicateFunc) PredicateFunc {
	return func(s core.State) bool {
		return p(s) && other(s)
	}
}

// Or connective
func (p PredicateFunc) Or(other PredicateFunc) PredicateFunc {
	return func(s core.State) bool {
		return p(s) || other(s)
	}
}

// Not connective
func (p PredicateFunc) Not() PredicateFunc {
	return func(s core.State) bool {
		return !p(s)
	}
}

// Predicate struct to encapsulate the name along with the function
type Predicate struct {
	Name  string
	Check PredicateFunc
}

// step to store in the segment
type hierarchyStep struct {
	state      string
	action     string
	reward     bool
	outOfSpace bool
	nextState  string
}

// Policy to bias towards states specified hierarchy predicates
type HierarchyPolicy struct {
	predicates []Predicate

	qTables map[int]*QTable
	visits  map[int]*QTable

	alpha    float64
	discount float64
	epsilon  float64
	rand     *rand.Rand

	curPredicate  int
	traceSegments map[int][]*hierarchyStep
	targetReached bool
}

var _ core.Policy = &HierarchyPolicy{}

func NewHierarchyPolicy(alpha, discount, epsilon float64, predicates ...Predicate) *HierarchyPolicy {
	predicates = append([]Predicate{Init}, predicates...)
	out := &HierarchyPolicy{
		predicates: predicates,

		qTables: make(map[int]*QTable),
		visits:  make(map[int]*QTable),

		alpha:    alpha,
		discount: discount,
		epsilon:  epsilon,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),

		curPredicate:  0,
		traceSegments: make(map[int][]*hierarchyStep),
		targetReached: false,
	}
	for i := 0; i < len(predicates); i++ {
		out.qTables[i] = NewQTable()
		out.visits[i] = NewQTable()
	}
	return out
}

func (h *HierarchyPolicy) Reset() {
	h.qTables = make(map[int]*QTable)
	h.visits = make(map[int]*QTable)

	for i := 0; i < len(h.predicates); i++ {
		h.qTables[i] = NewQTable()
		h.visits[i] = NewQTable()
	}
	h.curPredicate = 0
	h.traceSegments = make(map[int][]*hierarchyStep)
	h.targetReached = false
}

func (h *HierarchyPolicy) ResetEpisode(_ *core.EpisodeContext) {
	h.traceSegments = make(map[int][]*hierarchyStep)
	h.curPredicate = 0
	h.targetReached = false
}

func (h *HierarchyPolicy) UpdateStep(sCtx *core.StepContext, state core.State, action core.Action, nextState core.State) {
	reward := false
	ourOfSpace := false
	curPredicate := h.curPredicate
	if !h.targetReached {
		nextPredicate := 0
		for i := len(h.predicates) - 1; i >= 0; i-- {
			if h.predicates[i].Check(nextState) {
				nextPredicate = i
				break
			}
		}
		if nextPredicate != h.curPredicate {
			ourOfSpace = true
		}
		if nextPredicate > h.curPredicate {
			reward = true
		}

		if nextPredicate == len(h.predicates)-1 {
			h.targetReached = true
		}
		h.curPredicate = nextPredicate
	}
	sCtx.AdditionalInfo["predicate_state"] = h.predicates[curPredicate].Name
	if _, ok := h.traceSegments[curPredicate]; !ok {
		h.traceSegments[curPredicate] = make([]*hierarchyStep, 0)
	}
	h.traceSegments[curPredicate] = append(h.traceSegments[curPredicate], &hierarchyStep{
		state:      state.Hash(),
		action:     action.Hash(),
		reward:     reward,
		outOfSpace: ourOfSpace,
		nextState:  nextState.Hash(),
	})
}

func (h *HierarchyPolicy) PickAction(sCtx *core.StepContext, state core.State, actions []core.Action) core.Action {
	if h.rand.Float64() < h.epsilon {
		i := h.rand.Intn(len(actions))
		return actions[i]
	}

	actionsMap := make(map[string]core.Action)
	availableActions := make([]string, len(actions))
	for i, a := range actions {
		aHash := a.Hash()
		actionsMap[aHash] = a
		availableActions[i] = aHash
	}
	maxAction, _ := h.qTables[h.curPredicate].MaxAmong(state.Hash(), availableActions, 1)
	if maxAction == "" {
		return nil
	}
	return actionsMap[maxAction]
}

func (h *HierarchyPolicy) UpdateEpisode(_ *core.EpisodeContext) {
	for i := range h.predicates {
		if _, ok := h.traceSegments[i]; !ok {
			continue
		}
		segmentLength := len(h.traceSegments[i])
		for j := segmentLength - 1; j > 0; j-- {
			step := h.traceSegments[i][j]

			t := h.visits[i].Get(step.state, step.action, 0) + 1
			h.visits[i].Set(step.state, step.action, t)
			q := h.qTables[i].Get(step.state, step.action, 0)
			_, nextMaxVal := h.qTables[i].Max(step.nextState, 0)
			if step.outOfSpace || j == segmentLength-1 {
				_, nextMaxVal = h.qTables[i].Max(step.nextState, 0)
			}

			reward := 1 / float64(t)
			if step.reward {
				reward += 2
			}

			q = (1-h.alpha)*q + h.alpha*util.MaxFloat(reward, h.discount*nextMaxVal)
			h.qTables[i].Set(step.state, step.action, q)
		}
	}
}

type HierarchyPolicyConstructor struct {
	Alpha      float64
	Discount   float64
	Epsilon    float64
	Predicates []Predicate
}

var _ core.PolicyConstructor = &HierarchyPolicyConstructor{}

func NewHierarchyPolicyConstructor(alpha, discount, epsilon float64, predicates ...Predicate) *HierarchyPolicyConstructor {
	return &HierarchyPolicyConstructor{
		Alpha:      alpha,
		Discount:   discount,
		Epsilon:    epsilon,
		Predicates: predicates,
	}
}

func (h *HierarchyPolicyConstructor) NewPolicy() core.Policy {
	return NewHierarchyPolicy(h.Alpha, h.Discount, h.Epsilon, h.Predicates...)
}

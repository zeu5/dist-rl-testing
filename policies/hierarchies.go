package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
)

type PredicateFunc func(core.State) bool

type Predicate struct {
	Name  string
	Check PredicateFunc
}

type hierarchyStep struct {
	state      core.State
	action     core.Action
	reward     bool
	outOfSpace bool
	nextState  core.State
}

type HierarchyPolicy struct {
	predicates []Predicate

	qTables map[int]*QTable
	visits  map[int]*QTable

	alpha    float64
	discount float64
	epsilon  float64
	oneTime  bool
	rand     *rand.Rand

	curPredicate  int
	traceSegments map[int][]*hierarchyStep
	targetReached bool
}

var _ core.Policy = &HierarchyPolicy{}

func NewHierarchyPolicy(alpha, discount, epsilon float64, oneTime bool, predicates ...Predicate) *HierarchyPolicy {
	return &HierarchyPolicy{
		predicates: predicates,

		qTables: make(map[int]*QTable),
		visits:  make(map[int]*QTable),

		alpha:    alpha,
		discount: discount,
		epsilon:  epsilon,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *HierarchyPolicy) Reset() {
	h.qTables = make(map[int]*QTable)
	h.visits = make(map[int]*QTable)
}

func (h *HierarchyPolicy) ResetEpisode(_ *core.EpisodeContext) {
	h.traceSegments = make(map[int][]*hierarchyStep)
	h.curPredicate = 0
	h.targetReached = false
}

func (h *HierarchyPolicy) UpdateStep(_ *core.StepContext, state core.State, action core.Action, nextState core.State) {
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

		if h.oneTime && nextPredicate == len(h.predicates)-1 {
			h.targetReached = true
		}
		h.curPredicate = nextPredicate
	}
	if _, ok := h.traceSegments[curPredicate]; !ok {
		h.traceSegments[curPredicate] = make([]*hierarchyStep, 0)
	}
	h.traceSegments[curPredicate] = append(h.traceSegments[curPredicate], &hierarchyStep{
		state:      state,
		action:     action,
		reward:     reward,
		outOfSpace: ourOfSpace,
		nextState:  nextState,
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
		for j := 0; j < segmentLength; j++ {
			step := h.traceSegments[i][j]
			stateHash := step.state.Hash()
			actionHash := step.action.Hash()
			nextStateHash := step.nextState.Hash()

			t := h.visits[i].Get(stateHash, actionHash, 0) + 1
			h.visits[i].Set(stateHash, actionHash, t)
			q := h.qTables[i].Get(stateHash, actionHash, 0)
			_, nextMaxVal := h.qTables[i].Max(nextStateHash, 0)
			if step.outOfSpace || j == segmentLength-1 {
				_, nextMaxVal = h.qTables[i].Max(nextStateHash, 0)
			}

			reward := 1 / float64(t)
			if step.reward {
				reward += 2
			}

			q = (1-h.alpha)*q + h.alpha*util.MaxFloat(reward, h.discount*nextMaxVal)
			h.qTables[i].Set(stateHash, actionHash, q)
		}
	}
}

type HierarchyPolicyConstructor struct {
	Alpha      float64
	Discount   float64
	Epsilon    float64
	OneTime    bool
	Predicates []Predicate
}

var _ core.PolicyConstructor = &HierarchyPolicyConstructor{}

func NewHierarchyPolicyConstructor(alpha, discount, epsilon float64, oneTime bool, predicates ...Predicate) *HierarchyPolicyConstructor {
	return &HierarchyPolicyConstructor{
		Alpha:      alpha,
		Discount:   discount,
		Epsilon:    epsilon,
		OneTime:    oneTime,
		Predicates: predicates,
	}
}

func (h *HierarchyPolicyConstructor) NewPolicy() core.Policy {
	return NewHierarchyPolicy(h.Alpha, h.Discount, h.Epsilon, h.OneTime, h.Predicates...)
}

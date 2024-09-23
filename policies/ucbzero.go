package policies

import (
	"math"
	"time"

	erand "golang.org/x/exp/rand"

	"github.com/zeu5/dist-rl-testing/core"
)

type UCBZeroParams struct {
	StateSize   int
	ActionsSize int
	Horizon     int
	Episodes    int
	Constant    float64
	Epsilon     float64
}

type UCBZeroPolicy struct {
	qTable *QTable
	visits *QTable
	rand   *erand.Rand
	params UCBZeroParams

	eta float64
}

func NewUCBZeroPolicy(params UCBZeroParams) *UCBZeroPolicy {
	eta := math.Log(
		float64(params.Horizon) * float64(params.ActionsSize) * float64(params.Episodes) * float64(params.StateSize),
	)

	return &UCBZeroPolicy{
		qTable: NewQTable(),
		visits: NewQTable(),
		rand:   erand.New(erand.NewSource(uint64(time.Now().UnixNano()))),
		params: params,

		eta: eta,
	}
}

var _ core.Policy = &UCBZeroPolicy{}

func (b *UCBZeroPolicy) ResetEpisode(_ *core.EpisodeContext) {
}

func (b *UCBZeroPolicy) UpdateEpisode(_ *core.EpisodeContext) {
}

func (b *UCBZeroPolicy) PickAction(step *core.StepContext, state core.State, actions []core.Action) core.Action {
	if b.rand.Float64() < b.params.Epsilon {
		i := b.rand.Intn(len(actions))
		return actions[i]
	}

	actionsMap := make(map[string]core.Action)
	availableActions := make([]string, len(actions))
	for i, a := range actions {
		aHash := a.Hash()
		actionsMap[aHash] = a
		availableActions[i] = aHash
	}
	maxAction, _ := b.qTable.MaxAmong(state.Hash(), availableActions, float64(b.params.Horizon))
	if maxAction == "" {
		return nil
	}

	return actionsMap[maxAction]
}

func (b *UCBZeroPolicy) UpdateStep(sCtx *core.StepContext, state core.State, action core.Action, nextState core.State) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	_, nextStateVal := b.qTable.Max(nextStateHash, float64(b.params.Horizon))
	if nextStateVal > float64(b.params.Horizon) {
		nextStateVal = float64(b.params.Horizon)
	}

	bonus := b.params.Constant * (math.Sqrt((math.Pow(float64(b.params.Horizon), 3) + b.eta) / float64(t)))
	alphaT := float64(b.params.Horizon+1) / (float64(b.params.Horizon) + t)
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	newVal := (1-alphaT)*curVal + alphaT*(nextStateVal+2*bonus)
	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *UCBZeroPolicy) Reset() {
	b.qTable = NewQTable()
	b.visits = NewQTable()
	b.rand = erand.New(erand.NewSource(uint64(time.Now().UnixNano())))
}

type UCBZeroPolicyConstructor struct {
	params UCBZeroParams
}

var _ core.PolicyConstructor = &UCBZeroPolicyConstructor{}

func NewUCBZeroPolicyConstructor(params UCBZeroParams) *UCBZeroPolicyConstructor {
	return &UCBZeroPolicyConstructor{
		params: params,
	}
}

func (b *UCBZeroPolicyConstructor) NewPolicy() core.Policy {
	return NewUCBZeroPolicy(b.params)
}

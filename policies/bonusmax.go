package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
)

type BonusPolicyGreedyReward struct {
	qTable   *QTable
	alpha    float64
	discount float64
	visits   *QTable
	epsilon  float64
	rand     *rand.Rand
}

var _ core.Policy = &BonusPolicyGreedyReward{}

func NewBonusPolicyGreedyReward(alpha, discount, epsilon float64) *BonusPolicyGreedyReward {
	return &BonusPolicyGreedyReward{
		qTable:   NewQTable(),
		alpha:    alpha,
		discount: discount,
		visits:   NewQTable(),
		epsilon:  epsilon,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *BonusPolicyGreedyReward) Record(path string) {
	b.qTable.Record(path)
}

func (b *BonusPolicyGreedyReward) Reset() {
	b.qTable = NewQTable()
	b.visits = NewQTable()
}

func (b *BonusPolicyGreedyReward) ResetEpisode(_ *core.EpisodeContext) {
}

func (b *BonusPolicyGreedyReward) PickAction(step *core.StepContext, state core.State, actions []core.Action) core.Action {
	if b.rand.Float64() < b.epsilon {
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
	maxAction, _ := b.qTable.MaxAmong(state.Hash(), availableActions, 1)
	if maxAction == "" {
		return nil
	}
	return actionsMap[maxAction]
}

func (b *BonusPolicyGreedyReward) UpdateStep(sCtx *core.StepContext, state core.State, action core.Action, nextState core.State) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	_, nextStateVal := b.qTable.Max(nextStateHash, 1)
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	newVal := (1-b.alpha)*curVal + b.alpha*max(1/t, b.discount*nextStateVal)
	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *BonusPolicyGreedyReward) UpdateEpisode(episode *core.EpisodeContext) {

}

type BonusPolicyGreedyRewardConstructor struct {
	alpha    float64
	discount float64
	epsilon  float64
}

var _ core.PolicyConstructor = &BonusPolicyGreedyRewardConstructor{}

func NewBonusPolicyGreedyRewardConstructor(alpha, discount, epsilon float64) *BonusPolicyGreedyRewardConstructor {
	return &BonusPolicyGreedyRewardConstructor{
		alpha:    alpha,
		discount: discount,
		epsilon:  epsilon,
	}
}

func (b *BonusPolicyGreedyRewardConstructor) NewPolicy() core.Policy {
	return NewBonusPolicyGreedyReward(b.alpha, b.discount, b.epsilon)
}

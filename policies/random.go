package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
)

type RandomPolicy struct {
	rand *rand.Rand
}

var _ core.Policy = &RandomPolicy{}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RandomPolicy) Reset() {}

func (r *RandomPolicy) UpdateEpisode(_ *core.EpisodeContext) {}

func (r *RandomPolicy) PickAction(step *core.StepContext, state core.State, actions []core.Action) core.Action {
	i := r.rand.Intn(len(actions))
	return actions[i]
}

func (r *RandomPolicy) UpdateStep(_ *core.StepContext, _ core.State, _ core.Action, _ core.State) {}

func (r *RandomPolicy) ResetEpisode(_ *core.EpisodeContext) {}

type RandomPolicyConstructor struct{}

func (r *RandomPolicyConstructor) NewPolicy() core.Policy {
	return NewRandomPolicy()
}

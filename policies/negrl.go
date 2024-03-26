package policies

import (
	"math"
	"time"

	"github.com/zeu5/dist-rl-testing/core"
	erand "golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/sampleuv"
)

// A fixed negative reward policy (-1) at all states
// The next action is chosen according to the softmax function
// With a temperature
type SoftMaxNegPolicy struct {
	QTable      map[string]map[string]float64
	Alpha       float64
	Gamma       float64
	Temperature float64

	rand erand.Source
}

// NewSoftMaxNegPolicy instantiated the SoftMaxNegPolicy
func NewSoftMaxNegPolicy(alpha, gamma, temperature float64) *SoftMaxNegPolicy {
	return &SoftMaxNegPolicy{
		QTable:      make(map[string]map[string]float64),
		Alpha:       alpha,
		Gamma:       gamma,
		Temperature: temperature,
		rand:        erand.NewSource(uint64(time.Now().UnixMilli())),
	}
}

// Checking interface compatibility
var _ core.Policy = &SoftMaxNegPolicy{}

func (s *SoftMaxNegPolicy) Reset() {
	s.QTable = make(map[string]map[string]float64)
	s.rand = erand.NewSource(uint64(time.Now().UnixMilli()))
}

// Reset clears the QTable
func (s *SoftMaxNegPolicy) ResetEpisode(_ *core.EpisodeContext) {
}

func (s *SoftMaxNegPolicy) UpdateEpisode(_ *core.EpisodeContext) {
	// if episode%200 == 0 {
	// 	fmt.Printf("Smallest value: %f\n", s.smallest)
	// }
}

func (s *SoftMaxNegPolicy) PickAction(step *core.StepContext, state core.State, actions []core.Action) core.Action {
	stateHash := state.Hash()

	if _, ok := s.QTable[stateHash]; !ok {
		s.QTable[stateHash] = make(map[string]float64)
	}

	// Initializing QTable entry to 0 if it does not exist
	for _, a := range actions {
		aName := a.Hash()
		if _, ok := s.QTable[stateHash][aName]; !ok {
			s.QTable[stateHash][aName] = 0
		}
	}

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))
	largestValue := s.QTable[stateHash][actions[0].Hash()]

	for i := 0; i < len(actions); i++ {
		action := actions[i]
		val := s.QTable[stateHash][action.Hash()]
		vals[i] = val
		if val > largestValue {
			largestValue = val
		}
	}

	// Normalizing
	for i := 0; i < len(vals); i++ {
		vals[i] = vals[i] - largestValue
		vals[i] = math.Exp(vals[i])
		sum += vals[i]
	}

	// Computing weights for each action
	for i, v := range vals {
		weights[i] = v / sum
	}
	// using the sampleuv library to sample based on the weights
	i, ok := sampleuv.NewWeighted(weights, s.rand).Take()
	if !ok {
		return nil
	}
	return actions[i]
}

func (s *SoftMaxNegPolicy) UpdateStep(sCtx *core.StepContext, state core.State, action core.Action, nextState core.State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := s.QTable[stateHash]; !ok {
		s.QTable[stateHash] = make(map[string]float64)
	}
	if _, ok := s.QTable[stateHash][actionKey]; !ok {
		s.QTable[stateHash][actionKey] = 0
	}
	curVal := s.QTable[stateHash][actionKey]
	max := float64(0)
	if _, ok := s.QTable[nextStateHash]; ok {
		for _, val := range s.QTable[nextStateHash] {
			if val > max {
				max = val
			}
		}
	}
	// the update with -1 reward
	nextVal := (1-s.Alpha)*curVal + s.Alpha*(-1+s.Gamma*max)
	s.QTable[stateHash][actionKey] = nextVal
}

type SoftMaxNegFreqPolicy struct {
	*SoftMaxNegPolicy
	Freq map[string]int
}

var _ core.Policy = &SoftMaxNegFreqPolicy{}

func NewSoftMaxNegFreqPolicy(alpha, gamma, temp float64) *SoftMaxNegFreqPolicy {
	return &SoftMaxNegFreqPolicy{
		SoftMaxNegPolicy: NewSoftMaxNegPolicy(alpha, gamma, temp),
		Freq:             make(map[string]int),
	}
}

func (t *SoftMaxNegFreqPolicy) UpdateStep(sCtx *core.StepContext, state core.State, action core.Action, nextState core.State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := t.QTable[stateHash]; !ok {
		t.QTable[stateHash] = make(map[string]float64)
	}
	if _, ok := t.QTable[stateHash][actionKey]; !ok {
		t.QTable[stateHash][actionKey] = 0
	}
	curVal := t.QTable[stateHash][actionKey]
	max := float64(0)
	if _, ok := t.QTable[nextStateHash]; ok {
		for _, val := range t.QTable[nextStateHash] {
			if val > max {
				max = val
			}
		}
	}
	if _, ok := t.Freq[nextStateHash]; !ok {
		t.Freq[nextStateHash] = 0
	}
	t.Freq[nextStateHash] += 1
	reward := float64(-1 * t.Freq[nextStateHash])

	nextVal := float64(0)

	// the update with -1 reward
	nextVal = (1-t.Alpha)*curVal + t.Alpha*(reward+t.Gamma*max)
	t.QTable[stateHash][actionKey] = nextVal
}

type SoftMaxNegFreqPolicyConstructor struct {
	alpha float64
	gamma float64
	temp  float64
}

var _ core.PolicyConstructor = &SoftMaxNegFreqPolicyConstructor{}

func NewSoftMaxNegFreqPolicyConstructor(alpha, gamma, temp float64) *SoftMaxNegFreqPolicyConstructor {
	return &SoftMaxNegFreqPolicyConstructor{
		alpha: alpha,
		gamma: gamma,
		temp:  temp,
	}
}

func (s *SoftMaxNegFreqPolicyConstructor) NewPolicy() core.Policy {
	return NewSoftMaxNegFreqPolicy(s.alpha, s.gamma, s.temp)
}

type SoftMaxNegPolicyConstructor struct {
	alpha float64
	gamma float64
	temp  float64
}

var _ core.PolicyConstructor = &SoftMaxNegPolicyConstructor{}

func NewSoftMaxNegPolicyConstructor(alpha, gamma, temp float64) *SoftMaxNegPolicyConstructor {
	return &SoftMaxNegPolicyConstructor{
		alpha: alpha,
		gamma: gamma,
		temp:  temp,
	}
}

func (s *SoftMaxNegPolicyConstructor) NewPolicy() core.Policy {
	return NewSoftMaxNegPolicy(s.alpha, s.gamma, s.temp)
}

package analysis

import (
	"path"
	"strconv"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/policies"
	"github.com/zeu5/dist-rl-testing/util"
)

type predicateDataset struct {
	FirstTimeStepToFinal int
	FirstEpisodeToFinal  int

	FinalPredicateStates    []int
	FinalPredicateTimesteps []int

	PredicateEpisodes  map[string]int
	PredicateTimesteps map[string]int
}

func (p *predicateDataset) Copy() *predicateDataset {
	return &predicateDataset{
		FirstTimeStepToFinal: p.FirstTimeStepToFinal,
		FirstEpisodeToFinal:  p.FirstEpisodeToFinal,

		FinalPredicateStates:    util.CopyIntSlice(p.FinalPredicateStates),
		FinalPredicateTimesteps: util.CopyIntSlice(p.FinalPredicateTimesteps),

		PredicateEpisodes:  util.CopyStringIntMap(p.PredicateEpisodes),
		PredicateTimesteps: util.CopyStringIntMap(p.PredicateTimesteps),
	}
}

type PredicateAnalyzer struct {
	predicates []policies.Predicate
	painter    core.Painter

	dataset      *predicateDataset
	finalStates  map[string]bool
	lastTimeStep int
}

var _ core.Analyzer = &PredicateAnalyzer{}

func NewPredicateAnalyzer(painter core.Painter, predicates ...policies.Predicate) *PredicateAnalyzer {
	out := &PredicateAnalyzer{
		predicates: append([]policies.Predicate{policies.Init}, predicates...),
		painter:    painter,
		dataset: &predicateDataset{
			PredicateEpisodes:       make(map[string]int),
			PredicateTimesteps:      make(map[string]int),
			FinalPredicateStates:    make([]int, 0),
			FinalPredicateTimesteps: make([]int, 0),
			FirstTimeStepToFinal:    -1,
			FirstEpisodeToFinal:     -1,
		},
		finalStates:  make(map[string]bool),
		lastTimeStep: 0,
	}

	for _, pred := range out.predicates {
		out.dataset.PredicateEpisodes[pred.Name] = 0
		out.dataset.PredicateTimesteps[pred.Name] = 0
	}
	return out
}

func (p *PredicateAnalyzer) Reset() {
	p.dataset = &predicateDataset{
		PredicateEpisodes:       make(map[string]int),
		PredicateTimesteps:      make(map[string]int),
		FinalPredicateStates:    make([]int, 0),
		FinalPredicateTimesteps: make([]int, 0),
		FirstTimeStepToFinal:    -1,
		FirstEpisodeToFinal:     -1,
	}
	for _, pred := range p.predicates {
		p.dataset.PredicateEpisodes[pred.Name] = 0
		p.dataset.PredicateTimesteps[pred.Name] = 0
	}
	p.finalStates = make(map[string]bool)
	p.lastTimeStep = 0
}

func (p *PredicateAnalyzer) Analyze(eCtx *core.EpisodeContext, trace *core.Trace) {
	curPredicate := 0
	targetReached := false
	// map of number of timesteps spent at each predicate
	predicateTimesteps := make(map[int]int)
	for i := 0; i < trace.Len(); i++ {
		step := trace.Step(i)
		state := step.State
		nextPredicate := curPredicate

		if !targetReached {
			for i, predicate := range p.predicates {
				if predicate.Check(state) {
					nextPredicate = i
				}
			}
		}

		// If we satisfy the final predicate
		if nextPredicate == len(p.predicates)-1 {
			targetReached = true
			// Update first step to final
			if p.dataset.FirstTimeStepToFinal == -1 {
				p.dataset.FirstTimeStepToFinal = p.lastTimeStep + i
			}
			if p.dataset.FirstEpisodeToFinal == -1 {
				p.dataset.FirstEpisodeToFinal = eCtx.Episode
			}

			// Update final states
			colorMap := make(map[int]string)
			for node, ns := range state.(*core.PartitionState).NodeStates {
				colorMap[node] = p.painter(ns).Hash()
			}
			stateHash := util.JsonHash(colorMap)
			if _, ok := p.finalStates[stateHash]; !ok {
				p.finalStates[stateHash] = true
			}
		}

		// update counter for how many steps spent in this predicate
		if _, ok := predicateTimesteps[curPredicate]; !ok {
			predicateTimesteps[curPredicate] = 0
		}
		predicateTimesteps[curPredicate]++

		curPredicate = nextPredicate
	}

	for predIndex, timesteps := range predicateTimesteps {
		pred := p.predicates[predIndex]

		p.dataset.PredicateEpisodes[pred.Name]++
		p.dataset.PredicateTimesteps[pred.Name] += timesteps
	}

	p.lastTimeStep += trace.Len()
	p.dataset.FinalPredicateStates = append(p.dataset.FinalPredicateStates, len(p.finalStates))
	p.dataset.FinalPredicateTimesteps = append(p.dataset.FinalPredicateTimesteps, p.lastTimeStep)
}

func (p *PredicateAnalyzer) DataSet() core.DataSet {
	return p.dataset.Copy()
}

type PredicateAnalyzerConstructor struct {
	Painter    core.Painter
	Predicates []policies.Predicate
}

var _ core.AnalyzerConstructor = &PredicateAnalyzerConstructor{}

func (p *PredicateAnalyzerConstructor) NewAnalyzer(_ string, _ int) core.Analyzer {
	return NewPredicateAnalyzer(p.Painter, p.Predicates...)
}

func NewPredicateAnalyzerConstructor(painter core.Painter, predicates []policies.Predicate) *PredicateAnalyzerConstructor {
	return &PredicateAnalyzerConstructor{
		Painter:    painter,
		Predicates: predicates,
	}
}

type PredicateComparator struct {
	savePath string
}

var _ core.Comparator = &PredicateComparator{}

func NewPredicateComparator(savePath string, hierarchyName string) *PredicateComparator {
	return &PredicateComparator{
		savePath: path.Join(savePath, "predicate_comparison_"+hierarchyName+".json"),
	}
}

func (p *PredicateComparator) Compare(experiments []string, datasets []core.DataSet) {
	// Write to file
	out := make(map[string]*predicateDataset)
	for i, name := range experiments {
		out[name] = datasets[i].(*predicateDataset)
	}

	util.SaveJson(p.savePath, out)
}

type PredicateComparatorConstructor struct {
	savePath      string
	hierarchyName string
}

var _ core.ComparatorConstructor = &PredicateComparatorConstructor{}

func (p *PredicateComparatorConstructor) NewComparator(run int) core.Comparator {
	return NewPredicateComparator(path.Join(p.savePath, strconv.Itoa(run)), p.hierarchyName)
}

func NewPredicateComparatorConstructor(hierarchyName string, savePath string) *PredicateComparatorConstructor {
	return &PredicateComparatorConstructor{
		savePath:      savePath,
		hierarchyName: hierarchyName,
	}
}

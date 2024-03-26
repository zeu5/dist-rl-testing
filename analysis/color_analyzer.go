package analysis

import (
	"path"
	"strconv"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
)

type colorAnalyzerDataset struct {
	Timesteps    []int
	UniqueStates []int
}

func (c *colorAnalyzerDataset) Copy() *colorAnalyzerDataset {
	return &colorAnalyzerDataset{
		Timesteps:    util.CopyIntSlice(c.Timesteps),
		UniqueStates: util.CopyIntSlice(c.UniqueStates),
	}
}

type ColorAnalyzer struct {
	painter core.Painter
	states  map[string]bool
	dataset *colorAnalyzerDataset
}

var _ core.Analyzer = &ColorAnalyzer{}

func NewColorAnalyzer(painter core.Painter) *ColorAnalyzer {
	return &ColorAnalyzer{
		painter: painter,
		states:  make(map[string]bool),
		dataset: &colorAnalyzerDataset{
			Timesteps:    make([]int, 0),
			UniqueStates: make([]int, 0),
		},
	}
}

func (c *ColorAnalyzer) Reset() {
	c.states = make(map[string]bool)
}

func (c *ColorAnalyzer) Analyze(eCtx *core.EpisodeContext, trace *core.Trace) {
	for i := 0; i < trace.Len(); i++ {
		step := trace.Step(i)
		ps := step.State.(*core.PartitionState)

		colorMap := make(map[int]string)
		for node, ns := range ps.NodeStates {
			colorMap[node] = c.painter(ns).Hash()
		}
		stateHash := util.JsonHash(colorMap)

		c.states[stateHash] = true
	}
	lastTimeStep := 0
	if len(c.dataset.Timesteps) > 0 {
		lastTimeStep = c.dataset.Timesteps[len(c.dataset.Timesteps)-1]
	}
	c.dataset.Timesteps = append(c.dataset.Timesteps, lastTimeStep+trace.Len())
	c.dataset.UniqueStates = append(c.dataset.UniqueStates, len(c.states))
}

func (c *ColorAnalyzer) DataSet() core.DataSet {
	return c.dataset.Copy()
}

type ColorAnalyzerConstructor struct {
	painter core.Painter
}

func NewCovertAnalyzerConstructor(painter core.Painter) *ColorAnalyzerConstructor {
	return &ColorAnalyzerConstructor{
		painter: painter,
	}
}

var _ core.AnalyzerConstructor = &ColorAnalyzerConstructor{}

func (c *ColorAnalyzerConstructor) NewAnalyzer(_ int) core.Analyzer {
	return NewColorAnalyzer(c.painter)
}

type ColorComparator struct {
	savePath string
}

var _ core.Comparator = &ColorComparator{}

func NewColorComparator(savePath string) *ColorComparator {
	return &ColorComparator{
		savePath: path.Join(savePath, "color_analyzer.json"),
	}
}

func (c *ColorComparator) Compare(experimentNames []string, datasets []core.DataSet) {
	out := make(map[string]*colorAnalyzerDataset)
	for i, name := range experimentNames {
		out[name] = datasets[i].(*colorAnalyzerDataset)
	}

	util.SaveJson(c.savePath, out)
}

type ColorComparatorConstructor struct {
	savePath string
}

var _ core.ComparatorConstructor = &ColorComparatorConstructor{}

func (c *ColorComparatorConstructor) NewComparator(run int) core.Comparator {
	return NewColorComparator(path.Join(c.savePath, strconv.Itoa(run)))
}

func NewCoverageComparatorConstructor(savePath string) *ColorComparatorConstructor {
	return &ColorComparatorConstructor{
		savePath: savePath,
	}
}

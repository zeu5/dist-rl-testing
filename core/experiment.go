package core

import "time"

type ParallelExperiment struct {
	Name        string
	Environment EnvironmentConstructor
	Policy      PolicyConstructor
}

type DataSet interface{}

type Analyzer interface {
	Analyze(*EpisodeContext, *Trace)
	DataSet() DataSet
	Reset()
}

type AnalyzerConstructor interface {
	// new analyzer based on experiment name and run
	NewAnalyzer(string, int) Analyzer
}

type Comparator interface {
	Compare([]string, []DataSet)
}

type ComparatorConstructor interface {
	NewComparator(int) Comparator
}

type ParallelComparison struct {
	Experiments []*ParallelExperiment
	Analyzers   map[string]AnalyzerConstructor
	Comparators map[string]ComparatorConstructor
}

type RunConfig struct {
	Episodes       int
	Horizon        int
	EpisodeTimeout time.Duration

	ThresholdConsecutiveErrors   int
	ThresholdConsecutiveTimeouts int
}

func NewParallelComparison() *ParallelComparison {
	return &ParallelComparison{
		Analyzers:   make(map[string]AnalyzerConstructor),
		Comparators: make(map[string]ComparatorConstructor),
		Experiments: make([]*ParallelExperiment, 0),
	}
}

func (c *ParallelComparison) AddExperiment(e *ParallelExperiment) {
	c.Experiments = append(c.Experiments, e)
}

func (c *ParallelComparison) AddAnalysis(name string, a AnalyzerConstructor, cmp ComparatorConstructor) {
	c.Analyzers[name] = a
	c.Comparators[name] = cmp
}

type Experiment struct {
	Name        string
	Environment Environment
	Policy      Policy
}

type Comparison struct {
	Experiments []*Experiment
	Analyzers   map[string]Analyzer
	Comparators map[string]Comparator
}

func NewComparison() *Comparison {
	return &Comparison{
		Analyzers:   make(map[string]Analyzer),
		Comparators: make(map[string]Comparator),
		Experiments: make([]*Experiment, 0),
	}
}

func (c *Comparison) AddExperiment(e *Experiment) {
	c.Experiments = append(c.Experiments, e)
}

func (c *Comparison) AddAnalysis(name string, a Analyzer, cmp Comparator) {
	c.Analyzers[name] = a
	c.Comparators[name] = cmp
}

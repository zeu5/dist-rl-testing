package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zeu5/dist-rl-testing/util"
)

var (
	ErrTooManyTimeouts = errors.New("too many timeouts")
	ErrTooManyErrors   = errors.New("too many errors")
)

type experimentRunContext struct {
	run       int
	ctx       context.Context
	cancel    context.CancelFunc
	analyzers map[string]Analyzer

	writer *util.ParallelOutput

	*RunConfig
}

type ExperimentResult struct {
	CompletedEpisodes    int
	TotalEpisodes        int
	ErrorEpisodes        int
	TimeoutEpisodes      int
	TotalTimeSteps       int
	BoundReachedEpisodes int

	Error    error
	Datasets map[string]DataSet
}

func (r *ExperimentResult) IsError() bool {
	return r.Error != nil
}

func (e *Experiment) run(ctx *experimentRunContext) *ExperimentResult {
	result := &ExperimentResult{
		Datasets: make(map[string]DataSet),
	}
	e.Policy.Reset()

	consecutiveErrors := 0
	consecutiveTimeouts := 0
	totalTimeSteps := (ctx.Episodes + 1) * ctx.Horizon
EpisodeLoop:
	for episode := 0; result.TotalTimeSteps <= totalTimeSteps; episode++ {
		select {
		case <-ctx.ctx.Done():
			result.Error = errors.New("context cancelled")
			break EpisodeLoop
		default:
		}

		ctx.writer.Set(fmt.Sprintf(
			"Experiment: %-40s, Run %d, Timesteps: %d/%d, Episode %d, Error: %d, Timedout: %d, OurOfBounds: %d",
			e.Name, ctx.run, result.TotalTimeSteps, totalTimeSteps, episode, result.ErrorEpisodes, result.TimeoutEpisodes, result.BoundReachedEpisodes,
		))
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx.ctx, ctx.EpisodeTimeout)
		eCtx := NewEpisodeContext(timeoutCtx)
		eCtx.Run = ctx.run
		eCtx.Episode = episode
		eCtx.StartTimeStep = result.TotalTimeSteps

		go func(eCtx *EpisodeContext) {
			defer func() {
				if r := recover(); r != nil {
					eCtx.Error(fmt.Errorf("errored in episode %d: %#v", eCtx.Episode, r))
				}
			}()

			state, err := e.Environment.Reset(eCtx)
			if err != nil {
				eCtx.Error(fmt.Errorf("error resetting the env: %s", err))
				return
			}
			e.Policy.ResetEpisode(eCtx)
			for step := 0; step < ctx.Horizon; step++ {
				select {
				case <-eCtx.Context.Done():
					eCtx.Error(fmt.Errorf("context error: %s", eCtx.Context.Err()))
					return
				default:
				}

				sCtx := &StepContext{Step: step, EpisodeContext: eCtx, AdditionalInfo: make(map[string]interface{})}
				action := e.Policy.PickAction(
					sCtx,
					state,
					state.Actions(),
				)
				nextState, err := e.Environment.Step(action, sCtx)
				if err != nil {
					eCtx.Error(fmt.Errorf("error executing step %d: %s", step, err))
					return
				}
				e.Policy.UpdateStep(sCtx, state, action, nextState)
				eCtx.Trace.AddStep(&Step{
					State:     state,
					Action:    action,
					NextState: nextState,
					Misc:      util.CopyStringMap(sCtx.AdditionalInfo),
				})
				state = nextState
			}
			e.Policy.UpdateEpisode(eCtx)
			eCtx.Finish()
		}(eCtx)

		errorred := false
		timedout := false
		select {
		case <-eCtx.Done():
			if eCtx.IsError() {
				if errors.Is(eCtx.err, ErrOutOfBounds) {
					result.BoundReachedEpisodes++
				} else {
					errorred = true
				}
			}
		case <-timeoutCtx.Done():
			timedout = true
		}
		timeoutCancel()

		if errorred {
			result.ErrorEpisodes++
			if consecutiveErrors++; consecutiveErrors >= ctx.ThresholdConsecutiveErrors {
				result.Error = ErrTooManyErrors
				break EpisodeLoop
			}
		} else {
			consecutiveErrors = 0
		}
		if timedout {
			result.TimeoutEpisodes++
			if consecutiveTimeouts++; consecutiveTimeouts >= ctx.ThresholdConsecutiveTimeouts {
				result.Error = ErrTooManyTimeouts
				break EpisodeLoop
			}
		} else {
			consecutiveTimeouts = 0
		}

		if !errorred && !timedout {
			result.TotalTimeSteps += eCtx.Trace.Len()
			result.CompletedEpisodes++
		}
		result.TotalEpisodes++

		for _, a := range ctx.analyzers {
			a.Analyze(eCtx, eCtx.Trace)
		}
	}
	if result.Error != nil {
		ctx.writer.TrySet(fmt.Sprintf("Experiment: %s, Run %d, Error: %v", e.Name, ctx.run, result.Error))
	}

	for name, a := range ctx.analyzers {
		result.Datasets[name] = a.DataSet()
	}

	e.Policy.Reset()
	return result
}

func (c *Comparison) Run(ctx context.Context, runs int, rConfig *RunConfig) {
	for run := 0; run < runs; run++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		results := make(map[string]*ExperimentResult)

		// Run experiments
		for _, e := range c.Experiments {
			select {
			case <-ctx.Done():
				return
			default:
			}
			ctx := &experimentRunContext{
				run:       run,
				ctx:       ctx,
				analyzers: make(map[string]Analyzer),
				RunConfig: rConfig,
			}

			for name, aC := range c.Analyzers {
				aC.Reset()
				ctx.analyzers[name] = aC
			}

			results[e.Name] = e.run(ctx)
		}

		// Gather datasets to run comparisons
		datasets := make(map[string][]DataSet)
		analyzerNames := make([]string, 0)
		for name := range c.Analyzers {
			analyzerNames = append(analyzerNames, name)
		}
		experimentNames := make([]string, 0)
		for name, result := range results {
			experimentNames = append(experimentNames, name)
			for _, name := range analyzerNames {
				if _, ok := datasets[name]; !ok {
					datasets[name] = make([]DataSet, 0)
				}
				if result.IsError() {
					datasets[name] = append(datasets[name], nil)
				} else {
					datasets[name] = append(datasets[name], result.Datasets[name])
				}
			}
		}
		for name, c := range c.Comparators {
			c.Compare(experimentNames, datasets[name])
		}
	}
}

// parallelWorker is a worker that runs experiments
type parallelWorker struct {
	id int
}

// parallelWork is a struct that contains all the information needed to run an experiment
type parallelWork struct {
	experiment *ParallelExperiment
	comp       *ParallelComparison
	runNumber  int
	writer     *util.ParallelOutput
	rConfig    *RunConfig
	wg         *sync.WaitGroup
}

// parallelResult is a struct that contains the result of running an experiment
type parallelResult struct {
	experimentName string
	run            int
	result         *ExperimentResult
}

// Worker main loop that consumes work from a channel
func (w *parallelWorker) run(ctx context.Context, workCh <-chan *parallelWork, resultsCh chan<- *parallelResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case work, more := <-workCh:
			if !more {
				return
			}
			result := w.runWork(ctx, work)
			resultsCh <- result
		}
	}
}

// Run an experiment by constructing the experiment context, *Experiment
func (w *parallelWorker) runWork(ctx context.Context, work *parallelWork) *parallelResult {
	eCtx, eCancel := context.WithCancel(ctx)
	eRunCtx := &experimentRunContext{
		run:       work.runNumber,
		ctx:       eCtx,
		cancel:    eCancel,
		analyzers: make(map[string]Analyzer),
		writer:    work.writer,
		RunConfig: work.rConfig,
	}

	for name, aC := range work.comp.Analyzers {
		eRunCtx.analyzers[name] = aC.NewAnalyzer(work.experiment.Name, w.id)
	}

	// Construct the experiment
	exp := &Experiment{
		Name:        work.experiment.Name,
		Environment: work.experiment.Environment.NewEnvironment(eCtx, w.id),
		Policy:      work.experiment.Policy.NewPolicy(),
	}

	// Run the experiment
	result := exp.run(eRunCtx)
	work.wg.Done()
	eCancel()

	return &parallelResult{
		experimentName: work.experiment.Name,
		run:            work.runNumber,
		result:         result,
	}
}

func (c *ParallelComparison) Run(ctx context.Context, runs int, rConfig *RunConfig, parallelism int) {
	for run := 0; run < runs; run++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Create workers and channels
		wg := new(sync.WaitGroup)
		writer := util.NewTerminalPrinter(1 * time.Second)
		writer.Start(ctx)
		writer.Write(fmt.Sprintf("Run %d\n", run))

		parallelism = util.MinInt(parallelism, len(c.Experiments))

		workCh := make(chan *parallelWork, parallelism)
		resultsCh := make(chan *parallelResult, parallelism)

		// Start workers
		workers := make([]*parallelWorker, parallelism)
		for i := 0; i < parallelism; i++ {
			workers[i] = &parallelWorker{id: i}
			go workers[i].run(ctx, workCh, resultsCh)
		}

		results := make(map[string]*ExperimentResult)

		// Gather results
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case result, more := <-resultsCh:
					if !more {
						return
					}
					results[result.experimentName] = result.result
				}
			}
		}()

		// Run experiments by sending work to workers
		for _, e := range c.Experiments {
			wg.Add(1)
			select {
			case <-ctx.Done():
				return
			case workCh <- &parallelWork{
				experiment: e,
				comp:       c,
				runNumber:  run,
				rConfig:    rConfig,
				wg:         wg,
				writer:     writer.NewOutput(),
			}:
			}
		}

		// Wait for all work to finish
		wg.Wait()
		close(resultsCh)
		close(workCh)
		writer.Stop()

		// Gather datasets to run comparisons
		datasets := make(map[string][]DataSet)
		analyzerNames := make([]string, 0)
		for name := range c.Analyzers {
			analyzerNames = append(analyzerNames, name)
		}
		experimentNames := make([]string, 0)
		for name, result := range results {
			experimentNames = append(experimentNames, name)
			for _, name := range analyzerNames {
				if _, ok := datasets[name]; !ok {
					datasets[name] = make([]DataSet, 0)
				}
				if result.IsError() {
					datasets[name] = append(datasets[name], nil)
				} else {
					datasets[name] = append(datasets[name], result.Datasets[name])
				}
			}
		}
		for name, c := range c.Comparators {
			select {
			case <-ctx.Done():
				return
			default:
			}
			c.NewComparator(run).Compare(experimentNames, datasets[name])
		}
	}

}

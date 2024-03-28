package analysis

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/zeu5/dist-rl-testing/core"
)

type PrintDebugAnalyzer struct {
	// savePath is the path to save the trace
	savePath string
	exp      string
	// will save the trace to the file only after the episode number exceeds this threshold
	thresholdEpisode int
}

var _ core.Analyzer = &PrintDebugAnalyzer{}

func NewPrintDebugAnalyzer(savePath string, threshold int) *PrintDebugAnalyzer {
	// create a traces directory under save path
	os.MkdirAll(path.Join(savePath, "traces"), 0755)
	return &PrintDebugAnalyzer{
		savePath:         path.Join(savePath, "traces"),
		thresholdEpisode: threshold,
	}
}

func (a *PrintDebugAnalyzer) Analyze(ctx *core.EpisodeContext, trace *core.Trace) {
	if ctx.Episode < a.thresholdEpisode {
		return
	}
	buf := new(bytes.Buffer)

	for i := 0; i < trace.Len(); i++ {
		step := trace.Step(i)
		buf.WriteString(fmt.Sprintf("Step %d\n%s\n", i, stepToString(step.State, step.Action, step.NextState)))
		buf.WriteString("\n")
	}
	fileName := fmt.Sprintf("%d_trace_%d.txt", ctx.Run, ctx.Episode)
	if a.exp != "" {
		fileName = fmt.Sprintf("%d_%s_trace_%d.txt", ctx.Run, a.exp, ctx.Episode)
	}
	file := path.Join(a.savePath, fileName)
	os.WriteFile(file, buf.Bytes(), 0644)
}

func stepToString(state core.State, action core.Action, nextState core.State) string {
	return fmt.Sprintf("State: \n%s\nAction: %s\n\nNext State: \n%s\n", stateToString(state), actionToString(action), stateToString(nextState))
}

func stateToString(state core.State) string {
	ps, ok := state.(*core.PartitionState)
	if !ok {
		return "not a partition state"
	}
	out := ""
	for i, c := range ps.NodeColors {
		switch color := c.(type) {
		case *core.InActiveColor:
			out += fmt.Sprintf("%d: Inactive\n", i)
		case *core.ComposedColor:
			cs := "{ "
			for j, v := range color.Map() {
				cs += fmt.Sprintf("%s: %#v, ", j, v)
			}
			cs += "}"
			out += fmt.Sprintf("%d: %s\n", i, cs)
		}
	}
	return out
}

func actionToString(action core.Action) string {
	switch a := action.(type) {
	case *core.ChangePartitionAction:
		return "ChangePartitionAction"
	case *core.StaySamePartitionAction:
		return "StaySamePartitionAction"
	case *core.StartAction:
		return "StartAction"
	case *core.StopAction:
		return fmt.Sprintf("StopAction: %s", a.Color)
	case *core.RequestAction:
		return "RequestAction"
	default:
		return "Unknown Action"
	}
}

func (a *PrintDebugAnalyzer) DataSet() core.DataSet {
	return nil
}

func (a *PrintDebugAnalyzer) Reset() {
	// do nothing
}

type PrintDebugAnalyzerConstructor struct {
	SavePath         string
	ThresholdEpisode int
}

var _ core.AnalyzerConstructor = &PrintDebugAnalyzerConstructor{}

func NewPrintDebugAnalyzerConstructor(savePath string, thresholdEpisode int) *PrintDebugAnalyzerConstructor {
	return &PrintDebugAnalyzerConstructor{
		SavePath:         savePath,
		ThresholdEpisode: thresholdEpisode,
	}
}

func (c *PrintDebugAnalyzerConstructor) NewAnalyzer(exp string, _ int) core.Analyzer {
	os.MkdirAll(path.Join(c.SavePath, "traces"), 0755)
	return &PrintDebugAnalyzer{
		savePath:         path.Join(c.SavePath, "traces"),
		exp:              exp,
		thresholdEpisode: c.ThresholdEpisode,
	}
}

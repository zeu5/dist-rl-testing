package analysis

import (
	"bytes"
	"encoding/json"
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
	// create a traces directory under save path if not exists
	if _, err := os.Stat(path.Join(savePath, "traces")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(savePath, "traces"), 0755)
	}
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
		buf.WriteString(fmt.Sprintf("Step %d\n%s\n", i, stepToString(step)))
		buf.WriteString("\n")
	}
	fileName := fmt.Sprintf("%d_trace_%d.txt", ctx.Run, ctx.Episode)
	if a.exp != "" {
		fileName = fmt.Sprintf("%d_%s_trace_%d.txt", ctx.Run, a.exp, ctx.Episode)
	}
	file := path.Join(a.savePath, fileName)
	os.WriteFile(file, buf.Bytes(), 0644)
}

func stepToString(step *core.Step) string {
	return fmt.Sprintf(
		"State: \n%s\nAction: %s\n\nNext State: \n%s\nAdditional Info:\n%s\n",
		stateToString(step.State),
		actionToString(step.Action),
		stateToString(step.NextState),
		addInfoToString(step.Misc),
	)
}

func addInfoToString(addInfo map[string]interface{}) string {
	out := ""
	if _, ok := addInfo["messages_delivered"]; ok {

		messagesDelivered := addInfo["messages_delivered"].(int)
		messagesDropped := addInfo["messages_dropped"].(int)

		out += fmt.Sprintf("Messages Delivered: %d\nMessages Dropped: %d\n", messagesDelivered, messagesDropped)
	}
	if _, ok := addInfo["predicate_state"]; ok {
		out += fmt.Sprintf("Predicate State: %s\n", addInfo["predicate_state"].(string))
	}
	return out
}

func stateToString(state core.State) string {
	ps, ok := state.(*core.PartitionState)
	if !ok {
		return "not a partition state"
	}
	out := ""
	partitionMap := make(map[int][]int)
	for node, p := range ps.NodePartitions {
		if _, ok := partitionMap[p]; !ok {
			partitionMap[p] = make([]int, 0)
		}
		partitionMap[p] = append(partitionMap[p], node)
	}
	partString := "["
	for i := 0; i < len(partitionMap); i++ {
		partString += fmt.Sprintf(" [%v] ", partitionMap[i])
	}
	partString += "]"
	out += fmt.Sprintf("Partition: %s\n", partString)
	for i := 1; i <= ps.NumNodes; i++ {
		c := ps.NodeColors[i]
		switch color := c.(type) {
		case *core.InActiveColor:
			out += fmt.Sprintf("%d: Inactive\n", i)
		case *core.ComposedColor:

			bs, _ := json.Marshal(color.Map())
			out += fmt.Sprintf("%d: {%s}\n", i, string(bs))
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
	if _, err := os.Stat(path.Join(c.SavePath, "traces")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(c.SavePath, "traces"), 0755)
	}
	return &PrintDebugAnalyzer{
		savePath:         path.Join(c.SavePath, "traces"),
		exp:              exp,
		thresholdEpisode: c.ThresholdEpisode,
	}
}

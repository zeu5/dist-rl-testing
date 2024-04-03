package analysis

import (
	"fmt"
	"os"
	"path"

	"github.com/zeu5/dist-rl-testing/core"
)

type BugSpec struct {
	Name  string
	Check func(*core.Trace) bool
}

type BugAnalyzer struct {
	bugs     []BugSpec
	savePath string
	exp      string
}

var _ core.Analyzer = &BugAnalyzer{}

func NewBugAnalyzer(savePath string, bugs ...BugSpec) *BugAnalyzer {
	if _, err := os.Stat(path.Join(savePath, "bugs")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(savePath, "bugs"), 0755)
	}
	return &BugAnalyzer{
		bugs:     bugs,
		savePath: path.Join(savePath, "bugs"),
	}
}

func (ba *BugAnalyzer) Analyze(eCtx *core.EpisodeContext, trace *core.Trace) {
	for _, bug := range ba.bugs {
		if bug.Check(trace) {
			traceString := traceToString(trace)
			fileName := path.Join(ba.savePath, fmt.Sprintf("%d_%s_bug_%d.txt", eCtx.Run, bug.Name, eCtx.Episode))
			if ba.exp != "" {
				fileName = path.Join(ba.savePath, fmt.Sprintf("%d_%s_%s_bug_%d.txt", eCtx.Run, ba.exp, bug.Name, eCtx.Episode))
			}

			os.WriteFile(fileName, []byte(traceString), 0644)
		}
	}
}

func (*BugAnalyzer) DataSet() core.DataSet {
	return nil
}

func (*BugAnalyzer) Reset() {}

type BugAnalyzerConstructor struct {
	SavePath string
	Bugs     []BugSpec
}

var _ core.AnalyzerConstructor = &BugAnalyzerConstructor{}

func NewBugAnalyzerConstructor(savePath string, bugs ...BugSpec) *BugAnalyzerConstructor {
	return &BugAnalyzerConstructor{
		SavePath: savePath,
		Bugs:     bugs,
	}
}

func (e *BugAnalyzerConstructor) NewAnalyzer(exp string, _ int) core.Analyzer {
	if _, err := os.Stat(path.Join(e.SavePath, "bugs")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(e.SavePath, "bugs"), 0755)
	}
	return &BugAnalyzer{
		savePath: path.Join(e.SavePath, "bugs"),
		exp:      exp,
		bugs:     e.Bugs,
	}
}

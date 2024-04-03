package analysis

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/zeu5/dist-rl-testing/core"
	"github.com/zeu5/dist-rl-testing/util"
)

type ErrorAnalyzer struct {
	savePath string
	exp      string
}

var _ core.Analyzer = &ErrorAnalyzer{}

func NewErrorAnalyzer(savePath string) *ErrorAnalyzer {
	if _, err := os.Stat(path.Join(savePath, "errors")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(savePath, "errors"), 0755)
	}
	return &ErrorAnalyzer{
		savePath: path.Join(savePath, "errors"),
	}
}

func (a *ErrorAnalyzer) Analyze(ctx *core.EpisodeContext, trace *core.Trace) {
	buf := new(bytes.Buffer)
	err := trace.Error()
	if err == nil {
		return
	}
	if logError, ok := err.(*util.LogError); ok {
		buf.WriteString(fmt.Sprintf("Error: %s\nLogs:\n%s\n", logError.Err, logError.Logs))
	} else {
		buf.WriteString(fmt.Sprintf("Error: %s\n", err))
	}

	buf.WriteString(traceToString(trace))

	fileName := fmt.Sprintf("%d_error_%d.txt", ctx.Run, ctx.Episode)
	if a.exp != "" {
		fileName = fmt.Sprintf("%d_%s_error_%d.txt", ctx.Run, a.exp, ctx.Episode)
	}
	file := path.Join(a.savePath, fileName)
	os.WriteFile(file, buf.Bytes(), 0644)
}

func (a *ErrorAnalyzer) DataSet() core.DataSet {
	return nil
}

func (a *ErrorAnalyzer) Reset() {
	// do nothing
}

type ErrorAnalyzerConstructor struct {
	SavePath string
}

var _ core.AnalyzerConstructor = &ErrorAnalyzerConstructor{}

func NewErrorAnalyzerConstructor(savePath string) *ErrorAnalyzerConstructor {
	return &ErrorAnalyzerConstructor{
		SavePath: savePath,
	}
}

func (e *ErrorAnalyzerConstructor) NewAnalyzer(exp string, _ int) core.Analyzer {
	if _, err := os.Stat(path.Join(e.SavePath, "errors")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(e.SavePath, "errors"), 0755)
	}
	return &ErrorAnalyzer{
		savePath: path.Join(e.SavePath, "errors"),
		exp:      exp,
	}
}

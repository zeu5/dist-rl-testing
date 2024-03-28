package util

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gosuri/uilive"
)

type TerminalPrinter struct {
	parallelOutputs []*ParallelOutput
	frequency       time.Duration
	doneCh          chan struct{}

	writer  *uilive.Writer
	writers []io.Writer
}

func NewTerminalPrinter(frequency time.Duration) *TerminalPrinter {
	return &TerminalPrinter{
		parallelOutputs: make([]*ParallelOutput, 0),
		frequency:       frequency,
		doneCh:          make(chan struct{}),

		writer:  uilive.New(),
		writers: make([]io.Writer, 0),
	}
}

func (t *TerminalPrinter) NewOutput() *ParallelOutput {
	out := NewParallelOutput()
	t.parallelOutputs = append(t.parallelOutputs, out)
	t.writers = append(t.writers, t.writer.Newline())
	return out
}

func (p *TerminalPrinter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-p.doneCh:
				p.writer.Stop()
				return
			case <-ctx.Done():
				p.writer.Stop()
				return
			case <-time.After(p.frequency):
				p.print()
			}
		}
	}()
}

func (p *TerminalPrinter) Stop() {
	close(p.doneCh)
}

func (p *TerminalPrinter) Write(out string) {
	fmt.Fprintf(p.writer, "%s", out)
	p.writer.Flush()
}

func (p *TerminalPrinter) print() {
	// fmt.Printf("Printer print start\n")
	for i, output := range p.parallelOutputs {
		s := output.Get()
		// fmt.Printf("Writing: %s\n", s)
		fmt.Fprint(p.writers[i], s+"\n")
	}
	p.writer.Flush()
}

// PARALLEL OUTPUT
// used to update and print experiment outputs
type ParallelOutput struct {
	mu        *sync.Mutex
	printable string
}

func NewParallelOutput() *ParallelOutput {
	return &ParallelOutput{
		mu:        new(sync.Mutex),
		printable: "",
	}
}

// Set the output string (blocking)
func (p *ParallelOutput) Set(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printable = s
}

// Try to set the output string (non-blocking)
func (p *ParallelOutput) TrySet(s string) bool {
	success := p.mu.TryLock()
	if success {
		defer p.mu.Unlock()
		p.printable = s
		// fmt.Printf("output set to: %s\n", p.printable)
		return true
	}
	return false
}

// Get the output string (blocking)
func (p *ParallelOutput) Get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.printable
}

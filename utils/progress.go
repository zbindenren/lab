package utils

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// Progress draws a "done/total" step counter on stderr. Without a terminal on
// stderr it stays silent, so piped and redirected runs keep clean output.
type Progress struct {
	mu    sync.Mutex
	out   io.Writer
	total int
	done  int
	quiet bool
}

// NewProgress creates a counter for total steps.
func NewProgress(total int) *Progress {
	out := os.Stderr

	return &Progress{out: out, total: total, quiet: !term.IsTerminal(int(out.Fd()))}
}

// Step counts one finished step and redraws the counter with the step name.
func (p *Progress) Step(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.done++

	if p.quiet {
		return
	}

	// Redraw in place; \033[K wipes the longer name of the previous step.
	fmt.Fprintf(p.out, "\r\033[K%d/%d %s", p.done, p.total, name)
}

// Done ends the running line so subsequent output starts on a fresh one.
func (p *Progress) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.quiet {
		return
	}

	fmt.Fprintln(p.out)
}

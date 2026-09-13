package puff

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// progressTracker renders a single-line progress bar for multi-step flows
// like `puff upd`. Each completed step calls tick; the bar redraws in place.
// It is safe for concurrent use.
type progressTracker struct {
	w         io.Writer
	total     int
	completed int
	mu        sync.Mutex
	width     int
	label     string
}

// newProgressTracker starts a bar with the given total steps and label.
func newProgressTracker(w io.Writer, total int, label string) *progressTracker {
	return &progressTracker{w: w, total: total, label: label, width: 30}
}

// tick advances the bar by one completed step and redraws it.
func (p *progressTracker) tick() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed++
	p.draw()
}

// finish marks the tracker finished and moves to a new line.
func (p *progressTracker) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed = p.total
	p.draw()
	fmt.Fprintln(p.w)
}

// draw rewrites the current line with the current progress.
func (p *progressTracker) draw() {
	if p.total <= 0 {
		return
	}
	percent := float64(p.completed) / float64(p.total)
	if percent > 1 {
		percent = 1
	}
	filled := int(float64(p.width) * percent)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	fmt.Fprintf(p.w, "\r%s %s %3.0f%%", Cyan(p.w, bar), Dim(p.w, p.label), percent*100)
}
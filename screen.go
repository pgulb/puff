package puff

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
)

// screen renders a region of stdout in place, like a mini-TUI: a progress
// bar followed by per-repo status lines, all rewritten on every update. When
// the writer is not a terminal it falls back to plain line-by-line output with
// no bar and no escape codes, so piped output stays clean.
type screen struct {
	w         io.Writer
	isTTY     bool
	mu        sync.Mutex
	total     int
	completed int
	lines     []string
}

func newScreen(w io.Writer) *screen {
	return &screen{w: w, isTTY: ttyOf(w)}
}

func ttyOf(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	var st syscall.Stat_t
	return syscall.Fstat(int(f.Fd()), &st) == nil && (st.Mode & syscall.S_IFMT) == syscall.S_IFCHR
}

// begin saves the current cursor position; every subsequent redraw reprints
// the region below it.
func (s *screen) begin() {
	if !s.isTTY {
		return
	}
	fmt.Fprint(s.w, "\0337")
}

// tick advances the progress counter by one completed step and redraws.
func (s *screen) tick() {
	s.mu.Lock()
	s.completed++
	s.drawLocked()
	s.mu.Unlock()
}

// line appends a status line and redraws. In non-TTY mode it prints the line
// directly.
func (s *screen) line(format string, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isTTY {
		s.lines = append(s.lines, fmt.Sprintf(format, args...))
		s.drawLocked()
		return
	}
	fmt.Fprintln(s.w, fmt.Sprintf(format, args...))
}

// finish redraws at 100%, prints a trailing newline, and releases the region
// so subsequent output prints normally below it.
func (s *screen) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed = s.total
	s.drawLocked()
	if s.isTTY {
		fmt.Fprintln(s.w)
	}
}

func (s *screen) drawLocked() {
	if !s.isTTY {
		return
	}
	// Restore the saved cursor and clear everything below it, then reprint
	// the whole region so the display never scrolls.
	fmt.Fprint(s.w, "\0338\033[0J")
	if s.total > 0 {
		percent := float64(s.completed) / float64(s.total)
		if percent > 1 {
			percent = 1
		}
		filled := int(float64(30) * percent)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 30-filled)
		fmt.Fprintf(s.w, "%s %s %3.0f%%\n", Cyan(s.w, bar), Dim(s.w, "updating binaries"), percent*100)
	}
	for _, l := range s.lines {
		fmt.Fprintln(s.w, l)
	}
}
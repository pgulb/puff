package puff

import (
	"fmt"
	"io"
	"strings"
	"syscall"
)

// ANSI escape codes used for styling terminal output.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiCyan    = "\033[36m"
)

// Box-drawing glyphs used for bordered tables. When color is disabled these
// still render as clean ASCII, so piped output stays readable.
const (
	tl = "┌"
	tr = "┐"
	bl = "└"
	br = "┘"
	ml = "├"
	mr = "┤"
	mt = "┬"
	mb = "┴"
	mx = "┼"
	h  = "─"
	v  = "│"
)

// colorEnabled reports whether the given writer is a terminal and so can
// interpret ANSI escape codes. Output is uncolored when piped or redirected.
func colorEnabled(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	var st syscall.Stat_t
	// EAGAIN is harmless here; we only care about whether the fd is a terminal.
	return syscall.Fstat(int(f.Fd()), &st) == nil && (st.Mode & syscall.S_IFMT) == syscall.S_IFCHR
}

// styled wraps text in ANSI codes when color is enabled for w, otherwise
// returns the text unchanged.
func styled(w io.Writer, code, text string) string {
	if !colorEnabled(w) {
		return text
	}
	return code + text + ansiReset
}

// Bold prints text in bold when the writer is a terminal.
func Bold(w io.Writer, text string) string {
	return styled(w, ansiBold, text)
}

// Dim prints text in dim style when the writer is a terminal.
func Dim(w io.Writer, text string) string {
	return styled(w, ansiDim, text)
}

// Green prints text in green when the writer is a terminal.
func Green(w io.Writer, text string) string {
	return styled(w, ansiGreen, text)
}

// Red prints text in red when the writer is a terminal.
func Red(w io.Writer, text string) string {
	return styled(w, ansiRed, text)
}

// Yellow prints text in yellow when the writer is a terminal.
func Yellow(w io.Writer, text string) string {
	return styled(w, ansiYellow, text)
}

// Cyan prints text in cyan when the writer is a terminal.
func Cyan(w io.Writer, text string) string {
	return styled(w, ansiCyan, text)
}

// padRight pads text to width with spaces on the right.
func padRight(text string, width int) string {
	if len(text) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-len(text))
}

// padLeft pads text to width with spaces on the left.
func padLeft(text string, width int) string {
	if len(text) >= width {
		return text
	}
	return strings.Repeat(" ", width-len(text)) + text
}

// truncate shortens text to at most width characters, appending an ellipsis
// when truncation occurs.
func truncate(text string, width int) string {
	if len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "…"
}

// renderTable writes a bordered table to w. Headers are bolded and separated
// from the body by a horizontal rule. Column widths are fixed.
func renderTable(w io.Writer, headers []string, widths []int, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	// Top border.
	fmt.Fprint(w, tl)
	for i, width := range widths {
		fmt.Fprint(w, strings.Repeat(h, width+2))
		if i < len(widths)-1 {
			fmt.Fprint(w, mt)
		}
	}
	fmt.Fprintln(w, tr)

	// Header row.
	fmt.Fprint(w, v)
	for i, header := range headers {
		fmt.Fprint(w, " " + Bold(w, padRight(header, widths[i])) + " " + v)
	}
	fmt.Fprintln(w)

	// Header/body separator.
	fmt.Fprint(w, ml)
	for i, width := range widths {
		fmt.Fprint(w, strings.Repeat(h, width+2))
		if i < len(widths)-1 {
			fmt.Fprint(w, mx)
		}
	}
	fmt.Fprintln(w, mr)

	// Body rows.
	for _, row := range rows {
		fmt.Fprint(w, v)
		for i := 0; i < len(headers); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Fprint(w, " " + padRight(cell, widths[i]) + " " + v)
		}
		fmt.Fprintln(w)
	}

	// Bottom border.
	fmt.Fprint(w, bl)
	for i, width := range widths {
		fmt.Fprint(w, strings.Repeat(h, width+2))
		if i < len(widths)-1 {
			fmt.Fprint(w, mb)
		}
	}
	fmt.Fprintln(w, br)
}

// renderDynamicTable writes a bordered table with column widths computed from
// the widest cell in each column (plus a minimum width). Headers are bolded.
func renderDynamicTable(w io.Writer, headers []string, rows [][]string, minWidths ...int) {
	if len(rows) == 0 {
		return
	}
	widths := make([]int, len(headers))
	for i := range widths {
		if i < len(minWidths) {
			widths[i] = minWidths[i]
		} else {
			widths[i] = 8
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	renderTable(w, headers, widths, rows)
}
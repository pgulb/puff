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

// renderTable writes rows to w with each column padded to the given widths and
// separated by two spaces. The header row (if any) is bolded and followed by a
// rule line.
func renderTable(w io.Writer, headers []string, widths []int, rows [][]string) {
	if len(headers) > 0 {
		fmt.Fprint(w, Bold(w, padRight(headers[0], widths[0])))
		for i := 1; i < len(headers); i++ {
			fmt.Fprint(w, "  "+Bold(w, padRight(headers[i], widths[i])))
		}
		fmt.Fprintln(w)
		rule := ""
		for i, width := range widths {
			if i > 0 {
				rule += "  "
			}
			rule += strings.Repeat("─", width)
		}
		fmt.Fprintln(w, Dim(w, rule))
	}
	for _, row := range rows {
		fmt.Fprint(w, padRight(row[0], widths[0]))
		for i := 1; i < len(row); i++ {
			fmt.Fprint(w, "  "+padRight(row[i], widths[i]))
		}
		fmt.Fprintln(w)
	}
}

// renderDynamicTable writes rows with column widths computed from the widest
// cell in each column (plus a minimum width). Headers are bolded.
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
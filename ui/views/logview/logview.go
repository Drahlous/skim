package logview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"skim/filterfiles"
	"strings"
)

type LogView struct {
	Cursor int // which log line our cursor is pointing at
	Table  table.Model
	Lines  []string
}

func (v *LogView) Toggle() {
	return
}

func (v *LogView) CursorUp() int {
	if v.Cursor > 0 {
		v.Cursor--
	}
	return v.Cursor
}

func (v *LogView) CursorDown() int {
	if v.Cursor < v.GetMaxCursor() {
		v.Cursor++
	}
	return v.Cursor
}

func (v *LogView) CursorLeft() int {
	return 0
}

func (v *LogView) CursorRight() int {
	return 0
}

func (v *LogView) GetMaxCursor() int {
	return len(v.Lines) - 1
}

// FindNext returns the index of the next line, after Cursor and wrapping
// around to the start, whose text matches re. It scans the full line set
// regardless of any active filters, so it can find a match even while that
// line is currently hidden by hideUnmatched.
func (v *LogView) FindNext(re regexp.Regexp) (int, bool) {
	n := len(v.Lines)
	if n == 0 {
		return 0, false
	}
	for i := 1; i <= n; i++ {
		idx := (v.Cursor + i) % n
		if re.MatchString(v.Lines[idx]) {
			return idx, true
		}
	}
	return 0, false
}

// FindPrev is FindNext in reverse: it returns the index of the previous
// line, before Cursor and wrapping around to the end, whose text matches re.
func (v *LogView) FindPrev(re regexp.Regexp) (int, bool) {
	n := len(v.Lines)
	if n == 0 {
		return 0, false
	}
	for i := 1; i <= n; i++ {
		idx := ((v.Cursor-i)%n + n) % n
		if re.MatchString(v.Lines[idx]) {
			return idx, true
		}
	}
	return 0, false
}

var logStyle = lipgloss.NewStyle().
	Bold(false).
	Foreground(lipgloss.Color("#000000")).
	PaddingTop(0).
	PaddingLeft(0)

func (v *LogView) MakeTable(windowWidth int, windowHeight int, filters []filterfiles.Filter, hideUnmatched bool) table.Model {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Line", Width: windowWidth - 10}, // TODO: Avoid hardcoding this offset
	}

	rows := []table.Row{}

	// cursorRow tracks the position, within the visible rows actually
	// rendered, of the last row at or before v.Cursor's line index. v.Cursor
	// indexes into the full (unfiltered) line set, so when hideUnmatched
	// drops lines, a raw row-count of v.Cursor would overshoot; this keeps
	// the highlighted row aligned with the log line the cursor logically
	// points at, even when lines before it are hidden.
	cursorRow := 0

	for i, line := range v.Lines {
		// +1 Offset to make the first line number 1
		lineNumber := i + 1

		// Replace tabs with spaces
		line = strings.ReplaceAll(line, "\t", "    ")

		// Do any filters match this line?
		filter, match := filterfiles.GetMatchingFilter(filters, line)
		if match == true {

			// Style this log line with the color from the filter
			style := logStyle
			style.Background(lipgloss.Color(filter.BackColor))

			row := table.Row{fmt.Sprintf("%d", lineNumber), style.Render(line)}
			rows = append(rows, row)
		} else if !hideUnmatched {
			row := table.Row{fmt.Sprintf("%d", lineNumber), line}
			rows = append(rows, row)
		} else {
			continue
		}

		if i <= v.Cursor {
			cursorRow = len(rows) - 1
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		// TODO: Replace hardcoded offset with the size of the filter section
		table.WithHeight(windowHeight-12),
	)

	// Move the view to the visible row corresponding to the log cursor
	t.MoveDown(cursorRow)

	v.Table = t
	return t
}

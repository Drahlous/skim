package logview

import (
	"example/user/skim/filterfiles"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
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

func (v *LogView) GetMaxCursor() int {
	return len(v.Lines) - 1
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
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		// TODO: Replace hardcoded offset with the size of the filter section
		table.WithHeight(windowHeight-12),
	)

	// Move the view to the location of the log cursor
	t.MoveDown(v.Cursor)

	v.Table = t
	return t
}

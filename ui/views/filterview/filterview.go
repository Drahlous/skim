package filterview

import (
	"example/user/skim/filterfiles"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type FilterView struct {
	Cursor  int // which filter our cursor is pointing at
	Table   table.Model
	Filters []filterfiles.Filter
}

func (v *FilterView) Toggle() {
	filter := &v.Filters[v.Cursor]
	filter.IsEnabled = !filter.IsEnabled
}

func (v *FilterView) CursorUp() int {
	if v.Cursor > 0 {
		v.Cursor--
	}
	return v.Cursor
}

func (v *FilterView) CursorDown() int {
	if v.Cursor < v.GetMaxCursor() {
		v.Cursor++
	}
	return v.Cursor
}

func (v *FilterView) GetMaxCursor() int {
	return len(v.Filters) - 1
}

var filterStyle = lipgloss.NewStyle().
	Bold(false).
	Foreground(lipgloss.Color("#000000")).
	PaddingTop(0).
	PaddingLeft(0)

func (v *FilterView) MakeTable(windowWidth int, windowHeight int, filters []filterfiles.Filter) table.Model {
	columns := []table.Column{
		{Title: "", Width: 3},
		{Title: "Regex", Width: windowWidth - 9}, // TODO: Avoid hardcoding this offset
	}

	rows := []table.Row{}

	// Iterate over filters
	for i, filter := range filters {

		// Is this filter enabled?
		checked := " " // not selected
		if filters[i].IsEnabled {
			checked = "x" // this item is selected
		}

		style := filterStyle
		style.Background(lipgloss.Color(filter.BackColor))

		// Render the row
		row := table.Row{fmt.Sprintf("[%s]", checked), style.Render(filter.XML.Text)}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(5),
	)

	t.MoveDown(v.Cursor)

	v.Table = t
	return t
}

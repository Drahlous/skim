package ui

import (
	"bufio"
	"example/user/skim/filterfiles"
	"fmt"
	"os"

	// We'll shorten the package name to "tea" for ease of use
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model to store the application's state
type model struct {
	cursor  int // which filter our cursor is pointing at
	filters []filterfiles.Filter
	lines   []string
	table   table.Model
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// Define the initial state for the application
func initialModel(filters []filterfiles.Filter, scanner *bufio.Scanner) model {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return model{
		filters: filters,
		lines:   lines,
	}
}

// Now we'll define the Init method.
// Init can return a Cmd that might perform some initial I/O.
// For now, we don't need to do any I/O, so we'll return nil meaning "no command".
func (m model) Init() tea.Cmd {
	return nil
}

// The Update method is called when "things happen".
// It updates the model (state) in response to events.
// Update can also return a Cmd to make more things happen.
//
// In this example, we're moving the cursor when the user presses an arrow key.
//
// The "something happened" comes in the form of a Msg, which can be any type.
// Messages are the result of some I/O that took place, such as a keypress, timer tick, or server response.
// The "tea.KeyMsg" messages are automatically sent when keys are pressed.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Is it a key press?
	case tea.KeyMsg:

		// Which key was pressed?
		switch msg.String() {

		// These keys will exit the program
		case "ctrl+c", "q":
			return m, tea.Quit

		// "up" and "k" move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// Move the cursor down
		case "down", "j":
			if m.cursor < len(m.filters)-1 {
				m.cursor++
			}

			// Enter and spacebar toggle the selected state for the item under the cursor
		case "enter", " ":
			filter := &m.filters[m.cursor]
			filter.IsEnabled = !filter.IsEnabled
		}
	}
	return m, nil
}

func makeFilteredTable(m model) table.Model {

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Line", Width: 100},
	}

	rows := []table.Row{}

	for i, line := range m.lines {
		// Do any filters match this line?
		filter, match := filterfiles.GetMatchingFilter(m.filters, line)
		if match == true {

			// Style this log line with the color from the filter
			var style = lipgloss.NewStyle().
				Bold(false).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color(filter.BackColor)).
				PaddingTop(0).
				PaddingLeft(0)

			// +1 Offset to make the first line number 1
			row := table.Row{fmt.Sprintf("%d", i+1), style.Render(line)}
			rows = append(rows, row)
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	style.Selected = style.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	return t
}

// The View method will simply look at the model and return a string.
// The returned string is our UI.
// Bubble Tea takes care of redrawing and other logic.
func (m model) View() string {

	s := ""

	// Make table of filtered log lines
	m.table = makeFilteredTable(m)
	s += baseStyle.Render(m.table.View()) + "\n"

	// Filters
	s += "\nFilters\n\n"

	// Iterate over filters
	for i, filter := range m.filters {

		// Is the cursor pointing at this filter?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor is present
		}

		// Is this filter enabled?
		checked := " " // not selected
		if m.filters[i].IsEnabled {
			checked = "x" // this item is selected
		}

		var style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color(m.filters[i].BackColor)).
			PaddingTop(0).
			PaddingLeft(0)

			// Render the row
		row := fmt.Sprintf("%s [%s] %s\n", cursor, checked, style.Render(filter.XML.Text))
		s += row
	}

	// Footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

// Run the program by passing the initial model to tea.NewProgram, then run
func RunUI(filters []filterfiles.Filter, scanner *bufio.Scanner) {
	p := tea.NewProgram(initialModel(filters, scanner), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}

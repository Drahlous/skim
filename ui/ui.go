package ui

import (
	"bufio"
	"example/user/skim/filterfiles"
	"fmt"
	"os"
	"strings"

	// We'll shorten the package name to "tea" for ease of use
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Which window the cursor is active in
type Focus int

const (
	FilterFocus Focus = iota // Focus is in the Filters window
	LogFocus                 // Focus is in the Log window
	MaxFocus                 // Unused, represents the total number of focus entries
)

// Model to store the application's state
type model struct {
	filterCursor  int   // which filter our cursor is pointing at
	logCursor     int   // which log line our cursor is pointing at
	focus         Focus // which view is currently in focus
	filters       []filterfiles.Filter
	lines         []string
	logTable      table.Model
	filterTable   table.Model
	windowWidth   int
	windowHeight  int
	hideUnmatched bool // whether lines are displayed that do not match an active filter
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var filterStyle = lipgloss.NewStyle().
	Bold(false).
	Foreground(lipgloss.Color("#000000")).
	PaddingTop(0).
	PaddingLeft(0)

// Define the initial state for the application
func initialModel(filters []filterfiles.Filter, scanner *bufio.Scanner) model {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return model{
		filters:       filters,
		lines:         lines,
		hideUnmatched: true,
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
		var cursor *int
		var cursor_max int
		if m.focus == LogFocus {
			cursor = &m.logCursor
			// TODO: set this based on the visible lines instead of all lines
			cursor_max = len(m.lines) - 1
		} else if m.focus == FilterFocus {
			cursor = &m.filterCursor
			cursor_max = len(m.filters) - 1
		}

		// Which key was pressed?
		switch msg.String() {

		case "ctrl+c", "q":
			// These keys will exit the program
			return m, tea.Quit

		case "up", "k":
			// "up" and "k" move the cursor up
			if *cursor > 0 {
				*cursor--
			}

		case "down", "j":
			// Move the cursor down
			if *cursor < cursor_max {
				*cursor++
			}

		case "enter", " ":
			// Enter and spacebar toggle the selected state for the item under the cursor
			filter := &m.filters[m.filterCursor]
			filter.IsEnabled = !filter.IsEnabled

		case "tab":
			m.focus += 1
			m.focus %= MaxFocus

		case "h":
			// Toggle hiding unmatched lines
			m.hideUnmatched = !m.hideUnmatched
		}

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	}
	return m, nil
}

func makeFilteredTable(m model) table.Model {

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Line", Width: m.windowWidth - 10}, // TODO: Avoid hardcoding this offset
	}

	rows := []table.Row{}

	for i, line := range m.lines {
		// +1 Offset to make the first line number 1
		lineNumber := i + 1

		// Replace tabs with spaces
		line = strings.ReplaceAll(line, "\t", "    ")

		// Do any filters match this line?
		filter, match := filterfiles.GetMatchingFilter(m.filters, line)
		if match == true {

			// Style this log line with the color from the filter
			style := filterStyle
			style.Background(lipgloss.Color(filter.BackColor))

			row := table.Row{fmt.Sprintf("%d", lineNumber), style.Render(line)}
			rows = append(rows, row)
		} else if !m.hideUnmatched {
			row := table.Row{fmt.Sprintf("%d", lineNumber), line}
			rows = append(rows, row)
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		// TODO: Replace hardcoded offset with the size of the filter section
		table.WithHeight(m.windowHeight-12),
	)

	// Move the view to the location of the log cursor
	t.MoveDown(m.logCursor)

	return t
}

func makeFilters(m model) table.Model {
	columns := []table.Column{
		{Title: "", Width: 3},
		{Title: "Regex", Width: m.windowWidth - 9}, // TODO: Avoid hardcoding this offset
	}

	rows := []table.Row{}

	// Iterate over filters
	for i, filter := range m.filters {

		// Is this filter enabled?
		checked := " " // not selected
		if m.filters[i].IsEnabled {
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

	t.MoveDown(m.filterCursor)
	return t
}

// The View method will simply look at the model and return a string.
// The returned string is our UI.
// Bubble Tea takes care of redrawing and other logic.
func (m model) View() string {

	s := ""

	// Make table of filtered log lines
	m.logTable = makeFilteredTable(m)
	s += baseStyle.Render(m.logTable.View()) + "\n"

	m.filterTable = makeFilters(m)
	s += baseStyle.Render(m.filterTable.View()) + "\n"

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

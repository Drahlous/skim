package ui

import (
	"bufio"
	"example/user/skim/filterfiles"
	filterview "example/user/skim/ui/views/filterview"
	logview "example/user/skim/ui/views/logview"
	"fmt"
	"os"

	// We'll shorten the package name to "tea" for ease of use
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

type TableView interface {
	Toggle()
	CursorUp() int
	CursorDown() int
	GetMaxCursor() int
}

// Model to store the application's state
type model struct {
	log           logview.LogView
	filters       filterview.FilterView
	focus         Focus // which view is currently in focus
	windowWidth   int
	windowHeight  int
	hideUnmatched bool // whether lines are displayed that do not match an active filter
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
		filters: filterview.FilterView{
			Filters: filters,
			Cursor:  0,
		},
		log: logview.LogView{
			Lines:  lines,
			Cursor: 0,
		},
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

		var view TableView

		if m.focus == LogFocus {
			view = &m.log
		} else if m.focus == FilterFocus {
			view = &m.filters
		}

		// Which key was pressed?
		switch msg.String() {

		case "ctrl+c", "q":
			// These keys will exit the program
			return m, tea.Quit

		case "up", "k":
			// "up" and "k" move the cursor up
			view.CursorUp()

		case "down", "j":
			// Move the cursor down
			view.CursorDown()

		case "enter", " ":
			// Enter and spacebar toggle the selected state for the item under the cursor
			view.Toggle()

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

// The View method will simply look at the model and return a string.
// The returned string is our UI.
// Bubble Tea takes care of redrawing and other logic.
func (m model) View() string {

	s := ""

	// Make table of filtered log lines
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched)
	s += baseStyle.Render(m.log.Table.View()) + "\n"

	m.filters.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters)
	s += baseStyle.Render(m.filters.Table.View()) + "\n"

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

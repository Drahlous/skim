package ui

import (
	"bufio"
	"example/user/skim/filterfiles"
	"fmt"
	"os"

	// We'll shorten the package name to "tea" for ease of use
	tea "github.com/charmbracelet/bubbletea"
)

// Model to store the application's state
type model struct {
	cursor  int // which filter our cursor is pointing at
	filters []filterfiles.Filter
	lines   []string
}

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

// The View method will simply look at the model and return a string.
// The returned string is our UI.
// Bubble Tea takes care of redrawing and other logic.
func (m model) View() string {
	// Logfile Lines
	s := "Lines\n"

	for _, line := range m.lines {
		// Do any filters match this line?
		_, match := filterfiles.GetMatchingFilter(m.filters, line)
		if match == true {
			s += fmt.Sprintf("%s\n", line)
		}
	}

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

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, filter.XML.Text)
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

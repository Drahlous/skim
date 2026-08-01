package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"skim/filterfiles"
	"skim/keybindings"
	filterview "skim/ui/views/filterview"
	logview "skim/ui/views/logview"
	"strings"

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
	CursorLeft() int
	CursorRight() int
}

// activeScopes returns the keybinding scopes to check for the given focus,
// most specific first, so a view-scoped action can safely reuse a key
// already bound to a global action (see keybindings.Scope).
func activeScopes(focus Focus) []keybindings.Scope {
	switch focus {
	case FilterFocus:
		return []keybindings.Scope{keybindings.ScopeFilterView, keybindings.ScopeGlobal}
	case LogFocus:
		return []keybindings.Scope{keybindings.ScopeLogView, keybindings.ScopeGlobal}
	default:
		return []keybindings.Scope{keybindings.ScopeGlobal}
	}
}

// resolveAction finds the action bound to key that is valid for the given
// scopes, checking scopes in order (most specific first).
func resolveAction(km keybindings.KeyMap, scopes []keybindings.Scope, key string) (keybindings.Action, bool) {
	for _, scope := range scopes {
		for _, spec := range keybindings.Registry {
			if spec.Scope != scope {
				continue
			}
			for _, k := range km[spec.Action] {
				if k == key {
					return spec.Action, true
				}
			}
		}
	}
	return "", false
}

// renderStatusLine shows whether unmatched lines are currently being
// hidden and how many of the log's lines are visible, so filtering never
// silently makes lines disappear without a visible cue.
func renderStatusLine(m model) string {
	hideState := "OFF"
	if m.hideUnmatched {
		hideState = "ON"
	}

	total := len(m.log.Lines)
	shown := len(m.log.Table.Rows())

	return fmt.Sprintf("hide unmatched: %s  |  showing %d/%d lines", hideState, shown, total)
}

func renderKeyBindings(km keybindings.KeyMap) string {
	parts := []string{
		fmt.Sprintf("%s: quit", strings.Join(km[keybindings.Quit], "/")),
		fmt.Sprintf("%s: edit regex", strings.Join(km[keybindings.EditRegex], "/")),
		fmt.Sprintf("%s/%s: move column", strings.Join(km[keybindings.CursorLeft], ","), strings.Join(km[keybindings.CursorRight], ",")),
		fmt.Sprintf("%s: keybindings", strings.Join(km[keybindings.OpenKeybindingsScreen], "/")),
	}
	return strings.Join(parts, "  |  ")
}

// editorFinishedMsg is sent once the external $EDITOR process invoked to
// edit a filter's regex text has exited.
type editorFinishedMsg struct {
	err      error
	tempFile string
	index    int
}

// openRegexEditorCmd writes the filter's current regex text to a temp file
// and opens it in the user's $EDITOR (falling back to "vi"), suspending the
// UI while the editor runs.
func openRegexEditorCmd(index int, initialText string) tea.Cmd {
	tmpFile, err := os.CreateTemp("", "skim-regex-*.txt")
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: err}
		}
	}

	if _, err := tmpFile.WriteString(initialText); err != nil {
		tmpFile.Close()
		return func() tea.Msg {
			return editorFinishedMsg{err: err}
		}
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, tempFile: tmpFile.Name(), index: index}
	})
}

// Model to store the application's state
type model struct {
	log           logview.LogView
	filters       filterview.FilterView
	focus         Focus // which view is currently in focus
	windowWidth   int
	windowHeight  int
	hideUnmatched bool // whether lines are displayed that do not match an active filter
	keyMap        keybindings.KeyMap

	// Keybindings editor screen state
	editingKeybindings bool
	kbCursor           int  // which action row is selected
	kbCapturing        bool // waiting for a keypress to bind to the selected action
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

	keyMap, err := keybindings.Load()
	if err != nil {
		keyMap = keybindings.Defaults()
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
		keyMap:        keyMap,
	}
}

// Now we'll define the Init method.
// Init can return a Cmd that might perform some initial I/O.
// For now, we don't need to do any I/O, so we'll return nil meaning "no command".
func (m model) Init() tea.Cmd {
	return nil
}

// updateKeybindingsScreen handles key presses while the keybindings editor
// screen is open: it is either browsing the action list, or capturing the
// next keypress to bind to the selected action.
func (m model) updateKeybindingsScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.kbCapturing {
		key := msg.String()
		if key != "esc" {
			action := keybindings.Registry[m.kbCursor].Action
			m.keyMap[action] = []string{key}
			// Best-effort: if we can't persist, the rebinding still applies
			// for the rest of this session.
			keybindings.Save(m.keyMap)
		}
		m.kbCapturing = false
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		m.editingKeybindings = false

	case "up", "k":
		if m.kbCursor > 0 {
			m.kbCursor--
		}

	case "down", "j":
		if m.kbCursor < len(keybindings.Registry)-1 {
			m.kbCursor++
		}

	case "enter":
		m.kbCapturing = true
	}

	return m, nil
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

		if m.editingKeybindings {
			return m.updateKeybindingsScreen(msg)
		}

		var view TableView

		if m.focus == LogFocus {
			view = &m.log
		} else if m.focus == FilterFocus {
			view = &m.filters
		}

		action, ok := resolveAction(m.keyMap, activeScopes(m.focus), msg.String())
		if !ok {
			break
		}

		switch action {
		case keybindings.Quit:
			return m, tea.Quit

		case keybindings.CursorUp:
			view.CursorUp()

		case keybindings.CursorDown:
			view.CursorDown()

		case keybindings.CursorLeft:
			view.CursorLeft()

		case keybindings.CursorRight:
			view.CursorRight()

		case keybindings.Toggle:
			// Toggles the selected state for the item under the cursor
			view.Toggle()

		case keybindings.SwitchFocus:
			m.focus += 1
			m.focus %= MaxFocus

		case keybindings.ToggleHideUnmatched:
			m.hideUnmatched = !m.hideUnmatched

		case keybindings.EditRegex:
			// Edit the selected filter's regex in $EDITOR
			if len(m.filters.Filters) > 0 {
				selected := m.filters.Filters[m.filters.Cursor]
				return m, openRegexEditorCmd(m.filters.Cursor, selected.XML.Text)
			}

		case keybindings.OpenKeybindingsScreen:
			m.editingKeybindings = true
			m.kbCursor = 0
			m.kbCapturing = false
		}

	case editorFinishedMsg:
		if msg.tempFile != "" {
			defer os.Remove(msg.tempFile)
		}
		if msg.err == nil {
			content, err := os.ReadFile(msg.tempFile)
			if err == nil {
				newText := strings.TrimRight(string(content), "\n")
				m.filters.UpdateRegexText(msg.index, newText)
			}
		}

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	}
	return m, nil
}

// renderKeybindingsScreen renders the full-screen keybindings editor: every
// rebindable action, its current key(s), and the row under the cursor.
func (m model) renderKeybindingsScreen() string {
	var b strings.Builder

	b.WriteString("Keybindings  —  up/down: select   enter: rebind (esc cancels)   esc/q: close\n\n")

	for i, spec := range keybindings.Registry {
		cursor := "  "
		if i == m.kbCursor {
			cursor = "> "
		}

		keys := strings.Join(m.keyMap[spec.Action], ", ")
		if m.kbCapturing && i == m.kbCursor {
			keys = "press a key..."
		}

		b.WriteString(fmt.Sprintf("%s%-24s %s\n", cursor, spec.Description, keys))
	}

	return baseStyle.Render(b.String())
}

// The View method will simply look at the model and return a string.
// The returned string is our UI.
// Bubble Tea takes care of redrawing and other logic.
func (m model) View() string {

	if m.editingKeybindings {
		return m.renderKeybindingsScreen()
	}

	s := ""

	// Make table of filtered log lines
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched)
	s += baseStyle.Render(m.log.Table.View()) + "\n"

	s += baseStyle.Render(m.filters.Render(m.windowWidth, m.windowHeight)) + "\n"

	s += renderStatusLine(m) + "\n"
	s += renderKeyBindings(m.keyMap) + "\n"

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

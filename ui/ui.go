package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
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

	line := fmt.Sprintf("hide unmatched: %s  |  showing %d/%d lines", hideState, shown, total)
	if m.contextLines > 0 {
		line += fmt.Sprintf("  |  context: ±%d", m.contextLines)
	}
	if m.hasSearch {
		line += fmt.Sprintf("  |  search: /%s/", m.lastSearchText)
	}
	if m.filtersDirty {
		line += "  |  unsaved filter changes"
	}
	if m.saveStatus != "" {
		line += "  |  " + m.saveStatus
	}
	return line
}

// renderSearchPrompt shows the in-progress search pattern (or its compile
// error) in place of the help bar while the user is typing after "/".
func renderSearchPrompt(m model) string {
	if m.searchErr != "" {
		return fmt.Sprintf("/%s  (invalid regex: %s)", m.searchText, m.searchErr)
	}
	return fmt.Sprintf("/%s", m.searchText)
}

// displayKey renders a raw key string (as stored in a keybindings.KeyMap)
// for on-screen display. Some keys, like " " (space), are invisible or
// confusing when printed literally.
func displayKey(key string) string {
	if key == " " {
		return "space"
	}
	return key
}

// displayKeys renders a slice of raw key strings for on-screen display,
// joined with sep.
func displayKeys(keys []string, sep string) string {
	labels := make([]string, len(keys))
	for i, k := range keys {
		labels[i] = displayKey(k)
	}
	return strings.Join(labels, sep)
}

// keyBindingSep separates individual "key: description" entries in the
// bottom help bar.
const keyBindingSep = "  |  "

// renderKeyBindings builds the bottom help bar, showing only the bindings
// that are actually relevant to the currently focused pane (plus the
// always-available global bindings), so it doesn't advertise keys that do
// nothing in the current context. The result is wrapped to width without
// ever splitting a single "key: description" entry across lines (see
// packKeyBindings); width <= 0 leaves it as one unwrapped line.
func renderKeyBindings(km keybindings.KeyMap, focus Focus, width int) string {
	parts := []string{
		fmt.Sprintf("%s: quit", strings.Join(km[keybindings.Quit], "/")),
	}

	switch focus {
	case FilterFocus:
		parts = append(parts,
			fmt.Sprintf("%s: edit regex", strings.Join(km[keybindings.EditRegex], "/")),
			fmt.Sprintf("%s/%s: move column", strings.Join(km[keybindings.CursorLeft], ","), strings.Join(km[keybindings.CursorRight], ",")),
			fmt.Sprintf("%s: new filter", strings.Join(km[keybindings.NewFilter], "/")),
			fmt.Sprintf("%s: delete filter", strings.Join(km[keybindings.DeleteFilter], "/")),
			fmt.Sprintf("%s/%s: reorder", strings.Join(km[keybindings.MoveFilterUp], ","), strings.Join(km[keybindings.MoveFilterDown], ",")),
		)
	case LogFocus:
		parts = append(parts,
			fmt.Sprintf("%s: hide unmatched", strings.Join(km[keybindings.ToggleHideUnmatched], "/")),
			fmt.Sprintf("%s: search", strings.Join(km[keybindings.Search], "/")),
			fmt.Sprintf("%s/%s: next/prev match", strings.Join(km[keybindings.SearchNext], ","), strings.Join(km[keybindings.SearchPrev], ",")),
			fmt.Sprintf("%s/%s: context lines", strings.Join(km[keybindings.IncreaseContext], ","), strings.Join(km[keybindings.DecreaseContext], ",")),
		)
	}

	parts = append(parts,
		fmt.Sprintf("%s: save filters", strings.Join(km[keybindings.SaveFilters], "/")),
		fmt.Sprintf("%s: keybindings", strings.Join(km[keybindings.OpenKeybindingsScreen], "/")),
		fmt.Sprintf("%s: hide help", strings.Join(km[keybindings.ToggleHelp], "/")),
	)

	return packKeyBindings(parts, width)
}

// renderHelpHint is the collapsed form of the bottom help bar shown while
// help is hidden: just enough to tell the user how to bring it back, so it
// doesn't consume a full line of screen space by default.
func renderHelpHint(km keybindings.KeyMap) string {
	return fmt.Sprintf("%s: show keybindings", strings.Join(km[keybindings.ToggleHelp], "/"))
}

// packKeyBindings joins parts with keyBindingSep, greedily wrapping onto a
// new line whenever the next part would overflow width, so a narrow
// terminal breaks between entries rather than splitting one down the
// middle. A part wider than width by itself is placed alone on its line
// rather than being split. width <= 0 (no WindowSizeMsg received yet)
// disables wrapping entirely.
func packKeyBindings(parts []string, width int) string {
	if width <= 0 {
		return strings.Join(parts, keyBindingSep)
	}

	var lines []string
	line := ""
	for _, p := range parts {
		candidate := p
		if line != "" {
			candidate = line + keyBindingSep + p
		}
		if line != "" && lipgloss.Width(candidate) > width {
			lines = append(lines, line)
			line = p
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
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
	contextLines  int  // how many lines of context to show around a match when hideUnmatched is on
	keyMap        keybindings.KeyMap
	showHelp      bool // whether the full keybindings help bar is expanded

	// Keybindings editor screen state
	editingKeybindings bool
	kbCursor           int  // which action row is selected
	kbCapturing        bool // waiting for a keypress to bind to the selected action

	// Log search state
	searching      bool          // capturing a search pattern from the user
	searchText     string        // the pattern typed so far in the current input session
	searchErr      string        // set if searchText failed to compile as a regex
	lastSearch     regexp.Regexp // last successfully compiled search pattern
	lastSearchText string        // raw text of lastSearch, for display (lastSearch.String() includes the (?i) prefix)
	hasSearch      bool          // whether lastSearch is valid (n/N have something to jump to)

	// Filter persistence state
	filterFilePath string                               // where SaveFilters writes to
	fileMeta       filterfiles.TextAnalysisToolSettings // version/showOnlyFilteredLines to preserve on save
	filtersDirty   bool                                 // whether filters have changed since the last save
	saveStatus     string                               // last save attempt's outcome, shown in the status line
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// focusedStyle marks whichever pane (log or filters) currently has
// keyboard focus, since the two panes otherwise look identical and it's
// easy to lose track of which one your keypresses are going to.
var focusedStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("212"))

// paneStyle returns focusedStyle if want equals the model's current focus,
// otherwise the default baseStyle.
func (m model) paneStyle(want Focus) lipgloss.Style {
	if m.focus == want {
		return focusedStyle
	}
	return baseStyle
}

// Define the initial state for the application
func initialModel(filters []filterfiles.Filter, scanner *bufio.Scanner, filterFilePath string, fileMeta filterfiles.TextAnalysisToolSettings) model {
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
		hideUnmatched:  true,
		keyMap:         keyMap,
		filterFilePath: filterFilePath,
		fileMeta:       fileMeta,
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

// updateSearchInput handles key presses while a search pattern is being
// typed (after pressing "/" in the Log pane): every printable rune is
// appended to searchText, backspace removes the last one, esc cancels, and
// enter compiles the pattern and jumps to its first match after the cursor.
func (m model) updateSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchText = ""
		m.searchErr = ""

	case "enter":
		if m.searchText == "" {
			m.searching = false
			break
		}
		re, err := filterfiles.CompileRegex(m.searchText, false)
		if err != nil {
			// Stay in searching mode so the error actually renders (see
			// renderSearchPrompt, only shown while m.searching), and leave
			// any previously-successful search untouched so n/N still work
			// with it after a failed retry.
			m.searchErr = err.Error()
			break
		}
		m.searching = false
		m.searchErr = ""
		m.lastSearch = re
		m.lastSearchText = m.searchText
		m.hasSearch = true
		if idx, ok := m.log.FindNext(m.lastSearch); ok {
			m.log.Cursor = idx
		}

	case "backspace":
		if len(m.searchText) > 0 {
			r := []rune(m.searchText)
			m.searchText = string(r[:len(r)-1])
		}

	case " ":
		m.searchText += " "

	default:
		if len(msg.Runes) > 0 {
			m.searchText += string(msg.Runes)
		}
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

		if m.searching {
			return m.updateSearchInput(msg)
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
			// Toggles the selected state for the item under the cursor.
			// FilterView.Toggle is a no-op against an empty filter list
			// (reachable via d), which shouldn't be reported as a change.
			view.Toggle()
			if m.focus == FilterFocus && len(m.filters.Filters) > 0 {
				m.filtersDirty = true
				m.saveStatus = ""
			}

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

		case keybindings.Search:
			m.searching = true
			m.searchText = ""
			m.searchErr = ""

		case keybindings.SearchNext:
			if m.hasSearch {
				if idx, ok := m.log.FindNext(m.lastSearch); ok {
					m.log.Cursor = idx
				}
			}

		case keybindings.SearchPrev:
			if m.hasSearch {
				if idx, ok := m.log.FindPrev(m.lastSearch); ok {
					m.log.Cursor = idx
				}
			}

		case keybindings.NewFilter:
			m.filters.Add()
			m.filtersDirty = true
			m.saveStatus = ""
			return m, openRegexEditorCmd(m.filters.Cursor, "")

		case keybindings.DeleteFilter:
			if len(m.filters.Filters) > 0 {
				m.filters.Delete()
				m.filtersDirty = true
				m.saveStatus = ""
			}

		case keybindings.MoveFilterUp:
			if m.filters.MoveUp() {
				m.filtersDirty = true
				m.saveStatus = ""
			}

		case keybindings.MoveFilterDown:
			if m.filters.MoveDown() {
				m.filtersDirty = true
				m.saveStatus = ""
			}

		case keybindings.SaveFilters:
			if err := filterfiles.WriteFilterFile(m.filterFilePath, m.fileMeta, m.filters.Filters); err != nil {
				m.saveStatus = fmt.Sprintf("save failed: %v", err)
			} else {
				m.saveStatus = fmt.Sprintf("saved to %s", m.filterFilePath)
				m.filtersDirty = false
			}

		case keybindings.IncreaseContext:
			m.contextLines++

		case keybindings.DecreaseContext:
			if m.contextLines > 0 {
				m.contextLines--
			}

		case keybindings.ToggleHelp:
			m.showHelp = !m.showHelp
		}

	case tea.MouseMsg:
		// Scrolling while a modal input (keybindings editor or search) is
		// capturing keystrokes has no sensible target, so ignore it rather
		// than silently moving a cursor the user can't currently see move.
		if m.editingKeybindings || m.searching {
			break
		}

		var view TableView
		if m.focus == LogFocus {
			view = &m.log
		} else if m.focus == FilterFocus {
			view = &m.filters
		}

		switch msg.Type {
		case tea.MouseWheelUp:
			view.CursorUp()
		case tea.MouseWheelDown:
			view.CursorDown()
		}

	case editorFinishedMsg:
		if msg.tempFile != "" {
			defer os.Remove(msg.tempFile)
		}
		if msg.err == nil {
			content, err := os.ReadFile(msg.tempFile)
			if err == nil {
				newText := strings.TrimRight(string(content), "\n")
				if err := m.filters.UpdateRegexText(msg.index, newText); err == nil {
					m.filtersDirty = true
					m.saveStatus = ""
				}
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

		keys := displayKeys(m.keyMap[spec.Action], ", ")
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
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)
	s += m.paneStyle(LogFocus).Render(m.log.Table.View()) + "\n"

	counts := filterfiles.CountMatches(m.filters.Filters, m.log.Lines)
	s += m.paneStyle(FilterFocus).Render(m.filters.Render(m.windowWidth, m.windowHeight, counts)) + "\n"

	s += renderStatusLine(m) + "\n"

	switch {
	case m.searching:
		s += renderSearchPrompt(m) + "\n"
	case m.showHelp:
		s += renderKeyBindings(m.keyMap, m.focus, m.windowWidth) + "\n"
	default:
		s += renderHelpHint(m.keyMap) + "\n"
	}

	// Send the UI for rendering
	return s
}

// Run the program by passing the initial model to tea.NewProgram, then run
func RunUI(filters []filterfiles.Filter, scanner *bufio.Scanner, filterFilePath string, fileMeta filterfiles.TextAnalysisToolSettings, usingStdinLog bool) {
	opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
	if usingStdinLog {
		// The log's bufio.Scanner has already fully drained stdin above, so
		// stdin itself is no longer usable for keyboard input (and may not
		// have been a terminal in the first place). Read keypresses from the
		// controlling TTY instead.
		opts = append(opts, tea.WithInputTTY())
	}

	p := tea.NewProgram(initialModel(filters, scanner, filterFilePath, fileMeta), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}

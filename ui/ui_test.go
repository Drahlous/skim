package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"skim/filterfiles"
	"skim/keybindings"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// keyMsg builds a tea.KeyMsg whose String() matches the given key string
// (e.g. "q", "up", "enter", "ctrl+c", " "), for driving Update in tests the
// same way a real keypress would.
func keyMsg(key string) tea.KeyMsg {
	special := map[string]tea.KeyType{
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"enter": tea.KeyEnter, "tab": tea.KeyTab, "esc": tea.KeyEsc, "escape": tea.KeyEscape,
		"ctrl+c": tea.KeyCtrlC, " ": tea.KeySpace,
	}
	if t, ok := special[key]; ok {
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func mustFilter(t *testing.T, text string) filterfiles.Filter {
	t.Helper()
	re, err := filterfiles.CompileRegex(text, false)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return filterfiles.Filter{
		XML:       filterfiles.FilterXML{Text: text},
		Regex:     re,
		IsEnabled: true,
		BackColor: "#87CEFA",
	}
}

func newTestModel(t *testing.T, filters []filterfiles.Filter, lines string) model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	scanner := bufio.NewScanner(strings.NewReader(lines))
	return initialModel(filters, scanner, filepath.Join(t.TempDir(), "filters.tat"), filterfiles.TextAnalysisToolSettings{})
}

func TestActiveScopes(t *testing.T) {
	tests := []struct {
		name  string
		focus Focus
		want  []keybindings.Scope
	}{
		{"filter focus", FilterFocus, []keybindings.Scope{keybindings.ScopeFilterView, keybindings.ScopeGlobal}},
		{"log focus", LogFocus, []keybindings.Scope{keybindings.ScopeLogView, keybindings.ScopeGlobal}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activeScopes(tt.focus)
			if len(got) != len(tt.want) {
				t.Fatalf("activeScopes(%v) = %v, want %v", tt.focus, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("activeScopes(%v)[%d] = %v, want %v", tt.focus, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolveAction(t *testing.T) {
	km := keybindings.Defaults()

	tests := []struct {
		name       string
		scopes     []keybindings.Scope
		key        string
		wantAction keybindings.Action
		wantOK     bool
	}{
		{
			name:       "h in filter view scope resolves to cursor left",
			scopes:     []keybindings.Scope{keybindings.ScopeFilterView, keybindings.ScopeGlobal},
			key:        "h",
			wantAction: keybindings.CursorLeft,
			wantOK:     true,
		},
		{
			name:       "h in log view scope resolves to hide unmatched",
			scopes:     []keybindings.Scope{keybindings.ScopeLogView, keybindings.ScopeGlobal},
			key:        "h",
			wantAction: keybindings.ToggleHideUnmatched,
			wantOK:     true,
		},
		{
			name:       "global action resolves regardless of scope",
			scopes:     []keybindings.Scope{keybindings.ScopeGlobal},
			key:        "q",
			wantAction: keybindings.Quit,
			wantOK:     true,
		},
		{
			name:   "unbound key resolves to nothing",
			scopes: []keybindings.Scope{keybindings.ScopeGlobal},
			key:    "z",
			wantOK: false,
		},
		{
			name:   "view-scoped action does not resolve outside its scope",
			scopes: []keybindings.Scope{keybindings.ScopeGlobal},
			key:    "i", // edit_regex is ScopeFilterView-only
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, ok := resolveAction(km, tt.scopes, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("resolveAction(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if ok && action != tt.wantAction {
				t.Errorf("resolveAction(%q) = %v, want %v", tt.key, action, tt.wantAction)
			}
		})
	}
}

func TestRenderKeyBindings(t *testing.T) {
	km := keybindings.Defaults()

	t.Run("filter focus", func(t *testing.T) {
		out := renderKeyBindings(km, FilterFocus, 0)

		for _, want := range []string{"quit", "edit regex", "move column", "keybindings"} {
			if !strings.Contains(out, want) {
				t.Errorf("renderKeyBindings(FilterFocus) output missing %q, got: %q", want, out)
			}
		}
		if !strings.Contains(out, "ctrl+c/q") {
			t.Errorf("renderKeyBindings(FilterFocus) should show the quit keys, got: %q", out)
		}
		if strings.Contains(out, "hide unmatched") {
			t.Errorf("renderKeyBindings(FilterFocus) should not advertise the log-only hide-unmatched binding, got: %q", out)
		}
	})

	t.Run("log focus", func(t *testing.T) {
		out := renderKeyBindings(km, LogFocus, 0)

		for _, want := range []string{"quit", "hide unmatched", "keybindings"} {
			if !strings.Contains(out, want) {
				t.Errorf("renderKeyBindings(LogFocus) output missing %q, got: %q", want, out)
			}
		}
		if strings.Contains(out, "edit regex") || strings.Contains(out, "move column") {
			t.Errorf("renderKeyBindings(LogFocus) should not advertise filter-only bindings, got: %q", out)
		}
	})
}

func TestRenderStatusLine(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug")}
	m := newTestModel(t, filters, "debug: one\nnot matched\ndebug: two\n")

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	t.Run("hide unmatched on", func(t *testing.T) {
		if !m.hideUnmatched {
			t.Fatal("precondition: hideUnmatched should start true")
		}
		m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)
		out := renderStatusLine(m)

		if !strings.Contains(out, "hide unmatched: ON") {
			t.Errorf("status line missing hide-unmatched ON, got: %q", out)
		}
		if !strings.Contains(out, "showing 2/3 lines") {
			t.Errorf("status line = %q, want it to report 2/3 lines shown", out)
		}
	})

	t.Run("hide unmatched off", func(t *testing.T) {
		m.hideUnmatched = false
		m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)
		out := renderStatusLine(m)

		if !strings.Contains(out, "hide unmatched: OFF") {
			t.Errorf("status line missing hide-unmatched OFF, got: %q", out)
		}
		if !strings.Contains(out, "showing 3/3 lines") {
			t.Errorf("status line = %q, want it to report 3/3 lines shown", out)
		}
	})
}

func TestPaneStyleHighlightsFocusedPaneOnly(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	if m.focus != FilterFocus {
		t.Fatalf("precondition: focus = %v, want FilterFocus", m.focus)
	}
	if m.paneStyle(FilterFocus).GetBorderTopForeground() != focusedStyle.GetBorderTopForeground() {
		t.Error("paneStyle(FilterFocus) should be focusedStyle while FilterFocus is active")
	}
	if m.paneStyle(LogFocus).GetBorderTopForeground() != baseStyle.GetBorderTopForeground() {
		t.Error("paneStyle(LogFocus) should be baseStyle while FilterFocus is active")
	}

	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	if m.paneStyle(LogFocus).GetBorderTopForeground() != focusedStyle.GetBorderTopForeground() {
		t.Error("paneStyle(LogFocus) should be focusedStyle after switching to LogFocus")
	}
	if m.paneStyle(FilterFocus).GetBorderTopForeground() != baseStyle.GetBorderTopForeground() {
		t.Error("paneStyle(FilterFocus) should be baseStyle after switching to LogFocus")
	}
}

func TestInitialModel(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug")}
	m := newTestModel(t, filters, "line one\nline two\n")

	if m.focus != FilterFocus {
		t.Errorf("initial focus = %v, want FilterFocus", m.focus)
	}
	if !m.hideUnmatched {
		t.Error("initial hideUnmatched = false, want true")
	}
	if len(m.log.Lines) != 2 {
		t.Errorf("got %d log lines, want 2", len(m.log.Lines))
	}
	if len(m.filters.Filters) != 1 {
		t.Errorf("got %d filters, want 1", len(m.filters.Filters))
	}
	if len(m.keyMap) != len(keybindings.Registry) {
		t.Errorf("keyMap has %d actions, want %d", len(m.keyMap), len(keybindings.Registry))
	}
}

func TestUpdateQuit(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("Update(q) returned a nil Cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Update(q) did not return a command producing tea.QuitMsg")
	}
}

func TestUpdateSwitchFocus(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	if m.focus != FilterFocus {
		t.Fatalf("precondition: focus = %v, want FilterFocus", m.focus)
	}

	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	if m.focus != LogFocus {
		t.Errorf("focus after tab = %v, want LogFocus", m.focus)
	}

	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	if m.focus != FilterFocus {
		t.Errorf("focus after second tab = %v, want FilterFocus", m.focus)
	}
}

func TestUpdateCursorMovementRoutesToFocusedView(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}
	m := newTestModel(t, filters, "line one\nline two\nline three\n")

	// FilterFocus: j/k should move the filter cursor.
	newModel, _ := m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.filters.Cursor != 1 {
		t.Errorf("filters.Cursor after j = %d, want 1", m.filters.Cursor)
	}
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor changed to %d while FilterFocus, want unchanged 0", m.log.Cursor)
	}

	// Switch to LogFocus: j/k should move the log cursor instead.
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Errorf("log.Cursor after j in LogFocus = %d, want 1", m.log.Cursor)
	}
	if m.filters.Cursor != 1 {
		t.Errorf("filters.Cursor changed to %d while LogFocus, want unchanged 1", m.filters.Cursor)
	}
}

func TestUpdateHKeyIsContextSensitive(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")

	// FilterFocus: h moves the column selection, does not touch hideUnmatched.
	newModel, _ := m.Update(keyMsg("h"))
	m = newModel.(model)
	if !m.hideUnmatched {
		t.Error("hideUnmatched changed by h while FilterFocus, want unchanged (true)")
	}

	newModel, _ = m.Update(keyMsg("l"))
	m = newModel.(model)
	if int(m.filters.Column) != 1 {
		t.Errorf("filters.Column after l = %d, want 1 (CaseSensitiveColumn)", m.filters.Column)
	}

	// LogFocus: h toggles hideUnmatched.
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("h"))
	m = newModel.(model)
	if m.hideUnmatched {
		t.Error("hideUnmatched unchanged by h while LogFocus, want toggled to false")
	}
}

func TestUpdateToggleEnabledCheckbox(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")

	if !m.filters.Filters[0].IsEnabled {
		t.Fatal("precondition: filter should start enabled")
	}

	newModel, _ := m.Update(keyMsg("enter"))
	m = newModel.(model)
	if m.filters.Filters[0].IsEnabled {
		t.Error("filter still enabled after enter, want toggled off")
	}
}

func TestUpdateEditRegexReturnsCommand(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")

	_, cmd := m.Update(keyMsg("i"))
	if cmd == nil {
		t.Fatal("Update(i) with a selected filter returned a nil Cmd, want the regex editor command")
	}
}

func TestUpdateEditRegexNoOpWithoutFilters(t *testing.T) {
	m := newTestModel(t, nil, "line\n")

	_, cmd := m.Update(keyMsg("i"))
	if cmd != nil {
		t.Error("Update(i) with no filters returned a non-nil Cmd, want nil")
	}
}

func TestUpdateEditorFinishedMsgAppliesEditAndCleansUpTempFile(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")

	tmp, err := os.CreateTemp(t.TempDir(), "skim-regex-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmp.WriteString("goodbye\n"); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmp.Close()

	newModel, _ := m.Update(editorFinishedMsg{tempFile: tmp.Name(), index: 0})
	m = newModel.(model)

	if m.filters.Filters[0].XML.Text != "goodbye" {
		t.Errorf("XML.Text = %q, want %q", m.filters.Filters[0].XML.Text, "goodbye")
	}
	if !m.filters.Filters[0].Regex.MatchString("goodbye world") {
		t.Error("recompiled regex does not match the new text")
	}
	if _, err := os.Stat(tmp.Name()); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up after editorFinishedMsg was handled")
	}
}

func TestUpdateEditorFinishedMsgWithErrorLeavesFilterUnchanged(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")

	newModel, _ := m.Update(editorFinishedMsg{err: os.ErrInvalid})
	m = newModel.(model)

	if m.filters.Filters[0].XML.Text != "a" {
		t.Errorf("XML.Text = %q after errored edit, want unchanged %q", m.filters.Filters[0].XML.Text, "a")
	}
}

func TestMouseWheelScrollsFocusedView(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}
	m := newTestModel(t, filters, "line one\nline two\nline three\n")

	// FilterFocus: wheel should move the filter cursor.
	newModel, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	m = newModel.(model)
	if m.filters.Cursor != 1 {
		t.Errorf("filters.Cursor after wheel down = %d, want 1", m.filters.Cursor)
	}
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor changed to %d while FilterFocus, want unchanged 0", m.log.Cursor)
	}

	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelUp})
	m = newModel.(model)
	if m.filters.Cursor != 0 {
		t.Errorf("filters.Cursor after wheel up = %d, want 0", m.filters.Cursor)
	}

	// Switch to LogFocus: wheel should move the log cursor instead.
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Errorf("log.Cursor after wheel down in LogFocus = %d, want 1", m.log.Cursor)
	}
	if m.filters.Cursor != 0 {
		t.Errorf("filters.Cursor changed to %d while LogFocus, want unchanged 0", m.filters.Cursor)
	}
}

func TestMouseWheelIgnoredWhileSearching(t *testing.T) {
	m := newTestModel(t, nil, "line one\nline two\n")
	newModel, _ := m.Update(keyMsg("tab")) // LogFocus
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)

	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	m = newModel.(model)
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor changed to %d by a mouse wheel event while searching, want unchanged 0", m.log.Cursor)
	}
	if !m.searching {
		t.Error("searching became false after a mouse wheel event, want unchanged true")
	}
}

func TestMouseWheelIgnoredWhileEditingKeybindings(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}, "line\n")
	newModel, _ := m.Update(keyMsg("K"))
	m = newModel.(model)

	newModel, _ = m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	m = newModel.(model)
	if m.filters.Cursor != 0 {
		t.Errorf("filters.Cursor changed to %d by a mouse wheel event while editing keybindings, want unchanged 0", m.filters.Cursor)
	}
	if m.kbCursor != 0 {
		t.Errorf("kbCursor changed to %d by a mouse wheel event, want unchanged 0", m.kbCursor)
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := newTestModel(t, nil, "line\n")

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	if m.windowWidth != 120 || m.windowHeight != 40 {
		t.Errorf("windowWidth/Height = %d/%d, want 120/40", m.windowWidth, m.windowHeight)
	}
}

func TestDisplayKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{" ", "space"},
		{"enter", "enter"},
		{"ctrl+c", "ctrl+c"},
	}

	for _, tt := range tests {
		if got := displayKey(tt.key); got != tt.want {
			t.Errorf("displayKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestDisplayKeys(t *testing.T) {
	got := displayKeys([]string{"enter", " "}, ", ")
	want := "enter, space"
	if got != want {
		t.Errorf("displayKeys([enter,  ], \", \") = %q, want %q", got, want)
	}
}

func TestKeybindingsScreenShowsSpaceKeyByName(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	newModel, _ := m.Update(keyMsg("K"))
	m = newModel.(model)

	out := m.renderKeybindingsScreen()

	if strings.Contains(out, "enter, \n") || strings.Contains(out, "enter,  ") {
		t.Errorf("renderKeybindingsScreen still shows a bare trailing comma for the space key, got:\n%s", out)
	}
	if !strings.Contains(out, "enter, space") {
		t.Errorf("renderKeybindingsScreen should show the space key as \"space\", got:\n%s", out)
	}
}

func TestKeybindingsScreenOpenNavigateAndRebind(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	newModel, _ := m.Update(keyMsg("K"))
	m = newModel.(model)
	if !m.editingKeybindings {
		t.Fatal("editingKeybindings = false after K, want true")
	}
	if m.kbCursor != 0 {
		t.Errorf("kbCursor on open = %d, want 0", m.kbCursor)
	}

	// Navigate down twice.
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.kbCursor != 2 {
		t.Errorf("kbCursor after two j = %d, want 2", m.kbCursor)
	}

	action := keybindings.Registry[m.kbCursor].Action

	// Enter capture mode and rebind to "z".
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)
	if !m.kbCapturing {
		t.Fatal("kbCapturing = false after enter, want true")
	}

	newModel, _ = m.Update(keyMsg("z"))
	m = newModel.(model)
	if m.kbCapturing {
		t.Error("kbCapturing still true after a key was pressed, want false")
	}
	if len(m.keyMap[action]) != 1 || m.keyMap[action][0] != "z" {
		t.Errorf("keyMap[%v] = %v, want [z]", action, m.keyMap[action])
	}

	// The rebind should have persisted to disk.
	loaded, err := keybindings.Load()
	if err != nil {
		t.Fatalf("keybindings.Load() returned unexpected error: %v", err)
	}
	if len(loaded[action]) != 1 || loaded[action][0] != "z" {
		t.Errorf("persisted keyMap[%v] = %v, want [z]", action, loaded[action])
	}

	// Close the screen.
	newModel, _ = m.Update(keyMsg("q"))
	m = newModel.(model)
	if m.editingKeybindings {
		t.Error("editingKeybindings still true after q, want false")
	}
}

func TestKeybindingsScreenCaptureEscCancelsWithoutRebinding(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	newModel, _ := m.Update(keyMsg("K"))
	m = newModel.(model)

	action := keybindings.Registry[m.kbCursor].Action
	before := append([]string(nil), m.keyMap[action]...)

	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("esc"))
	m = newModel.(model)

	if m.kbCapturing {
		t.Error("kbCapturing still true after esc, want false")
	}
	if len(m.keyMap[action]) != len(before) {
		t.Errorf("keyMap[%v] changed after esc cancel: got %v, want unchanged %v", action, m.keyMap[action], before)
	}
	for i := range before {
		if m.keyMap[action][i] != before[i] {
			t.Errorf("keyMap[%v] changed after esc cancel: got %v, want unchanged %v", action, m.keyMap[action], before)
		}
	}
}

func TestSearchOpensCapturesAndJumpsOnEnter(t *testing.T) {
	m := newTestModel(t, nil, "alpha\nbravo\ncharlie\nbravo two\n")
	newModel, _ := m.Update(keyMsg("tab")) // LogFocus
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	if !m.searching {
		t.Fatal("searching = false after /, want true")
	}

	for _, r := range "bravo" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	if m.searchText != "bravo" {
		t.Fatalf("searchText = %q, want %q", m.searchText, "bravo")
	}

	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if m.searching {
		t.Error("searching still true after enter, want false")
	}
	if !m.hasSearch {
		t.Fatal("hasSearch = false after a valid pattern, want true")
	}
	if m.log.Cursor != 1 {
		t.Errorf("log.Cursor after search = %d, want 1 (first line containing %q)", m.log.Cursor, "bravo")
	}
}

func TestSearchBackspaceAndEscCancel(t *testing.T) {
	m := newTestModel(t, nil, "alpha\nbravo\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("a"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("b"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("backspace"))
	m = newModel.(model)
	if m.searchText != "a" {
		t.Fatalf("searchText after backspace = %q, want %q", m.searchText, "a")
	}

	newModel, _ = m.Update(keyMsg("esc"))
	m = newModel.(model)
	if m.searching {
		t.Error("searching still true after esc, want false")
	}
	if m.searchText != "" {
		t.Errorf("searchText after esc = %q, want empty", m.searchText)
	}
	if m.hasSearch {
		t.Error("hasSearch = true after esc cancel, want false (no pattern was ever submitted)")
	}
}

func TestSearchBackspaceRemovesWholeRuneNotJustOneByte(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)

	// "é" is a single rune but two UTF-8 bytes; a byte-based backspace would
	// leave a dangling, invalid partial byte instead of removing it whole.
	for _, r := range "café" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	if m.searchText != "café" {
		t.Fatalf("precondition: searchText = %q, want %q", m.searchText, "café")
	}

	newModel, _ = m.Update(keyMsg("backspace"))
	m = newModel.(model)

	if m.searchText != "caf" {
		t.Errorf("searchText after one backspace on %q = %q, want %q", "café", m.searchText, "caf")
	}
	if !utf8.ValidString(m.searchText) {
		t.Errorf("searchText %q is not valid UTF-8 after backspace", m.searchText)
	}
}

func TestSearchInvalidRegexSetsErrorAndStaysInSearchMode(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)

	for _, r := range "([unclosed" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if m.searchErr == "" {
		t.Error("searchErr is empty after an invalid regex, want an error message")
	}
	// Must stay in searching mode: renderSearchPrompt (the only place
	// searchErr is ever displayed) only renders while m.searching is true,
	// so leaving search mode here would make the error invisible to the
	// user despite being computed.
	if !m.searching {
		t.Error("searching = false after an invalid regex, want true so the error actually renders")
	}
}

func TestSearchInvalidRegexRetryDoesNotClobberPreviousSearch(t *testing.T) {
	m := newTestModel(t, nil, "alpha\nbravo\ncharlie\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	// A first, valid search.
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	for _, r := range "bravo" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)
	if !m.hasSearch {
		t.Fatal("precondition: hasSearch should be true after a valid search")
	}

	// Retry with an invalid pattern.
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	for _, r := range "([unclosed" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if !m.hasSearch {
		t.Error("hasSearch = false after a failed retry, want the previous valid search to remain usable")
	}
	if m.lastSearchText != "bravo" {
		t.Errorf("lastSearchText = %q after a failed retry, want unchanged %q", m.lastSearchText, "bravo")
	}

	// n should still work using the previous search.
	newModel, _ = m.Update(keyMsg("esc")) // dismiss the still-open error prompt first
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("n"))
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Errorf("log.Cursor after n post-failed-retry = %d, want 1 (bravo is still the active search)", m.log.Cursor)
	}
}

func TestSearchNextAndPrevJumpUsingLastSearch(t *testing.T) {
	m := newTestModel(t, nil, "alpha\nbravo\ncharlie\nbravo two\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	for _, r := range "bravo" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Fatalf("precondition: log.Cursor = %d, want 1", m.log.Cursor)
	}

	newModel, _ = m.Update(keyMsg("n"))
	m = newModel.(model)
	if m.log.Cursor != 3 {
		t.Errorf("log.Cursor after n = %d, want 3", m.log.Cursor)
	}

	newModel, _ = m.Update(keyMsg("N"))
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Errorf("log.Cursor after N = %d, want 1", m.log.Cursor)
	}
}

func TestSearchNextNoOpWithoutPriorSearch(t *testing.T) {
	m := newTestModel(t, nil, "alpha\nbravo\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("n"))
	m = newModel.(model)
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor after n with no prior search = %d, want unchanged 0", m.log.Cursor)
	}
}

func TestUpdateNewFilterOpensEditorAndMarksDirty(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	newModel, cmd := m.Update(keyMsg("a"))
	m = newModel.(model)

	if len(m.filters.Filters) != 2 {
		t.Fatalf("got %d filters after 'a', want 2", len(m.filters.Filters))
	}
	if !m.filtersDirty {
		t.Error("filtersDirty = false after adding a filter, want true")
	}
	if cmd == nil {
		t.Fatal("Update(a) returned a nil Cmd, want the regex editor command")
	}
}

func TestUpdateDeleteFilterMarksDirty(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}, "line\n")

	newModel, _ := m.Update(keyMsg("d"))
	m = newModel.(model)

	if len(m.filters.Filters) != 1 {
		t.Fatalf("got %d filters after 'd', want 1", len(m.filters.Filters))
	}
	if !m.filtersDirty {
		t.Error("filtersDirty = false after deleting a filter, want true")
	}
}

func TestUpdateDeleteFilterNoOpWithoutFilters(t *testing.T) {
	m := newTestModel(t, nil, "line\n")

	newModel, _ := m.Update(keyMsg("d"))
	m = newModel.(model)

	if m.filtersDirty {
		t.Error("filtersDirty = true after deleting from an empty filter list, want false (no-op)")
	}
}

func TestUpdateMoveFilterUpDown(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}, "line\n")
	newModel, _ := m.Update(keyMsg("j")) // cursor to filter 1 ("b")
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("["))
	m = newModel.(model)
	if m.filters.Filters[0].XML.Text != "b" || m.filters.Filters[1].XML.Text != "a" {
		t.Errorf("filter order after '[' = %v, want [b a]", m.filters.Filters)
	}
	if !m.filtersDirty {
		t.Error("filtersDirty = false after reordering, want true")
	}

	newModel, _ = m.Update(keyMsg("]"))
	m = newModel.(model)
	if m.filters.Filters[0].XML.Text != "a" || m.filters.Filters[1].XML.Text != "b" {
		t.Errorf("filter order after ']' = %v, want [a b] (back to original)", m.filters.Filters)
	}
}

func TestUpdateMoveFilterNoOpAtBoundaryDoesNotMarkDirty(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a"), mustFilter(t, "b")}, "line\n")
	// Cursor starts at 0 (the top): '[' (move up) should be a no-op.
	newModel, _ := m.Update(keyMsg("["))
	m = newModel.(model)
	if m.filtersDirty {
		t.Error("filtersDirty = true after a no-op move at the top boundary, want false")
	}

	// Move cursor to the bottom: ']' (move down) should now be a no-op.
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("]"))
	m = newModel.(model)
	if m.filtersDirty {
		t.Error("filtersDirty = true after a no-op move at the bottom boundary, want false")
	}
}

func TestUpdateToggleOnEmptyFilterListDoesNotMarkDirty(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")

	newModel, _ := m.Update(keyMsg("d")) // delete the only filter
	m = newModel.(model)
	if len(m.filters.Filters) != 0 {
		t.Fatalf("precondition: expected an empty filter list, got %d filters", len(m.filters.Filters))
	}
	// Deleting marks dirty (a real change); reset it to isolate Toggle's
	// own behavior against the now-empty list.
	m.filtersDirty = false

	newModel, _ = m.Update(keyMsg("enter")) // Toggle against an empty list
	m = newModel.(model)
	if m.filtersDirty {
		t.Error("filtersDirty = true after toggling against an empty filter list, want false (no-op)")
	}
}

func TestUpdateSaveFiltersSuccess(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")
	newModel, _ := m.Update(keyMsg("enter")) // toggle a filter off, so there's something to save
	m = newModel.(model)
	if !m.filtersDirty {
		t.Fatal("precondition: filtersDirty should be true after toggling a filter")
	}

	newModel, _ = m.Update(keyMsg("s"))
	m = newModel.(model)

	if m.filtersDirty {
		t.Error("filtersDirty = true after a successful save, want false")
	}
	if !strings.Contains(m.saveStatus, "saved") {
		t.Errorf("saveStatus = %q, want it to indicate success", m.saveStatus)
	}

	settings, err := filterfiles.ReadFilterFile(m.filterFilePath)
	if err != nil {
		t.Fatalf("ReadFilterFile on the saved path returned unexpected error: %v", err)
	}
	if settings.Filters[0].Enabled != "n" {
		t.Errorf("saved file has enabled=%q, want %q (the toggled-off state)", settings.Filters[0].Enabled, "n")
	}
}

func TestUpdateSaveFiltersFailure(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")
	m.filterFilePath = "/nonexistent/dir/filters.tat"

	newModel, _ := m.Update(keyMsg("s"))
	m = newModel.(model)

	if !strings.Contains(m.saveStatus, "failed") {
		t.Errorf("saveStatus = %q, want it to indicate failure", m.saveStatus)
	}
}

func TestRenderStatusLineShowsDirtyAndSaveStatus(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line one\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)

	if strings.Contains(renderStatusLine(m), "unsaved") {
		t.Error("status line shows unsaved changes before any edit, want clean state")
	}

	m.filtersDirty = true
	if !strings.Contains(renderStatusLine(m), "unsaved filter changes") {
		t.Error("status line does not show unsaved changes after filtersDirty = true")
	}

	m.saveStatus = "saved to /tmp/x.tat"
	out := renderStatusLine(m)
	if !strings.Contains(out, "saved to /tmp/x.tat") {
		t.Errorf("status line missing save status, got: %q", out)
	}
}

func TestUpdateContextLinesIncreaseDecrease(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")
	newModel, _ := m.Update(keyMsg("tab")) // LogFocus
	m = newModel.(model)

	if m.contextLines != 0 {
		t.Fatalf("precondition: contextLines = %d, want 0", m.contextLines)
	}

	newModel, _ = m.Update(keyMsg("+"))
	m = newModel.(model)
	if m.contextLines != 1 {
		t.Errorf("contextLines after + = %d, want 1", m.contextLines)
	}

	newModel, _ = m.Update(keyMsg("+"))
	m = newModel.(model)
	if m.contextLines != 2 {
		t.Errorf("contextLines after second + = %d, want 2", m.contextLines)
	}

	newModel, _ = m.Update(keyMsg("-"))
	m = newModel.(model)
	if m.contextLines != 1 {
		t.Errorf("contextLines after - = %d, want 1", m.contextLines)
	}
}

func TestUpdateContextLinesDoesNotGoNegative(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("-"))
	m = newModel.(model)
	if m.contextLines != 0 {
		t.Errorf("contextLines after - at zero = %d, want unchanged 0", m.contextLines)
	}
}

func TestRenderStatusLineShowsContext(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "a")}, "line one\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)

	if strings.Contains(renderStatusLine(m), "context:") {
		t.Error("status line shows a context indicator at the default of 0, want none")
	}

	m.contextLines = 2
	if !strings.Contains(renderStatusLine(m), "context: ±2") {
		t.Errorf("status line missing context indicator, got: %q", renderStatusLine(m))
	}
}

func TestViewShowsFilterMatchCounts(t *testing.T) {
	// Comparing against a bare "2" would be meaningless: the log pane's own
	// line-number column will contain that digit regardless of whether
	// match counts are wired up at all. Instead, render the filter pane
	// directly with the real computed counts and check View()'s output
	// contains that exact rendering, so this actually verifies View() wires
	// filterfiles.CountMatches through to filters.Render.
	filters := []filterfiles.Filter{mustFilter(t, "^debug")}
	m := newTestModel(t, filters, "debug: one\ndebug: two\nother\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	counts := filterfiles.CountMatches(m.filters.Filters, m.log.Lines)
	if counts[0] != 2 {
		t.Fatalf("precondition: CountMatches = %v, want [2]", counts)
	}
	// View() wraps the rendered pane in a bordered paneStyle box, so wrap
	// the expected value the same way rather than comparing raw content.
	wantFilterPane := m.paneStyle(FilterFocus).Render(m.filters.Render(m.windowWidth, m.windowHeight, counts))

	out := m.View()
	if !strings.Contains(out, wantFilterPane) {
		t.Errorf("View() output does not contain the filter pane rendered with real match counts")
	}
}

func TestViewDoesNotPanic(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "^debug")}, "debug: hello\nworld\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	out := m.View()
	if !strings.Contains(out, "^debug") {
		t.Errorf("View() output missing filter regex text, got:\n%s", out)
	}
	if !strings.Contains(out, "hide unmatched:") {
		t.Errorf("View() output missing status line, got:\n%s", out)
	}

	newModel, _ = m.Update(keyMsg("K"))
	m = newModel.(model)
	out = m.View()
	if !strings.Contains(out, "Keybindings") {
		t.Errorf("View() while editingKeybindings missing header, got:\n%s", out)
	}
}

// TestViewTotalHeightStaysWithinWindowWhenHelpExpands guards against a bug
// where expanding the keybindings help bar (which can wrap onto more than
// one line) grew the total rendered frame past windowHeight. Once that
// happens, Bubble Tea has to drop/shift lines it can't scroll back to fix,
// which desyncs its line-by-line redraw and can leave stale footer text
// stuck on screen even after toggling help back off (the collapsed hint
// line is always exactly one line, so the regression only shows up while
// expanded).
func TestViewTotalHeightStaysWithinWindowWhenHelpExpands(t *testing.T) {
	// Bubble Tea's own renderer splits View()'s output on "\n" and, if that
	// yields more elements than the terminal's height, drops lines from the
	// top since it can't navigate into scrollback -- so the relevant line
	// count is len(strings.Split(out, "\n")), not strings.Count(out, "\n").
	numLines := func(out string) int { return len(strings.Split(out, "\n")) }

	filters := []filterfiles.Filter{
		mustFilter(t, "^debug"),
		mustFilter(t, "goodbye"),
		mustFilter(t, "Hello"),
	}
	lines := "Hello World!\ndebug: this is a debug message\ngoodbye is what I would say here, if the file was over\ndebug: another debug message here\ndebug: getting ready to end the file\ngoodbye world!\n"
	m := newTestModel(t, filters, lines)
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 165, Height: 40})
	m = newModel.(model)

	collapsedLines := numLines(m.View())

	newModel, _ = m.Update(keyMsg("?"))
	m = newModel.(model)
	if !m.showHelp {
		t.Fatalf("precondition: showHelp = false after \"?\", want true")
	}
	expandedOut := m.View()
	expandedLines := numLines(expandedOut)

	if expandedLines > m.windowHeight {
		t.Errorf("expanded View() has %d lines, exceeds windowHeight %d (collapsed had %d)", expandedLines, m.windowHeight, collapsedLines)
	}
}

func TestViewShowsSearchPromptWhileSearching(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "^debug")}, "debug: hello\nworld\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("w"))
	m = newModel.(model)

	out := m.View()
	if !strings.Contains(out, "/w") {
		t.Errorf("View() while searching missing the in-progress pattern, got:\n%s", out)
	}
}

func TestJumpToTopAndBottom(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\nfour\n")
	newModel, _ := m.Update(keyMsg("tab")) // LogFocus
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.log.Cursor != 2 {
		t.Fatalf("precondition: log.Cursor = %d, want 2", m.log.Cursor)
	}

	newModel, _ = m.Update(keyMsg("g"))
	m = newModel.(model)
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor after g = %d, want 0", m.log.Cursor)
	}

	newModel, _ = m.Update(keyMsg("G"))
	m = newModel.(model)
	if m.log.Cursor != 3 {
		t.Errorf("log.Cursor after G = %d, want 3 (last line)", m.log.Cursor)
	}
}

func TestJumpToTopBottomNoOpOnEmptyLog(t *testing.T) {
	m := newTestModel(t, nil, "")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("G"))
	m = newModel.(model)
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor after G on empty log = %d, want unchanged 0", m.log.Cursor)
	}
}

func TestJumpToLineOpensCapturesAndJumpsOnEnter(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\nfour\nfive\n")
	newModel, _ := m.Update(keyMsg("tab")) // LogFocus
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	if !m.jumpingToLine {
		t.Fatal("jumpingToLine = false after :, want true")
	}

	for _, r := range "3" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	if m.jumpLineText != "3" {
		t.Fatalf("jumpLineText = %q, want %q", m.jumpLineText, "3")
	}

	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if m.jumpingToLine {
		t.Error("jumpingToLine still true after enter, want false")
	}
	if m.log.Cursor != 2 {
		t.Errorf("log.Cursor after jumping to line 3 = %d, want 2 (0-indexed)", m.log.Cursor)
	}
}

func TestJumpToLineClampsOutOfRangeInput(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	for _, r := range "999" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if m.log.Cursor != 2 {
		t.Errorf("log.Cursor after jumping past the end = %d, want 2 (clamped to last line)", m.log.Cursor)
	}
}

func TestJumpToLineEmptyInputJustClosesPrompt(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("enter")) // empty input just closes the prompt
	m = newModel.(model)
	if m.jumpingToLine {
		t.Error("jumpingToLine still true after enter on empty input, want false")
	}
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor changed to %d after empty-input enter, want unchanged 0", m.log.Cursor)
	}
}

func TestJumpToLineIgnoresNonDigitRunes(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	for _, r := range "1x2" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	if m.jumpLineText != "12" {
		t.Errorf("jumpLineText = %q, want %q (non-digit rune ignored)", m.jumpLineText, "12")
	}
}

func TestJumpToLineBackspaceAndEscCancel(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\n")
	newModel, _ := m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("1"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("2"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("backspace"))
	m = newModel.(model)
	if m.jumpLineText != "1" {
		t.Fatalf("jumpLineText after backspace = %q, want %q", m.jumpLineText, "1")
	}

	newModel, _ = m.Update(keyMsg("esc"))
	m = newModel.(model)
	if m.jumpingToLine {
		t.Error("jumpingToLine still true after esc, want false")
	}
	if m.jumpLineText != "" {
		t.Errorf("jumpLineText after esc = %q, want empty", m.jumpLineText)
	}
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor changed to %d after esc cancel, want unchanged 0", m.log.Cursor)
	}
}

func TestRenderStatusLineShowsLinePosition(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)

	if !strings.Contains(renderStatusLine(m), "line 1/3") {
		t.Errorf("status line missing initial position, got: %q", renderStatusLine(m))
	}

	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)
	if !strings.Contains(renderStatusLine(m), "line 2/3") {
		t.Errorf("status line after moving down = %q, want it to show line 2/3", renderStatusLine(m))
	}
}

func TestRenderStatusLineOmitsPositionOnEmptyLog(t *testing.T) {
	m := newTestModel(t, nil, "")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	m.log.MakeTable(m.windowWidth, m.windowHeight, m.filters.Filters, m.hideUnmatched, m.contextLines)

	if strings.Contains(renderStatusLine(m), "line ") {
		t.Errorf("status line on empty log shows a position indicator, want none: %q", renderStatusLine(m))
	}
}

func TestViewShowsJumpLinePromptWhileJumping(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "^debug")}, "debug: hello\nworld\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("2"))
	m = newModel.(model)

	out := m.View()
	if !strings.Contains(out, ":2") {
		t.Errorf("View() while jumping missing the in-progress line number, got:\n%s", out)
	}
}

func TestViewShowsActiveSearchInStatusLine(t *testing.T) {
	m := newTestModel(t, nil, "hello world\ngoodbye\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("tab"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("/"))
	m = newModel.(model)
	for _, r := range "hello" {
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	out := m.View()
	if !strings.Contains(out, "search: /hello/") {
		t.Errorf("View() after a committed search missing the active pattern in the status line, got:\n%s", out)
	}
}

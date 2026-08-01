package ui

import (
	"bufio"
	"os"
	"skim/filterfiles"
	"skim/keybindings"
	"strings"
	"testing"

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
	return initialModel(filters, scanner)
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
	out := renderKeyBindings(km)

	for _, want := range []string{"quit", "edit regex", "move column", "keybindings"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderKeyBindings output missing %q, got: %q", want, out)
		}
	}
	if !strings.Contains(out, "ctrl+c/q") {
		t.Errorf("renderKeyBindings should show the quit keys, got: %q", out)
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

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := newTestModel(t, nil, "line\n")

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	if m.windowWidth != 120 || m.windowHeight != 40 {
		t.Errorf("windowWidth/Height = %d/%d, want 120/40", m.windowWidth, m.windowHeight)
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

func TestViewDoesNotPanic(t *testing.T) {
	m := newTestModel(t, []filterfiles.Filter{mustFilter(t, "^debug")}, "debug: hello\nworld\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)

	out := m.View()
	if !strings.Contains(out, "^debug") {
		t.Errorf("View() output missing filter regex text, got:\n%s", out)
	}

	newModel, _ = m.Update(keyMsg("K"))
	m = newModel.(model)
	out = m.View()
	if !strings.Contains(out, "Keybindings") {
		t.Errorf("View() while editingKeybindings missing header, got:\n%s", out)
	}
}

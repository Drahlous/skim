package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"skim/filterfiles"
	"skim/keybindings"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestActiveScopesDefaultsToGlobalForUnknownFocus(t *testing.T) {
	got := activeScopes(Focus(99))
	want := []keybindings.Scope{keybindings.ScopeGlobal}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("activeScopes(unknown focus) = %v, want %v", got, want)
	}
}

func TestRenderSearchPromptShowsCompileError(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	m.searchText = "("
	m.searchErr = "missing closing paren"

	got := renderSearchPrompt(m)
	if !strings.Contains(got, "invalid regex: missing closing paren") {
		t.Errorf("renderSearchPrompt() = %q, want it to mention the compile error", got)
	}
}

func TestRenderJumpLinePromptShowsError(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	m.jumpLineText = "x"
	m.jumpLineErr = "not a number"

	got := renderJumpLinePrompt(m)
	if !strings.Contains(got, "not a number") {
		t.Errorf("renderJumpLinePrompt() = %q, want it to mention the parse error", got)
	}
}

func TestInitReturnsNilCmd(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestInitialModelFallsBackToDefaultsOnInvalidKeybindingsConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	skimDir := filepath.Join(configDir, "skim")
	if err := os.MkdirAll(skimDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skimDir, "keybindings.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader("line\n"))
	m := initialModel(nil, scanner, filepath.Join(t.TempDir(), "filters.tat"), filterfiles.TextAnalysisToolSettings{})

	if len(m.keyMap) != len(keybindings.Registry) {
		t.Errorf("keyMap has %d actions after falling back to defaults, want %d", len(m.keyMap), len(keybindings.Registry))
	}
}

func TestKeybindingsScreenNavigateUpAndLeft(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(keyMsg("K"))
	m = newModel.(model)

	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.kbCursor != 2 {
		t.Fatalf("precondition: kbCursor after two j = %d, want 2", m.kbCursor)
	}

	newModel, _ = m.Update(keyMsg("k"))
	m = newModel.(model)
	if m.kbCursor != 1 {
		t.Errorf("kbCursor after k = %d, want 1", m.kbCursor)
	}

	action := keybindings.Registry[m.kbCursor].Action
	if len(m.keyMap[action]) < 2 {
		t.Skipf("action %s does not have 2+ keys, can't test left/right navigation", action)
	}
	newModel, _ = m.Update(keyMsg("l"))
	m = newModel.(model)
	if m.kbKeyCursor != 1 {
		t.Fatalf("precondition: kbKeyCursor after l = %d, want 1", m.kbKeyCursor)
	}
	newModel, _ = m.Update(keyMsg("h"))
	m = newModel.(model)
	if m.kbKeyCursor != 0 {
		t.Errorf("kbKeyCursor after h = %d, want 0", m.kbKeyCursor)
	}
}

func TestKeybindingsScreenShowsCapturingPromptForSelectedAction(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("K"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("enter")) // start capturing a replacement key
	m = newModel.(model)
	if !m.kbCapturing {
		t.Fatal("precondition: kbCapturing should be true after enter")
	}

	out := m.View()
	if !strings.Contains(out, "press a key...") {
		t.Errorf("View() while capturing a keybinding missing the capture prompt, got:\n%s", out)
	}
}

func TestSearchEnterWithEmptyTextClosesSearch(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(keyMsg("/"))
	m = newModel.(model)
	if !m.searching {
		t.Fatal("precondition: searching should be true after /")
	}

	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)
	if m.searching {
		t.Error("searching still true after enter on empty search text, want false")
	}
	if m.hasSearch {
		t.Error("hasSearch = true after confirming an empty search, want false")
	}
}

func TestSearchSpaceCharacterIsAppended(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(keyMsg("/"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("a"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg(" "))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("b"))
	m = newModel.(model)

	if m.searchText != "a b" {
		t.Errorf("searchText = %q, want %q", m.searchText, "a b")
	}
}

func TestJumpToLineOverflowShowsNotANumberError(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\n")
	newModel, _ := m.Update(keyMsg(":"))
	m = newModel.(model)
	for _, r := range "99999999999999999999" { // overflows int, strconv.Atoi errors
		newModel, _ = m.Update(keyMsg(string(r)))
		m = newModel.(model)
	}

	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if !m.jumpingToLine {
		t.Error("jumpingToLine became false after an unparseable number, want to stay open with an error")
	}
	if m.jumpLineErr == "" {
		t.Error("jumpLineErr is empty after an overflowing line number, want \"not a number\"")
	}
}

func TestJumpToLineZeroClampsToFirstLine(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\n")
	newModel, _ := m.Update(keyMsg("j")) // move off line 0 first
	m = newModel.(model)
	if m.log.Cursor == 0 {
		t.Fatal("precondition: cursor should have moved off 0")
	}

	newModel, _ = m.Update(keyMsg(":"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("0"))
	m = newModel.(model)
	newModel, _ = m.Update(keyMsg("enter"))
	m = newModel.(model)

	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor after jumping to line 0 = %d, want 0 (clamped)", m.log.Cursor)
	}
}

func TestCursorUpMovesLogFocusCursor(t *testing.T) {
	m := newTestModel(t, nil, "one\ntwo\nthree\n")
	newModel, _ := m.Update(keyMsg("j"))
	m = newModel.(model)
	if m.log.Cursor != 1 {
		t.Fatalf("precondition: cursor after j = %d, want 1", m.log.Cursor)
	}

	newModel, _ = m.Update(keyMsg("up"))
	m = newModel.(model)
	if m.log.Cursor != 0 {
		t.Errorf("log.Cursor after up = %d, want 0", m.log.Cursor)
	}
}

func TestMouseEventWhileEditingFilterWithPickerClosedIsNoOp(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	// colorPicker.open defaults to false here.

	newModel, cmd := m.Update(tea.MouseMsg{Type: tea.MouseWheelUp})
	m = newModel.(model)
	if cmd != nil {
		t.Error("mouse wheel while editing a filter (picker closed) returned a non-nil Cmd, want no-op")
	}
	if m.filterEditor.cursor != fieldColor {
		t.Errorf("filterEditor.cursor changed to %v, want unchanged %v", m.filterEditor.cursor, fieldColor)
	}
}

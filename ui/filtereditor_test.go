package ui

import (
	"path/filepath"
	"skim/filterfiles"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterEditorCursorNavigation(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldRegex}

	m = update(t, m, keyMsg("up"))
	if m.filterEditor.cursor != fieldDescription {
		t.Errorf("cursor after up = %v, want %v", m.filterEditor.cursor, fieldDescription)
	}

	m = update(t, m, keyMsg("up"))
	if m.filterEditor.cursor != fieldDescription {
		t.Errorf("cursor after up at the top row = %v, want unchanged %v", m.filterEditor.cursor, fieldDescription)
	}

	m = update(t, m, keyMsg("down"))
	if m.filterEditor.cursor != fieldRegex {
		t.Errorf("cursor after down = %v, want %v", m.filterEditor.cursor, fieldRegex)
	}

	m.filterEditor.cursor = maxFilterEditorField - 1
	m = update(t, m, keyMsg("down"))
	if m.filterEditor.cursor != maxFilterEditorField-1 {
		t.Errorf("cursor after down at the bottom row = %v, want unchanged %v", m.filterEditor.cursor, maxFilterEditorField-1)
	}
}

func TestOpenExternalEditorForHoveredFieldNoFiltersIsNoOp(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldRegex}

	newModel, cmd := m.Update(keyMsg("ctrl+e"))
	m = newModel.(model)
	if cmd != nil {
		t.Error("ctrl+e with no filters returned a non-nil Cmd, want no-op")
	}
}

func TestOpenExternalEditorForHoveredDescriptionField(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.filters.Filters[0].XML.Description = "existing description"
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldDescription}

	newModel, cmd := m.Update(keyMsg("ctrl+e"))
	m = newModel.(model)
	if cmd == nil {
		t.Fatal("ctrl+e while hovering the description field returned a nil Cmd, want the external editor command")
	}
}

func TestActivateFilterEditorFieldNoFiltersClosesEditor(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldEnabled}

	m = update(t, m, keyMsg("enter"))
	if m.editingFilter {
		t.Error("enter with no filters left editingFilter true, want it closed")
	}
}

func TestFilterEditorToggleCaseSensitive(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^Abc")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldCaseSensitive}

	if m.filters.Filters[0].CaseSensitive {
		t.Fatal("precondition: filter should start case-insensitive")
	}
	m = update(t, m, keyMsg("enter"))
	if !m.filters.Filters[0].CaseSensitive {
		t.Error("CaseSensitive still false after enter on the Case sensitive field")
	}
	if !m.filtersDirty {
		t.Error("filtersDirty = false after toggling CaseSensitive, want true")
	}
	// The regex must be recompiled with the new case sensitivity so the
	// toggle actually changes matching behavior, not just the flag.
	if m.filters.Filters[0].Regex.MatchString("abc") {
		t.Error("regex still matches lowercase after enabling case sensitivity, want recompiled case-sensitive regex")
	}
	if !m.filters.Filters[0].Regex.MatchString("Abc") {
		t.Error("recompiled case-sensitive regex does not match the original-case text")
	}
}

func TestFilterEditorToggleEnabled(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldEnabled}

	if !m.filters.Filters[0].IsEnabled {
		t.Fatal("precondition: filter should start enabled")
	}
	m = update(t, m, keyMsg("enter"))
	if m.filters.Filters[0].IsEnabled {
		t.Error("IsEnabled still true after enter on the Enabled field")
	}
	if !m.filtersDirty {
		t.Error("filtersDirty = false after toggling IsEnabled, want true")
	}
}

func TestFilterEditorTextInputEscDiscardsInProgressEdit(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldDescription}
	m = update(t, m, keyMsg("enter"), keyMsg("x"), keyMsg("y"))

	m = update(t, m, keyMsg("esc"))
	if m.filterEditor.editingText {
		t.Error("esc did not exit text-edit mode")
	}
	if m.filterEditor.textBuf != "" {
		t.Errorf("textBuf = %q after esc, want cleared", m.filterEditor.textBuf)
	}
	if m.filters.Filters[0].XML.Description != "" {
		t.Errorf("Description = %q after discarding an in-progress edit, want unchanged empty", m.filters.Filters[0].XML.Description)
	}
}

func TestFilterEditorTextInputBackspace(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldDescription}
	m = update(t, m, keyMsg("enter"), keyMsg("h"), keyMsg("i"), keyMsg("backspace"))

	if m.filterEditor.textBuf != "h" {
		t.Errorf("textBuf after backspace = %q, want %q", m.filterEditor.textBuf, "h")
	}
}

func TestRenderFilterEditorNoFiltersSelected(t *testing.T) {
	m := newTestModel(t, nil, "line\n")
	newModel, _ := m.Update(tea_WindowSize())
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{}

	out := m.View()
	if !strings.Contains(out, "No filter selected") {
		t.Errorf("View() with editingFilter true and no filters missing placeholder text, got:\n%s", out)
	}
}

func TestRenderFilterEditorHeaderWhileEditingText(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea_WindowSize())
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldDescription}
	m = update(t, m, keyMsg("enter"))

	out := m.View()
	if !strings.Contains(out, "ctrl+e: edit in $EDITOR   esc: discard") {
		t.Errorf("View() while editing a text field missing the edit-mode header, got:\n%s", out)
	}
}

func TestRenderFilterEditorShowsRegexCompileError(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea_WindowSize())
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldRegex}
	m = update(t, m, keyMsg("enter"), keyMsg("("), keyMsg("enter"))

	out := m.View()
	if !strings.Contains(out, "invalid regex:") {
		t.Errorf("View() with an invalid in-progress regex missing the compile-error annotation, got:\n%s", out)
	}
}

func TestRenderFilterEditorShowsInProgressTextBuffer(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea_WindowSize())
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldDescription}
	m = update(t, m, keyMsg("enter"), keyMsg("h"), keyMsg("i"))

	out := m.View()
	if !strings.Contains(out, "[hi]") {
		t.Errorf("View() while typing a description missing the in-progress buffer, got:\n%s", out)
	}
}

func TestRenderFilterEditorUncheckedCheckbox(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea_WindowSize())
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{}

	if m.filters.Filters[0].CaseSensitive {
		t.Fatal("precondition: filter should start with CaseSensitive false")
	}
	out := m.View()
	if !strings.Contains(out, "Case sensitive: [ ]") {
		t.Errorf("View() missing unchecked checkbox rendering, got:\n%s", out)
	}
}

func TestOpenFilterFieldEditorCmdFallsBackToViWhenEditorUnset(t *testing.T) {
	t.Setenv("EDITOR", "")

	cmd := openFilterFieldEditorCmd(fieldDescription, "hello")
	if cmd == nil {
		t.Fatal("openFilterFieldEditorCmd returned a nil Cmd")
	}
	// Calling the Cmd only builds a tea "run this in the terminal" message
	// (see bubbletea's Exec/ExecProcess) -- it does not actually spawn the
	// editor process, so this is safe to invoke without a real $EDITOR.
	if msg := cmd(); msg == nil {
		t.Error("invoking the Cmd returned a nil Msg")
	}
}

func TestOpenFilterFieldEditorCmdErrorsWhenTempFileCannotBeCreated(t *testing.T) {
	// Point TMPDIR at a path that doesn't exist so os.CreateTemp fails,
	// exercising openFilterFieldEditorCmd's temp-file-creation error path.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	cmd := openFilterFieldEditorCmd(fieldRegex, "hello")
	msg, ok := cmd().(filterFieldEditorFinishedMsg)
	if !ok {
		t.Fatalf("Cmd() returned %#v, want a filterFieldEditorFinishedMsg", msg)
	}
	if msg.err == nil {
		t.Error("filterFieldEditorFinishedMsg.err is nil, want the CreateTemp error")
	}
}

// tea_WindowSize returns a WindowSizeMsg large enough for View() to render
// without panicking, matching the size other View() tests in this package use.
func tea_WindowSize() tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: 120, Height: 40}
}

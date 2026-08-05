package ui

import (
	"skim/filterfiles"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestColorPickerCellAtBoundaries(t *testing.T) {
	startX, startY := colorPickerGridStartX(), colorPickerGridStartY()

	tests := []struct {
		name string
		x, y int
		ok   bool
	}{
		{"row above grid", startX, startY - 1, false},
		{"col left of grid", startX - 1, startY, false},
		{"row past last row", startX, startY + colorPickerRows()*colorPickerColStride, false},
		{"in the gap between swatches", startX + colorPickerCellW, startY, false},
		{"col past last column", startX + colorPickerCols*colorPickerColStride, startY, false},
		{"first cell resolves to index 0", startX, startY, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, ok := colorPickerCellAt(tt.x, tt.y)
			if ok != tt.ok {
				t.Fatalf("colorPickerCellAt(%d, %d) ok = %v, want %v (idx=%d)", tt.x, tt.y, ok, tt.ok, idx)
			}
			if tt.ok && idx != 0 {
				t.Errorf("colorPickerCellAt(%d, %d) = %d, want 0", tt.x, tt.y, idx)
			}
		})
	}
}

func TestUpdateColorPickerEscCloses(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"))
	if !m.filterEditor.colorPicker.open {
		t.Fatal("precondition: color picker should be open")
	}

	m = update(t, m, keyMsg("esc"))
	if m.filterEditor.colorPicker.open {
		t.Error("esc did not close the color picker")
	}
}

func TestUpdateColorPickerCursorClampsAtGridEdges(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"))

	// Force the cursor to the top-left cell, then confirm up/left are no-ops.
	m.filterEditor.colorPicker.cursor = 0
	m = update(t, m, keyMsg("up"))
	if m.filterEditor.colorPicker.cursor != 0 {
		t.Errorf("up at the top row moved cursor to %d, want unchanged 0", m.filterEditor.colorPicker.cursor)
	}
	m = update(t, m, keyMsg("left"))
	if m.filterEditor.colorPicker.cursor != 0 {
		t.Errorf("left at the leftmost column moved cursor to %d, want unchanged 0", m.filterEditor.colorPicker.cursor)
	}

	// Force the cursor to the bottom-right cell (last palette entry), then
	// confirm down/right are no-ops.
	last := len(colorPalette) - 1
	m.filterEditor.colorPicker.cursor = last
	m = update(t, m, keyMsg("down"))
	if m.filterEditor.colorPicker.cursor != last {
		t.Errorf("down at the bottom row moved cursor to %d, want unchanged %d", m.filterEditor.colorPicker.cursor, last)
	}
	m = update(t, m, keyMsg("right"))
	if m.filterEditor.colorPicker.cursor != last {
		t.Errorf("right at the last cell moved cursor to %d, want unchanged %d", m.filterEditor.colorPicker.cursor, last)
	}
}

func TestUpdateColorPickerCursorMovesWithinGrid(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"))

	// Start somewhere in the interior of the grid (not on any edge row/col)
	// so up/left have room to actually move the cursor.
	start := colorPickerCols + 1
	m.filterEditor.colorPicker.cursor = start

	m = update(t, m, keyMsg("up"))
	if want := start - colorPickerCols; m.filterEditor.colorPicker.cursor != want {
		t.Errorf("cursor after up = %d, want %d", m.filterEditor.colorPicker.cursor, want)
	}

	m.filterEditor.colorPicker.cursor = start
	m = update(t, m, keyMsg("left"))
	if want := start - 1; m.filterEditor.colorPicker.cursor != want {
		t.Errorf("cursor after left = %d, want %d", m.filterEditor.colorPicker.cursor, want)
	}
}

func TestColorPaletteIndexForUnknownHexReturnsZero(t *testing.T) {
	if got := colorPaletteIndexFor("#123456"); got != 0 {
		t.Errorf("colorPaletteIndexFor(unknown hex) = %d, want 0", got)
	}
}

func TestUpdateColorPickerMouseMotionMovesCursor(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"))

	x := colorPickerGridStartX() + colorPickerColStride // second column
	y := colorPickerGridStartY()
	newModel, _ := m.Update(tea.MouseMsg{X: x, Y: y, Type: tea.MouseMotion})
	m = newModel.(model)

	if m.filterEditor.colorPicker.cursor != 1 {
		t.Errorf("cursor after hovering the second swatch = %d, want 1", m.filterEditor.colorPicker.cursor)
	}
}

func TestUpdateColorPickerMouseIgnoredWhileCustomEditing(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"), keyMsg("c"))
	if !m.filterEditor.colorPicker.customEditing {
		t.Fatal("precondition: custom hex entry should be active")
	}

	x, y := colorPickerGridStartX(), colorPickerGridStartY()
	newModel, _ := m.Update(tea.MouseMsg{X: x, Y: y, Type: tea.MouseLeft})
	m = newModel.(model)

	if !m.filterEditor.colorPicker.customEditing {
		t.Error("a mouse click while typing a custom hex value left customEditing, want it to stay active (mouse has no target there)")
	}
	if !m.filterEditor.colorPicker.open {
		t.Error("a mouse click while typing a custom hex value closed the picker, want no-op")
	}
}

func TestUpdateColorPickerCustomInputEscReturnsToGrid(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"), keyMsg("c"), keyMsg("a"), keyMsg("b"))
	if !m.filterEditor.colorPicker.customEditing {
		t.Fatal("precondition: custom hex entry should be active")
	}

	m = update(t, m, keyMsg("esc"))
	if m.filterEditor.colorPicker.customEditing {
		t.Error("esc did not exit custom hex entry")
	}
	if !m.filterEditor.colorPicker.open {
		t.Error("esc from custom hex entry also closed the picker, want it to stay open on the grid")
	}
	if m.filterEditor.colorPicker.customBuf != "" {
		t.Errorf("customBuf = %q after esc, want cleared", m.filterEditor.colorPicker.customBuf)
	}
}

func TestUpdateColorPickerCustomInputBackspace(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"), keyMsg("c"), keyMsg("a"), keyMsg("b"), keyMsg("backspace"))

	if got := m.filterEditor.colorPicker.customBuf; got != "a" {
		t.Errorf("customBuf after backspace = %q, want %q", got, "a")
	}

	// Backspacing an already-empty buffer must not underflow/panic.
	m = update(t, m, keyMsg("backspace"), keyMsg("backspace"))
	if got := m.filterEditor.colorPicker.customBuf; got != "" {
		t.Errorf("customBuf after over-backspacing = %q, want empty", got)
	}
}

func TestUpdateColorPickerCustomInputIgnoresNonHexRunes(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"), keyMsg("c"), keyMsg("z"), keyMsg("1"))

	if got := m.filterEditor.colorPicker.customBuf; got != "1" {
		t.Errorf("customBuf = %q after typing a non-hex rune then a hex rune, want %q", got, "1")
	}
}

func TestRenderColorPickerGridViaView(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"))

	out := m.View()
	if !strings.Contains(out, "Pick a Color") {
		t.Errorf("View() with the color picker open missing header, got:\n%s", out)
	}
	if !strings.Contains(out, "Selected: "+colorPalette[m.filterEditor.colorPicker.cursor]) {
		t.Errorf("View() with the color picker open missing selected-swatch line, got:\n%s", out)
	}
}

func TestRenderColorPickerCustomInputViaView(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "a")}
	m := newTestModel(t, filters, "line\n")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(model)
	m.editingFilter = true
	m.filterEditor = filterEditorState{cursor: fieldColor}
	m = update(t, m, keyMsg("enter"), keyMsg("c"), keyMsg("1"), keyMsg("2"))

	out := m.View()
	if !strings.Contains(out, "Custom Hex Color") {
		t.Errorf("View() during custom hex entry missing header, got:\n%s", out)
	}
	if !strings.Contains(out, "#12") {
		t.Errorf("View() during custom hex entry missing in-progress buffer, got:\n%s", out)
	}

	// Confirming a too-short value sets customErr, which should also render.
	m = update(t, m, keyMsg("enter"))
	out = m.View()
	if !strings.Contains(out, "hex color must be 6 digits") {
		t.Errorf("View() with an invalid custom hex value missing error message, got:\n%s", out)
	}
}

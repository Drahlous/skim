package filterview

import (
	"skim/filterfiles"
	"strings"
	"testing"
)

func mustFilter(t *testing.T, text string, caseSensitive bool, enabled bool, backColor string) filterfiles.Filter {
	t.Helper()
	re, err := filterfiles.CompileRegex(text, caseSensitive)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return filterfiles.Filter{
		XML:           filterfiles.FilterXML{Text: text},
		Regex:         re,
		IsEnabled:     enabled,
		CaseSensitive: caseSensitive,
		BackColor:     backColor,
	}
}

func TestCursorUpDown(t *testing.T) {
	v := FilterView{Filters: []filterfiles.Filter{
		mustFilter(t, "a", false, true, "#000000"),
		mustFilter(t, "b", false, true, "#000000"),
		mustFilter(t, "c", false, true, "#000000"),
	}}

	if got := v.CursorUp(); got != 0 {
		t.Errorf("CursorUp() at top = %d, want 0", got)
	}
	if got := v.CursorDown(); got != 1 {
		t.Errorf("CursorDown() = %d, want 1", got)
	}
	if got := v.CursorDown(); got != 2 {
		t.Errorf("CursorDown() = %d, want 2", got)
	}
	if got := v.CursorDown(); got != 2 {
		t.Errorf("CursorDown() past the end = %d, want 2 (should clamp)", got)
	}
}

func TestGetMaxCursor(t *testing.T) {
	tests := []struct {
		name    string
		filters []filterfiles.Filter
		want    int
	}{
		{"empty", nil, -1},
		{"one filter", []filterfiles.Filter{mustFilter(t, "a", false, true, "#000000")}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := FilterView{Filters: tt.filters}
			if got := v.GetMaxCursor(); got != tt.want {
				t.Errorf("GetMaxCursor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCursorLeftRightClampsToColumnRange(t *testing.T) {
	v := FilterView{}

	if v.Column != EnabledColumn {
		t.Fatalf("default Column = %v, want EnabledColumn", v.Column)
	}

	if got := v.CursorLeft(); got != int(EnabledColumn) {
		t.Errorf("CursorLeft() at leftmost column = %d, want %d (should not go below EnabledColumn)", got, EnabledColumn)
	}

	if got := v.CursorRight(); got != int(CaseSensitiveColumn) {
		t.Errorf("CursorRight() = %d, want %d", got, CaseSensitiveColumn)
	}
	if got := v.CursorRight(); got != int(ExcludingColumn) {
		t.Errorf("CursorRight() = %d, want %d", got, ExcludingColumn)
	}
	if got := v.CursorRight(); got != int(ExcludingColumn) {
		t.Errorf("CursorRight() past the end = %d, want %d (should clamp)", got, ExcludingColumn)
	}

	if got := v.CursorLeft(); got != int(CaseSensitiveColumn) {
		t.Errorf("CursorLeft() back = %d, want %d", got, CaseSensitiveColumn)
	}
	if got := v.CursorLeft(); got != int(EnabledColumn) {
		t.Errorf("CursorLeft() back = %d, want %d", got, EnabledColumn)
	}
}

func TestToggleEnabledColumn(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "a", false, true, "#000000")},
		Column:  EnabledColumn,
	}

	v.Toggle()
	if v.Filters[0].IsEnabled {
		t.Error("IsEnabled = true after Toggle, want false")
	}

	v.Toggle()
	if !v.Filters[0].IsEnabled {
		t.Error("IsEnabled = false after second Toggle, want true")
	}
}

func TestToggleCaseSensitiveColumnRecompilesRegex(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "hello", false, true, "#000000")},
		Column:  CaseSensitiveColumn,
	}

	if !v.Filters[0].Regex.MatchString("HELLO") {
		t.Fatal("precondition failed: expected case-insensitive match before toggling")
	}

	v.Toggle()

	if !v.Filters[0].CaseSensitive {
		t.Error("CaseSensitive = false after Toggle, want true")
	}
	if v.Filters[0].Regex.MatchString("HELLO") {
		t.Error("regex still matches different case after enabling case sensitivity")
	}
	if !v.Filters[0].Regex.MatchString("hello") {
		t.Error("regex no longer matches same-case text after enabling case sensitivity")
	}

	v.Toggle()
	if v.Filters[0].CaseSensitive {
		t.Error("CaseSensitive = true after second Toggle, want false")
	}
	if !v.Filters[0].Regex.MatchString("HELLO") {
		t.Error("regex should match different case again after disabling case sensitivity")
	}
}

func TestToggleExcludingColumn(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "heartbeat", false, true, "#000000")},
		Column:  ExcludingColumn,
	}

	v.Toggle()
	if !v.Filters[0].Excluding {
		t.Error("Excluding = false after Toggle, want true")
	}

	v.Toggle()
	if v.Filters[0].Excluding {
		t.Error("Excluding = true after second Toggle, want false")
	}
}

func TestAddInsertsAfterCursorAndMovesCursorToIt(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
		},
		Cursor: 0,
	}

	v.Add()

	if len(v.Filters) != 3 {
		t.Fatalf("got %d filters, want 3", len(v.Filters))
	}
	if v.Cursor != 1 {
		t.Errorf("Cursor after Add = %d, want 1 (the new filter, inserted after the old cursor)", v.Cursor)
	}
	if v.Filters[0].XML.Text != "a" || v.Filters[2].XML.Text != "b" {
		t.Errorf("existing filters reordered unexpectedly: %+v", v.Filters)
	}
	if v.Filters[1].XML.Text != "" {
		t.Errorf("new filter text = %q, want empty", v.Filters[1].XML.Text)
	}
	if v.Filters[1].IsEnabled {
		t.Error("new filter should start disabled")
	}
	if v.Column != EnabledColumn {
		t.Errorf("Column after Add = %v, want EnabledColumn", v.Column)
	}
}

func TestAddOnEmptyList(t *testing.T) {
	v := FilterView{}

	v.Add()

	if len(v.Filters) != 1 {
		t.Fatalf("got %d filters, want 1", len(v.Filters))
	}
	if v.Cursor != 0 {
		t.Errorf("Cursor after Add on empty list = %d, want 0", v.Cursor)
	}
}

func TestDeleteRemovesAndClampsCursor(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
		},
		Cursor: 1,
	}

	v.Delete()

	if len(v.Filters) != 1 {
		t.Fatalf("got %d filters, want 1", len(v.Filters))
	}
	if v.Filters[0].XML.Text != "a" {
		t.Errorf("remaining filter = %q, want %q", v.Filters[0].XML.Text, "a")
	}
	if v.Cursor != 0 {
		t.Errorf("Cursor after deleting the last row = %d, want 0 (clamped)", v.Cursor)
	}
}

func TestDeleteOnEmptyListIsNoOp(t *testing.T) {
	v := FilterView{}
	v.Delete() // should not panic
	if len(v.Filters) != 0 {
		t.Errorf("got %d filters, want 0", len(v.Filters))
	}
}

func TestDeleteLastFilterLeavesEmptyList(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "a", false, true, "#000000")},
		Cursor:  0,
	}

	v.Delete()

	if len(v.Filters) != 0 {
		t.Fatalf("got %d filters, want 0", len(v.Filters))
	}
	if v.Cursor != 0 {
		t.Errorf("Cursor after deleting the only filter = %d, want 0", v.Cursor)
	}

	// Toggle must not panic against an empty list.
	v.Toggle()
}

func TestMoveUpAndDown(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
			mustFilter(t, "c", false, true, "#000000"),
		},
		Cursor: 1,
	}

	v.MoveUp()
	if v.Cursor != 0 {
		t.Errorf("Cursor after MoveUp = %d, want 0", v.Cursor)
	}
	texts := []string{v.Filters[0].XML.Text, v.Filters[1].XML.Text, v.Filters[2].XML.Text}
	if texts[0] != "b" || texts[1] != "a" || texts[2] != "c" {
		t.Errorf("filter order after MoveUp = %v, want [b a c]", texts)
	}

	v.MoveDown()
	if v.Cursor != 1 {
		t.Errorf("Cursor after MoveDown = %d, want 1", v.Cursor)
	}
	texts = []string{v.Filters[0].XML.Text, v.Filters[1].XML.Text, v.Filters[2].XML.Text}
	if texts[0] != "a" || texts[1] != "b" || texts[2] != "c" {
		t.Errorf("filter order after MoveDown = %v, want [a b c] (back to original)", texts)
	}
}

func TestMoveUpAtTopIsNoOp(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
		},
		Cursor: 0,
	}
	if moved := v.MoveUp(); moved {
		t.Error("MoveUp() at top returned true, want false (no-op)")
	}
	if v.Cursor != 0 {
		t.Errorf("Cursor after MoveUp at top = %d, want unchanged 0", v.Cursor)
	}
	if v.Filters[0].XML.Text != "a" {
		t.Errorf("filter order changed after a no-op MoveUp: %v", v.Filters)
	}
}

func TestMoveDownAtBottomIsNoOp(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
		},
		Cursor: 1,
	}
	if moved := v.MoveDown(); moved {
		t.Error("MoveDown() at bottom returned true, want false (no-op)")
	}
	if v.Cursor != 1 {
		t.Errorf("Cursor after MoveDown at bottom = %d, want unchanged 1", v.Cursor)
	}
	if v.Filters[1].XML.Text != "b" {
		t.Errorf("filter order changed after a no-op MoveDown: %v", v.Filters)
	}
}

func TestMoveUpDownOnEmptyOrSingleListIsNoOp(t *testing.T) {
	empty := FilterView{}
	if empty.MoveUp() || empty.MoveDown() {
		t.Error("MoveUp/MoveDown on an empty list returned true, want false")
	}

	single := FilterView{Filters: []filterfiles.Filter{mustFilter(t, "a", false, true, "#000000")}}
	if single.MoveUp() || single.MoveDown() {
		t.Error("MoveUp/MoveDown on a single-element list returned true, want false")
	}
	if single.Filters[0].XML.Text != "a" {
		t.Error("single-element list should be unaffected by MoveUp/MoveDown")
	}
}

func TestMoveUpDownReturnTrueWhenTheyActuallyMove(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "a", false, true, "#000000"),
			mustFilter(t, "b", false, true, "#000000"),
		},
		Cursor: 1,
	}
	if moved := v.MoveUp(); !moved {
		t.Error("MoveUp() with room to move returned false, want true")
	}
	if moved := v.MoveDown(); !moved {
		t.Error("MoveDown() with room to move returned false, want true")
	}
}

func TestRenderShowsSelectedColumnAsBraces(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "^debug", false, true, "#87CEFA"),
			mustFilter(t, "goodbye", false, false, "#FF0000"),
		},
		Cursor: 0,
		Column: EnabledColumn,
	}

	out := v.Render(120, 30, nil)

	if !strings.Contains(out, "{x}") {
		t.Errorf("expected selected enabled checkbox to render as {x}, got:\n%s", out)
	}
	if strings.Contains(out, "{ }") {
		t.Errorf("unselected checkboxes should use brackets, not braces, got:\n%s", out)
	}
	if !strings.Contains(out, "[ ]") {
		t.Errorf("expected the disabled second filter's checkbox to render as [ ], got:\n%s", out)
	}
	if !strings.Contains(out, "^debug") || !strings.Contains(out, "goodbye") {
		t.Errorf("expected both filters' regex text in output, got:\n%s", out)
	}
}

func TestRenderMovesSelectionWithColumn(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "^debug", false, true, "#87CEFA")},
		Cursor:  0,
		Column:  CaseSensitiveColumn,
	}

	out := v.Render(120, 30, nil)

	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least a header and one row, got %d lines", len(lines))
	}
	row := lines[1]

	if !strings.HasPrefix(row, "[x]") {
		t.Errorf("expected the (unselected) enabled checkbox first in the row, got: %q", row)
	}
	if !strings.Contains(row, "{ }") {
		t.Errorf("expected the selected case-sensitivity checkbox to render as { }, got: %q", row)
	}
}

func TestRenderMarksExcludingFilters(t *testing.T) {
	excluding := mustFilter(t, "heartbeat", false, true, "#000000")
	excluding.Excluding = true

	v := FilterView{
		Filters: []filterfiles.Filter{
			excluding,
			mustFilter(t, "ERROR", false, true, "#FF0000"),
		},
		Cursor: 0,
	}

	out := v.Render(120, 30, nil)
	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a header and two rows, got %d lines", len(lines))
	}

	if !strings.HasSuffix(strings.TrimRight(lines[1], " "), "[x]") {
		t.Errorf("expected the excluding filter's row to end with a checked Excl checkbox, got: %q", lines[1])
	}
	if !strings.HasSuffix(strings.TrimRight(lines[2], " "), "[ ]") {
		t.Errorf("expected the non-excluding filter's row to end with an unchecked Excl checkbox, got: %q", lines[2])
	}
}

func TestRenderShowsDescription(t *testing.T) {
	f := mustFilter(t, "heartbeat", false, true, "#000000")
	f.XML.Description = "noisy health checks"

	v := FilterView{Filters: []filterfiles.Filter{f}}

	out := v.Render(120, 30, nil)

	if !strings.Contains(out, "noisy health checks") {
		t.Errorf("expected the filter's description in output, got:\n%s", out)
	}
}

func TestRenderShowsMatchCounts(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{
			mustFilter(t, "^debug", false, true, "#87CEFA"),
			mustFilter(t, "goodbye", false, true, "#FF0000"),
		},
	}

	out := v.Render(120, 30, []int{214, 3})

	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a header and two rows, got %d lines", len(lines))
	}
	if !strings.Contains(lines[1], "214") {
		t.Errorf("row 0 missing its match count, got: %q", lines[1])
	}
	if !strings.Contains(lines[2], "3") {
		t.Errorf("row 1 missing its match count, got: %q", lines[2])
	}
}

func TestRenderMatchCountsDefaultToZeroWhenMissing(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "a", false, true, "#000000")},
	}

	// nil and short counts slices should both be handled without panicking.
	out := v.Render(120, 30, nil)
	if !strings.Contains(out, "0") {
		t.Errorf("expected a zero count with nil counts, got: %q", out)
	}

	out = v.Render(120, 30, []int{})
	if !strings.Contains(out, "0") {
		t.Errorf("expected a zero count with an empty counts slice, got: %q", out)
	}
}

func TestRenderScrollsToKeepCursorVisible(t *testing.T) {
	var filters []filterfiles.Filter
	for i := 0; i < 10; i++ {
		filters = append(filters, mustFilter(t, "a", false, true, "#000000"))
	}
	v := FilterView{Filters: filters, Cursor: 9}

	out := v.Render(80, 30, nil)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// header + visibleHeight rows
	if len(lines) != 1+visibleHeight {
		t.Fatalf("got %d lines, want %d (header + %d visible rows)", len(lines), 1+visibleHeight, visibleHeight)
	}
}

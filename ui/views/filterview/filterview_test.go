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
	if got := v.CursorRight(); got != int(CaseSensitiveColumn) {
		t.Errorf("CursorRight() past the end = %d, want %d (should clamp)", got, CaseSensitiveColumn)
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

func TestUpdateRegexText(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "hello", false, true, "#000000")},
	}

	if err := v.UpdateRegexText(0, "goodbye"); err != nil {
		t.Fatalf("UpdateRegexText returned unexpected error: %v", err)
	}
	if v.Filters[0].XML.Text != "goodbye" {
		t.Errorf("XML.Text = %q, want %q", v.Filters[0].XML.Text, "goodbye")
	}
	if !v.Filters[0].Regex.MatchString("goodbye world") {
		t.Error("recompiled regex does not match the new text")
	}
	if v.Filters[0].Regex.MatchString("hello world") {
		t.Error("recompiled regex still matches the old text")
	}
}

func TestUpdateRegexTextPreservesCaseSensitivity(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "hello", true, true, "#000000")},
	}

	if err := v.UpdateRegexText(0, "Hello"); err != nil {
		t.Fatalf("UpdateRegexText returned unexpected error: %v", err)
	}
	if v.Filters[0].Regex.MatchString("HELLO") {
		t.Error("regex should remain case-sensitive after UpdateRegexText")
	}
	if !v.Filters[0].Regex.MatchString("Hello") {
		t.Error("regex should match the exact-case new text")
	}
}

func TestUpdateRegexTextInvalidRegexLeavesFilterUnchanged(t *testing.T) {
	v := FilterView{
		Filters: []filterfiles.Filter{mustFilter(t, "hello", false, true, "#000000")},
	}

	err := v.UpdateRegexText(0, "([unclosed")
	if err == nil {
		t.Fatal("UpdateRegexText with invalid regex returned no error")
	}
	if v.Filters[0].XML.Text != "hello" {
		t.Errorf("XML.Text = %q after failed update, want unchanged %q", v.Filters[0].XML.Text, "hello")
	}
}

func TestUpdateRegexTextOutOfRange(t *testing.T) {
	v := FilterView{Filters: []filterfiles.Filter{mustFilter(t, "hello", false, true, "#000000")}}

	if err := v.UpdateRegexText(5, "goodbye"); err == nil {
		t.Fatal("UpdateRegexText with out-of-range index returned no error")
	}
	if err := v.UpdateRegexText(-1, "goodbye"); err == nil {
		t.Fatal("UpdateRegexText with negative index returned no error")
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

	out := v.Render(120, 30)

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

	out := v.Render(120, 30)

	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least a header and one row, got %d lines", len(lines))
	}
	row := lines[1]

	if !strings.HasPrefix(row, "[x]") {
		t.Errorf("expected the (unselected) enabled checkbox first in the row, got: %q", row)
	}
	if !strings.HasSuffix(strings.TrimRight(row, " "), "{ }") {
		t.Errorf("expected the selected case-sensitivity checkbox last in the row as { }, got: %q", row)
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

	out := v.Render(120, 30)

	if !strings.Contains(out, "! heartbeat") {
		t.Errorf("expected the excluding filter's regex to be marked with \"! \", got:\n%s", out)
	}
	if strings.Contains(out, "! ERROR") {
		t.Errorf("non-excluding filter should not be marked, got:\n%s", out)
	}
}

func TestRenderScrollsToKeepCursorVisible(t *testing.T) {
	var filters []filterfiles.Filter
	for i := 0; i < 10; i++ {
		filters = append(filters, mustFilter(t, "a", false, true, "#000000"))
	}
	v := FilterView{Filters: filters, Cursor: 9}

	out := v.Render(80, 30)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// header + visibleHeight rows
	if len(lines) != 1+visibleHeight {
		t.Fatalf("got %d lines, want %d (header + %d visible rows)", len(lines), 1+visibleHeight, visibleHeight)
	}
}

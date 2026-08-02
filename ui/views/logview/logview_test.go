package logview

import (
	"regexp"
	"skim/filterfiles"
	"strings"
	"testing"
)

func mustRegex(t *testing.T, text string) regexp.Regexp {
	t.Helper()
	re, err := filterfiles.CompileRegex(text, false)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return re
}

func TestCursorUpDown(t *testing.T) {
	v := LogView{Lines: []string{"a", "b", "c"}}

	if got := v.CursorUp(); got != 0 {
		t.Errorf("CursorUp() at top = %d, want 0 (should not go negative)", got)
	}

	if got := v.CursorDown(); got != 1 {
		t.Errorf("CursorDown() = %d, want 1", got)
	}
	if got := v.CursorDown(); got != 2 {
		t.Errorf("CursorDown() = %d, want 2", got)
	}
	if got := v.CursorDown(); got != 2 {
		t.Errorf("CursorDown() past the end = %d, want 2 (should clamp at max)", got)
	}

	if got := v.CursorUp(); got != 1 {
		t.Errorf("CursorUp() = %d, want 1", got)
	}
}

func TestCursorLeftRightAreNoOps(t *testing.T) {
	v := LogView{Lines: []string{"a"}, Cursor: 0}

	if got := v.CursorLeft(); got != 0 {
		t.Errorf("CursorLeft() = %d, want 0", got)
	}
	if got := v.CursorRight(); got != 0 {
		t.Errorf("CursorRight() = %d, want 0", got)
	}
	if v.Cursor != 0 {
		t.Errorf("Cursor changed to %d after CursorLeft/Right, want unchanged", v.Cursor)
	}
}

func TestToggleIsNoOp(t *testing.T) {
	v := LogView{Lines: []string{"a"}}
	// Should not panic and should not alter any observable state.
	v.Toggle()
}

func TestGetMaxCursor(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
	}{
		{"empty", nil, -1},
		{"one line", []string{"a"}, 0},
		{"three lines", []string{"a", "b", "c"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := LogView{Lines: tt.lines}
			if got := v.GetMaxCursor(); got != tt.want {
				t.Errorf("GetMaxCursor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func mustFilter(t *testing.T, text string, backColor string) filterfiles.Filter {
	t.Helper()
	re, err := filterfiles.CompileRegex(text, false)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return filterfiles.Filter{
		XML:       filterfiles.FilterXML{Text: text},
		Regex:     re,
		IsEnabled: true,
		BackColor: backColor,
	}
}

func TestMakeTableHidesOrShowsUnmatchedLines(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"debug: one", "info: two", "debug: three"}}

	t.Run("hides unmatched lines", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, true)
		rows := table.Rows()
		if len(rows) != 2 {
			t.Fatalf("got %d rows, want 2 (only matching lines)", len(rows))
		}
		for _, row := range rows {
			if !strings.Contains(row[1], "debug") {
				t.Errorf("row %v does not look like a matched debug line", row)
			}
		}
	})

	t.Run("shows unmatched lines", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, false)
		rows := table.Rows()
		if len(rows) != 3 {
			t.Fatalf("got %d rows, want 3 (all lines)", len(rows))
		}
	})
}

func TestMakeTableLineNumbersAreOneIndexed(t *testing.T) {
	v := LogView{Lines: []string{"first", "second"}}
	table := v.MakeTable(100, 30, nil, false)
	rows := table.Rows()

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0][0] != "1" {
		t.Errorf("first row line number = %q, want %q", rows[0][0], "1")
	}
	if rows[1][0] != "2" {
		t.Errorf("second row line number = %q, want %q", rows[1][0], "2")
	}
}

func TestMakeTableReplacesTabsWithSpaces(t *testing.T) {
	v := LogView{Lines: []string{"a\tb"}}
	table := v.MakeTable(100, 30, nil, false)
	rows := table.Rows()

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if strings.Contains(rows[0][1], "\t") {
		t.Errorf("row content %q still contains a tab character", rows[0][1])
	}
	if !strings.Contains(rows[0][1], "a    b") {
		t.Errorf("row content %q does not contain the expected tab-expanded text", rows[0][1])
	}
}

func TestFindNextWrapsAndSkipsCurrentLine(t *testing.T) {
	v := LogView{Lines: []string{"apple", "banana", "cherry", "banana split"}, Cursor: 1}
	re := mustRegex(t, "banana")

	idx, ok := v.FindNext(re)
	if !ok {
		t.Fatal("FindNext() found no match, want one")
	}
	if idx != 3 {
		t.Errorf("FindNext() = %d, want 3 (the next banana after Cursor, not Cursor's own line)", idx)
	}

	v.Cursor = 3
	idx, ok = v.FindNext(re)
	if !ok {
		t.Fatal("FindNext() found no match, want one")
	}
	if idx != 1 {
		t.Errorf("FindNext() from the last match = %d, want 1 (wraps around to the start)", idx)
	}
}

func TestFindPrevWrapsAndSkipsCurrentLine(t *testing.T) {
	v := LogView{Lines: []string{"apple", "banana", "cherry", "banana split"}, Cursor: 3}
	re := mustRegex(t, "banana")

	idx, ok := v.FindPrev(re)
	if !ok {
		t.Fatal("FindPrev() found no match, want one")
	}
	if idx != 1 {
		t.Errorf("FindPrev() = %d, want 1 (the previous banana before Cursor)", idx)
	}

	v.Cursor = 1
	idx, ok = v.FindPrev(re)
	if !ok {
		t.Fatal("FindPrev() found no match, want one")
	}
	if idx != 3 {
		t.Errorf("FindPrev() from the first match = %d, want 3 (wraps around to the end)", idx)
	}
}

func TestFindNextNoMatch(t *testing.T) {
	v := LogView{Lines: []string{"apple", "banana"}}
	re := mustRegex(t, "nonexistent")

	if _, ok := v.FindNext(re); ok {
		t.Error("FindNext() found a match, want none")
	}
	if _, ok := v.FindPrev(re); ok {
		t.Error("FindPrev() found a match, want none")
	}
}

func TestFindNextEmptyLog(t *testing.T) {
	v := LogView{}
	re := mustRegex(t, "anything")

	if _, ok := v.FindNext(re); ok {
		t.Error("FindNext() on an empty log found a match, want none")
	}
	if _, ok := v.FindPrev(re); ok {
		t.Error("FindPrev() on an empty log found a match, want none")
	}
}

func TestMakeTableCursorRowTracksVisibleLineWhenLinesAreHidden(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "keep", "#87CEFA")}
	// Lines 0 and 2 are hidden by hideUnmatched; only "keep one" (1) and
	// "keep two" (3) are visible.
	v := LogView{Lines: []string{"drop", "keep one", "drop", "keep two"}, Cursor: 2}

	table := v.MakeTable(100, 30, filters, true)

	// Cursor (2) points at a hidden line ("drop"); the highlighted row
	// should land on the last visible row at or before it: "keep one".
	if got := table.SelectedRow(); got[1] != "keep one" {
		t.Errorf("SelectedRow() = %v, want row for %q", got, "keep one")
	}

	v.Cursor = 3
	table = v.MakeTable(100, 30, filters, true)
	if got := table.SelectedRow(); got[1] != "keep two" {
		t.Errorf("SelectedRow() = %v, want row for %q", got, "keep two")
	}
}

package logview

import (
	"skim/filterfiles"
	"strings"
	"testing"
)

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

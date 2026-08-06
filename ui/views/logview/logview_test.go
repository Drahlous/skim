package logview

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"skim/filterfiles"
	"strconv"
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
		table := v.MakeTable(100, 30, filters, true, 0)
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
		table := v.MakeTable(100, 30, filters, false, 0)
		rows := table.Rows()
		if len(rows) != 3 {
			t.Fatalf("got %d rows, want 3 (all lines)", len(rows))
		}
	})
}

func TestMakeTableLineNumbersAreOneIndexed(t *testing.T) {
	v := LogView{Lines: []string{"first", "second"}}
	table := v.MakeTable(100, 30, nil, false, 0)
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

func TestMakeTableLineNumberColumnWidensForLargeLineCounts(t *testing.T) {
	numLines := 500000
	lines := make([]string, numLines)
	for i := range lines {
		lines[i] = "line"
	}
	v := LogView{Lines: lines, Cursor: numLines - 1}
	tbl := v.MakeTable(100, 30, nil, false, 0)
	view := tbl.View()

	want := strconv.Itoa(numLines)
	if !strings.Contains(view, want) {
		t.Errorf("rendered view does not contain the full line number %q; got:\n%s", want, view)
	}
	if strings.Contains(view, "…") {
		t.Errorf("rendered view contains a truncation ellipsis, want the full line number visible:\n%s", view)
	}
}

func TestMakeTableFillsExactlyWindowWidth(t *testing.T) {
	windowWidth := 100
	v := LogView{Lines: []string{"hello"}}
	tbl := v.MakeTable(windowWidth, 30, nil, false, 0)

	// MakeTable's own render doesn't include ui.go's pane border (that's
	// applied afterward by ui.go's paneStyle.Render) -- so its content
	// should exactly fill windowWidth minus that border, with nothing left
	// over and nothing overflowing it. This is the invariant tableChromeWidth
	// exists to preserve; a wrong offset here would either waste terminal
	// columns or push the table wider than the pane border can absorb.
	want := windowWidth - paneBorderStyle.GetHorizontalFrameSize()
	for _, line := range strings.Split(tbl.View(), "\n") {
		if got := lipgloss.Width(line); got != want {
			t.Errorf("line %q rendered at width %d, want %d (windowWidth - pane border)", line, got, want)
		}
	}
}

func TestMakeTableReplacesTabsWithSpaces(t *testing.T) {
	v := LogView{Lines: []string{"a\tb"}}
	table := v.MakeTable(100, 30, nil, false, 0)
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

func TestMakeTableSanitizesControlCharacters(t *testing.T) {
	v := LogView{Lines: []string{"mixed CRLF-ish line\r trailing"}}
	table := v.MakeTable(100, 30, nil, false, 0)
	rows := table.Rows()

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if strings.ContainsAny(rows[0][1], "\r\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f\x1f") {
		t.Errorf("row content %q still contains a raw control character", rows[0][1])
	}
	if !strings.Contains(rows[0][1], "mixed CRLF-ish line") || !strings.Contains(rows[0][1], "trailing") {
		t.Errorf("row content %q lost surrounding text", rows[0][1])
	}
}

func TestMakeTableStripsAnsiEscapeSequences(t *testing.T) {
	v := LogView{Lines: []string{"\x1b[33mWARNING\x1b[0m: something happened"}}
	table := v.MakeTable(100, 30, nil, false, 0)
	rows := table.Rows()

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if strings.ContainsAny(rows[0][1], "\x1b[]") {
		t.Errorf("row content %q still contains ANSI escape sequence junk", rows[0][1])
	}
	if !strings.Contains(rows[0][1], "WARNING") || !strings.Contains(rows[0][1], "something happened") {
		t.Errorf("row content %q lost surrounding text", rows[0][1])
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

	table := v.MakeTable(100, 30, filters, true, 0)

	// Cursor (2) points at a hidden line ("drop"); the highlighted row
	// should land on the last visible row at or before it: "keep one".
	if got := table.SelectedRow(); got[1] != "keep one" {
		t.Errorf("SelectedRow() = %v, want row for %q", got, "keep one")
	}

	v.Cursor = 3
	table = v.MakeTable(100, 30, filters, true, 0)
	if got := table.SelectedRow(); got[1] != "keep two" {
		t.Errorf("SelectedRow() = %v, want row for %q", got, "keep two")
	}
}

func mustExcludingFilter(t *testing.T, text string) filterfiles.Filter {
	t.Helper()
	re, err := filterfiles.CompileRegex(text, false)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return filterfiles.Filter{
		XML:       filterfiles.FilterXML{Text: text},
		Regex:     re,
		IsEnabled: true,
		Excluding: true,
	}
}

func TestMakeTableHidesExcludedLinesEvenWithHideUnmatchedOff(t *testing.T) {
	filters := []filterfiles.Filter{
		mustFilter(t, "^debug", "#87CEFA"),
		mustExcludingFilter(t, "noisy"),
	}
	v := LogView{Lines: []string{"debug: one", "noisy: skip me", "info: two"}}

	// hideUnmatched is off, so ordinarily every line would show; the
	// excluding filter should still remove its match.
	table := v.MakeTable(100, 30, filters, false, 0)
	rows := table.Rows()

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (excluded line dropped)", len(rows))
	}
	for _, row := range rows {
		if strings.Contains(row[1], "noisy") {
			t.Errorf("row %v should have been excluded, but is present", row)
		}
	}
}

func TestMakeTableExcludingBeatsHighlightingMatch(t *testing.T) {
	filters := []filterfiles.Filter{
		mustExcludingFilter(t, "heartbeat"),
		mustFilter(t, "heartbeat", "#87CEFA"), // would otherwise highlight the same line
	}
	v := LogView{Lines: []string{"heartbeat: ok", "other line"}}

	table := v.MakeTable(100, 30, filters, false, 0)
	rows := table.Rows()

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 (heartbeat line excluded despite also matching a highlighting filter)", len(rows))
	}
	if rows[0][1] != "other line" {
		t.Errorf("remaining row = %v, want the non-excluded line", rows[0])
	}
}

func TestMakeTableContextLinesShowsNeighborsOfAMatch(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"info: zero", "info: one", "debug: two", "info: three", "info: four"}}

	t.Run("zero context matches existing hide-unmatched behavior", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, true, 0)
		rows := table.Rows()
		if len(rows) != 1 {
			t.Fatalf("got %d rows, want 1 (only the match)", len(rows))
		}
		if rows[0][0] != "3" {
			t.Errorf("row line number = %q, want %q", rows[0][0], "3")
		}
	})

	t.Run("context 1 includes one neighbor on each side", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, true, 1)
		rows := table.Rows()
		if len(rows) != 3 {
			t.Fatalf("got %d rows, want 3 (match plus one line of context each side), got: %v", len(rows), rows)
		}
		got := []string{rows[0][0], rows[1][0], rows[2][0]}
		want := []string{"2", "3", "4"}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d line number = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("context is clamped at the ends of the log", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, true, 10)
		rows := table.Rows()
		if len(rows) != 5 {
			t.Fatalf("got %d rows, want 5 (the whole log, context clamped at both ends)", len(rows))
		}
	})

	t.Run("context is irrelevant when hideUnmatched is off", func(t *testing.T) {
		table := v.MakeTable(100, 30, filters, false, 2)
		rows := table.Rows()
		if len(rows) != 5 {
			t.Fatalf("got %d rows, want 5 (everything already shown)", len(rows))
		}
	})
}

func TestMakeTableContextLinesIncludesUnmatchedNeighborContent(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"info: before", "debug: match"}}

	table := v.MakeTable(100, 30, filters, true, 1)
	rows := table.Rows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if !strings.Contains(rows[0][1], "info: before") {
		t.Errorf("context row = %v, want it to contain the unmatched neighbor's text", rows[0])
	}
	if !strings.Contains(rows[1][1], "debug: match") {
		t.Errorf("matched row = %v, want it to contain the matched line's text", rows[1])
	}
}

func genLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return lines
}

func TestMakeTableWindowsRowsForLargeLogs(t *testing.T) {
	lines := genLines(1000)
	v := LogView{Lines: lines, Cursor: 500}

	table := v.MakeTable(100, 30, nil, false, 0)
	rows := table.Rows()

	if len(rows) >= len(lines) {
		t.Fatalf("got %d rows, want far fewer than the full %d-line log (windowing should have kicked in)", len(rows), len(lines))
	}
	if got := table.SelectedRow(); got[1] != "line 501" {
		t.Errorf("SelectedRow() = %v, want the row for line 501 (Cursor=500, 1-indexed)", got)
	}
	if v.ShownCount != len(lines) {
		t.Errorf("ShownCount = %d, want %d (every line, since hideUnmatched is off)", v.ShownCount, len(lines))
	}
}

func TestMakeTableWindowNearTopOfLog(t *testing.T) {
	lines := genLines(1000)
	v := LogView{Lines: lines, Cursor: 0}

	table := v.MakeTable(100, 30, nil, false, 0)
	rows := table.Rows()

	if len(rows) == 0 || rows[0][1] != "line 1" {
		t.Fatalf("first windowed row = %v, want line 1 (window should start at the top when the cursor is there)", rows[0])
	}
	if got := table.SelectedRow(); got[1] != "line 1" {
		t.Errorf("SelectedRow() = %v, want line 1", got)
	}
}

func TestMakeTableWindowNearBottomOfLog(t *testing.T) {
	lines := genLines(1000)
	v := LogView{Lines: lines, Cursor: 999}

	table := v.MakeTable(100, 30, nil, false, 0)
	rows := table.Rows()

	if len(rows) == 0 || rows[len(rows)-1][1] != "line 1000" {
		t.Fatalf("last windowed row = %v, want line 1000 (window should end at the bottom when the cursor is there)", rows[len(rows)-1])
	}
	if got := table.SelectedRow(); got[1] != "line 1000" {
		t.Errorf("SelectedRow() = %v, want line 1000", got)
	}
}

func TestShownCountReflectsFullLogNotJustTheWindow(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	lines := make([]string, 1000)
	for i := range lines {
		if i%2 == 0 {
			lines[i] = fmt.Sprintf("debug: line %d", i+1)
		} else {
			lines[i] = fmt.Sprintf("info: line %d", i+1)
		}
	}
	v := LogView{Lines: lines, Cursor: 500}

	v.MakeTable(100, 30, filters, true, 0)

	if v.ShownCount != 500 {
		t.Errorf("ShownCount = %d, want 500 (half the lines match ^debug)", v.ShownCount)
	}
	if len(v.Table.Rows()) >= v.ShownCount {
		t.Errorf("got %d rows, want fewer than ShownCount (%d) -- windowing should still apply", len(v.Table.Rows()), v.ShownCount)
	}
}

func TestMakeTableWindowedCursorSnapsToVisibleLineWhenHidden(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "keep", "#87CEFA")}
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("drop %d", i)
	}
	lines[500] = "keep this one"
	// Cursor points at a hidden line just after the only match; it should
	// snap to the nearest preceding visible row, same as the small-log
	// behavior in TestMakeTableCursorRowTracksVisibleLineWhenLinesAreHidden,
	// but exercised on a log large enough to actually trigger windowing.
	v := LogView{Lines: lines, Cursor: 501}

	table := v.MakeTable(100, 30, filters, true, 0)
	if got := table.SelectedRow(); got[1] != "keep this one" {
		t.Errorf("SelectedRow() = %v, want the only visible line", got)
	}
}

func TestMatchCacheReusedWhenFiltersUnchanged(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"debug: one", "info: two"}}

	v.MakeTable(100, 30, filters, true, 0)
	first := v.matchCache

	v.MakeTable(100, 30, filters, true, 0)
	second := v.matchCache

	if len(first) == 0 || &first[0] != &second[0] {
		t.Error("matchCache was rebuilt even though the filter set didn't change")
	}
}

func TestMatchCacheInvalidatesWhenFilterRegexChanges(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"debug: one", "info: two"}}

	table := v.MakeTable(100, 30, filters, true, 0)
	if rows := table.Rows(); len(rows) != 1 || !strings.Contains(rows[0][1], "debug") {
		t.Fatalf("precondition: got rows %v, want just the debug line", rows)
	}

	filters[0] = mustFilter(t, "^info", "#87CEFA")
	table = v.MakeTable(100, 30, filters, true, 0)
	rows := table.Rows()
	if len(rows) != 1 || !strings.Contains(rows[0][1], "info") {
		t.Errorf("cache was not invalidated after the filter's regex changed: rows = %v", rows)
	}
}

func TestMatchCacheInvalidatesWhenFilterEnabledToggles(t *testing.T) {
	filters := []filterfiles.Filter{mustFilter(t, "^debug", "#87CEFA")}
	v := LogView{Lines: []string{"debug: one", "info: two"}}

	v.MakeTable(100, 30, filters, true, 0)

	filters[0].IsEnabled = false
	table := v.MakeTable(100, 30, filters, true, 0)
	if rows := table.Rows(); len(rows) != 0 {
		t.Errorf("got %d rows after disabling the only filter, want 0: %v", len(rows), rows)
	}
}

func TestMatchCountsMatchesCountMatches(t *testing.T) {
	filters := []filterfiles.Filter{
		mustFilter(t, "^debug", "#87CEFA"),
		mustFilter(t, "^info", "#90EE90"),
	}
	lines := []string{"debug: one", "info: two", "debug: three", "other"}
	v := LogView{Lines: lines}

	got := v.MatchCounts(filters)
	want := filterfiles.CountMatches(filters, lines)

	if len(got) != len(want) {
		t.Fatalf("MatchCounts() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MatchCounts()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestMatchCountsCountsExcludedLinesLikeCountMatches(t *testing.T) {
	// filterfiles.CountMatches doesn't check IsExcluded -- a line that
	// matches both a highlighting filter and an excluding filter still
	// counts toward the highlighting filter's total, even though it would
	// never actually be rendered. MatchCounts must preserve that, not
	// silently start excluding such lines from the count.
	filters := []filterfiles.Filter{
		mustFilter(t, "heartbeat", "#87CEFA"),
		mustExcludingFilter(t, "heartbeat"),
	}
	lines := []string{"heartbeat: ok"}
	v := LogView{Lines: lines}

	got := v.MatchCounts(filters)
	want := filterfiles.CountMatches(filters, lines)
	if got[0] != want[0] {
		t.Errorf("MatchCounts()[0] = %d, want %d (should match CountMatches despite the line also being excluded)", got[0], want[0])
	}
}

func TestMakeTableExcludedLineStaysHiddenEvenAsContext(t *testing.T) {
	// A line that would otherwise be pulled in as context around a nearby
	// match must still be dropped if it independently matches an excluding
	// filter: exclusion is unconditional, regardless of *why* a line would
	// otherwise have been shown.
	filters := []filterfiles.Filter{
		mustFilter(t, "^debug", "#87CEFA"),
		mustExcludingFilter(t, "secret"),
	}
	v := LogView{Lines: []string{"secret: redacted", "debug: match"}}

	table := v.MakeTable(100, 30, filters, true, 1)
	rows := table.Rows()

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 (the excluded neighbor must not appear as context), rows: %v", len(rows), rows)
	}
	if !strings.Contains(rows[0][1], "debug: match") {
		t.Errorf("remaining row = %v, want the matched line", rows[0])
	}
}

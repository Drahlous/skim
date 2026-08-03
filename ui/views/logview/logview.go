package logview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"skim/filterfiles"
	"sort"
	"strconv"
	"strings"
)

type LogView struct {
	Cursor int // which log line our cursor is pointing at
	Table  table.Model
	Lines  []string

	// ShownCount is the total number of lines MakeTable's last call
	// considered "shown" (matched, not excluded), across the whole log --
	// not just the ones actually turned into table.Rows (see MakeTable's
	// windowing). Callers that want an accurate "N of M lines shown" count
	// (e.g. the status line) should read this rather than
	// len(Table.Rows()), which only reflects the rendered window.
	ShownCount int

	// matchCache holds, for each line in Lines (same index), whether it's
	// excluded and which filter (if any) wins its highlight -- computed by
	// ensureMatchCache and reused across calls until the filter set
	// actually changes, so repeated renders between keystrokes (cursor
	// movement, a window resize) don't repeatedly re-run every filter's
	// regex against every line.
	matchCache    []matchState
	matchCacheKey string

	// matchCounts and matchCountsKey cache MatchCounts' result under the
	// same fingerprint as matchCache, so a repeated call (MatchCounts is
	// invoked on every render, not just when filters change) is an O(1)
	// copy instead of re-tallying all of matchCache from scratch.
	matchCounts    []int
	matchCountsKey string

	// shownIndices holds the indices (into Lines, ascending) of every line
	// that is currently shown (per hideUnmatched/contextLines) and not
	// excluded -- i.e. exactly the lines MakeTable would otherwise have to
	// rediscover with an O(len(Lines)) scan on every call. Built by
	// ensureShownIndices and reused until the filter set, hideUnmatched, or
	// contextLines actually change, so a pure cursor-movement render can
	// locate the cursor's row and its visible window with a binary search
	// and a bounded walk instead of scanning every line in the log.
	shownIndices    []int
	shownIndicesKey string
}

// matchState is one line's cached result against the current filter set.
type matchState struct {
	excluded    bool
	filterIndex int // index into the filters slice of the highlighting match, or -1
}

func (v *LogView) Toggle() {
	return
}

func (v *LogView) CursorUp() int {
	if v.Cursor > 0 {
		v.Cursor--
	}
	return v.Cursor
}

func (v *LogView) CursorDown() int {
	if v.Cursor < v.GetMaxCursor() {
		v.Cursor++
	}
	return v.Cursor
}

func (v *LogView) CursorLeft() int {
	return 0
}

func (v *LogView) CursorRight() int {
	return 0
}

func (v *LogView) GetMaxCursor() int {
	return len(v.Lines) - 1
}

// FindNext returns the index of the next line, after Cursor and wrapping
// around to the start, whose text matches re. It scans the full line set
// regardless of any active filters, so it can find a match even while that
// line is currently hidden by hideUnmatched.
func (v *LogView) FindNext(re regexp.Regexp) (int, bool) {
	n := len(v.Lines)
	if n == 0 {
		return 0, false
	}
	for i := 1; i <= n; i++ {
		idx := (v.Cursor + i) % n
		if re.MatchString(v.Lines[idx]) {
			return idx, true
		}
	}
	return 0, false
}

// FindPrev is FindNext in reverse: it returns the index of the previous
// line, before Cursor and wrapping around to the end, whose text matches re.
func (v *LogView) FindPrev(re regexp.Regexp) (int, bool) {
	n := len(v.Lines)
	if n == 0 {
		return 0, false
	}
	for i := 1; i <= n; i++ {
		idx := ((v.Cursor-i)%n + n) % n
		if re.MatchString(v.Lines[idx]) {
			return idx, true
		}
	}
	return 0, false
}

// filtersCacheKey builds a cheap fingerprint of filters' match-relevant
// fields (their regex source text, enabled, and excluding state -- order
// matters too, since matching is first-enabled-filter-wins), used to detect
// whether a cached matchState slice is still valid. It's O(filters), not
// O(lines), so computing it on every MakeTable/MatchCounts call is fine.
func filtersCacheKey(filters []filterfiles.Filter) string {
	var b strings.Builder
	for _, f := range filters {
		b.WriteString(f.Regex.String())
		b.WriteByte(0)
		if f.IsEnabled {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		if f.Excluding {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		b.WriteByte(0)
	}
	return b.String()
}

// ensureMatchCache (re)computes v.matchCache if the filter set has changed
// (or the cache has never been built, or Lines has changed length) since
// the last call, otherwise leaves the existing cache in place.
func (v *LogView) ensureMatchCache(filters []filterfiles.Filter) {
	key := filtersCacheKey(filters)
	if key == v.matchCacheKey && len(v.matchCache) == len(v.Lines) {
		return
	}

	cache := make([]matchState, len(v.Lines))
	for i, line := range v.Lines {
		idx, _ := filterfiles.GetMatchingFilterIndex(filters, line)
		cache[i] = matchState{
			excluded:    filterfiles.IsExcluded(filters, line),
			filterIndex: idx,
		}
	}
	v.matchCache = cache
	v.matchCacheKey = key
}

// MatchCounts returns, for each filter (by index), how many lines it is the
// highlighting match for -- the same result as filterfiles.CountMatches,
// but computed from (and populating) the per-line match cache so it doesn't
// re-run every filter's regex against every line a second time in the same
// frame that MakeTable already did. The tally itself is cached under the
// same key as matchCache, since MatchCounts is called on every render (see
// ui.go) and re-summing all of matchCache every time would otherwise be an
// O(len(Lines)) cost on every keystroke even when nothing changed.
func (v *LogView) MatchCounts(filters []filterfiles.Filter) []int {
	v.ensureMatchCache(filters)

	if v.matchCountsKey == v.matchCacheKey && v.matchCounts != nil {
		return append([]int(nil), v.matchCounts...)
	}

	counts := make([]int, len(filters))
	for _, ms := range v.matchCache {
		if ms.filterIndex >= 0 {
			counts[ms.filterIndex]++
		}
	}
	v.matchCounts = counts
	v.matchCountsKey = v.matchCacheKey
	return append([]int(nil), counts...)
}

var logStyle = lipgloss.NewStyle().
	Bold(false).
	Foreground(lipgloss.Color("#000000")).
	PaddingTop(0).
	PaddingLeft(0)

// shownLines reports, for each line index, whether it should be rendered:
// every line if hideUnmatched is off, otherwise any line that matches an
// enabled filter plus up to contextLines lines immediately before/after each
// match (grep -C style), so a hidden match's surrounding context isn't lost
// along with it. cache must already reflect the current filter set (see
// ensureMatchCache).
func shownLines(cache []matchState, hideUnmatched bool, contextLines int) []bool {
	n := len(cache)
	shown := make([]bool, n)
	if !hideUnmatched {
		for i := range shown {
			shown[i] = true
		}
		return shown
	}

	for i, ms := range cache {
		if ms.filterIndex >= 0 {
			lo, hi := i-contextLines, i+contextLines
			if lo < 0 {
				lo = 0
			}
			if hi >= n {
				hi = n - 1
			}
			for j := lo; j <= hi; j++ {
				shown[j] = true
			}
		}
	}
	return shown
}

// ensureShownIndices (re)computes v.shownIndices -- the indices of every
// shown, non-excluded line, in ascending order -- if the filter set,
// hideUnmatched, or contextLines have changed (or the cache has never been
// built) since the last call; otherwise it leaves the existing cache in
// place. Callers must call ensureMatchCache first, since this reads
// v.matchCache.
func (v *LogView) ensureShownIndices(filters []filterfiles.Filter, hideUnmatched bool, contextLines int) {
	key := v.matchCacheKey + "|" + strconv.FormatBool(hideUnmatched) + "|" + strconv.Itoa(contextLines)
	if key == v.shownIndicesKey && v.shownIndices != nil {
		return
	}

	shown := shownLines(v.matchCache, hideUnmatched, contextLines)
	indices := make([]int, 0, len(v.Lines))
	for i, ms := range v.matchCache {
		if shown[i] && !ms.excluded {
			indices = append(indices, i)
		}
	}
	v.shownIndices = indices
	v.shownIndicesKey = key
}

// buildRow formats and, if the line has a highlighting match, styles a
// single line into the table.Row bubbles/table will render.
func buildRow(i int, line string, ms matchState, filters []filterfiles.Filter) table.Row {
	lineNumber := i + 1
	line = strings.ReplaceAll(line, "\t", "    ")

	if ms.filterIndex >= 0 {
		style := logStyle
		style.Background(lipgloss.Color(filters[ms.filterIndex].BackColor))
		return table.Row{fmt.Sprintf("%d", lineNumber), style.Render(line)}
	}
	return table.Row{fmt.Sprintf("%d", lineNumber), line}
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func (v *LogView) MakeTable(windowWidth int, windowHeight int, filters []filterfiles.Filter, hideUnmatched bool, contextLines int) table.Model {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Line", Width: windowWidth - 10}, // TODO: Avoid hardcoding this offset
	}

	v.ensureMatchCache(filters)
	v.ensureShownIndices(filters, hideUnmatched, contextLines)

	// bubbles/table's own UpdateViewport only ever renders rows in
	// [cursor-height, cursor+height] (see its renderRow/UpdateViewport) --
	// building a fully formatted/styled table.Row for every shown line in
	// the whole log, only for almost all of them to never actually be
	// joined into the rendered output, is wasted work that scales with the
	// log's size instead of the terminal's. shownIndices (cached by
	// ensureShownIndices, and only rebuilt when filters/hideUnmatched/
	// contextLines change) lets us find cursorRow -- the position, among
	// shown/non-excluded lines, of the last such line at or before
	// v.Cursor -- with a binary search instead of an O(len(Lines)) scan.
	shownCount := len(v.shownIndices)
	v.ShownCount = shownCount

	// rank is the number of shownIndices <= v.Cursor; cursorRow is the last
	// such line's 0-based position, or 0 if none precede the cursor (this
	// matches the "snap to the nearest visible line" behavior when the
	// cursor itself sits on a hidden/excluded line).
	rank := sort.Search(shownCount, func(i int) bool { return v.shownIndices[i] > v.Cursor })
	cursorRow := 0
	if rank > 0 {
		cursorRow = rank - 1
	}

	// TODO: Replace hardcoded offset with the size of the filter section
	height := windowHeight - 12
	if height < 1 {
		height = 1
	}
	start := clamp(cursorRow-height, 0, cursorRow)
	end := clamp(cursorRow+height, cursorRow, shownCount)

	rows := make([]table.Row, 0, end-start)
	for _, i := range v.shownIndices[start:end] {
		rows = append(rows, buildRow(i, v.Lines[i], v.matchCache[i], filters))
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// cursorRow was computed relative to the full shown set; rows only
	// covers [start, end), so re-anchor it to the window before handing it
	// to the table.
	t.MoveDown(cursorRow - start)

	v.Table = t
	return t
}

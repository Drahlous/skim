package logview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"skim/filterfiles"
	"strings"
)

type LogView struct {
	Cursor int // which log line our cursor is pointing at
	Table  table.Model
	Lines  []string

	// matchCache holds, for each line in Lines (same index), whether it's
	// excluded and which filter (if any) wins its highlight -- computed by
	// ensureMatchCache and reused across calls until the filter set
	// actually changes, so repeated renders between keystrokes (cursor
	// movement, a window resize) don't repeatedly re-run every filter's
	// regex against every line.
	matchCache    []matchState
	matchCacheKey string
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
// frame that MakeTable already did.
func (v *LogView) MatchCounts(filters []filterfiles.Filter) []int {
	v.ensureMatchCache(filters)

	counts := make([]int, len(filters))
	for _, ms := range v.matchCache {
		if ms.filterIndex >= 0 {
			counts[ms.filterIndex]++
		}
	}
	return counts
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

func (v *LogView) MakeTable(windowWidth int, windowHeight int, filters []filterfiles.Filter, hideUnmatched bool, contextLines int) table.Model {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Line", Width: windowWidth - 10}, // TODO: Avoid hardcoding this offset
	}

	v.ensureMatchCache(filters)

	rows := []table.Row{}
	shown := shownLines(v.matchCache, hideUnmatched, contextLines)

	// cursorRow tracks the position, within the visible rows actually
	// rendered, of the last row at or before v.Cursor's line index. v.Cursor
	// indexes into the full (unfiltered) line set, so when hideUnmatched
	// drops lines, a raw row-count of v.Cursor would overshoot; this keeps
	// the highlighted row aligned with the log line the cursor logically
	// points at, even when lines before it are hidden.
	cursorRow := 0

	for i, line := range v.Lines {
		if !shown[i] {
			continue
		}

		// +1 Offset to make the first line number 1
		lineNumber := i + 1

		// Replace tabs with spaces
		line = strings.ReplaceAll(line, "\t", "    ")

		ms := v.matchCache[i]

		// A line matching an enabled excluding filter is always hidden,
		// regardless of hideUnmatched, context radius, or whether it would
		// otherwise match a highlighting filter or be pulled in as context
		// around a nearby match.
		if ms.excluded {
			continue
		}

		// Do any filters match this line? Lines shown only as context around
		// a match (or because hideUnmatched is off) render plainly.
		var row table.Row
		if ms.filterIndex >= 0 {
			// Style this log line with the color from the filter
			style := logStyle
			style.Background(lipgloss.Color(filters[ms.filterIndex].BackColor))
			row = table.Row{fmt.Sprintf("%d", lineNumber), style.Render(line)}
		} else {
			row = table.Row{fmt.Sprintf("%d", lineNumber), line}
		}
		rows = append(rows, row)

		if i <= v.Cursor {
			cursorRow = len(rows) - 1
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		// TODO: Replace hardcoded offset with the size of the filter section
		table.WithHeight(windowHeight-12),
	)

	// Move the view to the visible row corresponding to the log cursor
	t.MoveDown(cursorRow)

	v.Table = t
	return t
}

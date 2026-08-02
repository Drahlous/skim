# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

skim is a terminal UI tool for skimming plaintext log files, similar to TextAnalysisTool (TAT) or glogg. It reads a log file plus a TextAnalysisTool.NET `.tat` filter file (XML), and renders the log in a scrollable table with lines colored/highlighted according to matching filters.

## Commands

```sh
go build -v ./...              # build (mirrors .github/workflows/go.yml CI)
go run . -filter <path> -log <path>   # run with a specific filter/log file
go run .                       # run with defaults (./examples/simple_filter_two.tat, ./examples/simple_longer.log)
go vet ./...                   # static analysis
go test ./...                  # run unit tests (mirrors .github/workflows/go.yml CI)
go run ./tools/gentestlogs      # regenerate synthetic medium/large/huge .log+.tat fixtures into testdata/ (gitignored; not committed)
```

CI (`.github/workflows/go.yml`) runs `go build -v ./...` then `go test -v -race ./...` on push/PR to `main`; a separate `.github/workflows/release.yml` builds and attaches the `skim` binary to GitHub releases on tag pushes.

## Architecture

Entry point `skim.go` wires together two independent packages and hands off to the Bubble Tea UI loop:

1. **`filterfiles`** — parses TextAnalysisTool `.tat` filter files (XML) into `FilterXML` structs, then compiles each into a `Filter` (adds a compiled `regexp.Regexp` and a `#RRGGBB` background color derived from the XML's `backColor` attribute). `GetMatchingFilter` finds the first enabled filter whose regex matches a given line.
2. **`ui`** — a [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture) program. `ui.go` defines the top-level `model` (implements `Init`/`Update`/`View`), which owns two sub-views:
   - `ui/views/logview` — renders log lines as a `bubbles/table`, coloring each row by its matching filter's `BackColor` and hiding non-matching lines when `hideUnmatched` is set.
   - `ui/views/filterview` — renders the filter list as a `bubbles/table`, showing each filter's enabled checkbox and regex text, colored by its own `BackColor`.

Both sub-views implement a shared `TableView` interface (`Toggle()`, `CursorUp() int`, `CursorDown() int`) defined in `ui/ui.go`, which lets `Update` route key events (arrows/`j`/`k`/`enter`/space) to whichever view currently has focus without type-switching. `tab` cycles focus between `FilterFocus` and `LogFocus`; `h` toggles hiding unmatched log lines; `q`/`ctrl+c` quits.

Log lines and filters are held entirely in memory (the whole log file is scanned into `[]string` up front in `initialModel`), and the `bubbles/table` model is rebuilt from scratch on every `View()` call rather than mutated incrementally.

## Notable gotchas

- `filterview.MakeTable` and `logview.MakeTable` both compute column widths using hardcoded offsets from `windowWidth`/`windowHeight` (see `// TODO` comments) — if you change table styling/borders, these offsets will likely need adjusting too.
- `filterfiles.GetMatchingFilter` returns the *first* enabled filter that matches; filter order in the `.tat` file matters.
- `logview.LogView` caches each line's match/exclusion result (`matchCache`, keyed by `filtersCacheKey`) so repeated renders between keystrokes don't re-run every filter's regex against every line. If you add a new way for a line's rendering to depend on the filter set, make sure it either reads from that cache or is otherwise covered by `filtersCacheKey`'s fingerprint — otherwise a stale cache hit will show outdated results after a filter change. `LogView.MatchCounts` reuses the same cache and must keep matching `filterfiles.CountMatches`'s semantics (it counts a line for its winning highlighting filter regardless of whether the line is also excluded).
- `logview.MakeTable` only builds `table.Row`s for a window of shown lines around the cursor (mirroring `bubbles/table`'s own `UpdateViewport`, which likewise only ever renders `[cursor-height, cursor+height]`) — not every shown line in the whole log. `Table.Rows()` therefore reflects the current window, not the true total; use `LogView.ShownCount` (set by `MakeTable`) for an accurate "N of M lines shown" count instead of `len(Table.Rows())`.

## Potential future optimizations (unmeasured -- verify before relying on these)

After the per-line match cache and the windowed row-building (see gotchas above), the remaining per-render cost at huge (1M-line) scale is dominated by a few O(lines) passes that are cheap per line (cache lookups only, no regex/formatting) but still touch every line on every render, even a pure cursor-movement keystroke that changes nothing about which lines are shown. These are documented as ideas, not benchmarked yet -- follow the same experiment-before-implementing approach used for the cache and windowing changes (see this branch's git history) before shipping any of them, since the actual win at each spot is unverified:

- **Cache `LogView.MatchCounts`'s result, not just the per-line data it reads.** It currently re-tallies all N cached entries into a fresh `[]int` on every call, even when the filter set hasn't changed since the last one. Caching the resulting counts slice under the same invalidation key `ensureMatchCache` already tracks (`matchCacheKey`) would make a repeated call an O(1) lookup instead. Smallest, lowest-risk of the three -- reuses existing cache machinery directly. Watch for: callers must not be able to mutate the cached slice (return a copy, or otherwise make it safe to hand out repeatedly).

- **Precompute a shown-line-rank prefix array so `cursorRow`/`ShownCount` are O(1) instead of O(n).** `MakeTable`'s first pass (finding where the cursor sits among shown/non-excluded lines, and the total shown count) currently walks `v.Lines` from index 0 on every call. A cumulative array (`shownPrefix[i]` = count of shown, non-excluded lines before index `i`), built once and cached, would turn that into array indexing. This is likely the biggest remaining win, but also the most involved: `shown` depends on filters *and* `hideUnmatched` *and* `contextLines` (see `shownLines`), while the existing cache key only covers filters -- the key would need to widen to include all three, or a hide-unmatched/context-lines toggle with an unchanged filter set would silently serve a stale prefix array. The hidden-line-cursor-snapping behavior (see `TestMakeTableCursorRowTracksVisibleLineWhenLinesAreHidden` / `TestMakeTableWindowedCursorSnapsToVisibleLineWhenHidden`) needs to keep working through the prefix-array lookup, not just a linear scan.

- **Precompute one `lipgloss.Style` per filter, and skip `strings.ReplaceAll` when a line has no tab.** `buildRow` currently builds a fresh `Style` per matched row (`style := logStyle; style.Background(...)`) even though there are usually only a handful of distinct filter colors, and unconditionally runs `ReplaceAll(line, "\t", "    ")` even on lines with no tab at all. Precomputing one Style per filter and guarding the replace with `strings.IndexByte(line, '\t') == -1` are both small, contained changes to `buildRow`. Likely the smallest impact of the three (this loop is already bounded to the visible window, not the whole file), but essentially free to try.

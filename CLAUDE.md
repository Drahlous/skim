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

- `filterview.Render` and `logview.MakeTable` both compute column widths using offsets derived from real style values (`filterChromeWidth`/`tableChromeWidth`) rather than bare numbers — if you change table styling/borders, check those helpers still match. `filterview.Render` always emits exactly `filterview.VisibleHeight` data rows (padding blank ones when there are fewer filters, windowing when there are more; see its doc comment) — this constancy is what lets `logview.MakeTable`'s own height budget (`chromeLines`) stay correct regardless of filter count, so don't reintroduce a variable-height `Render` without updating that budget too.
- `filterfiles.GetMatchingFilter` returns the *first* enabled filter that matches; filter order in the `.tat` file matters.
- `logview.LogView` caches each line's match/exclusion result (`matchCache`, keyed by `filtersCacheKey`) so repeated renders between keystrokes don't re-run every filter's regex against every line. If you add a new way for a line's rendering to depend on the filter set, make sure it either reads from that cache or is otherwise covered by `filtersCacheKey`'s fingerprint — otherwise a stale cache hit will show outdated results after a filter change. `LogView.MatchCounts` reuses the same cache and must keep matching `filterfiles.CountMatches`'s semantics (it counts a line for its winning highlighting filter regardless of whether the line is also excluded).
- `logview.MakeTable` only builds `table.Row`s for a window of shown lines around the cursor (mirroring `bubbles/table`'s own `UpdateViewport`, which likewise only ever renders `[cursor-height, cursor+height]`) — not every shown line in the whole log. `Table.Rows()` therefore reflects the current window, not the true total; use `LogView.ShownCount` (set by `MakeTable`) for an accurate "N of M lines shown" count instead of `len(Table.Rows())`.
- **Bubble Tea's `standardRenderer` diffs each new frame against its own cached `lastRender`, and trusts that cache completely — it has no way to notice the real terminal no longer matches it.** That assumption breaks in two known ways in this codebase, and will break again the same way for any future code path with the same shape:
  - *Frame taller than the terminal.* If `View()`'s total line count exceeds `windowHeight`, the renderer has to drop/shift lines it can't scroll back to, permanently desyncing its diff — fixed once for the expandable keybindings help bar (see `249e230`, and `TestViewTotalHeightStaysWithinWindowWhenHelpExpands`) by sizing the log/filter panes around the footer's *actual* rendered line count instead of assuming it's always one line, and fixed again for the filter pane itself (issue #57) by making `filterview.Render` always emit a constant number of rows (`filterview.VisibleHeight`, padded/windowed as needed) instead of a count that grew with the filter list, and having `ui.go`'s `View` subtract that constant from the log pane's height budget the same way it already does for the footer (see `TestViewTotalHeightStaysWithinWindowAsFilterCountGrows`). Any future variable-height chrome (another wrapping status/footer line, a growable panel) needs the same treatment: compute its real line count and subtract it from the pane height budget *before* rendering panes, not after — or, as with the filter pane, make it constant-height in the first place and budget for that constant.
  - *Something outside Bubble Tea's control draws to the terminal.* `openFilterFieldEditorCmd` (in `ui/filtereditor.go`) uses `tea.ExecProcess` to hand the terminal to `$EDITOR`, for editing a filter editor text field (description/regex). Bubble Tea's `ReleaseTerminal`/`RestoreTerminal` cycle around that never actually flips its internal `altScreenActive` flag off and back on, so the `enterAltScreen()` call on return is a no-op (see `bubbletea@v0.24.2/standard_renderer.go`'s `enterAltScreen`/`tea.go`'s `RestoreTerminal`) — it skips the `ClearScreen()` + cache-reset (`repaint()`) it would normally do, leaving the renderer diffing the next frame against a `lastRender` that no longer reflects what `$EDITOR` left on screen. Fixed by having the `filterFieldEditorFinishedMsg` handler in `ui.go`'s `Update` explicitly return `tea.ClearScreen` to force the repaint Bubble Tea itself failed to trigger (see `assertClearsScreen` in `ui_test.go`). Any future use of `tea.ExecProcess`/`ReleaseTerminal` (another external-editor or pager integration) needs the same explicit `tea.ClearScreen` on return — don't assume Bubble Tea will detect the handoff and repaint on its own.
  
  General rule: any change that could make a frame taller than the terminal, or that hands the real terminal to something Bubble Tea doesn't control, must force a full repaint (size the frame to fit, or return `tea.ClearScreen`) rather than relying on Bubble Tea to notice — it won't.

## Potential future optimizations (unmeasured -- verify before relying on these)

After the per-line match cache and the windowed row-building (see gotchas above), the remaining per-render cost at huge (1M-line) scale is dominated by a few O(lines) passes that are cheap per line (cache lookups only, no regex/formatting) but still touch every line on every render, even a pure cursor-movement keystroke that changes nothing about which lines are shown. These are documented as ideas, not benchmarked yet -- follow the same experiment-before-implementing approach used for the cache and windowing changes (see this branch's git history) before shipping any of them, since the actual win at each spot is unverified:

- **Cache `LogView.MatchCounts`'s result, not just the per-line data it reads.** It currently re-tallies all N cached entries into a fresh `[]int` on every call, even when the filter set hasn't changed since the last one. Caching the resulting counts slice under the same invalidation key `ensureMatchCache` already tracks (`matchCacheKey`) would make a repeated call an O(1) lookup instead. Smallest, lowest-risk of the three -- reuses existing cache machinery directly. Watch for: callers must not be able to mutate the cached slice (return a copy, or otherwise make it safe to hand out repeatedly).

- **Precompute a shown-line-rank prefix array so `cursorRow`/`ShownCount` are O(1) instead of O(n).** `MakeTable`'s first pass (finding where the cursor sits among shown/non-excluded lines, and the total shown count) currently walks `v.Lines` from index 0 on every call. A cumulative array (`shownPrefix[i]` = count of shown, non-excluded lines before index `i`), built once and cached, would turn that into array indexing. This is likely the biggest remaining win, but also the most involved: `shown` depends on filters *and* `hideUnmatched` *and* `contextLines` (see `shownLines`), while the existing cache key only covers filters -- the key would need to widen to include all three, or a hide-unmatched/context-lines toggle with an unchanged filter set would silently serve a stale prefix array. The hidden-line-cursor-snapping behavior (see `TestMakeTableCursorRowTracksVisibleLineWhenLinesAreHidden` / `TestMakeTableWindowedCursorSnapsToVisibleLineWhenHidden`) needs to keep working through the prefix-array lookup, not just a linear scan.

- **Precompute one `lipgloss.Style` per filter, and skip `strings.ReplaceAll` when a line has no tab.** `buildRow` currently builds a fresh `Style` per matched row (`style := logStyle; style.Background(...)`) even though there are usually only a handful of distinct filter colors, and unconditionally runs `ReplaceAll(line, "\t", "    ")` even on lines with no tab at all. Precomputing one Style per filter and guarding the replace with `strings.IndexByte(line, '\t') == -1` are both small, contained changes to `buildRow`. Likely the smallest impact of the three (this loop is already bounded to the visible window, not the whole file), but essentially free to try.

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
```

There are currently no automated tests in the repo (no `*_test.go` files). CI (`.github/workflows/go.yml`) only runs `go build -v ./...` on push/PR to `main`; a separate `.github/workflows/release.yml` builds and attaches the `skim` binary to GitHub releases on tag pushes.

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

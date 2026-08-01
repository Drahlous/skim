# Getting started

This page covers what you see when skim opens, how to move around, and the one habit — hiding unmatched lines — that makes filters actually useful. For how to build filters that find something specific, see the [tutorial](./tutorial-triage-a-log.md); for the full `.tat` format, see [filter files](./filter-files.md).

## Launching skim

skim takes two arguments: a log file to read, and a `.tat` filter file describing what to highlight in it.

```sh
skim -log path/to/your.log -filter path/to/your-filters.tat
```

Run with no arguments and skim opens against the bundled example log and filter file, so you can explore the UI without pointing it at anything of your own:

```sh
go run .
```

## The two panes

skim's screen is split into two panes, plus a status line and a help bar at the bottom:

```
┌─────────────────────────────────────────────┐
│  #     Line                                  │   <- Log pane
│  1     Hello World!                          │
│  2     debug: this is a debug message        │
│  ...                                          │
└─────────────────────────────────────────────┘
┌─────────────────────────────────────────────┐
│      Regex                            Aa     │   <- Filters pane
│  [x] ^debug                           [ ]     │
│  [x] goodbye                          [ ]     │
└─────────────────────────────────────────────┘
hide unmatched: ON  |  showing 4/6 lines
q: quit  h: hide unmatched  K: keybindings
```

- **Log pane** (top) — every line of the log file, numbered, colored by whichever filter matched it.
- **Filters pane** (bottom) — one row per filter from your `.tat` file: an enabled checkbox, the regex text (colored with that filter's own background color), and a case-sensitivity checkbox.

The pane with keyboard focus is outlined in a highlighted border (pink by default). Press `tab` to switch focus between them.

Here's that same layout rendered for real, mid-investigation — three filters isolating request errors, warnings, and one specific request's full lifecycle out of a much noisier log (this is the end state of the [tutorial](./tutorial-triage-a-log.md)):

![skim showing three enabled filters highlighting matching lines in a log, with the status line reporting 13 of 44 lines shown](../screenshots/skim.png)

## Moving around

These work in both panes:

| Key | Action |
| --- | --- |
| `up` / `k` | move cursor up |
| `down` / `j` | move cursor down |
| `tab` | switch focus between Log and Filters |
| `q` / `ctrl+c` | quit |

In the **Filters** pane, `left`/`h` and `right`/`l` move the cursor between the enabled checkbox and the case-sensitivity checkbox for the selected filter, and `enter`/`space` toggles whichever one is selected.

This is the full default keymap — every action shown here can be rebound. See [keybindings](./keybindings.md).

## Hiding the noise

This is the core workflow the rest of the docs build on. Press `h` while the **Log** pane is focused to toggle `hide unmatched`:

- **ON** (the default) — only lines that match at least one *enabled* filter are shown.
- **OFF** — every line in the log is shown, with matching lines still colored.

The status line always tells you which mode you're in and how many lines are currently visible out of the total: `hide unmatched: ON | showing 4/6 lines`.

In practice this means: start with `hide unmatched` on and no filters (or all filters disabled) to see nothing, then enable filters one at a time to pull exactly the lines you care about out of the log. You never need to scroll past everything else to find them.

## Editing a filter's regex live

With the **Filters** pane focused, move the cursor to a filter row and press `i` to open that filter's regex text in `$EDITOR` (falls back to `vi` if unset). Save and quit the editor, and skim recompiles the regex and re-renders the log immediately — no restart needed.

This is the fastest way to iterate when you don't yet know the exact pattern you're looking for: tweak the regex, look at what now matches, tweak again.

## Next steps

- Walk through a realistic investigation end-to-end in the [tutorial](./tutorial-triage-a-log.md).
- Learn the full `.tat` filter file format in [filter files](./filter-files.md), including colors and case sensitivity.
- Customize or look up every action in [keybindings](./keybindings.md).

# Keybindings

Every action in skim is rebindable. This page lists the defaults and explains how rebinding and persistence work.

## Default keymap

| Action | Default keys | Scope | Description |
| --- | --- | --- | --- |
| Quit | `ctrl+c`, `q` | global | Quit skim |
| Move cursor up | `up`, `k` | global | Move the cursor up in the focused pane |
| Move cursor down | `down`, `j` | global | Move the cursor down in the focused pane |
| Move column left | `left`, `h` | global | Move the column cursor left (Filters pane only — see below) |
| Move column right | `right`, `l` | global | Move the column cursor right (Filters pane only) |
| Toggle selection | `enter`, `space` | global | Toggle the checkbox under the cursor in the Filters pane |
| Switch focus | `tab` | global | Cycle keyboard focus between the Log and Filters panes |
| Hide unmatched lines | `h` | Log pane only | Toggle whether log lines with no matching enabled filter are shown |
| Edit filter | `i` | Filters pane only | Open the filter editor for the selected filter |
| Edit keybindings | `K` | global | Open the keybindings editor screen |
| Search log | `/` | Log pane only | Start typing an ad-hoc regex search, independent of the `.tat` filters |
| Jump to next match | `n` | Log pane only | Move the cursor to the next line matching the last search |
| Jump to previous match | `N` | Log pane only | Move the cursor to the previous line matching the last search |
| New filter | `a` | Filters pane only | Insert a new, disabled filter after the cursor and open the filter editor for it |
| Delete filter | `d` | Filters pane only | Remove the filter under the cursor |
| Move filter up | `[` | Filters pane only | Swap the filter under the cursor with the one above it |
| Move filter down | `]` | Filters pane only | Swap the filter under the cursor with the one below it |
| Save filters to file | `s` | global | Write the current filter set back to the `.tat` file skim was launched with |
| Show more context around matches | `+` | Log pane only | Increase the number of unmatched lines shown around each match when hide-unmatched is on |
| Show less context around matches | `-` | Log pane only | Decrease the context radius (down to 0) |
| Jump to top | `g` | Log pane only | Move the cursor to the first log line |
| Jump to bottom | `G` | Log pane only | Move the cursor to the last log line |
| Jump to line number | `:` | Log pane only | Start typing a 1-indexed line number; `enter` jumps to it (clamped to the log's bounds), `esc` cancels |

Two actions use `h` for different things depending on which pane has focus: **move column left** in the Filters pane, **hide unmatched lines** in the Log pane. skim resolves this by checking pane-specific bindings before global ones, so both can share the same key without conflict — see "scope" in the table above. If you rebind one, the other is unaffected.

The help bar at the bottom of the screen always reflects your current bindings and only shows the actions relevant to the pane you're focused on.

## Editing a filter

Press `i` (default) on a filter row, or `a` to create a new one, to open the filter editor:

1. `up`/`k` and `down`/`j` move between fields: description, regex, case sensitivity, exclusion, enabled, color.
2. `enter` on **Description** or **Regex** starts typing; `enter` again confirms it (an invalid regex stays in edit mode showing the compile error instead of being discarded), `esc` discards the in-progress edit of just that field. `ctrl+e` on either field — whether you're already typing or just have the cursor on the row — suspends skim and opens the field's current text in `$EDITOR` (falls back to `vi`); save and quit applies the result immediately, the same as pressing `enter` (an invalid regex still drops into edit mode with the error shown, rather than being silently discarded).
3. `enter`/`space` on **Case sensitive**, **Excluding**, or **Enabled** toggles it immediately.
4. `enter` on **Color** opens a swatch grid: `up`/`down`/`left`/`right` (or `hjkl`) move the selection, the mouse can hover and click a swatch directly, `c` switches to typing an exact `#RRGGBB` hex value, and `enter`/click applies the selection. `esc` backs out to the field list without changing the color.
5. `esc` from the field list closes the editor. Each field applies as soon as it's confirmed, so there's no separate "save" for the form itself.

## Rebinding a key

Press `K` (default) from anywhere to open the keybindings editor:

1. `up`/`k` and `down`/`j` move between actions in the list.
2. `enter` starts capturing — the next key you press is bound to the selected action, replacing its previous binding.
3. `esc` while capturing cancels without changing anything; `esc`/`q` from the list closes the editor.

Each action currently holds a single key (rebinding replaces it, rather than adding an alternate). Rebinding takes effect immediately for the rest of the session, and skim best-effort persists it to disk — if the write fails (e.g. no writable config directory), the new binding still applies until you quit, but won't survive a restart.

## Where bindings are stored

skim persists custom bindings as JSON to your OS's standard user config directory (via Go's [`os.UserConfigDir`](https://pkg.go.dev/os#UserConfigDir)):

- Linux: `$XDG_CONFIG_HOME/skim/keybindings.json` (typically `~/.config/skim/keybindings.json`)
- macOS: `~/Library/Application Support/skim/keybindings.json`
- Windows: `%AppData%\skim\keybindings.json`

The file only needs to contain the actions you've overridden — anything absent falls back to its default. Deleting the file (or the actions inside it) restores defaults for the next launch.

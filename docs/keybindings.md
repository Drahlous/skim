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
| Edit regex in `$EDITOR` | `i` | Filters pane only | Open the selected filter's regex text in `$EDITOR` |
| Edit keybindings | `K` | global | Open the keybindings editor screen |
| Search log | `/` | Log pane only | Start typing an ad-hoc regex search, independent of the `.tat` filters |
| Jump to next match | `n` | Log pane only | Move the cursor to the next line matching the last search |
| Jump to previous match | `N` | Log pane only | Move the cursor to the previous line matching the last search |

Two actions use `h` for different things depending on which pane has focus: **move column left** in the Filters pane, **hide unmatched lines** in the Log pane. skim resolves this by checking pane-specific bindings before global ones, so both can share the same key without conflict — see "scope" in the table above. If you rebind one, the other is unaffected.

The help bar at the bottom of the screen always reflects your current bindings and only shows the actions relevant to the pane you're focused on.

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

# Filter files

skim's filters live in a `.tat` file — the same XML format used by [TextAnalysisTool.NET](https://textanalysistool.com/). This page documents the format as skim actually interprets it, and how to build filter files by hand or live in the UI.

## File structure

```xml
<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<TextAnalysisTool.NET version="2023-04-25" showOnlyFilteredLines="False">
  <filters>
    <filter enabled="y" excluding="n" description="" backColor="87cefa" type="matches_text" case_sensitive="n" regex="y" text="^debug" />
    <filter enabled="y" excluding="n" description="" backColor="ff0000" type="matches_text" case_sensitive="n" regex="y" text="goodbye" />
  </filters>
</TextAnalysisTool.NET>
```

Every `<filter>` is a single row in the Filters pane, in file order. **Order matters for highlighting**: for each log line, skim highlights it with the *first enabled, non-excluding filter whose regex matches*, top to bottom. If two highlighting filters could both match a line, whichever appears earlier in the file wins. Exclusion (below) is independent of this order — every enabled excluding filter is checked regardless of position.

## Attributes skim uses

| Attribute | Meaning |
| --- | --- |
| `enabled` | `"y"` or `"n"`. Disabled filters never match, and (with hide-unmatched on) their lines are hidden along with everything else unmatched. Toggle live with `enter`/`space` in the Filters pane. |
| `case_sensitive` | `"y"` or `"n"`. When `"n"` (the default), the regex is compiled with an `(?i)` case-insensitive flag. Toggle live with the case-sensitivity checkbox in the Filters pane. |
| `excluding` | `"y"` or `"n"`. A matching line from an enabled `excluding="y"` filter is **always hidden** — regardless of `hide unmatched`, and regardless of whether the line would otherwise match a highlighting filter. Use it for noise you never want to see (health checks, heartbeats) rather than relying on hide-unmatched, which only hides lines that match *nothing*. In the Filters pane, an excluding filter's regex is prefixed with `! ` so it isn't mistaken for a highlighting filter that just never matches. |
| `backColor` | A 6-digit hex color **without** a leading `#` (e.g. `87cefa`, not `#87cefa`), applied as the background of any log line that matches. Excluded lines are never shown, so `backColor` has no effect on the *log*, but it's still required and still applied to the filter's own row in the Filters pane — where the row text (including the `! ` exclusion marker) renders in black. Avoid a dark `backColor` like `000000` on an excluding filter, or its row becomes unreadable. |
| `text` | The regex pattern to match against each log line. Go's [`regexp` syntax](https://pkg.go.dev/regexp/syntax) (RE2) applies — not .NET regex syntax, even though the file format comes from a .NET tool. Edit live with `i` in the Filters pane. |

## Attributes kept for TAT compatibility, not currently acted on

These are parsed from the file and preserved if you round-trip it, but skim doesn't change behavior based on them today:

- `regex` — in TAT this toggles whether `text` is treated as a literal string or a regex. skim always compiles `text` as a regex.
- `description`, `type` — carried through, not displayed or used anywhere in skim's UI.

## Writing filters

You can hand-write a `.tat` file with any text editor, or build one from inside skim:

1. Start from a file with as many `<filter>` rows as you expect to need — skim doesn't currently support adding or removing filter rows from the UI, only editing the ones already in the file. A row with a placeholder `text` and `enabled="n"` works well as a starting point.
2. Run skim against your log and that filter file.
3. `tab` to the Filters pane, move the cursor to a row, and press `i` to edit its regex in `$EDITOR`. Save and quit to apply it immediately.
4. Press `enter`/`space` on the enabled checkbox to turn the filter on and see it take effect in the Log pane.
5. Repeat, watching `showing X/Y lines` in the status line as a signal for whether a regex is too broad or too narrow.

To reorder filters (which changes which one "wins" on overlapping matches) or add/remove rows entirely, edit the `.tat` file directly and reload skim.

See the [tutorial](./tutorial-triage-a-log.md) for this whole process applied to a real scenario.

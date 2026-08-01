# Tutorial: triage a log

This walkthrough builds a filter set from scratch to answer a concrete question, the way you'd actually use skim during an investigation. It uses two files bundled in this repo:

- `examples/tutorial/checkout-service.log` — 44 lines from a checkout service, with a burst of failures buried in routine traffic and health-check noise.
- `examples/tutorial/tutorial-start.tat` — three empty, disabled filter rows to build on.

There's also `examples/tutorial/tutorial-solution.tat`, a finished filter set you can compare against or skip straight to.

## The scenario

Checkout requests have been failing intermittently. You've been told it's "around 09:15" and that it looks payment-related. Your job: confirm what's failing and why, and pull together the full picture of at least one failed request.

## Step 1 — look at the raw log

Launch skim against the tutorial files:

```sh
go run . -log examples/tutorial/checkout-service.log -filter examples/tutorial/tutorial-start.tat
```

All three filters in `tutorial-start.tat` start disabled, and `hide unmatched` defaults to on, so the Log pane opens empty (`showing 0/44 lines`) — nothing is enabled to match against yet.

With the Log pane focused, press `h` to turn hide-unmatched **off**. Now all 44 lines are visible. Scroll through with `j`/`down`. You can see the shape of the problem — `WARN`/`ERROR` lines around two different requests — but it's mixed in with `DEBUG [heartbeat]` lines every couple of entries and routine `INFO` traffic from unrelated requests. This is the noise skim is built to cut through.

Press `h` again to turn hide-unmatched back **on** before continuing — you'll build the signal back up deliberately, one filter at a time.

## Step 2 — isolate the errors

Press `tab` to move focus to the Filters pane. The cursor starts on the first row (red, `EDIT_ME`).

1. Press `i` to open that filter's regex in `$EDITOR`.
2. Replace `EDIT_ME` with `ERROR`, then save and quit the editor.
3. Press `enter` (or `space`) to enable the filter — the cursor is already on the enabled checkbox column, so this flips `[ ]` to `[x]`.

The Log pane immediately updates to `showing 6/44 lines`: every line containing `ERROR`, colored red. You can see two requests each produced a timeout error with a two-line stack trace.

## Step 3 — bring in the warnings

Move the cursor down (`j`) to the second filter row (gold).

1. `i` → replace `EDIT_ME` with `WARN` → save and quit.
2. `enter`/`space` to enable it.

Now `showing 10/44 lines`: the 6 `ERROR` lines plus 4 `WARN` lines (`payment-gateway response slow ... retrying`), gold before each request's red timeout. Both failing requests show the same pattern — two slow-response warnings, then a timeout.

## Step 4 — pull the full story for one request

Errors and warnings tell you *what* went wrong, but not the surrounding context — when the request came in, what it was for, how it finally resolved. Note the request ID from one of the failures, e.g. `req-1184`.

Move down (`j`) to the third filter row (blue):

1. `i` → replace `EDIT_ME` with `req-1184` → save and quit.
2. `enter`/`space` to enable it.

`showing 13/44 lines` — three new lines appeared: `req-1184`'s `request received`, `cart total calculated`, and `request completed: 502 Bad Gateway` entries. Its `WARN`/`ERROR` lines are still red/gold, not blue, even though they also contain `req-1184`.

That's filter order at work: skim colors each line using the **first enabled filter that matches it**, top to bottom in the file. `ERROR` and `WARN` are listed before your new filter, so they claim those lines first; the `req-1184` filter only gets to color what's left over.

## Step 5 — reorder to change precedence

To make the trace filter "win" for everything belonging to `req-1184` — so you can visually follow one request's entire lifecycle in blue, independent of severity — move it earlier in the file. skim can't reorder filter rows from the UI, so edit `tutorial-start.tat` directly: cut the third `<filter ... text="req-1184" />` line and paste it above the `ERROR` filter, then reload skim (it re-reads the filter file on launch).

The total shown stays `13/44` — the same 13 lines still match *something* — but now all 8 lines mentioning `req-1184` (including its warnings and error) are blue, and the `ERROR`/`WARN` filters only claim the remaining lines from the other failing request, `req-1182`.

Check your result — or skip straight to it — with the bundled solution:

```sh
go run . -log examples/tutorial/checkout-service.log -filter examples/tutorial/tutorial-solution.tat
```

## What this demonstrated

- Building a filter set incrementally, going from an unfiltered wall of text to exactly the lines relevant to the question.
- `hide unmatched` plus enabling filters one at a time as the core way to cut noise without losing anything.
- Editing regexes live with `i` and seeing results immediately, with no restart.
- Filter order determining which filter "claims" (and colors) a line when more than one would match — and that reordering is a `.tat` file edit, not a UI action.

From here, `examples/tutorial/tutorial-solution.tat` is a reasonable starting point to adapt for a real service: swap in your own error signatures and request-ID pattern, and save it alongside the logs you pull it out for next time.

See [filter files](./filter-files.md) for the full `.tat` format reference, and [keybindings](./keybindings.md) for everything used here plus the rest of the keymap.

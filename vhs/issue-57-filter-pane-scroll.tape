# Demonstrates the fix for https://github.com/Drahlous/skim/issues/57 (filter
# pane rendering stale/duplicated rows once the filter count grows past 4).
# Requires the synthetic testdata fixtures this repro used in the issue --
# these are gitignored (see CLAUDE.md), so generate them first:
#   go run ./tools/gentestlogs

Output screenshots/demo-issue-57-fix.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 650
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 50ms

Hide
Type "clear"
Enter
Show

Type "./skim -filter testdata/logs/filters.tat -log testdata/logs/huge.log"
Sleep 500ms
Enter
Sleep 1200ms

# Focus starts on the Log pane; move to the Filters pane, which starts with
# the 4 filters from the issue's repro (^debug/^info/^warn/^error).
Tab
Sleep 800ms

# Add a 5th filter -- this is the exact step that used to overflow the
# frame by one line and desync Bubble Tea's renderer (duplicated/missing
# filter rows on the next keystroke).
Type "a"
Sleep 600ms
Escape
Sleep 900ms

# Move down and add a 6th, same as the issue's repro.
Down
Sleep 400ms
Type "a"
Sleep 600ms
Escape
Sleep 900ms

# Scroll all the way down through the filter list -- the pane keeps a
# constant height and scrolls smoothly, the same way the log pane does,
# instead of duplicating/dropping rows.
Down@100ms 7
Sleep 1200ms

Up@100ms 7
Sleep 1000ms

Tab
Sleep 1000ms
Type "q"

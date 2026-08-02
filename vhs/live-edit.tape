Output screenshots/demo-live-edit.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 550
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 60ms

Hide
Type "clear"
Enter
Show

Type "./skim -log examples/tutorial/checkout-service.log -filter examples/tutorial/tutorial-start.tat"
Sleep 500ms
Enter
Sleep 1200ms

# Cursor starts on the first (red) filter row, which just has a placeholder
# regex. "i" opens the filter editor.
Type "i"
Sleep 800ms

# Move down to the Regex field and edit it in place.
Down
Sleep 400ms
Enter
Sleep 400ms
Backspace 7
Type "ERROR"
Sleep 400ms
Enter
Sleep 400ms
Escape
Sleep 800ms

# Enable the filter to see it take effect
Enter
Sleep 1800ms

# Second filter row: same live-edit workflow for WARN
Down
Sleep 500ms
Type "i"
Sleep 800ms
Down
Sleep 400ms
Enter
Sleep 400ms
Backspace 7
Type "WARN"
Sleep 400ms
Enter
Sleep 400ms
Escape
Sleep 800ms
Enter
Sleep 1800ms

Tab
Sleep 2000ms
Type "q"

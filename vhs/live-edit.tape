Output screenshots/demo-live-edit.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 550
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 60ms

Hide
Type "export EDITOR=vi && clear"
Enter
Show

Type "./skim -log examples/tutorial/checkout-service.log -filter examples/tutorial/tutorial-start.tat"
Sleep 500ms
Enter
Sleep 1200ms

# Cursor starts on the first (red) filter row, which just has a placeholder regex
Type "i"
Sleep 800ms
Type "ciwERROR"
Escape
Sleep 400ms
Type ":wq"
Enter
Sleep 1200ms

# Enable the filter to see it take effect
Enter
Sleep 1800ms

# Second filter row: same live-edit workflow for WARN
Down
Sleep 500ms
Type "i"
Sleep 800ms
Type "ciwWARN"
Escape
Sleep 400ms
Type ":wq"
Enter
Sleep 1200ms
Enter
Sleep 1800ms

Tab
Sleep 2000ms
Type "q"

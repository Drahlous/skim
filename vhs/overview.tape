Output screenshots/demo-overview.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 650
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 50ms

Hide
Type "export EDITOR=vi && clear"
Enter
Show

Type "./skim -log examples/tutorial/checkout-service.log -filter examples/tutorial/tutorial-solution.tat"
Sleep 500ms
Enter
Sleep 1200ms

# Focus starts on the Filters pane; move to the Log pane first
Tab
Sleep 800ms

Screenshot screenshots/skim.png

Down 4
Sleep 1200ms

# Reveal the noise hide-unmatched normally cuts out
Type "h"
Sleep 1800ms

# Hide it again
Type "h"
Sleep 1200ms

# Move to the Filters pane and disable the WARN filter to show live effect
Tab
Sleep 600ms
Down 2
Sleep 600ms
Enter
Sleep 1500ms
Enter
Sleep 1000ms

Tab
Sleep 1200ms
Type "q"

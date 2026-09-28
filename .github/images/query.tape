Output query.gif
Output query.mp4

Set FontSize 32
Set Width 1800
Set Height 1200
Set TypingSpeed 200ms

Hide
Type "fx testdata/users.json"
Enter
Show

Sleep 1.5s

# Open the query input and complete a path.
Type "."
Sleep 1s
Type "u"
Sleep 1s
Tab
Sleep 1.5s
Type "[]."
Sleep 1s
Tab
Sleep 800ms
Tab
Sleep 800ms
Tab
Sleep 1.5s
Enter
Sleep 2s

# Cancel restores the original.
Type "."
Sleep 500ms
Escape
Sleep 1s

# Filter and map with expressions.
Type "."
Sleep 500ms
Type "users ?.act"
Sleep 1s
Tab
Sleep 1.5s
Type " @.n"
Sleep 1s
Tab
Sleep 1.5s
Enter
Sleep 3s

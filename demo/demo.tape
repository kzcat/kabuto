# kabuto demo recording (charmbracelet/vhs)
# For CI/reproducible builds, replace /tmp/kabuto-demo with a freshly built ./kabuto.
# Regenerate with: vhs demo/demo.tape

Output demo/kabuto.gif

Set Width 1200
Set Height 700
Set FontSize 16
Set TypingSpeed 50ms

# Launch kabuto in interactive watch mode (30s refresh)
Type "/tmp/kabuto-demo -w 30"
Enter
Sleep 3s

# Cycle color mode
Type "c"
Sleep 2s

# Toggle section 1
Type "1"
Sleep 2s

# Show help overlay
Type "?"
Sleep 2s

# Quit
Type "q"
Sleep 1s

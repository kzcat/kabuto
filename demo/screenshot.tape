# Regenerate the README screenshot:  vhs demo/screenshot.tape
# Requires: a built ./kabuto binary and a Nerd Font (for the braille charts),
# e.g. `brew install --cask font-jetbrains-mono-nerd-font`.
Output "/tmp/kabuto-shot.gif"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 15
Set Width 1500
Set Height 840
Set Padding 16
Sleep 800ms
Type "COLORTERM=truecolor ./kabuto --lang en -s japan -s us -s commodity"
Enter
Sleep 4500ms
Screenshot "demo/screenshot.png"
Sleep 400ms

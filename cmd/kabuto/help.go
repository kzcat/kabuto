package main

import "strings"

var helpLines = []string{
	"q / Esc    Quit",
	"r          Refetch",
	"c          Color mode (normal/JP/none)",
	"+ / -      Columns +1 / -1",
	"=          Auto columns",
	"1-9        Show section (toggle)",
	"0 / a      All sections",
	"f          Full height toggle",
	"Space      Pause / resume",
	"? / h      This help",
	"",
	"Press any key to close",
}

// overlayHelp renders a centered help box over the frame.
func overlayHelp(frame string, termWidth, termRows int) string {
	// Determine box dimensions
	maxW := 0
	for _, l := range helpLines {
		if len(l) > maxW {
			maxW = len(l)
		}
	}
	boxW := maxW + 4 // 2 border + 2 padding
	boxH := len(helpLines) + 2

	// Build box lines
	box := make([]string, boxH)
	box[0] = "┌" + strings.Repeat("─", boxW-2) + "┐"
	for i, l := range helpLines {
		pad := boxW - 4 - len(l)
		if pad < 0 {
			pad = 0
		}
		box[i+1] = "│ " + l + strings.Repeat(" ", pad) + " │"
	}
	box[boxH-1] = "└" + strings.Repeat("─", boxW-2) + "┘"

	// Overlay onto frame lines
	lines := strings.Split(frame, "\n")
	// Ensure we have at least termRows lines
	for len(lines) < termRows {
		lines = append(lines, "")
	}

	startRow := (termRows - boxH) / 2
	startCol := (termWidth - boxW) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	for i, bline := range box {
		row := startRow + i
		if row >= len(lines) {
			break
		}
		orig := lines[row]
		// Ensure orig is wide enough
		for len(orig) < termWidth {
			orig += " "
		}
		// Splice the box line in
		pre := ""
		if startCol > 0 && startCol <= len(orig) {
			pre = orig[:startCol]
		}
		post := ""
		end := startCol + boxW
		if end < len(orig) {
			post = orig[end:]
		}
		lines[row] = pre + bline + post
	}
	return strings.Join(lines, "\n")
}

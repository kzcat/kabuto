package main

import (
	"strings"

	"github.com/kzcat/kabuto/internal/i18n"
)

func buildHelpLines(lang string) []string {
	return []string{
		"q / Esc    " + i18n.T(lang, "key_quit"),
		"r          " + i18n.T(lang, "key_refetch"),
		"c          " + i18n.T(lang, "key_color"),
		"+ / -      " + i18n.T(lang, "key_columns"),
		"=          " + i18n.T(lang, "key_auto_cols"),
		"1-9        " + i18n.T(lang, "key_section"),
		"0 / a      " + i18n.T(lang, "key_all"),
		"f          " + i18n.T(lang, "key_fullheight"),
		"Space      " + i18n.T(lang, "key_pause"),
		"n / Tab    " + i18n.T(lang, "key_sel_next"),
		"b          " + i18n.T(lang, "key_sel_prev"),
		"Enter      " + i18n.T(lang, "key_detail"),
		"p          " + i18n.T(lang, "key_preset"),
		"Esc        " + i18n.T(lang, "key_back"),
		"? / h      " + i18n.T(lang, "key_help"),
		"",
		i18n.T(lang, "key_close"),
	}
}

// overlayHelp renders a centered help box over the frame.
func overlayHelp(frame string, termWidth, termRows int, lang string) string {
	lines := buildHelpLines(lang)
	// Determine box dimensions
	maxW := 0
	for _, l := range lines {
		w := len(l)
		if w > maxW {
			maxW = w
		}
	}
	boxW := maxW + 4 // 2 border + 2 padding
	boxH := len(lines) + 2

	// Build box lines
	box := make([]string, boxH)
	box[0] = "┌" + strings.Repeat("─", boxW-2) + "┐"
	for i, l := range lines {
		pad := boxW - 4 - len(l)
		if pad < 0 {
			pad = 0
		}
		box[i+1] = "│ " + l + strings.Repeat(" ", pad) + " │"
	}
	box[boxH-1] = "└" + strings.Repeat("─", boxW-2) + "┘"

	// Overlay onto frame lines
	frameLines := strings.Split(frame, "\n")
	// Ensure we have at least termRows lines
	for len(frameLines) < termRows {
		frameLines = append(frameLines, "")
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
		if row >= len(frameLines) {
			break
		}
		orig := frameLines[row]
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
		frameLines[row] = pre + bline + post
	}
	return strings.Join(frameLines, "\n")
}

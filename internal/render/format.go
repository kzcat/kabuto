package render

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/kzcat/kabuto/internal/i18n"
)

// stringWidth returns the display width (fullwidth=2, halfwidth=1).
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isWide checks whether a rune is East Asian Wide/Fullwidth.
func isWide(r rune) bool {
	if r >= 0x1100 && r <= 0x115F {
		return true
	}
	if r >= 0x2E80 && r <= 0x303E {
		return true
	}
	if r >= 0x3040 && r <= 0x33BF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0xA000 && r <= 0xA4CF {
		return true
	}
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	if r >= 0xF900 && r <= 0xFAFF {
		return true
	}
	if r >= 0xFE30 && r <= 0xFE6F {
		return true
	}
	if r >= 0xFF01 && r <= 0xFF60 {
		return true
	}
	if r >= 0xFFE0 && r <= 0xFFE6 {
		return true
	}
	if r >= 0x20000 && r <= 0x2FFFF {
		return true
	}
	if r >= 0x30000 && r <= 0x3FFFF {
		return true
	}
	// halfwidth katakana
	if r >= 0xFF61 && r <= 0xFF9F {
		return false
	}
	_ = unicode.Han
	return false
}

// truncWidth truncates s to fit within the given display width.
func truncWidth(s string, width int) string {
	w := 0
	var b strings.Builder
	for _, r := range s {
		rw := 1
		if isWide(r) {
			rw = 2
		}
		if w+rw > width {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

// padRight pads s with spaces on the right to the given display width.
func padRight(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

// padLeft pads s with spaces on the left to the given display width.
func padLeft(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return strings.Repeat(" ", width-sw) + s
}

func fmtNum(value float64, decimals int) string {
	return fmtNumLang(value, decimals, "en")
}

func fmtNumLang(value float64, decimals int, lang string) string {
	neg := value < 0
	if neg {
		value = -value
	}
	s := fmt.Sprintf("%.*f", decimals, value)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	n := len(intPart)
	grp := i18n.GroupSep(lang)
	dec := i18n.DecimalSep(lang)
	if n > 3 {
		var buf strings.Builder
		rem := n % 3
		if rem > 0 {
			buf.WriteString(intPart[:rem])
			if n > rem {
				buf.WriteString(grp)
			}
		}
		for i := rem; i < n; i += 3 {
			buf.WriteString(intPart[i : i+3])
			if i+3 < n {
				buf.WriteString(grp)
			}
		}
		intPart = buf.String()
	}
	result := intPart
	if len(parts) > 1 {
		result += dec + parts[1]
	}
	if neg {
		result = "-" + result
	}
	return result
}

func fmtChange(value float64, decimals int) string {
	return fmtChangeLang(value, decimals, "en")
}

func fmtChangeLang(value float64, decimals int, lang string) string {
	if value > 0 {
		return "+" + fmtNumLang(value, decimals, lang)
	}
	return fmtNumLang(value, decimals, lang)
}

func fmtPct(value float64) string {
	s := fmt.Sprintf("%.2f%%", math.Abs(value))
	if value > 0 {
		return "+" + s
	} else if value < 0 {
		return "-" + s
	}
	return s
}

// fmtPctPlain formats a percentage without a sign (e.g. "1.28%").
func fmtPctPlain(value float64) string {
	return fmt.Sprintf("%.2f%%", math.Abs(value))
}

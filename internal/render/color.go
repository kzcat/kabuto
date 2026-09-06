package render

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// meterBar renders a btop-style horizontal meter '█████░░░' representing the
// magnitude of pct (clamped to +/-maxPct). The filled portion is colored by
// gain/loss (depth-aware); the empty portion is bright black. width is the total
// cell count. Returns "" when width<=0. useColor=false returns plain runes.
func meterBar(pct float64, width int, th Theme, useColor, redGreen bool, depth int) string {
	if width <= 0 {
		return ""
	}
	const maxPct = 3.0 // +/-3% maps to a full bar
	mag := math.Abs(pct) / maxPct
	if mag > 1 {
		mag = 1
	}
	filled := int(math.Round(mag * float64(width)))
	if filled > width {
		filled = width
	}
	if !useColor {
		return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	}
	base := baseRGBForTheme(pct, redGreen, th)
	var fillClr string
	if depth != 0 {
		fillClr = fgDepth(base, depth)
	} else {
		fillClr = colorForTheme(pct, true, redGreen, th)
	}
	var b strings.Builder
	b.WriteString(fillClr)
	b.WriteString(strings.Repeat("█", filled))
	b.WriteString(th.BrightBlk)
	b.WriteString(strings.Repeat("░", width-filled))
	b.WriteString(th.Reset)
	return b.String()
}

// colorFor returns the ANSI color escape for the given change value (supports rg inversion).
func colorFor(change float64, useColor, redGreen bool) string {
	return colorForTheme(change, useColor, redGreen, defaultTheme)
}

// colorForTheme returns the color escape using the given Theme.
func colorForTheme(change float64, useColor, redGreen bool, th Theme) string {
	if !useColor {
		return ""
	}
	up, down := th.UpColor, th.DownColor
	if redGreen {
		up, down = down, up
	}
	if change > 0 {
		return up
	} else if change < 0 {
		return down
	}
	return ""
}

// arrow returns an indicator symbol based on the change direction.
func arrow(change float64) string {
	if change > 0 {
		return "▲"
	} else if change < 0 {
		return "▼"
	}
	return "・"
}

// truecolorSupported checks if COLORTERM env is truecolor or 24bit.
func truecolorSupported() bool {
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	return ct == "truecolor" || ct == "24bit"
}

// Color depth levels.
const (
	depthTruecolor = 24 // 24-bit RGB
	depth256       = 8  // 256-color (xterm)
	depth16        = 4  // 16-color (basic ANSI)
)

// colorDepth detects the terminal color depth from environment:
//
//	truecolor (COLORTERM=truecolor|24bit) -> 256 (TERM contains "256") -> 16 (otherwise).
func colorDepth() int {
	if truecolorSupported() {
		return depthTruecolor
	}
	if strings.Contains(os.Getenv("TERM"), "256") {
		return depth256
	}
	return depth16
}

// rgbTo256 converts an RGB color to the nearest xterm-256 color index.
// It picks the closer of the 6x6x6 color cube and the 24-step grayscale ramp.
func rgbTo256(r, g, b int) int {
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return v
	}
	r, g, b = clamp(r), clamp(g), clamp(b)

	// 6x6x6 color cube. Cube level thresholds are 0,95,135,175,215,255.
	cubeIdx := func(v int) int {
		if v < 48 {
			return 0
		}
		if v < 115 {
			return 1
		}
		return (v - 35) / 40
	}
	cubeVal := func(i int) int {
		if i == 0 {
			return 0
		}
		return 55 + i*40
	}
	ri, gi, bi := cubeIdx(r), cubeIdx(g), cubeIdx(b)
	cr, cg, cb := cubeVal(ri), cubeVal(gi), cubeVal(bi)
	cubeDist := (cr-r)*(cr-r) + (cg-g)*(cg-g) + (cb-b)*(cb-b)
	cubeColor := 16 + 36*ri + 6*gi + bi

	// grayscale ramp 232..255 (values 8,18,...,238).
	avg := (r + g + b) / 3
	gi2 := (avg - 3) / 10
	if gi2 < 0 {
		gi2 = 0
	}
	if gi2 > 23 {
		gi2 = 23
	}
	gv := 8 + gi2*10
	grayDist := (gv-r)*(gv-r) + (gv-g)*(gv-g) + (gv-b)*(gv-b)
	grayColor := 232 + gi2

	if grayDist < cubeDist {
		return grayColor
	}
	return cubeColor
}

// ansi16Palette is the standard 16-color ANSI palette (approximate RGB).
var ansi16Palette = [16][3]int{
	{0, 0, 0}, {170, 0, 0}, {0, 170, 0}, {170, 85, 0},
	{0, 0, 170}, {170, 0, 170}, {0, 170, 170}, {170, 170, 170},
	{85, 85, 85}, {255, 85, 85}, {85, 255, 85}, {255, 255, 85},
	{85, 85, 255}, {255, 85, 255}, {85, 255, 255}, {255, 255, 255},
}

// rgbTo16 converts an RGB color to the nearest standard 16-color ANSI index (0..15).
func rgbTo16(r, g, b int) int {
	best, bestDist := 0, 1<<30
	for i, c := range ansi16Palette {
		d := (c[0]-r)*(c[0]-r) + (c[1]-g)*(c[1]-g) + (c[2]-b)*(c[2]-b)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}

// fgDepth returns a foreground-color escape for the given RGB at the given color depth.
// truecolor: ESC[38;2;r;g;bm (unchanged from fg24).
// 256:       ESC[38;5;Nm with a 6x6x6/grayscale approximation.
// 16:        ESC[3Nm / ESC[9Nm nearest-ANSI.
func fgDepth(c [3]int, depth int) string {
	switch depth {
	case depth256:
		return fmt.Sprintf("\033[38;5;%dm", rgbTo256(c[0], c[1], c[2]))
	case depth16:
		idx := rgbTo16(c[0], c[1], c[2])
		if idx < 8 {
			return fmt.Sprintf("\033[3%dm", idx)
		}
		return fmt.Sprintf("\033[9%dm", idx-8)
	default: // depthTruecolor
		return fg24(c)
	}
}

// baseRGBFor returns the base RGB for the given change value (supports rg inversion).
func baseRGBFor(change float64, redGreen bool) [3]int {
	return baseRGBForTheme(change, redGreen, defaultTheme)
}

// baseRGBForTheme returns the RGB using the given Theme.
func baseRGBForTheme(change float64, redGreen bool, th Theme) [3]int {
	up, down := th.UpRGB, th.DownRGB
	if redGreen {
		up, down = down, up
	}
	if change < 0 {
		return down
	}
	return up // up/flat uses the up color as base
}

// fg24 returns a truecolor foreground escape sequence.
func fg24(c [3]int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
}

// gradientRGB linearly interpolates from base (top row) to ~50% darker (bottom row) for the given row.
func gradientRGB(base [3]int, row, rows int) [3]int {
	if rows <= 1 {
		return base
	}
	// t=0 (top row) -> 1 (bottom row). Bottom row is ~50% of base.
	t := float64(row) / float64(rows-1)
	factor := 1.0 - 0.5*t
	var out [3]int
	for i := 0; i < 3; i++ {
		out[i] = int(math.Round(float64(base[i]) * factor))
	}
	return out
}

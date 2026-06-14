package render

// Theme holds color escapes and RGB for rendering tiles/charts.
type Theme struct {
	Name      string
	UpColor   string // ANSI escape for positive change
	DownColor string // ANSI escape for negative change
	UpRGB     [3]int // truecolor RGB for positive change gradient
	DownRGB   [3]int // truecolor RGB for negative change gradient
	BoldWhite string
	BrightBlk string
	Reset     string
	Reverse   string
	Bold      string
}

// DefaultTheme returns the default theme (green/red).
var defaultTheme = Theme{
	Name:      "default",
	UpColor:   "\033[32m",
	DownColor: "\033[31m",
	UpRGB:     [3]int{0, 200, 0},
	DownRGB:   [3]int{220, 40, 40},
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var monoTheme = Theme{
	Name:      "mono",
	UpColor:   "",
	DownColor: "",
	UpRGB:     [3]int{180, 180, 180},
	DownRGB:   [3]int{180, 180, 180},
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var lightTheme = Theme{
	Name:      "light",
	UpColor:   "\033[32;1m", // bright green for light bg
	DownColor: "\033[31;1m", // bright red for light bg
	UpRGB:     [3]int{0, 150, 0},
	DownRGB:   [3]int{200, 0, 0},
	BoldWhite: "\033[1;30m", // dark bold for light bg
	BrightBlk: "\033[37m",   // lighter gray for light bg
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var highcontrastTheme = Theme{
	Name:      "highcontrast",
	UpColor:   "\033[34;1m",     // bright blue (color-blind-friendly)
	DownColor: "\033[38;5;208m", // orange (color-blind-friendly)
	UpRGB:     [3]int{50, 100, 255},
	DownRGB:   [3]int{255, 140, 0},
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

// ThemeByName returns a Theme by name. Unknown names fall back to "default".
func ThemeByName(name string) Theme {
	switch name {
	case "mono":
		return monoTheme
	case "light":
		return lightTheme
	case "highcontrast":
		return highcontrastTheme
	default:
		return defaultTheme
	}
}

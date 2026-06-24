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

// Popular terminal palettes. UpColor/DownColor stay as robust 16-color escapes
// for the change text; UpRGB/DownRGB carry each palette's signature colors for
// the truecolor chart gradient (degraded to 256/16 color by the renderer).
var draculaTheme = Theme{
	Name:      "dracula",
	UpColor:   "\033[92m",
	DownColor: "\033[91m",
	UpRGB:     [3]int{80, 250, 123}, // #50fa7b
	DownRGB:   [3]int{255, 85, 85},  // #ff5555
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var nordTheme = Theme{
	Name:      "nord",
	UpColor:   "\033[32m",
	DownColor: "\033[31m",
	UpRGB:     [3]int{163, 190, 140}, // #a3be8c
	DownRGB:   [3]int{191, 97, 106},  // #bf616a
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var gruvboxTheme = Theme{
	Name:      "gruvbox",
	UpColor:   "\033[92m",
	DownColor: "\033[91m",
	UpRGB:     [3]int{184, 187, 38}, // #b8bb26
	DownRGB:   [3]int{251, 73, 52},  // #fb4934
	BoldWhite: "\033[1;37m",
	BrightBlk: "\033[90m",
	Reset:     "\033[0m",
	Reverse:   "\033[7m",
	Bold:      "\033[1m",
}

var solarizedTheme = Theme{
	Name:      "solarized",
	UpColor:   "\033[32m",
	DownColor: "\033[31m",
	UpRGB:     [3]int{133, 153, 0}, // #859900
	DownRGB:   [3]int{220, 50, 47}, // #dc322f
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
	case "dracula":
		return draculaTheme
	case "nord":
		return nordTheme
	case "gruvbox":
		return gruvboxTheme
	case "solarized":
		return solarizedTheme
	default:
		return defaultTheme
	}
}

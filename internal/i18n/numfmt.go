package i18n

// Locale number formatting: thousands separator and decimal point per language.
// For fr, we use ASCII space (U+0020) as the grouping separator to keep width
// math simple and stable (not a non-breaking or thin space).

type numFormat struct {
	Group   string // thousands separator
	Decimal string // decimal point
}

var numFormats = map[string]numFormat{
	"en": {",", "."},
	"ja": {",", "."},
	"zh": {",", "."},
	"ko": {",", "."},
	"es": {",", "."},
	"de": {".", ","},
	"fr": {" ", ","}, // ASCII space U+0020 for grouping (see comment above)
}

// GroupSep returns the thousands grouping separator for lang.
func GroupSep(lang string) string {
	if f, ok := numFormats[normLang(lang)]; ok {
		return f.Group
	}
	return ","
}

// DecimalSep returns the decimal separator for lang.
func DecimalSep(lang string) string {
	if f, ok := numFormats[normLang(lang)]; ok {
		return f.Decimal
	}
	return "."
}

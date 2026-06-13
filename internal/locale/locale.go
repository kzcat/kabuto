// Package locale provides home-market detection and timezone resolution for kabuto.
package locale

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kzcat/kabuto/internal/symbols"
)

// DetectCountry reads $LC_ALL then $LANG and extracts the 2-letter country code
// from an "xx_YY" locale string (e.g. "ja_JP.UTF-8" → "JP"). Defaults to "US".
func DetectCountry() string {
	for _, env := range []string{os.Getenv("LC_ALL"), os.Getenv("LANG")} {
		if cc := countryFromLocale(env); cc != "" {
			return cc
		}
	}
	return "US"
}

// countryFromLocale extracts the YY part from "xx_YY[.codeset]" as an uppercase
// 2-letter code. Returns "" if not present.
func countryFromLocale(s string) string {
	if s == "" {
		return ""
	}
	// strip codeset/modifier: "ja_JP.UTF-8" → "ja_JP"
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	i := strings.IndexByte(s, '_')
	if i < 0 || i+1 >= len(s) {
		return ""
	}
	cc := s[i+1:]
	if len(cc) != 2 {
		return ""
	}
	return strings.ToUpper(cc)
}

// ResolveCountry returns the effective country code: flag override > env > "US".
// The result is always uppercased.
func ResolveCountry(override string) string {
	if override != "" {
		return strings.ToUpper(override)
	}
	return DetectCountry()
}

// ResolveLocation returns the display location. With a non-empty tz it resolves
// the IANA name; on failure it prints an English error to stderr and falls back
// to time.Local. Empty tz → time.Local.
func ResolveLocation(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kabuto: invalid timezone %q: %v (falling back to local time)\n", tz, err)
		return time.Local
	}
	return loc
}

// homeSectionForCountry maps a country code to the section that should be moved
// to the front. Returns "" when there is no mapping.
func homeSectionForCountry(cc string) string {
	switch cc {
	case "JP":
		return "japan"
	case "US":
		return "us"
	case "GB", "DE", "FR", "EU", "CH", "IT", "RU":
		return "europe"
	case "HK", "CN", "TW", "KR", "IN", "SG", "MY", "ID", "TH", "AU", "NZ":
		return "asia"
	case "TR", "IL", "SA", "CA", "MX", "BR":
		return "mideast-america"
	default:
		return ""
	}
}

// HomeFirstOrder returns a copy of symbols.SectionOrder with the country's home
// section moved to the front (preserving the relative order of the rest). When
// the home section is "us", "us-futures" is placed immediately after "us".
func HomeFirstOrder(cc string) []string {
	home := homeSectionForCountry(cc)
	if home == "" {
		// default US
		home = "us"
	}

	// front sections in priority order
	front := []string{home}
	if home == "us" {
		front = append(front, "us-futures")
	}
	frontSet := map[string]bool{}
	for _, s := range front {
		frontSet[s] = true
	}

	out := make([]string, 0, len(symbols.SectionOrder))
	out = append(out, front...)
	for _, s := range symbols.SectionOrder {
		if !frontSet[s] {
			out = append(out, s)
		}
	}
	return out
}

// CryptoItems returns the crypto section items reordered for the given country:
// BTC-JPY first for JP, otherwise BTC-USD first. symbols.Sections is left
// unmodified.
func CryptoItems(cc string) []symbols.Item {
	src := symbols.Sections["crypto"].Items
	items := make([]symbols.Item, len(src))
	copy(items, src)

	want := "BTC-USD"
	if cc == "JP" {
		want = "BTC-JPY"
	}
	for i, it := range items {
		if it.Symbol == want {
			// move to front, preserving the relative order of the rest
			head := items[i]
			rest := append([]symbols.Item{}, items[:i]...)
			rest = append(rest, items[i+1:]...)
			return append([]symbols.Item{head}, rest...)
		}
	}
	return items
}

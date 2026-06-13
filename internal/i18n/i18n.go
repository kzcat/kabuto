// Package i18n provides UI language detection and translation catalogs for kabuto.
//
// Supported languages: en, ja, zh (Simplified), ko, de, fr, es. English (en) is
// the base/fallback. Language (label) is independent of country (--country order).
package i18n

import (
	"os"
	"strings"
)

// Supported lists the bundled UI languages. en is the base/fallback.
var Supported = []string{"en", "ja", "zh", "ko", "de", "fr", "es"}

// isSupported reports whether lang is one of the bundled languages.
func isSupported(lang string) bool {
	for _, l := range Supported {
		if l == lang {
			return true
		}
	}
	return false
}

// LangFromLocale extracts the leading language subtag from a locale-ish string
// and normalizes it to a supported language code, or "en" when unknown/empty.
//
// Examples:
//
//	"ja_JP.UTF-8" -> "ja"
//	"zh_CN"       -> "zh"
//	"zh_Hans"     -> "zh"
//	"zh_Hant"     -> "zh" (treated as Simplified per spec)
//	"de_DE"       -> "de"
//	"JA"          -> "ja"
//	"xx_YY"       -> "en"
//	""            -> "en"
func LangFromLocale(s string) string {
	if s == "" {
		return "en"
	}
	// Strip codeset (".UTF-8") and modifier ("@euro") if present.
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	// Leading language subtag is before the first "_" or "-".
	if i := strings.IndexAny(s, "_-"); i >= 0 {
		s = s[:i]
	}
	lang := strings.ToLower(strings.TrimSpace(s))
	if isSupported(lang) {
		return lang
	}
	return "en"
}

// DetectLang reads $LC_ALL, then $LANG, then $LANGUAGE (first colon-separated
// element) and returns the first supported language found, or "en".
func DetectLang() string {
	if v := os.Getenv("LC_ALL"); v != "" {
		if lang := LangFromLocale(v); lang != "en" || strings.HasPrefix(strings.ToLower(v), "en") {
			return lang
		}
	}
	if v := os.Getenv("LANG"); v != "" {
		if lang := LangFromLocale(v); lang != "en" || strings.HasPrefix(strings.ToLower(v), "en") {
			return lang
		}
	}
	if v := os.Getenv("LANGUAGE"); v != "" {
		// LANGUAGE is a colon-separated priority list; use the first element.
		first := v
		if i := strings.Index(v, ":"); i >= 0 {
			first = v[:i]
		}
		if first != "" {
			if lang := LangFromLocale(first); lang != "en" || strings.HasPrefix(strings.ToLower(first), "en") {
				return lang
			}
		}
	}
	return "en"
}

// ResolveLang returns the effective UI language with priority:
// flag (--lang) > environment (DetectLang) > default "en".
// flagVal is normalized like LangFromLocale ("ja_JP"/"JA" -> "ja"); unknown -> "en".
func ResolveLang(flagVal string) string {
	if strings.TrimSpace(flagVal) != "" {
		return LangFromLocale(flagVal)
	}
	return DetectLang()
}

// normLang maps an empty language to "en" so callers can pass "" for default.
func normLang(lang string) string {
	if lang == "" {
		return "en"
	}
	return lang
}

// T returns the UI string for key in lang, falling back to en, then to key.
func T(lang, key string) string {
	lang = normLang(lang)
	if m, ok := uiCatalog[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if m, ok := uiCatalog["en"]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return key
}

// SectionTitle returns the localized title for a section key, falling back to en.
func SectionTitle(lang, sectionKey string) string {
	lang = normLang(lang)
	if m, ok := sectionCatalog[lang]; ok {
		if s, ok := m[sectionKey]; ok {
			return s
		}
	}
	if m, ok := sectionCatalog["en"]; ok {
		if s, ok := m[sectionKey]; ok {
			return s
		}
	}
	return sectionKey
}

// SymbolName returns the localized name for a Yahoo symbol. Resolution order:
// lang catalog -> en catalog -> fallbackEnglishName (symbols.Item.Name).
func SymbolName(lang, symbol, fallbackEnglishName string) string {
	lang = normLang(lang)
	if m, ok := symbolCatalog[lang]; ok {
		if s, ok := m[symbol]; ok {
			return s
		}
	}
	if m, ok := symbolCatalog["en"]; ok {
		if s, ok := m[symbol]; ok {
			return s
		}
	}
	return fallbackEnglishName
}

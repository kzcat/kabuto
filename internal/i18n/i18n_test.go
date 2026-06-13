package i18n

import (
	"os"
	"testing"

	"github.com/kzcat/kabuto/internal/symbols"
)

// TestSymbolCatalogCoverage は全51銘柄×7言語がカバーされていることを検証する。
func TestSymbolCatalogCoverage(t *testing.T) {
	// Collect all symbols from sections.
	var allSyms []string
	for _, key := range symbols.SectionOrder {
		sec := symbols.Sections[key]
		for _, item := range sec.Items {
			allSyms = append(allSyms, item.Symbol)
		}
	}
	if len(allSyms) != 51 {
		t.Fatalf("expected 51 symbols, got %d", len(allSyms))
	}

	for _, lang := range Supported {
		for _, sym := range allSyms {
			name := SymbolName(lang, sym, "FB")
			if name == "FB" {
				t.Errorf("lang=%s symbol=%s: got fallback 'FB', expected localized name", lang, sym)
			}
		}
	}

	// en must have all symbols explicitly.
	enMap := symbolCatalog["en"]
	for _, sym := range allSyms {
		if _, ok := enMap[sym]; !ok {
			t.Errorf("en catalog missing symbol %s", sym)
		}
	}
}

// TestResolveLangPriority は --lang > env > en の優先順位を検証する。
func TestResolveLangPriority(t *testing.T) {
	// Explicit flag overrides env.
	if got := ResolveLang("ja"); got != "ja" {
		t.Errorf("ResolveLang('ja') = %q, want 'ja'", got)
	}

	// Unknown flag -> en.
	if got := ResolveLang("xx"); got != "en" {
		t.Errorf("ResolveLang('xx') = %q, want 'en'", got)
	}

	// Empty flag -> detect from env.
	os.Setenv("LANG", "de_DE.UTF-8")
	os.Setenv("LC_ALL", "")
	os.Setenv("LANGUAGE", "")
	defer os.Unsetenv("LANG")
	if got := ResolveLang(""); got != "de" {
		t.Errorf("ResolveLang('') with LANG=de_DE.UTF-8 = %q, want 'de'", got)
	}
}

// TestFallbackChain はフォールバック挙動を検証する。
func TestFallbackChain(t *testing.T) {
	// Unknown key -> key itself.
	if got := T("ja", "nope"); got != "nope" {
		t.Errorf("T('ja','nope') = %q, want 'nope'", got)
	}

	// Unknown lang -> en fallback.
	if got := T("xx", "clock"); got != "Clock" {
		t.Errorf("T('xx','clock') = %q, want 'Clock'", got)
	}

	// SectionTitle unknown lang -> en fallback.
	if got := SectionTitle("xx", "japan"); got != "Japan" {
		t.Errorf("SectionTitle('xx','japan') = %q, want 'Japan'", got)
	}

	// SectionTitle unknown key -> key itself.
	if got := SectionTitle("en", "nonexist"); got != "nonexist" {
		t.Errorf("SectionTitle('en','nonexist') = %q, want 'nonexist'", got)
	}

	// SymbolName unknown lang -> en fallback.
	if got := SymbolName("xx", "^N225", "fallback"); got != "Nikkei 225" {
		t.Errorf("SymbolName('xx','^N225','fallback') = %q, want 'Nikkei 225'", got)
	}

	// SymbolName unknown symbol -> fallbackEnglishName.
	if got := SymbolName("en", "NOSYM", "MyFB"); got != "MyFB" {
		t.Errorf("SymbolName('en','NOSYM','MyFB') = %q, want 'MyFB'", got)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kzcat/kabuto/internal/symbols"
)

func TestParseAddSpec(t *testing.T) {
	tests := []struct {
		spec    string
		want    ItemConf
		wantErr bool
	}{
		{"TSLA", ItemConf{Symbol: "TSLA", Decimals: 2}, false},
		{"AAPL:US", ItemConf{Symbol: "AAPL", Country: "US", Decimals: 2}, false},
		{"7203.T:JP:2", ItemConf{Symbol: "7203.T", Country: "JP", Decimals: 2}, false},
		{"BTC-USD:us:4", ItemConf{Symbol: "BTC-USD", Country: "US", Decimals: 4}, false},
		{"", ItemConf{}, true},
		{":US", ItemConf{}, true},
		{"X:US:abc", ItemConf{}, true},
	}
	for _, tt := range tests {
		got, err := ParseAddSpec(tt.spec)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseAddSpec(%q) expected error", tt.spec)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAddSpec(%q) unexpected error: %v", tt.spec, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseAddSpec(%q) = %+v, want %+v", tt.spec, got, tt.want)
		}
	}
}

func TestParse_Valid(t *testing.T) {
	data := []byte(`{
		"lang": "ja",
		"country": "JP",
		"theme": "mono",
		"range": "5d",
		"source": "yahoo",
		"sections": [{"key":"watch","title":"Watchlist","items":[{"name":"Tesla","symbol":"TSLA","country":"US","decimals":2}]}],
		"section_order": ["watch","japan","us"]
	}`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if cfg.Lang != "ja" {
		t.Errorf("Lang = %q, want ja", cfg.Lang)
	}
	if cfg.Country != "JP" {
		t.Errorf("Country = %q, want JP", cfg.Country)
	}
	if cfg.Theme != "mono" {
		t.Errorf("Theme = %q, want mono", cfg.Theme)
	}
	if cfg.Range != "5d" {
		t.Errorf("Range = %q, want 5d", cfg.Range)
	}
	if cfg.Source != "yahoo" {
		t.Errorf("Source = %q, want yahoo", cfg.Source)
	}
	if len(cfg.Sections) != 1 || cfg.Sections[0].Key != "watch" {
		t.Errorf("Sections unexpected: %+v", cfg.Sections)
	}
	if len(cfg.SectionOrder) != 3 || cfg.SectionOrder[0] != "watch" {
		t.Errorf("SectionOrder unexpected: %v", cfg.SectionOrder)
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse([]byte(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoad_NoFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
}

func TestLoad_TempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"lang":"ko","theme":"light"}`), 0644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Lang != "ko" || cfg.Theme != "light" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{bad`), 0644)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON file")
	}
}

func TestToItem(t *testing.T) {
	ic := ItemConf{Symbol: "TSLA", Country: "US", Decimals: 2}
	item := ic.ToItem()
	if item.Name != "TSLA" {
		t.Errorf("Name = %q, want TSLA (fallback from Symbol)", item.Name)
	}
	ic2 := ItemConf{Name: "Tesla", Symbol: "TSLA", Decimals: 2}
	item2 := ic2.ToItem()
	if item2.Name != "Tesla" {
		t.Errorf("Name = %q, want Tesla", item2.Name)
	}
}

func TestRegisterSections(t *testing.T) {
	secs := []SectionConf{
		{Key: "testcustom", Title: "Test Custom", Items: []ItemConf{
			{Name: "Foo", Symbol: "FOO", Country: "US", Decimals: 2},
		}},
	}
	RegisterSections(secs)
	sec, ok := symbols.Sections["testcustom"]
	if !ok {
		t.Fatal("expected testcustom in Sections")
	}
	if len(sec.Items) != 1 || sec.Items[0].Symbol != "FOO" {
		t.Errorf("unexpected items: %+v", sec.Items)
	}
	// Clean up
	delete(symbols.Sections, "testcustom")
}

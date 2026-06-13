package locale

import (
	"testing"
)

func TestDetectCountry(t *testing.T) {
	cases := []struct {
		name   string
		lcAll  string
		lang   string
		setAll bool
		setLng bool
		want   string
	}{
		{name: "ja_JP from LANG", lang: "ja_JP.UTF-8", setLng: true, want: "JP"},
		{name: "en_US from LANG", lang: "en_US", setLng: true, want: "US"},
		{name: "empty defaults US", want: "US"},
		{name: "LC_ALL takes precedence", lcAll: "de_DE.UTF-8", lang: "ja_JP.UTF-8", setAll: true, setLng: true, want: "DE"},
		{name: "C locale ignored", lang: "C", setLng: true, want: "US"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// clear both so unset cases truly default
			t.Setenv("LC_ALL", "")
			t.Setenv("LANG", "")
			if c.setAll {
				t.Setenv("LC_ALL", c.lcAll)
			}
			if c.setLng {
				t.Setenv("LANG", c.lang)
			}
			if got := DetectCountry(); got != c.want {
				t.Errorf("DetectCountry() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolveCountry(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "ja_JP.UTF-8")
	if got := ResolveCountry(""); got != "JP" {
		t.Errorf("env-derived country = %q, want JP", got)
	}
	if got := ResolveCountry("de"); got != "DE" {
		t.Errorf("flag override should win and uppercase: got %q, want DE", got)
	}
}

func TestHomeFirstOrder(t *testing.T) {
	cases := []struct {
		cc        string
		wantFirst string
	}{
		{"JP", "japan"},
		{"US", "us"},
		{"DE", "europe"},
		{"HK", "asia"},
		{"BR", "mideast-america"},
		{"ZZ", "us"}, // unknown → default US
	}
	for _, c := range cases {
		order := HomeFirstOrder(c.cc)
		if order[0] != c.wantFirst {
			t.Errorf("HomeFirstOrder(%q)[0] = %q, want %q", c.cc, order[0], c.wantFirst)
		}
		// us-futures must follow us immediately when us is home
		if c.wantFirst == "us" {
			if order[1] != "us-futures" {
				t.Errorf("us-futures should follow us: %v", order[:3])
			}
		}
		// all sections preserved
		if len(order) != 9 {
			t.Errorf("HomeFirstOrder(%q) len = %d, want 9", c.cc, len(order))
		}
	}
}

func TestCryptoItems(t *testing.T) {
	jp := CryptoItems("JP")
	if jp[0].Symbol != "BTC-JPY" {
		t.Errorf("JP crypto first = %q, want BTC-JPY", jp[0].Symbol)
	}
	us := CryptoItems("US")
	if us[0].Symbol != "BTC-USD" {
		t.Errorf("US crypto first = %q, want BTC-USD", us[0].Symbol)
	}
	if len(jp) != len(us) {
		t.Errorf("crypto item count mismatch: %d vs %d", len(jp), len(us))
	}
}

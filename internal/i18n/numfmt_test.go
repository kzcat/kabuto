package i18n

import "testing"

func TestGroupSep(t *testing.T) {
	cases := []struct {
		lang, want string
	}{
		{"en", ","},
		{"ja", ","},
		{"de", "."},
		{"fr", " "},
		{"zh", ","},
		{"ko", ","},
		{"es", ","},
		{"", ","},
	}
	for _, c := range cases {
		if got := GroupSep(c.lang); got != c.want {
			t.Errorf("GroupSep(%q) = %q, want %q", c.lang, got, c.want)
		}
	}
}

func TestDecimalSep(t *testing.T) {
	cases := []struct {
		lang, want string
	}{
		{"en", "."},
		{"ja", "."},
		{"de", ","},
		{"fr", ","},
		{"zh", "."},
		{"ko", "."},
		{"es", "."},
		{"", "."},
	}
	for _, c := range cases {
		if got := DecimalSep(c.lang); got != c.want {
			t.Errorf("DecimalSep(%q) = %q, want %q", c.lang, got, c.want)
		}
	}
}

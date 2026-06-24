package render

import "testing"

// The extra built-in palettes must resolve by name and carry distinct gradient
// colors (otherwise --theme would be a no-op for them).
func TestThemeByNameExtraBuiltins(t *testing.T) {
	def := ThemeByName("default")
	for _, name := range []string{"dracula", "nord", "gruvbox", "solarized"} {
		th := ThemeByName(name)
		if th.Name != name {
			t.Errorf("ThemeByName(%q).Name = %q, want %q", name, th.Name, name)
		}
		if th.UpRGB == def.UpRGB && th.DownRGB == def.DownRGB {
			t.Errorf("theme %q shares default up/down RGB; should be distinct", name)
		}
		if th.UpRGB == ([3]int{}) || th.DownRGB == ([3]int{}) {
			t.Errorf("theme %q has an unset (zero) gradient color", name)
		}
		if th.Reset == "" {
			t.Errorf("theme %q missing Reset escape", name)
		}
	}
}

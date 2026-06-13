package render

import (
	"strings"
	"testing"
	"time"
)

// TestHeaderTimezoneOffset は Options.Loc に渡した location のオフセットがヘッダーに
// コロン付き(例 +09:00)で併記され、"Updated:" を含むことを検証する(ネットワーク不要)。
func TestHeaderTimezoneOffset(t *testing.T) {
	loc := time.FixedZone("X", 9*3600)
	out := RenderDashboard(nil, []string{"japan"}, Options{NoColor: true, TermWidth: 80, Loc: loc})
	header := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(header, "Updated:") {
		t.Errorf("header should contain 'Updated:': %q", header)
	}
	if !strings.Contains(header, "+09:00") {
		t.Errorf("header should contain offset '+09:00': %q", header)
	}
	if !strings.Contains(header, "kabuto") {
		t.Errorf("header should contain brand 'kabuto': %q", header)
	}
}

// TestHeaderNegativeOffset は負オフセットも -05:00 形式で出ることを検証する。
func TestHeaderNegativeOffset(t *testing.T) {
	loc := time.FixedZone("Y", -5*3600)
	out := RenderDashboard(nil, []string{"us"}, Options{NoColor: true, TermWidth: 80, Loc: loc})
	header := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(header, "-05:00") {
		t.Errorf("header should contain offset '-05:00': %q", header)
	}
}

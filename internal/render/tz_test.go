package render

import (
	"strings"
	"testing"
	"time"
)

// TestHeaderTimezoneOffset verifies that the offset from Options.Loc is shown in the header
// with colons (e.g. +09:00) alongside "Updated:" (no network required).
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

// TestHeaderNegativeOffset verifies that negative offsets are displayed in -05:00 form.
func TestHeaderNegativeOffset(t *testing.T) {
	loc := time.FixedZone("Y", -5*3600)
	out := RenderDashboard(nil, []string{"us"}, Options{NoColor: true, TermWidth: 80, Loc: loc})
	header := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(header, "-05:00") {
		t.Errorf("header should contain offset '-05:00': %q", header)
	}
}

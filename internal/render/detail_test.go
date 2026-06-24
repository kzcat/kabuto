package render

import (
	"strings"
	"testing"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

func TestRenderDashboardSelHighlight(t *testing.T) {
	// Register a minimal section for testing
	symbols.RegisterSection(symbols.Section{
		Key:   "test_sel",
		Title: "Test",
		Items: []symbols.Item{
			{Symbol: "A", Name: "ItemA", Decimals: 2},
			{Symbol: "B", Name: "ItemB", Decimals: 2},
		},
	})
	defer func() { delete(symbols.Sections, "test_sel") }()

	data := map[string]*fetcher.Result{
		"A": {Price: 100, Change: 1.5, ChangePct: 1.5, PrevClose: 98.5, Series: []float64{99, 100, 100.5}},
		"B": {Price: 200, Change: -2, ChangePct: -1.0, PrevClose: 202, Series: []float64{201, 200, 199}},
	}

	// SelIndex=0 should not panic
	opt := Options{
		NoColor:   false,
		TermWidth: 80,
		TermRows:  24,
		Watch:     true,
		SelIndex:  0,
	}
	out := RenderDashboard(data, []string{"test_sel"}, opt)
	if out == "" {
		t.Fatal("expected non-empty output with SelIndex=0")
	}

	// SelIndex=-1: no highlight, should not panic
	opt.SelIndex = -1
	out = RenderDashboard(data, []string{"test_sel"}, opt)
	if out == "" {
		t.Fatal("expected non-empty output with SelIndex=-1")
	}

	// SelIndex out of range (too high): should not panic
	opt.SelIndex = 99
	out = RenderDashboard(data, []string{"test_sel"}, opt)
	if out == "" {
		t.Fatal("expected non-empty output with SelIndex=99")
	}

	// NoColor mode: no crash
	opt.NoColor = true
	opt.SelIndex = 0
	out = RenderDashboard(data, []string{"test_sel"}, opt)
	if out == "" {
		t.Fatal("expected non-empty output with NoColor+SelIndex=0")
	}
}

func TestRenderDashboardDetailView(t *testing.T) {
	symbols.RegisterSection(symbols.Section{
		Key:   "test_det",
		Title: "Test",
		Items: []symbols.Item{
			{Symbol: "X", Name: "ItemX", Decimals: 2},
			{Symbol: "Y", Name: "ItemY", Decimals: 2},
		},
	})
	defer func() { delete(symbols.Sections, "test_det") }()

	data := map[string]*fetcher.Result{
		"X": {Price: 150, Change: 2, ChangePct: 1.3, PrevClose: 148, Currency: "USD", Series: []float64{148, 149, 150, 150.5}},
	}

	// DetailView=true, SelIndex=0
	opt := Options{
		TermWidth:  80,
		TermRows:   24,
		Watch:      true,
		SelIndex:   0,
		DetailView: true,
	}
	out := RenderDashboard(data, []string{"test_det"}, opt)
	if !strings.Contains(out, "ItemX") {
		t.Fatal("detail view should contain item name")
	}

	// DetailView=true, SelIndex=-1: should fall through to grid (no detail)
	opt.SelIndex = -1
	out = RenderDashboard(data, []string{"test_det"}, opt)
	// Should contain both items in grid
	if strings.Contains(out, "Prev:") {
		t.Fatal("should not show detail view when SelIndex=-1")
	}

	// DetailView=true, SelIndex out of range: grid
	opt.SelIndex = 99
	out = RenderDashboard(data, []string{"test_det"}, opt)
	if strings.Contains(out, "Prev:") {
		t.Fatal("should not show detail view when SelIndex out of range")
	}

	// DetailView with nil result (SelIndex=1 has no data): should not panic
	opt.SelIndex = 1
	out = RenderDashboard(data, []string{"test_det"}, opt)
	if !strings.Contains(out, "ItemY") {
		t.Fatal("detail view should show item name even with nil result")
	}

	// NoColor detail
	opt.NoColor = true
	opt.SelIndex = 0
	out = RenderDashboard(data, []string{"test_det"}, opt)
	if !strings.Contains(out, "ItemX") {
		t.Fatal("NoColor detail should contain item name")
	}
}

func TestExtractInner(t *testing.T) {
	// Simple case with Unicode box drawing
	line := "\033[90m│\033[0m hello \033[90m│\033[0m"
	inner := extractInner(line)
	// Inner includes the escape sequences between border chars
	expected := "\033[0m hello \033[90m"
	if inner != expected {
		t.Fatalf("extractInner: want %q, got %q", expected, inner)
	}
	// ASCII fallback
	line2 := "|content|"
	inner2 := extractInner(line2)
	if inner2 != "content" {
		t.Fatalf("extractInner ASCII: want 'content', got %q", inner2)
	}
}

func init() {
	// Ensure fetcher.Result is usable
	_ = fetcher.Result{}
}

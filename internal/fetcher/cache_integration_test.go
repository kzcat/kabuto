package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type stubSource struct {
	name  string
	price float64
	fail  bool
	calls atomic.Int32
}

func (s *stubSource) Name() string { return s.name }
func (s *stubSource) Fetch(ctx context.Context, symbol string, rng Range, client *http.Client) (*Result, error) {
	s.calls.Add(1)
	if s.fail {
		return nil, fmt.Errorf("offline")
	}
	return &Result{Price: s.price}, nil
}
func isolatedCache(t *testing.T) {
	t.Helper()
	old := CacheDirOverride
	CacheDirOverride = t.TempDir()
	t.Cleanup(func() { CacheDirOverride = old })
}

func TestFetchCacheSourcePolicy(t *testing.T) {
	isolatedCache(t)
	yahoo := &stubSource{name: "yahoo", price: 100}
	stooq := &stubSource{name: "stooq", price: 200}
	for _, tc := range []struct {
		sources []Source
		want    float64
		name    string
	}{
		{[]Source{yahoo}, 100, "yahoo"},
		{[]Source{stooq}, 200, "stooq"},
		{[]Source{yahoo, stooq}, 100, "yahoo"},
		{[]Source{stooq, yahoo}, 200, "stooq"},
	} {
		for i := 0; i < 2; i++ {
			r := FetchAll([]string{"TEST"}, Range1D, tc.sources...)["TEST"]
			if r == nil || r.Price != tc.want || r.Source != tc.name || r.Stale || r.FetchedAt.IsZero() {
				t.Fatalf("wrong source result: %+v", r)
			}
		}
	}
	if yahoo.calls.Load() != 2 || stooq.calls.Load() != 2 {
		t.Fatalf("policy caches must be independent and reusable: yahoo=%d stooq=%d", yahoo.calls.Load(), stooq.calls.Load())
	}
	FetchAll([]string{"TEST"}, Range1Y, yahoo)
	if yahoo.calls.Load() != 3 {
		t.Fatal("range must be part of cache key")
	}
}

func TestFetchStaleCacheDoesNotRenewAndRecovers(t *testing.T) {
	isolatedCache(t)
	source := &stubSource{name: "test", fail: true, price: 200}
	stamp := time.Now().Add(-2 * time.Hour).Round(0)
	cacheSet("TEST", Range1D, &Result{Price: 100, Source: "test", FetchedAt: stamp}, source)
	path := filepath.Join(CacheDirOverride, cacheKey("TEST", Range1D, source)+".json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		r := FetchAll([]string{"TEST"}, Range1D, source)["TEST"]
		if r == nil || r.Price != 100 || !r.Stale || !r.FetchedAt.Equal(stamp) {
			t.Fatalf("stale metadata lost: %+v", r)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || source.calls.Load() != 2 {
		t.Fatal("failed fetch renewed cache or suppressed retry")
	}
	source.fail = false
	r := FetchAll([]string{"TEST"}, Range1D, source)["TEST"]
	if r == nil || r.Price != 200 || r.Stale || !r.FetchedAt.After(stamp) {
		t.Fatalf("did not recover: %+v", r)
	}
	cached, fresh := cacheGet("TEST", Range1D, source)
	if !fresh || cached.Stale || cached.Price != 200 {
		t.Fatalf("recovery not cached: %+v fresh=%v", cached, fresh)
	}
}

func TestFetchAllContextCancelsActiveAndQueuedRequests(t *testing.T) {
	started := make(chan struct{}, maxWorkers*2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	BaseURLOverride = []string{server.URL + "/first/%s", server.URL + "/fallback/%s"}
	defer func() { BaseURLOverride = nil }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	syms := make([]string, maxWorkers+5)
	for i := range syms {
		syms[i] = fmt.Sprint(i)
	}
	go func() { FetchAllContext(ctx, syms, Range1D, &YahooSource{}); close(done) }()
	for i := 0; i < maxWorkers; i++ {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("requests did not start")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation did not stop active/queued work")
	}
	select {
	case <-started:
		t.Fatal("started additional work after cancellation")
	default:
	}
}

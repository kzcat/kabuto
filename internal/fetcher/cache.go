package fetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const cacheTTL = 60 * time.Second

// CacheDirOverride allows tests to set a custom cache directory.
var CacheDirOverride string

type cacheEntry struct {
	Result    *Result   `json:"result"`
	Timestamp time.Time `json:"timestamp"`
}

func cacheDir() string {
	if CacheDirOverride != "" {
		return CacheDirOverride
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "kabuto")
}

func cacheKey(symbol string, rng Range, sources ...Source) string {
	if len(sources) == 0 {
		sources = []Source{&YahooSource{}}
	}
	names := make([]string, len(sources))
	for i, src := range sources {
		names[i] = src.Name()
	}
	// Version the namespace to exclude legacy entries with incorrect semantics.
	key, _ := json.Marshal([]any{"v2", symbol, rng.String(), names})
	h := sha256.Sum256(key)
	return hex.EncodeToString(h[:8])
}

func cacheGet(symbol string, rng Range, sources ...Source) (*Result, bool) {
	dir := cacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, cacheKey(symbol, rng, sources...)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	if entry.Result == nil {
		return nil, false
	}
	age := time.Since(entry.Timestamp)
	fresh := age >= 0 && age < cacheTTL
	entry.Result.FetchedAt = entry.Timestamp
	entry.Result.Stale = !fresh
	return entry.Result, fresh
}

func cacheSet(symbol string, rng Range, r *Result, sources ...Source) {
	dir := cacheDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	stamp := r.FetchedAt
	if stamp.IsZero() {
		stamp = time.Now()
	}
	entry := cacheEntry{Result: r, Timestamp: stamp}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(dir, cacheKey(symbol, rng, sources...)+".json")
	file, err := os.CreateTemp(dir, ".cache-*")
	if err != nil {
		return
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		_ = os.Rename(file.Name(), path)
	}
}

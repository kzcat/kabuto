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

func cacheKey(symbol string, rng Range) string {
	h := sha256.Sum256([]byte(symbol + "|" + rng.String()))
	return hex.EncodeToString(h[:8])
}

func cacheGet(symbol string, rng Range) (*Result, bool) {
	dir := cacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, cacheKey(symbol, rng)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	fresh := time.Since(entry.Timestamp) < cacheTTL
	return entry.Result, fresh
}

func cacheSet(symbol string, rng Range, r *Result) {
	dir := cacheDir()
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	entry := cacheEntry{Result: r, Timestamp: time.Now()}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(dir, cacheKey(symbol, rng)+".json")
	_ = os.WriteFile(path, data, 0o644)
}

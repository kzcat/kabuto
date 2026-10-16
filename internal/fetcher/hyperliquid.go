package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// hlPrefix marks HyperLiquid perpetual symbols: "hl:<dex>:<coin>", e.g.
// "hl:xyz:JP225" for the JP225 perpetual on the xyz HIP-3 dex.
const hlPrefix = "hl:"

// HyperliquidURLOverride is a test hook for the HyperLiquid info endpoint.
var HyperliquidURLOverride string

// hlCurrency maps HyperLiquid coins to the currency their price is quoted in
// (the underlying index's points), when known.
var hlCurrency = map[string]string{
	"xyz:JP225": "JPY",
}

// HyperliquidSource fetches HyperLiquid perpetuals, which trade around the
// clock. The quote is the oracle price, which tracks the underlying index; the
// history comes from trade candles.
type HyperliquidSource struct{}

func (h *HyperliquidSource) Name() string { return "hyperliquid" }

func (h *HyperliquidSource) Fetch(ctx context.Context, symbol string, rng Range, client *http.Client) (*Result, error) {
	coin, ok := strings.CutPrefix(symbol, hlPrefix)
	dex, _, hasDex := strings.Cut(coin, ":")
	if !ok || !hasDex || dex == "" {
		return nil, fmt.Errorf("hyperliquid: unsupported symbol %s", symbol)
	}

	var meta struct {
		Universe []struct {
			Name string `json:"name"`
		} `json:"universe"`
	}
	var ctxs []struct {
		OraclePx  string `json:"oraclePx"`
		PrevDayPx string `json:"prevDayPx"`
	}
	var raw []json.RawMessage
	if err := hlPost(ctx, client, map[string]any{"type": "metaAndAssetCtxs", "dex": dex}, &raw); err != nil {
		return nil, err
	}
	if len(raw) != 2 || json.Unmarshal(raw[0], &meta) != nil || json.Unmarshal(raw[1], &ctxs) != nil {
		return nil, fmt.Errorf("hyperliquid: unexpected metaAndAssetCtxs response for %s", symbol)
	}
	idx := -1
	for i, u := range meta.Universe {
		if u.Name == coin {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(ctxs) {
		return nil, fmt.Errorf("hyperliquid: %s not listed", coin)
	}
	price, err := strconv.ParseFloat(ctxs[idx].OraclePx, 64)
	if err != nil || math.IsNaN(price) || math.IsInf(price, 0) || price <= 0 {
		return nil, fmt.Errorf("hyperliquid: invalid oracle price %q for %s", ctxs[idx].OraclePx, coin)
	}

	now := time.Now()
	interval, span := hlCandleWindow(rng)
	var candles []struct {
		Open  string `json:"o"`
		Close string `json:"c"`
	}
	req := map[string]any{"type": "candleSnapshot", "req": map[string]any{
		"coin": coin, "interval": interval,
		"startTime": now.Add(-span).UnixMilli(), "endTime": now.UnixMilli(),
	}}
	if err := hlPost(ctx, client, req, &candles); err != nil {
		return nil, err
	}
	series := make([]float64, 0, len(candles))
	for _, c := range candles {
		if v, err := strconv.ParseFloat(c.Close, 64); err == nil {
			series = append(series, v)
		}
	}

	// A perpetual has no session close: 1d compares with the price 24h ago,
	// longer ranges with the open of the window (like Yahoo's range start).
	var prevClose float64
	if rng == Range1D {
		prevClose, _ = strconv.ParseFloat(ctxs[idx].PrevDayPx, 64)
	} else if len(candles) > 0 {
		prevClose, _ = strconv.ParseFloat(candles[0].Open, 64)
	}
	r := &Result{
		Price:     price,
		PrevClose: prevClose,
		Time:      now.In(time.Local).Format("15:04"),
		Epoch:     now.Unix(),
		Series:    series,
		Currency:  hlCurrency[coin],
		Source:    h.Name(),
	}
	if prevClose > 0 {
		r.Change = price - prevClose
		r.ChangePct = r.Change / prevClose * 100
	} else {
		r.ChangeUnavailable = true
	}
	return r, nil
}

// hlCandleWindow returns the candle interval and look-back span for rng.
func hlCandleWindow(rng Range) (string, time.Duration) {
	const day = 24 * time.Hour
	switch rng {
	case Range5D:
		return "15m", 5 * day
	case Range1Mo:
		return "1d", 30 * day
	case Range6Mo:
		return "1d", 182 * day
	case Range1Y:
		return "1w", 365 * day
	default:
		return "5m", day
	}
}

// hlPost sends an info request and decodes the JSON response into out.
func hlPost(ctx context.Context, client *http.Client, body any, out any) error {
	u := "https://api.hyperliquid.xyz/info"
	if HyperliquidURLOverride != "" {
		u = HyperliquidURLOverride
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("hyperliquid: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("hyperliquid: decode error: %w", err)
	}
	return nil
}

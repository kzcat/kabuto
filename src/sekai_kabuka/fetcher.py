"""Yahoo Finance からデータ取得"""

import json
import urllib.parse
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone, timedelta

JST = timezone(timedelta(hours=9))
# ブラウザ完全偽装 UA や curl UA は Yahoo 側で 429 になる。素の Mozilla/5.0 のみ通る
_UA = "Mozilla/5.0"
# query1 はネットワークによってレート制限(429)されることがあるため query2 を優先
_URLS = (
    "https://query2.finance.yahoo.com/v8/finance/chart/{}?interval=1d&range=2d",
    "https://query1.finance.yahoo.com/v8/finance/chart/{}?interval=1d&range=2d",
)
_TIMEOUT = 10
_MAX_WORKERS = 8


def _fetch_one(symbol: str) -> dict | None:
    """1銘柄取得。query2→query1 の順に試行。失敗時None。"""
    quoted = urllib.parse.quote(symbol, safe="")
    for url_tpl in _URLS:
        req = urllib.request.Request(url_tpl.format(quoted), headers={"User-Agent": _UA})
        try:
            with urllib.request.urlopen(req, timeout=_TIMEOUT) as resp:
                data = json.loads(resp.read())
            meta = data["chart"]["result"][0]["meta"]
            price = meta["regularMarketPrice"]
            prev_close = meta["chartPreviousClose"]
            market_time = meta["regularMarketTime"]
            dt = datetime.fromtimestamp(market_time, tz=JST)
            return {
                "price": price,
                "prev_close": prev_close,
                "change": price - prev_close,
                "change_pct": (price - prev_close) / prev_close * 100 if prev_close else 0,
                "time": dt.strftime("%H:%M"),
            }
        except Exception:
            continue
    return None


def fetch_all(symbols: list[str]) -> dict[str, dict | None]:
    """複数銘柄を並列取得。{symbol: result or None}"""
    results = {}
    with ThreadPoolExecutor(max_workers=_MAX_WORKERS) as pool:
        futures = {pool.submit(_fetch_one, s): s for s in symbols}
        for f in futures:
            results[futures[f]] = f.result()
    return results

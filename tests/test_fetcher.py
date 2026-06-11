"""fetcher のパーステスト(ネットワーク不要)"""

import json
import unittest
from unittest.mock import patch, MagicMock
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from sekai_kabuka.fetcher import _fetch_one


FIXTURE = json.dumps({
    "chart": {
        "result": [{
            "meta": {
                "regularMarketPrice": 39500.50,
                "chartPreviousClose": 39000.00,
                "regularMarketTime": 1718100000,
            }
        }]
    }
}).encode()

FIXTURE_ERROR = json.dumps({"chart": {"result": None, "error": {"code": "Not Found"}}}).encode()


class TestFetchOne(unittest.TestCase):
    @patch("sekai_kabuka.fetcher.urllib.request.urlopen")
    def test_parse_success(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = FIXTURE
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = _fetch_one("^N225")
        self.assertIsNotNone(result)
        self.assertAlmostEqual(result["price"], 39500.50)
        self.assertAlmostEqual(result["prev_close"], 39000.00)
        self.assertAlmostEqual(result["change"], 500.50)
        self.assertAlmostEqual(result["change_pct"], 500.50 / 39000.00 * 100)
        self.assertIn(":", result["time"])

    @patch("sekai_kabuka.fetcher.urllib.request.urlopen")
    def test_parse_failure(self, mock_urlopen):
        mock_urlopen.side_effect = Exception("timeout")
        result = _fetch_one("^N225")
        self.assertIsNone(result)

    @patch("sekai_kabuka.fetcher.urllib.request.urlopen")
    def test_invalid_json(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = FIXTURE_ERROR
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = _fetch_one("INVALID")
        self.assertIsNone(result)


if __name__ == "__main__":
    unittest.main()

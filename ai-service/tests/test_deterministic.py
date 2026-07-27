import asyncio
import unittest

from app.providers.deterministic import DeterministicProvider
from app.schemas import ReviewRequest


class DeterministicProviderTest(unittest.TestCase):
    def test_confirms_high_quality_candidate(self) -> None:
        provider = DeterministicProvider()
        request = ReviewRequest(
            symbol="BTC_USDT",
            features={
                "ruleScore": 84,
                "dataQualityScore": 92,
                "spoofScore": 18,
                "reasons": ["MULTI_TIMEFRAME_BULLISH"],
                "riskFlags": [],
            },
        )
        response = asyncio.run(provider.review(request))
        self.assertEqual(response.decision, "CONFIRM")
        self.assertEqual(response.risk_level, "MEDIUM")

    def test_rejects_low_quality_candidate(self) -> None:
        provider = DeterministicProvider()
        request = ReviewRequest(
            symbol="ALT_USDT",
            features={
                "ruleScore": 82,
                "dataQualityScore": 40,
                "spoofScore": 75,
                "reasons": [],
                "riskFlags": ["ORDER_BOOK_NOT_SYNCED"],
            },
        )
        response = asyncio.run(provider.review(request))
        self.assertEqual(response.decision, "REJECT")


if __name__ == "__main__":
    unittest.main()

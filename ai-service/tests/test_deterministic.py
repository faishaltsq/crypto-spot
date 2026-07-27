import asyncio
import time
import unittest

from pydantic import ValidationError

from app.config import Settings
from app.providers.deterministic import DeterministicProvider
from app.providers.common import parse_chat_content
from app.schemas import ReviewRequest, ReviewResponse
from app.service import ReviewService


def request(**overrides: object) -> ReviewRequest:
    features = {
        "pair": "BTC_USDT",
        "timeframe": "15m",
        "market_regime": "bullish",
        "final_score": 84,
        "dynamic_threshold": 70,
        "rule_score": 84,
        "model_probability": 0.7,
        "data_quality": 92,
        "volume_features": {"relative_volume_1m": 1.8},
        "order_flow_features": {"volume_delta_ratio_1m": 0.2},
        "liquidity_features": {"liquidity_score": 80},
        "spoof_risk": 18,
        "supporting_evidence": ["VOLUME_EXPANSION"],
        "contradicting_evidence": [],
        "risk_flags": [],
    }
    features.update(overrides)
    return ReviewRequest(feature_version="feature-v2", features=features)


class DeterministicProviderTest(unittest.TestCase):
    def test_confirms_high_quality_candidate(self) -> None:
        response = asyncio.run(DeterministicProvider().review(request()))
        self.assertEqual(response.decision, "CONFIRM")
        self.assertEqual(response.provider, "deterministic")

    def test_rejects_low_quality_candidate(self) -> None:
        response = asyncio.run(DeterministicProvider().review(request(data_quality=40, spoof_risk=75)))
        self.assertEqual(response.decision, "REJECT")

    def test_rejects_raw_or_secret_fields(self) -> None:
        with self.assertRaises(ValidationError):
            request(api_secret="must-not-cross-boundary")

    def test_validates_unknown_decision(self) -> None:
        with self.assertRaises(ValidationError):
            ReviewResponse.model_validate({
                "decision": "PROMOTE", "confidence": 0.5, "summary": "bad",
                "supporting_reason_codes": [], "contradicting_reason_codes": [], "risk_flags": [],
                "provider": "test", "model": "test", "latency_ms": 0, "fallback": False,
            })

    def test_invalid_json_is_rejected(self) -> None:
        with self.assertRaises(ValueError):
            parse_chat_content({"choices": [{"message": {"content": "not-json"}}]})

    def test_missing_required_field_is_rejected(self) -> None:
        with self.assertRaises(ValidationError):
            ReviewResponse.model_validate({
                "decision": "WAIT", "confidence": 0.5,
                "supporting_reason_codes": [], "contradicting_reason_codes": [], "risk_flags": [],
                "provider": "test", "model": "test", "latency_ms": 0, "fallback": False,
            })


class ReviewServiceTest(unittest.TestCase):
    def test_disabled_uses_deterministic_without_key(self) -> None:
        service = ReviewService(Settings(ai_enabled=False, ai_provider="deepseek"))
        response = asyncio.run(service.review(request()))
        self.assertEqual(response.provider, "deterministic")
        self.assertFalse(response.fallback)

    def test_deepseek_missing_key_falls_back_without_retry(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", ai_model="deepseek-chat"))
        response = asyncio.run(service.review(request()))
        self.assertTrue(response.fallback)
        self.assertEqual(response.provider_error_code, "AI_PROVIDER_MISCONFIGURED")

    def test_grok_missing_key_falls_back_without_retry(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="grok", ai_model="grok-2"))
        response = asyncio.run(service.review(request()))
        self.assertTrue(response.fallback)
        self.assertEqual(response.provider_error_code, "AI_PROVIDER_MISCONFIGURED")

    def test_cache_hit_and_duplicate_request(self) -> None:
        service = ReviewService(Settings(ai_enabled=False, ai_cache_ttl_seconds=300))
        first = asyncio.run(service.review(request()))
        second = asyncio.run(service.review(request()))
        self.assertIs(first, second)
        self.assertEqual(len(service.cache), 1)

    def test_cache_miss_when_feature_changes(self) -> None:
        service = ReviewService(Settings(ai_enabled=False, ai_cache_ttl_seconds=300))
        asyncio.run(service.review(request()))
        asyncio.run(service.review(request(rule_score=85, final_score=85)))
        self.assertEqual(len(service.cache), 2)

    def test_circuit_opens_and_half_open_probe_closes(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", deepseek_api_key="x", ai_model="test", ai_circuit_breaker_failures=1, ai_circuit_breaker_reset_seconds=1))
        async def fails(_: ReviewRequest) -> object:
            raise TimeoutError("secret=x")
        service.provider.review = fails  # type: ignore[method-assign]
        failed = asyncio.run(service.review(request()))
        self.assertEqual(failed.provider_error_code, "AI_PROVIDER_ERROR")
        self.assertEqual(service.circuit.state.value, "OPEN")
        time.sleep(1.01)
        async def valid(_: ReviewRequest) -> object:
            return {"decision": "WAIT", "confidence": 0.5, "summary": "review", "supporting_reason_codes": [], "contradicting_reason_codes": [], "risk_flags": []}
        service.provider.review = valid  # type: ignore[method-assign]
        recovered = asyncio.run(service.review(request(rule_score=83, final_score=83)))
        self.assertFalse(recovered.fallback)
        self.assertEqual(service.circuit.state.value, "CLOSED")

    def test_retry_then_fallback(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", deepseek_api_key="x", ai_model="test", ai_max_retries=1))
        calls = 0
        async def fails(_: ReviewRequest) -> object:
            nonlocal calls
            calls += 1
            raise TimeoutError("network failure")
        service.provider.review = fails  # type: ignore[method-assign]
        response = asyncio.run(service.review(request()))
        self.assertEqual(calls, 2)
        self.assertTrue(response.fallback)

    def test_invalid_provider_response_falls_back(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", deepseek_api_key="x", ai_model="test"))
        async def invalid(_: ReviewRequest) -> object:
            return {"decision": "PROMOTE"}
        service.provider.review = invalid  # type: ignore[method-assign]
        response = asyncio.run(service.review(request()))
        self.assertTrue(response.fallback)
        self.assertEqual(response.provider_error_code, "AI_PROVIDER_ERROR")

    def test_provider_error_is_redacted_to_code(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", deepseek_api_key="secret-value", ai_model="test"))
        async def fails(_: ReviewRequest) -> object:
            raise TimeoutError("secret-value")
        service.provider.review = fails  # type: ignore[method-assign]
        response = asyncio.run(service.review(request()))
        self.assertNotIn("secret-value", response.summary)
        self.assertEqual(response.provider_error_code, "AI_PROVIDER_ERROR")

    def test_fallback_preserves_review_audit_versions(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="grok", ai_model="test"))
        response = asyncio.run(service.review(request()))
        self.assertEqual(response.prompt_version, "ai-review-v1")
        self.assertEqual(response.schema_version, "ai-review-schema-v1")

    def test_disabled_provider_mode_uses_deterministic(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="disabled"))
        response = asyncio.run(service.review(request()))
        self.assertEqual(response.provider, "deterministic")

    def test_circuit_open_skips_provider(self) -> None:
        service = ReviewService(Settings(ai_enabled=True, ai_provider="deepseek", deepseek_api_key="x", ai_model="test", ai_circuit_breaker_failures=1))
        service.circuit.failure()
        calls = 0
        async def unexpected(_: ReviewRequest) -> object:
            nonlocal calls
            calls += 1
            return {}
        service.provider.review = unexpected  # type: ignore[method-assign]
        response = asyncio.run(service.review(request()))
        self.assertEqual(calls, 0)
        self.assertEqual(response.provider_error_code, "AI_CIRCUIT_OPEN")


if __name__ == "__main__":
    unittest.main()

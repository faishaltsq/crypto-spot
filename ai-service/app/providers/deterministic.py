from app.providers.base import Provider
from app.schemas import ReviewRequest, ReviewResponse


class DeterministicProvider(Provider):
    def __init__(self, provider_name: str = "deterministic") -> None:
        self.provider_name = provider_name

    async def review(self, request: ReviewRequest) -> ReviewResponse:
        features = request.features
        decision = "WAIT"
        if features.data_quality < 60 or features.spoof_risk >= 70:
            decision = "REJECT"
        elif features.data_quality >= 70 and features.rule_score >= 80 and features.spoof_risk < 50:
            decision = "CONFIRM"
        return ReviewResponse(
            decision=decision,
            confidence=max(0.0, min(features.rule_score / 100, 1.0)),
            summary="Deterministic review of supplied rule and data-quality features.",
            supporting_reason_codes=features.supporting_evidence,
            contradicting_reason_codes=features.contradicting_evidence,
            risk_flags=features.risk_flags,
            provider=self.provider_name,
            model="rule-review-v1",
            latency_ms=0,
            fallback=False,
        )

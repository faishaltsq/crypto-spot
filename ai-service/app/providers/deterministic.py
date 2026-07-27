from app.providers.base import Provider
from app.schemas import ReviewRequest, ReviewResponse


class DeterministicProvider(Provider):
    def __init__(self, provider_name: str = "deterministic") -> None:
        self.provider_name = provider_name

    async def review(self, request: ReviewRequest) -> ReviewResponse:
        features = request.features
        score = float(features.get("ruleScore", 0) or 0)
        quality = float(features.get("dataQualityScore", 0) or 0)
        spoof = float(features.get("spoofScore", 100) or 100)
        reasons = list(features.get("reasons", []) or [])
        flags = list(features.get("riskFlags", []) or [])

        decision = "WAIT"
        risk = "HIGH"
        if quality >= 70 and score >= 80 and spoof < 50:
            decision = "CONFIRM"
            risk = "MEDIUM"
        elif quality < 60 or spoof >= 70:
            decision = "REJECT"
            risk = "HIGH"
        elif score >= 70:
            risk = "MEDIUM"

        return ReviewResponse(
            decision=decision,
            confidence=max(0.0, min(score / 100, 1.0)),
            riskLevel=risk,
            reasonCodes=reasons[:8],
            riskFlags=flags[:8],
            summary=(
                f"Rule review for {request.symbol}. "
                f"Score {score:.1f}, data quality {quality:.1f}, "
                f"and spoof heuristic {spoof:.1f}."
            ),
            provider=self.provider_name,
            model="rule-review-v1",
        )

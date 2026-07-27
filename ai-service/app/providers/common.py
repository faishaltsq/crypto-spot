import json
from typing import Any

from app.schemas import ReviewRequest


SYSTEM_PROMPT = """
You review a precomputed SPOT crypto market candidate.
You do not invent prices, news, positions, or indicators.
The rule engine is primary. You may only CONFIRM, REJECT, or WAIT.
Reject incomplete data and high spoofing risk.
Treat this as paper analysis, not guaranteed financial advice.
Return JSON only with:
decision, confidence, riskLevel, reasonCodes, riskFlags, summary.
""".strip()


def user_payload(request: ReviewRequest) -> str:
    compact: dict[str, Any] = {
        "symbol": request.symbol,
        "features": request.features,
    }
    return json.dumps(compact, separators=(",", ":"), ensure_ascii=False)


def parse_chat_content(payload: dict[str, Any]) -> dict[str, Any]:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        raise ValueError("provider returned no choices")
    message = choices[0].get("message", {})
    content = message.get("content")
    if not isinstance(content, str) or not content.strip():
        raise ValueError("provider returned empty content")
    return json.loads(content)

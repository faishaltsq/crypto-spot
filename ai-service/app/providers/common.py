import json
from typing import Any

from app.schemas import ReviewRequest

SYSTEM_PROMPT = """
You review a precomputed SPOT crypto market candidate.
Rule engine is authoritative. You are review-only: never create orders, prices, thresholds, or signals.
Do not remove risk flags or blocked reasons. Use only supplied feature summary.
Return JSON only with decision, confidence, summary, supporting_reason_codes,
contradicting_reason_codes, risk_flags. Decision is CONFIRM, REJECT, WAIT, or UNAVAILABLE.
""".strip()


def user_payload(request: ReviewRequest) -> str:
    return json.dumps(request.model_dump(), separators=(",", ":"), ensure_ascii=True)


def parse_chat_content(payload: dict[str, Any]) -> dict[str, Any]:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        raise ValueError("PROVIDER_EMPTY_RESPONSE")
    content = choices[0].get("message", {}).get("content")
    if not isinstance(content, str) or not content.strip():
        raise ValueError("PROVIDER_EMPTY_RESPONSE")
    return json.loads(content)

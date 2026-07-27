from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

PROMPT_VERSION = "ai-review-v1"
SCHEMA_VERSION = "ai-review-schema-v1"


class ReviewFeatures(BaseModel):
    model_config = ConfigDict(extra="forbid")

    pair: str = Field(min_length=1, max_length=32)
    timeframe: str = Field(min_length=1, max_length=16)
    market_regime: str = Field(min_length=1, max_length=32)
    final_score: float = Field(ge=0, le=100)
    dynamic_threshold: float = Field(ge=0, le=100)
    rule_score: float = Field(ge=0, le=100)
    model_probability: float = Field(ge=0, le=1)
    data_quality: float = Field(ge=0, le=100)
    volume_features: dict[str, float]
    order_flow_features: dict[str, float]
    liquidity_features: dict[str, float]
    spoof_risk: float = Field(ge=0, le=100)
    supporting_evidence: list[str] = Field(max_length=8)
    contradicting_evidence: list[str] = Field(max_length=8)
    risk_flags: list[str] = Field(max_length=8)


class ReviewRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    feature_version: str = Field(min_length=1, max_length=64)
    prompt_version: str = PROMPT_VERSION
    features: ReviewFeatures


class ReviewResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    decision: Literal["CONFIRM", "REJECT", "WAIT", "UNAVAILABLE"]
    confidence: float = Field(ge=0, le=1)
    summary: str = Field(min_length=1, max_length=600)
    supporting_reason_codes: list[str] = Field(max_length=8)
    contradicting_reason_codes: list[str] = Field(max_length=8)
    risk_flags: list[str] = Field(max_length=8)
    provider: str = Field(min_length=1, max_length=32)
    model: str = Field(min_length=1, max_length=128)
    latency_ms: int = Field(ge=0)
    fallback: bool
    fallback_reason: str | None = Field(default=None, max_length=64)
    provider_error_code: str | None = Field(default=None, max_length=64)
    prompt_version: str = PROMPT_VERSION
    schema_version: str = SCHEMA_VERSION


REVIEW_JSON_SCHEMA = {
    "name": "spot_signal_review",
    "strict": True,
    "schema": {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "decision": {"type": "string", "enum": ["CONFIRM", "REJECT", "WAIT", "UNAVAILABLE"]},
            "confidence": {"type": "number", "minimum": 0, "maximum": 1},
            "summary": {"type": "string", "maxLength": 600},
            "supporting_reason_codes": {"type": "array", "items": {"type": "string"}, "maxItems": 8},
            "contradicting_reason_codes": {"type": "array", "items": {"type": "string"}, "maxItems": 8},
            "risk_flags": {"type": "array", "items": {"type": "string"}, "maxItems": 8},
        },
        "required": ["decision", "confidence", "summary", "supporting_reason_codes", "contradicting_reason_codes", "risk_flags"],
    },
}

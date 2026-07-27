from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field


class ReviewRequest(BaseModel):
    symbol: str
    features: dict[str, Any]


class ReviewResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    decision: Literal["CONFIRM", "REJECT", "WAIT"]
    confidence: float = Field(ge=0, le=1)
    risk_level: Literal["LOW", "MEDIUM", "HIGH"] = Field(alias="riskLevel")
    reason_codes: list[str] = Field(default_factory=list, alias="reasonCodes")
    risk_flags: list[str] = Field(default_factory=list, alias="riskFlags")
    summary: str
    provider: str
    model: str


REVIEW_JSON_SCHEMA: dict[str, Any] = {
    "name": "spot_signal_review",
    "strict": True,
    "schema": {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "decision": {
                "type": "string",
                "enum": ["CONFIRM", "REJECT", "WAIT"],
            },
            "confidence": {
                "type": "number",
                "minimum": 0,
                "maximum": 1,
            },
            "riskLevel": {
                "type": "string",
                "enum": ["LOW", "MEDIUM", "HIGH"],
            },
            "reasonCodes": {
                "type": "array",
                "items": {"type": "string"},
                "maxItems": 8,
            },
            "riskFlags": {
                "type": "array",
                "items": {"type": "string"},
                "maxItems": 8,
            },
            "summary": {
                "type": "string",
                "maxLength": 600,
            },
        },
        "required": [
            "decision",
            "confidence",
            "riskLevel",
            "reasonCodes",
            "riskFlags",
            "summary",
        ],
    },
}

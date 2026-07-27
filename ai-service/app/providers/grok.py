import httpx

from app.providers.base import Provider
from app.providers.common import SYSTEM_PROMPT, parse_chat_content, user_payload
from app.schemas import REVIEW_JSON_SCHEMA, ReviewRequest, ReviewResponse


class GrokProvider(Provider):
    def __init__(self, api_key: str, model: str, timeout: int) -> None:
        if not api_key:
            raise ValueError("AI_PROVIDER_MISCONFIGURED")
        if not model:
            raise ValueError("AI_MODEL is required for Grok")
        self.api_key = api_key
        self.model = model
        self.timeout = timeout

    async def review(self, request: ReviewRequest) -> ReviewResponse:
        body = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_payload(request)},
            ],
            "response_format": {
                "type": "json_schema",
                "json_schema": REVIEW_JSON_SCHEMA,
            },
            "temperature": 0,
            "max_tokens": 700,
        }
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            response = await client.post(
                "https://api.x.ai/v1/chat/completions",
                json=body,
                headers=headers,
            )
            response.raise_for_status()
            parsed = parse_chat_content(response.json())

        return parsed

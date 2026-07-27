import logging

from app.config import Settings
from app.providers.base import Provider
from app.providers.deepseek import DeepSeekProvider
from app.providers.deterministic import DeterministicProvider
from app.providers.grok import GrokProvider
from app.schemas import ReviewRequest, ReviewResponse

logger = logging.getLogger(__name__)


class ReviewService:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings
        self.provider = self._build_provider(settings)
        self.fallback = DeterministicProvider("fallback")

    @staticmethod
    def _build_provider(settings: Settings) -> Provider:
        provider = settings.ai_provider.strip().lower()
        if provider == "deepseek":
            return DeepSeekProvider(
                settings.ai_api_key,
                settings.ai_model,
                settings.ai_timeout_seconds,
            )
        if provider == "grok":
            return GrokProvider(
                settings.ai_api_key,
                settings.ai_model,
                settings.ai_timeout_seconds,
            )
        return DeterministicProvider("deterministic")

    async def review(self, request: ReviewRequest) -> ReviewResponse:
        try:
            return await self.provider.review(request)
        except Exception as exc:
            logger.exception("AI provider failed: %s", exc)
            return await self.fallback.review(request)

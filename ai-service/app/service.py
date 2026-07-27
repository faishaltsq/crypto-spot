import asyncio
import hashlib
import json
import time
from dataclasses import dataclass
from enum import Enum
from typing import Any

import httpx
from pydantic import ValidationError

from app.config import Settings
from app.providers.base import Provider
from app.providers.deepseek import DeepSeekProvider
from app.providers.deterministic import DeterministicProvider
from app.providers.grok import GrokProvider
from app.schemas import PROMPT_VERSION, SCHEMA_VERSION, ReviewRequest, ReviewResponse


class CircuitState(str, Enum):
    CLOSED = "CLOSED"
    OPEN = "OPEN"
    HALF_OPEN = "HALF_OPEN"


@dataclass
class CircuitBreaker:
    threshold: int
    reset_seconds: int
    failures: int = 0
    opened_at: float | None = None
    half_open_in_flight: bool = False

    def allow_request(self) -> bool:
        if self.opened_at is None:
            return True
        if time.monotonic() - self.opened_at < self.reset_seconds:
            return False
        if self.half_open_in_flight:
            return False
        self.half_open_in_flight = True
        return True

    def success(self) -> None:
        self.failures = 0
        self.opened_at = None
        self.half_open_in_flight = False

    def failure(self) -> None:
        self.failures += 1
        self.half_open_in_flight = False
        if self.failures >= self.threshold or self.opened_at is not None:
            self.opened_at = time.monotonic()

    @property
    def state(self) -> CircuitState:
        if self.opened_at is None:
            return CircuitState.CLOSED
        if time.monotonic() - self.opened_at >= self.reset_seconds:
            return CircuitState.HALF_OPEN
        return CircuitState.OPEN


class ReviewService:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings
        self.provider, self.provider_error = self._build_provider(settings)
        self.fallback = DeterministicProvider()
        self.cache: dict[str, tuple[float, ReviewResponse]] = {}
        self.circuit = CircuitBreaker(settings.ai_circuit_breaker_failures, settings.ai_circuit_breaker_reset_seconds)

    @staticmethod
    def _build_provider(settings: Settings) -> tuple[Provider, str | None]:
        if not settings.ai_enabled or settings.ai_provider in {"disabled", "deterministic"}:
            return DeterministicProvider(), None
        try:
            if settings.ai_provider == "deepseek":
                return DeepSeekProvider(settings.deepseek_api_key, settings.ai_model, settings.ai_timeout_seconds), None
            if settings.ai_provider == "grok":
                return GrokProvider(settings.grok_api_key, settings.ai_model, settings.ai_timeout_seconds), None
        except ValueError:
            return DeterministicProvider(), "AI_PROVIDER_MISCONFIGURED"
        return DeterministicProvider(), "AI_PROVIDER_MISCONFIGURED"

    def _cache_key(self, request: ReviewRequest) -> str:
        payload = {
            "pair": request.features.pair,
            "timeframe": request.features.timeframe,
            "feature_version": request.feature_version,
            "feature_snapshot": request.features.model_dump(),
            "prompt_version": request.prompt_version,
            "provider": self.settings.ai_provider,
            "model": self.settings.ai_model,
        }
        encoded = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()
        return hashlib.sha256(encoded).hexdigest()

    async def review(self, request: ReviewRequest) -> ReviewResponse:
        cache_key = self._cache_key(request)
        cached = self.cache.get(cache_key)
        if cached and cached[0] > time.monotonic():
            return cached[1]

        if self.provider_error:
            return await self._fallback(request, self.provider_error)
        if not self.settings.ai_enabled or self.settings.ai_provider in {"disabled", "deterministic"}:
            response = await self.fallback.review(request)
            self._store(cache_key, response)
            return response
        if not self.circuit.allow_request():
            return await self._fallback(request, "AI_CIRCUIT_OPEN")

        started_at = time.monotonic()
        for attempt in range(self.settings.ai_max_retries + 1):
            try:
                raw = await self.provider.review(request)
                response = ReviewResponse.model_validate({
                    **raw,
                    "provider": self.settings.ai_provider,
                    "model": self.settings.ai_model,
                    "latency_ms": int((time.monotonic() - started_at) * 1000),
                    "fallback": False,
                    "prompt_version": PROMPT_VERSION,
                    "schema_version": SCHEMA_VERSION,
                })
                self.circuit.success()
                self._store(cache_key, response)
                return response
            except (TimeoutError, httpx.TimeoutException, httpx.HTTPError, ValidationError, ValueError, json.JSONDecodeError):
                if attempt == self.settings.ai_max_retries:
                    self.circuit.failure()
                    return await self._fallback(request, "AI_PROVIDER_ERROR")
        return await self._fallback(request, "AI_PROVIDER_ERROR")

    def _store(self, cache_key: str, response: ReviewResponse) -> None:
        if self.settings.ai_cache_ttl_seconds > 0:
            self.cache[cache_key] = (time.monotonic() + self.settings.ai_cache_ttl_seconds, response)

    async def _fallback(self, request: ReviewRequest, reason: str) -> ReviewResponse:
        fallback = await self.fallback.review(request)
        return fallback.model_copy(update={
            "fallback": True,
            "fallback_reason": reason,
            "provider_error_code": reason,
        })

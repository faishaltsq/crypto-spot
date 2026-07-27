from functools import lru_cache

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    ai_enabled: bool = False
    ai_provider: str = "deterministic"
    deepseek_api_key: str = ""
    grok_api_key: str = ""
    ai_model: str = ""
    ai_timeout_seconds: int = Field(default=10, ge=1, le=60)
    ai_max_retries: int = Field(default=1, ge=0, le=3)
    ai_cache_ttl_seconds: int = Field(default=300, ge=0, le=3600)
    ai_circuit_breaker_failures: int = Field(default=5, ge=1, le=100)
    ai_circuit_breaker_reset_seconds: int = Field(default=60, ge=1, le=3600)

    model_config = SettingsConfigDict(env_file=".env", extra="ignore", case_sensitive=False)

    @field_validator("ai_provider")
    @classmethod
    def validate_provider(cls, provider: str) -> str:
        normalized = provider.strip().lower()
        if normalized not in {"deterministic", "deepseek", "grok", "disabled"}:
            return "disabled"
        return normalized


@lru_cache
def get_settings() -> Settings:
    return Settings()

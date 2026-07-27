from fastapi import FastAPI

from app.config import get_settings
from app.schemas import ReviewRequest, ReviewResponse
from app.service import ReviewService

settings = get_settings()
service = ReviewService(settings)

app = FastAPI(
    title="Crypto Spot Signal AI Review",
    version="1.0.0",
)


@app.get("/health")
async def health() -> dict[str, str]:
    return {
        "status": "ok",
        "provider": settings.ai_provider,
        "mode": "review-only",
    }


@app.post(
    "/review",
    response_model=ReviewResponse,
    response_model_by_alias=True,
)
async def review(request: ReviewRequest) -> ReviewResponse:
    return await service.review(request)

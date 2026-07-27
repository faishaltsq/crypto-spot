from abc import ABC, abstractmethod

from app.schemas import ReviewRequest, ReviewResponse


class Provider(ABC):
    @abstractmethod
    async def review(self, request: ReviewRequest) -> ReviewResponse:
        raise NotImplementedError

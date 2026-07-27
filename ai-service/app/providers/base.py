from abc import ABC, abstractmethod

from typing import Any

from app.schemas import ReviewRequest


class Provider(ABC):
    @abstractmethod
    async def review(self, request: ReviewRequest) -> Any:
        raise NotImplementedError

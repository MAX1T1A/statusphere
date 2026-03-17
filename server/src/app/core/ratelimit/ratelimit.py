import time
from collections import defaultdict, deque
from functools import wraps

from fastapi import HTTPException, Request


class RateLimiter:
    def __init__(self) -> None:
        self._hits: dict[str, deque[float]] = defaultdict(deque)

    def _clean(self, key: str, window: float) -> None:
        q = self._hits[key]
        cutoff = time.monotonic() - window
        while q and q[0] < cutoff:
            q.popleft()

    def check(self, key: str, limit: int, window: float) -> bool:
        self._clean(key, window)
        if len(self._hits[key]) >= limit:
            return False
        self._hits[key].append(time.monotonic())
        return True


limiter = RateLimiter()


def limit(max_calls: int, window: float = 60.0):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            request: Request | None = kwargs.get("request")
            if request is None:
                for a in args:
                    if isinstance(a, Request):
                        request = a
                        break

            ip = request.client.host if request and request.client else "unknown"
            key = f"{func.__name__}:{ip}"

            if not limiter.check(key, max_calls, window):
                raise HTTPException(429, "rate limit exceeded")

            return await func(*args, **kwargs)

        return wrapper

    return decorator

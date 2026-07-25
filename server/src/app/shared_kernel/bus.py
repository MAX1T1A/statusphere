from typing import Any, Awaitable, Callable, Protocol

from app.shared_kernel.operation import Operation

Handler = Callable[[Operation], Awaitable[Any]]
Middleware = Callable[[Operation, Handler], Awaitable[Any]]


class UseCase(Protocol):
    async def execute(self, operation: Any) -> Any: ...


class UseCaseBus:
    def __init__(self) -> None:
        self._handlers: dict[type[Operation], UseCase] = {}
        self._middlewares: list[Middleware] = []

    def use(self, middleware: Middleware) -> None:
        self._middlewares.append(middleware)

    def register(self, operation_type: type[Operation], use_case: UseCase) -> None:
        self._handlers[operation_type] = use_case

    async def dispatch(self, operation: Operation) -> Any:
        use_case = self._handlers[type(operation)]

        async def call(op: Operation) -> Any:
            return await use_case.execute(op)

        handler: Handler = call
        for middleware in reversed(self._middlewares):
            handler = _bind(middleware, handler)
        return await handler(operation)


def _bind(middleware: Middleware, nxt: Handler) -> Handler:
    async def wrapped(operation: Operation) -> Any:
        return await middleware(operation, nxt)

    return wrapped

from typing import Callable, Optional

from app.modules.chats.application.dto import MessageDTO
from app.modules.chats.application.interfaces import IChatsUnitOfWork, IMessageDelivery
from app.modules.chats.domain.message import normalize_text
from app.shared_kernel.operation import AuthenticatedOperation


class SendMessage(AuthenticatedOperation):
    room_token: str
    to_account: str
    text: str
    from_name: str


class SendMessageUseCase:
    def __init__(self, uow_factory: Callable[[], IChatsUnitOfWork], delivery: IMessageDelivery) -> None:
        self._uow_factory = uow_factory
        self._delivery = delivery

    async def execute(self, op: SendMessage) -> Optional[MessageDTO]:
        text = normalize_text(op.text)
        if not text:
            return None

        async with self._uow_factory() as uow:
            created = await uow.messages.add(op.room_token, op.actor.account_id, op.to_account, text)

        at = created.isoformat()
        await self._delivery.deliver(op.room_token, op.actor.account_id, op.from_name, op.to_account, text, at)
        return MessageDTO(from_account=op.actor.account_id, to_account=op.to_account, text=text, at=at)

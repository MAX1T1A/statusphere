import asyncio


async def deliver_message(
    self, token: str, from_account: str, from_name: str, to_account: str, text: str, at: str
) -> None:
    room = self._storage.get_or_create(token)
    envelope = {
        "type": "msg",
        "from": from_account,
        "from_name": from_name,
        "to": to_account,
        "text": text,
        "at": at,
    }
    for subscriber in room.subscribers:
        if to_account and subscriber.account_id not in (to_account, from_account):
            continue
        try:
            subscriber.queue.put_nowait(envelope)
        except asyncio.QueueFull:
            pass

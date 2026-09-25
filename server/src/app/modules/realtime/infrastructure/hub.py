import asyncio
import time
from dataclasses import dataclass, field
from typing import AsyncGenerator

from app.modules.chats.public import IMessageDelivery
from app.modules.photo.public import IPhotoBroadcast
from app.modules.presence.application.interfaces import IPresenceBroadcast

# A device that connects mid-room has missed every publish that came before it, so it
# reads everyone else as offline until their next tick (up to their own --interval).
# Replaying the last snapshot closes that gap; past this age it's not "just missed a
# tick" anymore; it's a device that stopped publishing, so it stays out of the replay.
REPLAY_MAX_AGE = 60.0


@dataclass
class _Subscriber:
    device_id: str
    account_id: str
    queue: asyncio.Queue


@dataclass
class _Room:
    token: str
    subscribers: list[_Subscriber] = field(default_factory=list)
    last_snapshot: dict[str, tuple[float, dict]] = field(default_factory=dict)


class RealtimeHub(IMessageDelivery, IPresenceBroadcast, IPhotoBroadcast):
    def __init__(self) -> None:
        self._rooms: dict[str, _Room] = {}

    def _get_or_create(self, token: str) -> _Room:
        if token not in self._rooms:
            self._rooms[token] = _Room(token=token)
        return self._rooms[token]

    def _remove_subscriber(self, token: str, device_id: str) -> None:
        room = self._rooms.get(token)
        if room:
            room.subscribers = [s for s in room.subscribers if s.device_id != device_id]

    async def subscribe(self, token: str, device_id: str, account_id: str) -> AsyncGenerator[dict, None]:
        queue: asyncio.Queue = asyncio.Queue(maxsize=128)
        room = self._get_or_create(token)
        now = time.monotonic()
        for other_device, (seen_at, snapshot) in room.last_snapshot.items():
            if other_device != device_id and now - seen_at <= REPLAY_MAX_AGE:
                try:
                    queue.put_nowait(snapshot)
                except asyncio.QueueFull:
                    pass
        room.subscribers.append(_Subscriber(device_id=device_id, account_id=account_id, queue=queue))
        try:
            while True:
                data = await queue.get()
                yield data
        finally:
            self._remove_subscriber(token, device_id)

    async def publish(
        self, room_token: str, account_id: str, account_name: str, device_id: str, data: dict
    ) -> None:
        room = self._get_or_create(room_token)
        stamped = {**data, "account_id": account_id, "account_name": account_name, "device_id": device_id}
        room.last_snapshot[device_id] = (time.monotonic(), stamped)
        for subscriber in room.subscribers:
            if subscriber.device_id != device_id:
                try:
                    subscriber.queue.put_nowait(stamped)
                except asyncio.QueueFull:
                    pass

    async def publish_photo(
        self, room_token: str, account_id: str, photo_id: str, created_at: str, expires_at: str
    ) -> None:
        room = self._get_or_create(room_token)
        envelope = {
            "type": "photo_status",
            "account_id": account_id,
            "photo_id": photo_id,
            "created_at": created_at,
            "expires_at": expires_at,
        }
        for subscriber in room.subscribers:
            try:
                subscriber.queue.put_nowait(envelope)
            except asyncio.QueueFull:
                pass

    async def deliver(
        self, room_token: str, from_account: str, from_name: str, to_account: str, text: str, at: str
    ) -> None:
        room = self._get_or_create(room_token)
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

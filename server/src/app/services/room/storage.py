import asyncio
from dataclasses import dataclass, field


@dataclass
class Subscriber:
    device_id: str
    queue: asyncio.Queue


@dataclass
class Room:
    token: str
    subscribers: list[Subscriber] = field(default_factory=list)


class RoomStorage:
    def __init__(self) -> None:
        self._rooms: dict[str, Room] = {}

    def get_or_create(self, token: str) -> Room:
        if token not in self._rooms:
            self._rooms[token] = Room(token=token)
        return self._rooms[token]

    def remove_subscriber(self, token: str, device_id: str) -> None:
        room = self._rooms.get(token)
        if room:
            room.subscribers = [s for s in room.subscribers if s.device_id != device_id]

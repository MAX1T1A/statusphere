import asyncio

from app.services.room import RoomManager
from app.services.room.storage import Subscriber


async def test_publish_overrides_device_id_and_skips_sender():
    rm = RoomManager()
    room = rm._storage.get_or_create("room1")

    q_other = asyncio.Queue()
    q_sender = asyncio.Queue()
    room.subscribers.append(Subscriber(device_id="other", queue=q_other))
    room.subscribers.append(Subscriber(device_id="sender", queue=q_sender))

    await rm.publish("room1", "sender", {"device_id": "SPOOF", "x": 1})

    assert q_sender.empty(), "sender must not receive its own frame"
    msg = q_other.get_nowait()
    assert msg["device_id"] == "sender"
    assert msg["x"] == 1

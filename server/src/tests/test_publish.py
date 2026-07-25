import asyncio

from app.services.room import RoomManager
from app.services.room.storage import Subscriber


async def test_publish_stamps_identity_and_skips_sender():
    rm = RoomManager()
    room = rm._storage.get_or_create("room1")

    q_other = asyncio.Queue()
    q_sender = asyncio.Queue()
    room.subscribers.append(Subscriber(device_id="other", account_id="acc-2", queue=q_other))
    room.subscribers.append(Subscriber(device_id="sender", account_id="acc-1", queue=q_sender))

    await rm.publish("room1", "acc-1", "Max", "sender", {"device_id": "SPOOF", "account_id": "SPOOF", "x": 1})

    assert q_sender.empty(), "sender must not receive its own frame"
    msg = q_other.get_nowait()
    assert msg["device_id"] == "sender"
    assert msg["account_id"] == "acc-1"
    assert msg["account_name"] == "Max"
    assert msg["x"] == 1


async def test_group_message_goes_to_everyone_incl_sender():
    rm = RoomManager()
    room = rm._storage.get_or_create("r")
    qa, qb, qc = asyncio.Queue(), asyncio.Queue(), asyncio.Queue()
    room.subscribers.append(Subscriber(device_id="da", account_id="A", queue=qa))
    room.subscribers.append(Subscriber(device_id="db", account_id="B", queue=qb))
    room.subscribers.append(Subscriber(device_id="dc", account_id="C", queue=qc))

    await rm.deliver_message("r", "A", "Alice", "", "hi all", "2026-01-01T00:00:00")

    for q in (qa, qb, qc):
        m = q.get_nowait()
        assert m["type"] == "msg" and m["to"] == "" and m["text"] == "hi all" and m["from"] == "A"


async def test_dm_goes_only_to_recipient_and_sender():
    rm = RoomManager()
    room = rm._storage.get_or_create("r")
    qa, qb, qc = asyncio.Queue(), asyncio.Queue(), asyncio.Queue()
    room.subscribers.append(Subscriber(device_id="da", account_id="A", queue=qa))
    room.subscribers.append(Subscriber(device_id="db", account_id="B", queue=qb))
    room.subscribers.append(Subscriber(device_id="dc", account_id="C", queue=qc))

    await rm.deliver_message("r", "A", "Alice", "B", "secret", "2026-01-01T00:00:00")

    assert qa.get_nowait()["text"] == "secret", "sender's devices see their own DM"
    assert qb.get_nowait()["to"] == "B", "recipient receives the DM"
    assert qc.empty(), "third party must not receive the DM"

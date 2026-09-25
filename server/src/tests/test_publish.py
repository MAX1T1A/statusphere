import asyncio
import contextlib

import app.modules.realtime.infrastructure.hub as hubmod
from app.modules.realtime.infrastructure.hub import REPLAY_MAX_AGE, RealtimeHub, _Subscriber


async def test_publish_stamps_identity_and_skips_sender():
    hub = RealtimeHub()
    room = hub._get_or_create("room1")

    q_other = asyncio.Queue()
    q_sender = asyncio.Queue()
    room.subscribers.append(_Subscriber(device_id="other", account_id="acc-2", queue=q_other))
    room.subscribers.append(_Subscriber(device_id="sender", account_id="acc-1", queue=q_sender))

    await hub.publish("room1", "acc-1", "Max", "sender", {"device_id": "SPOOF", "account_id": "SPOOF", "x": 1})

    assert q_sender.empty(), "sender must not receive its own frame"
    msg = q_other.get_nowait()
    assert msg["device_id"] == "sender"
    assert msg["account_id"] == "acc-1"
    assert msg["account_name"] == "Max"
    assert msg["x"] == 1


async def test_subscribe_replays_the_last_snapshot_to_a_new_joiner():
    hub = RealtimeHub()
    await hub.publish("room1", "acc-1", "Max", "device-1", {"x": 1})

    agen = hub.subscribe("room1", "device-2", "acc-2")
    replayed = await agen.__anext__()
    await agen.aclose()

    assert replayed == {"x": 1, "account_id": "acc-1", "account_name": "Max", "device_id": "device-1"}


async def _still_waiting(agen):
    # A real timeout would need the wall clock, which the stale-snapshot test freezes,
    # so "nothing replayed" is observed as "the generator is still parked on queue.get()"
    # after one tick of the loop, not by racing a timer against it.
    task = asyncio.ensure_future(agen.__anext__())
    await asyncio.sleep(0)
    still_waiting = not task.done()
    task.cancel()
    with contextlib.suppress(asyncio.CancelledError):
        await task
    return still_waiting


async def test_subscribe_skips_the_joiners_own_last_snapshot():
    hub = RealtimeHub()
    await hub.publish("room1", "acc-1", "Max", "device-1", {"x": 1})

    agen = hub.subscribe("room1", "device-1", "acc-1")
    assert await _still_waiting(agen)


async def test_subscribe_does_not_replay_a_stale_snapshot(monkeypatch):
    now = [0.0]
    monkeypatch.setattr(hubmod.time, "monotonic", lambda: now[0])
    hub = RealtimeHub()
    await hub.publish("room1", "acc-1", "Max", "device-1", {"x": 1})

    now[0] = REPLAY_MAX_AGE + 1
    agen = hub.subscribe("room1", "device-2", "acc-2")
    assert await _still_waiting(agen), "a device that stopped publishing must not look freshly online to a new joiner"


async def test_group_message_goes_to_everyone_incl_sender():
    hub = RealtimeHub()
    room = hub._get_or_create("r")
    qa, qb, qc = asyncio.Queue(), asyncio.Queue(), asyncio.Queue()
    room.subscribers.append(_Subscriber(device_id="da", account_id="A", queue=qa))
    room.subscribers.append(_Subscriber(device_id="db", account_id="B", queue=qb))
    room.subscribers.append(_Subscriber(device_id="dc", account_id="C", queue=qc))

    await hub.deliver("r", "A", "Alice", "", "hi all", "2026-01-01T00:00:00")

    for q in (qa, qb, qc):
        m = q.get_nowait()
        assert m["type"] == "msg" and m["to"] == "" and m["text"] == "hi all" and m["from"] == "A"


async def test_dm_goes_only_to_recipient_and_sender():
    hub = RealtimeHub()
    room = hub._get_or_create("r")
    qa, qb, qc = asyncio.Queue(), asyncio.Queue(), asyncio.Queue()
    room.subscribers.append(_Subscriber(device_id="da", account_id="A", queue=qa))
    room.subscribers.append(_Subscriber(device_id="db", account_id="B", queue=qb))
    room.subscribers.append(_Subscriber(device_id="dc", account_id="C", queue=qc))

    await hub.deliver("r", "A", "Alice", "B", "secret", "2026-01-01T00:00:00")

    assert qa.get_nowait()["text"] == "secret", "sender's devices see their own DM"
    assert qb.get_nowait()["to"] == "B", "recipient receives the DM"
    assert qc.empty(), "third party must not receive the DM"

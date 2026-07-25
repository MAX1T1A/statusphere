from app.modules.presence.infrastructure.sampler import Sampler


class FakeWriter:
    def __init__(self, fail: bool = False):
        self.fail = fail
        self.batches = []

    async def save_batch(self, rows):
        if self.fail:
            raise RuntimeError("db down")
        self.batches.append(rows)


async def test_flush_uses_token_identity_not_payload():
    writer = FakeWriter()
    sampler = Sampler(writer, interval=15)
    await sampler.put("room1", "acc-1", "token-dev", {"device_id": "SPOOFED", "x": 1})
    await sampler._flush()

    assert len(writer.batches) == 1
    room_token, account_id, device_id, data = writer.batches[0][0]
    assert room_token == "room1"
    assert account_id == "acc-1"
    assert device_id == "token-dev"
    assert data["x"] == 1


async def test_flush_requeues_on_failure():
    writer = FakeWriter(fail=True)
    sampler = Sampler(writer, interval=15)
    await sampler.put("room1", "acc-1", "dev1", {"a": 1})
    await sampler._flush()
    assert sampler._buffer, "buffer must retain data when the write fails"

    writer.fail = False
    await sampler._flush()
    assert len(writer.batches) == 1
    assert not sampler._buffer


async def test_put_keeps_only_latest_per_device():
    writer = FakeWriter()
    sampler = Sampler(writer, interval=15)
    await sampler.put("room1", "acc-1", "dev1", {"n": 1})
    await sampler.put("room1", "acc-1", "dev1", {"n": 2})
    await sampler._flush()
    assert len(writer.batches[0]) == 1
    assert writer.batches[0][0][3]["n"] == 2

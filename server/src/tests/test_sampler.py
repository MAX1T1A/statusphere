from app.services.sampler import Sampler


class FakeRepo:
    def __init__(self, fail: bool = False):
        self.fail = fail
        self.batches = []

    async def save_batch(self, rows):
        if self.fail:
            raise RuntimeError("db down")
        self.batches.append(rows)


async def test_flush_uses_token_device_id_not_payload():
    repo = FakeRepo()
    sampler = Sampler(repo)
    await sampler.put("room1", "token-dev", {"device_id": "SPOOFED", "device_name": "n", "x": 1})
    await sampler._flush()

    assert len(repo.batches) == 1
    room_token, device_id, device_name, data = repo.batches[0][0]
    assert room_token == "room1"
    assert device_id == "token-dev"
    assert device_name == "n"
    assert data["x"] == 1


async def test_flush_requeues_on_failure():
    repo = FakeRepo(fail=True)
    sampler = Sampler(repo)
    await sampler.put("room1", "dev1", {"a": 1})
    await sampler._flush()
    assert sampler._buffer, "buffer must retain data when the write fails"

    repo.fail = False
    await sampler._flush()
    assert len(repo.batches) == 1
    assert not sampler._buffer


async def test_put_keeps_only_latest_per_device():
    repo = FakeRepo()
    sampler = Sampler(repo)
    await sampler.put("room1", "dev1", {"n": 1})
    await sampler.put("room1", "dev1", {"n": 2})
    await sampler._flush()
    assert len(repo.batches[0]) == 1
    assert repo.batches[0][0][3]["n"] == 2

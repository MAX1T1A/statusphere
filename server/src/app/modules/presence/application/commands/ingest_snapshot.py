from app.modules.presence.application.interfaces import IPresenceBroadcast, ISampler
from app.shared_kernel.operation import AuthenticatedOperation


class IngestPresenceSnapshot(AuthenticatedOperation):
    room: str
    account_name: str
    snapshot: dict


class IngestPresenceSnapshotUseCase:
    def __init__(self, broadcast: IPresenceBroadcast, sampler: ISampler) -> None:
        self._broadcast = broadcast
        self._sampler = sampler

    async def execute(self, op: IngestPresenceSnapshot) -> None:
        await self._broadcast.publish(
            op.room, op.actor.account_id, op.account_name, op.actor.device_id, op.snapshot
        )
        await self._sampler.put(op.room, op.actor.account_id, op.actor.device_id, op.snapshot)

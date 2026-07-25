from dataclasses import dataclass

from app.shared_kernel.bus import UseCaseBus
from asyncpg.pool import Pool


@dataclass
class Container:
    pool: Pool
    bus: UseCaseBus


def build_container(pool: Pool) -> Container:
    bus = UseCaseBus()
    return Container(pool=pool, bus=bus)

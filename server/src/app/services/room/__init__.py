from .storage import RoomStorage
from .v1.deliver import deliver_message
from .v1.publish import publish
from .v1.subscribe import subscribe


class RoomManager:
    def __init__(self) -> None:
        self._storage = RoomStorage()

    publish = publish
    subscribe = subscribe
    deliver_message = deliver_message

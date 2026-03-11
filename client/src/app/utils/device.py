import os
import uuid


def get_device_id() -> str:
    if env_id := os.getenv("DEVICE_ID"):
        return env_id
    mac = uuid.getnode()
    return str(uuid.UUID(int=mac, version=1))

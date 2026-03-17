import hashlib
import hmac
import os
import secrets

_secret: bytes | None = None


def _get_secret() -> bytes:
    global _secret
    if _secret is None:
        raw = os.environ.get("AUTH_SECRET", "")
        if not raw:
            raise RuntimeError("AUTH_SECRET env is not set")
        _secret = raw.encode()
    return _secret


def _sign(payload: str) -> str:
    return hmac.new(_get_secret(), payload.encode(), hashlib.sha256).hexdigest()


def generate_device_id() -> str:
    return secrets.token_hex(8)


def generate_room_id() -> str:
    return secrets.token_hex(16)


def generate_token(room_id: str, device_id: str) -> str:
    payload = f"{room_id}:{device_id}"
    return f"{payload}:{_sign(payload)}"


def verify_token(token: str) -> tuple[str, str] | None:
    parts = token.split(":")
    if len(parts) != 3:
        return None
    room_id, device_id, sig = parts
    if not room_id or not device_id:
        return None
    if not hmac.compare_digest(sig, _sign(f"{room_id}:{device_id}")):
        return None
    return room_id, device_id

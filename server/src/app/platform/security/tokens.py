import hashlib
import hmac
import secrets
import time

from app.platform.config import get_settings

TOKEN_V2 = "v2"


def _sign(payload: str) -> str:
    secret = get_settings().auth_secret.encode()
    return hmac.new(secret, payload.encode(), hashlib.sha256).hexdigest()


def generate_device_id() -> str:
    return secrets.token_hex(8)


def generate_room_id() -> str:
    return secrets.token_hex(16)


def generate_account_id() -> str:
    return secrets.token_hex(16)


def generate_account_secret() -> str:
    return secrets.token_hex(32)


def verifier(secret: str) -> str:
    return _sign(f"account-secret:{secret}")


def check_secret(secret: str, stored_verifier: str) -> bool:
    return hmac.compare_digest(verifier(secret), stored_verifier)


def generate_account_token(account_id: str, device_id: str) -> str:
    payload = f"account:{account_id}:{device_id}"
    return f"{TOKEN_V2}:{account_id}:{device_id}:{_sign(payload)}"


def verify_account_token(token: str) -> tuple[str, str] | None:
    parts = token.split(":")
    if len(parts) != 4 or parts[0] != TOKEN_V2:
        return None
    _, account_id, device_id, sig = parts
    if not account_id or not device_id:
        return None
    if not hmac.compare_digest(sig, _sign(f"account:{account_id}:{device_id}")):
        return None
    return account_id, device_id


def sign_code(kind: str, subject: str, ttl_seconds: int) -> str:
    exp = int(time.time()) + ttl_seconds
    sig = _sign(f"{kind}:{subject}:{exp}")
    return f"{kind}:{subject}:{exp}:{sig}"


def verify_code(kind: str, code: str) -> str | None:
    prefix = f"{kind}:"
    if not code.startswith(prefix):
        return None
    try:
        body, sig = code[len(prefix):].rsplit(":", 1)
        subject, exp_raw = body.rsplit(":", 1)
        exp = int(exp_raw)
    except ValueError:
        return None
    if not subject or exp < int(time.time()):
        return None
    if not hmac.compare_digest(sig, _sign(f"{kind}:{subject}:{exp}")):
        return None
    return subject

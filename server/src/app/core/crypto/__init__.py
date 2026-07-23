import os
from functools import lru_cache

from app.core.config import get_settings
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

_NONCE_LEN = 12


@lru_cache
def _cipher() -> AESGCM:
    return AESGCM(get_settings().presence_key)


def encrypt(plaintext: bytes) -> bytes:
    nonce = os.urandom(_NONCE_LEN)
    return nonce + _cipher().encrypt(nonce, plaintext, None)


def decrypt(blob: bytes) -> bytes:
    return _cipher().decrypt(blob[:_NONCE_LEN], blob[_NONCE_LEN:], None)

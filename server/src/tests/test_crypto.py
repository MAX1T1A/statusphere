import json

import pytest

from app.platform import crypto


def test_encrypt_decrypt_roundtrip():
    payload = {"active_app": "kitty", "spotify_status": "playing"}
    blob = crypto.encrypt(json.dumps(payload).encode())
    assert isinstance(blob, bytes)
    assert blob != json.dumps(payload).encode()
    assert json.loads(crypto.decrypt(blob)) == payload


def test_each_encryption_uses_fresh_nonce():
    a = crypto.encrypt(b"same")
    b = crypto.encrypt(b"same")
    assert a != b
    assert crypto.decrypt(a) == crypto.decrypt(b) == b"same"


def test_tampered_ciphertext_rejected():
    blob = bytearray(crypto.encrypt(b"secret"))
    blob[-1] ^= 0x01
    with pytest.raises(Exception):
        crypto.decrypt(bytes(blob))

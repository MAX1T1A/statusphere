from app.core.auth.auth import (
    generate_device_id,
    generate_room_id,
    generate_token,
    verify_token,
)


def test_token_roundtrip():
    rid, did = generate_room_id(), generate_device_id()
    token = generate_token(rid, did)
    assert verify_token(token) == (rid, did)


def test_tampered_signature_rejected():
    token = generate_token("room", "dev")
    flipped = token[:-1] + ("0" if token[-1] != "0" else "1")
    assert verify_token(flipped) is None


def test_malformed_token_rejected():
    assert verify_token("a:b") is None
    assert verify_token("only") is None
    assert verify_token("") is None
    assert verify_token(":dev:sig") is None

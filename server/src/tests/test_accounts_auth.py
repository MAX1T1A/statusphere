from app.core.auth import auth


def test_account_token_roundtrip():
    aid, did = auth.generate_account_id(), auth.generate_device_id()
    token = auth.generate_account_token(aid, did)
    assert token.startswith("v2:")
    assert auth.verify_account_token(token) == (aid, did)


def test_account_token_rejects_non_v2_and_tamper():
    assert auth.verify_account_token("room:dev:sig") is None
    assert auth.verify_account_token("") is None
    token = auth.generate_account_token("acc", "dev")
    assert auth.verify_account_token(token[:-1] + ("0" if token[-1] != "0" else "1")) is None


def test_secret_verifier():
    secret = auth.generate_account_secret()
    v = auth.verifier(secret)
    assert v != secret
    assert auth.check_secret(secret, v) is True
    assert auth.check_secret("wrong", v) is False


def test_code_roundtrip_and_kind_isolation():
    code = auth.sign_code("link", "acc123", ttl_seconds=300)
    assert auth.verify_code("link", code) == "acc123"
    assert auth.verify_code("invite", code) is None


def test_expired_code_rejected():
    expired = auth.sign_code("invite", "room123", ttl_seconds=-1)
    assert auth.verify_code("invite", expired) is None

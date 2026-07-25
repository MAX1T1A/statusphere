from app.modules.rooms.application.interfaces import IInviteCodec
from app.platform.security import sign_code, verify_code

INVITE_KIND = "invite"
INVITE_TTL = 3600


class InviteCodec(IInviteCodec):
    def sign(self, room_id: str) -> str:
        return sign_code(INVITE_KIND, room_id, INVITE_TTL)

    def verify(self, code: str) -> str | None:
        return verify_code(INVITE_KIND, code)

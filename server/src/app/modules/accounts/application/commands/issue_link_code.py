from app.platform.security import sign_code
from app.shared_kernel.operation import AuthenticatedOperation

LINK_KIND = "link"
LINK_TTL = 300


class IssueLinkCode(AuthenticatedOperation):
    room: str = ""


class IssueLinkCodeUseCase:
    async def execute(self, op: IssueLinkCode) -> str:
        subject = f"{op.actor.account_id}:{op.actor.device_id}:{op.room}"
        return sign_code(LINK_KIND, subject, LINK_TTL)

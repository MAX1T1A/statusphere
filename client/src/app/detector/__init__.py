from dataclasses import dataclass

from .de_wm import detect_de_wm
from .distro import detect_distro
from .os import detect_os


@dataclass
class SystemContext:
    os_family: str
    distro: str | None
    de_wm: str | None


class SystemDetector:
    def detect(self) -> SystemContext:
        return SystemContext(os_family=detect_os(), distro=detect_distro(), de_wm=detect_de_wm())

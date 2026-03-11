import os


def detect_de_wm() -> str | None:
    checks = [
        ("HYPRLAND_INSTANCE_SIGNATURE", "hyprland"),
        ("SWAYSOCK", "sway"),
        ("GNOME_DESKTOP_SESSION_ID", "gnome"),
        ("KDE_FULL_SESSION", "kde"),
    ]

    for env_var, name in checks:
        if os.getenv(env_var):
            return name

    xdg = os.getenv("XDG_CURRENT_DESKTOP", "").lower()
    if xdg:
        return xdg

    return None

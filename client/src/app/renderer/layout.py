from rich.panel import Panel
from rich.table import Table


def build_layout(devices: dict[str, dict]) -> Panel:
    table = Table(expand=True, show_edge=False, pad_edge=False)
    table.add_column("Device", style="bold cyan", no_wrap=True)
    table.add_column("CPU", justify="right")
    table.add_column("Memory", justify="right")
    table.add_column("Load 1m", justify="right")
    table.add_column("Uptime", justify="right")
    table.add_column("Workspace", style="magenta")
    table.add_column("Window", style="dim", max_width=40, no_wrap=True)

    if not devices:
        table.add_row("waiting for devices…", "", "", "", "", "", "")

    return Panel(table, title="[bold]statusphere[/bold]", subtitle="[dim]live feed[/dim]", border_style="blue")

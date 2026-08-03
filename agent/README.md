# Statusphere on a headless box

Same binary as the desktop client, just pointed at a server instead of a workstation. It joins your room like any other member, but instead of "what window is open" it reports how the hardware is doing. Shows up as a card in the TUI and in the bar widget, same as everyone else.

## Installing it on a server

Grab an invite from your own machine (`statusphere --invite`), then on the server:

```bash
curl -sSL https://raw.githubusercontent.com/MAX1T1A/statusphere/master/agent/install.sh \
  | sudo bash -s -- <invite> "example.com"
```

That's the whole setup: a `statusphere` system user, the binary in `/usr/local/bin`, joins the room, sets the name and marks the account as a server, drops `privacy.json` and `custom.json` templates in `~statusphere/.config/statusphere/` (worth editing `custom.json` for the box), installs the systemd unit, and enables it.

Safe to run again - it won't touch the user, configs, or unit if they're already there, and won't rejoin the room if an account already exists. It'll just update the binary and re-apply `--set-kind`/`--set-name`.

<details>
<summary>By hand, if you'd rather not pipe a script into sudo</summary>

```bash
useradd --system --create-home --home-dir /home/statusphere statusphere
install -m755 statusphere-linux-amd64 /usr/local/bin/statusphere

sudo -u statusphere -H statusphere --join <invite>
sudo -u statusphere -H statusphere --set-name "example.com"
sudo -u statusphere -H statusphere --set-kind server

install -m600 -o statusphere -g statusphere privacy.example.json /home/statusphere/.config/statusphere/privacy.json
install -m600 -o statusphere -g statusphere custom.example.json /home/statusphere/.config/statusphere/custom.json

install -m644 statusphere-agent.service /etc/systemd/system/
systemctl enable --now statusphere-agent
```

</details>

## Polling interval

`--interval 30s` in the unit is how often it collects and publishes. The desktop default of two seconds makes no sense on a server - there's nothing to watch live, and running `docker ps` that often costs more than the information is worth. Upper bound is 2 minutes: the room drops a device after five minutes of silence, and an interval close to that makes the card flicker offline.

Each field in `custom.json` has its own `repeat_seconds`, which overrides the poll interval - without it the command runs on every tick. Reasonable numbers: disk and docker checks in minutes, backups and certs in hours.

## What it reports

Built in: uptime, cpu, memory, disk (root), load average, core count. Everything else is a command in `custom.json` - there's an example next to it.

The thresholds it judges itself against live in `health.json`, same directory, created on first run:

```json
{
  "cpu_percent":  { "warn": 95, "crit": 99 },
  "memory_percent": { "warn": 90, "crit": 97 },
  "disk_percent": { "warn": 85, "crit": 95 },
  "load_per_core": { "warn": 1.5, "crit": 3 }
}
```

Zero out a pair to turn that metric off. The verdict (`ok`/`warn`/`crit`) is computed on the server itself and shipped to the room alongside the numbers, so everyone watching sees the same thing. A healthy machine sends nothing extra - silence is the "all good" state.

## What it can't do

Tell you it died. If the box goes down, the agent goes with it, and clients in the room just see the connection drop rather than a "server is down" message. That answer has to come from outside: the bar widget polls the server's own `/health` separately, and `/health/detail`, gated behind a device token, adds database state and how fresh the snapshot stream is.

## Privacy

Metrics go to the whole room like any other field. If something in `custom.json` shouldn't be visible, drop the command instead of counting on incognito - `custom: off` in the profile turns those fields off entirely.

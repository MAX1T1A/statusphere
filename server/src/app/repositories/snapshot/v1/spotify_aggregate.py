import json
from collections import defaultdict
from datetime import date, timedelta

from app.platform.crypto import decrypt

QUERY = """
    SELECT created_at::date AS day, data FROM snapshots
    WHERE room_token = $1 AND device_id = $2 AND created_at::date >= $3
"""


async def spotify_aggregate(self, room_token: str, device_id: str, since: date) -> dict:
    async with self.pool.acquire() as conn:
        rows = await conn.fetch(QUERY, room_token, device_id, since)

    interval = self.sample_interval
    per_day: dict = defaultdict(int)
    per_track: dict = defaultdict(int)
    per_artist: dict = defaultdict(int)
    per_album: dict = defaultdict(int)
    total = 0

    for row in rows:
        try:
            payload = json.loads(decrypt(row["data"]))
        except Exception:
            continue
        if payload.get("spotify_status") != "playing":
            continue

        total += 1
        per_day[row["day"]] += 1

        title = (payload.get("spotify_track") or "").strip()
        artist = (payload.get("spotify_artist") or "").strip()
        album = (payload.get("spotify_album") or "").strip()
        if title:
            per_track[(title, artist)] += 1
        if artist:
            per_artist[artist] += 1
        if album:
            per_album[album] += 1

    today = date.today()
    daily = []
    day = since
    while day <= today:
        daily.append({"day": str(day), "seconds": per_day.get(day, 0) * interval})
        day += timedelta(days=1)

    top_tracks = [
        {"title": title, "artist": artist, "seconds": count * interval}
        for (title, artist), count in sorted(per_track.items(), key=lambda kv: kv[1], reverse=True)[:3]
    ]
    top_artists = [
        {"artist": artist, "seconds": count * interval}
        for artist, count in sorted(per_artist.items(), key=lambda kv: kv[1], reverse=True)[:3]
    ]

    top_album = ""
    if per_album:
        top_album = max(per_album.items(), key=lambda kv: kv[1])[0]

    return {
        "total_seconds": total * interval,
        "daily": daily,
        "top_tracks": top_tracks,
        "top_artists": top_artists,
        "unique_tracks": len(per_track),
        "unique_artists": len(per_artist),
        "top_album": top_album,
    }

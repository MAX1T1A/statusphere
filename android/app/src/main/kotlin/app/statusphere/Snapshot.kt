package app.statusphere

import org.json.JSONArray
import org.json.JSONObject

enum class PlaybackStatus(val wire: String) { PLAYING("playing"), PAUSED("paused"), STOPPED("stopped") }

sealed interface Playback

data class Music(
    val track: String,
    val artist: String,
    val album: String,
    val artUrl: String,
    val status: PlaybackStatus,
    val positionSeconds: Int,
    val lengthSeconds: Int,
) : Playback

data class Video(
    val title: String,
    val channel: String,
    val status: PlaybackStatus,
    val positionSeconds: Int,
    val lengthSeconds: Int,
) : Playback

data class ForegroundApp(val label: String, val packageName: String)

data class Battery(val percent: Int, val charging: Boolean)

// Field names must match phoneSnapshot in client/mobile/snapshot.go.
data class PhoneSnapshot(
    val music: Music? = null,
    val video: Video? = null,
    val app: ForegroundApp? = null,
    val battery: Battery? = null,
) {
    fun playing(now: Playback?): PhoneSnapshot = copy(music = now as? Music, video = now as? Video)

    fun toJson(): String = JSONObject().apply {
        music?.let {
            put("music", JSONObject().apply {
                put("track", it.track)
                put("artist", it.artist)
                put("album", it.album)
                put("art_url", it.artUrl)
                put("status", it.status.wire)
                put("position_seconds", it.positionSeconds)
                put("length_seconds", it.lengthSeconds)
            })
        }
        video?.let {
            put("video", JSONObject().apply {
                put("title", it.title)
                put("channel", it.channel)
                put("status", it.status.wire)
                put("position_seconds", it.positionSeconds)
                put("length_seconds", it.lengthSeconds)
            })
        }
        app?.let {
            put("app", JSONObject().apply {
                put("label", it.label)
                put("package", it.packageName)
            })
        }
        battery?.let {
            put("battery", JSONObject().apply {
                put("percent", it.percent)
                put("charging", it.charging)
            })
        }
    }.toString()
}

data class Game(val name: String, val artUrl: String)

sealed interface Presence {
    data object Offline : Presence
    data class Incognito(val note: String) : Presence
    data class Online(val app: String, val music: Music?, val video: Video?, val game: Game?) : Presence
}

// Wire names must match client/internal/cardlayout/cardlayout.go.
enum class TileType(val wire: String) { SCALAR("scalar"), MUSIC("music"), GAME("game"), VIDEO("video"), PHOTO("photo"), PICTURE("picture") }

enum class ScalarForm(val wire: String) {
    RING("ring"),
    DIAL("dial"),
    BAR("bar"),
    NUMBER("number"),
    TEXT("text"),
    BIG("big"),
    CLOCK("clock"),
    WEATHER("weather"),
    WEATHER_LIVE("weatherLive"),
    MOON("moon"),
    SUN("sun"),
}

enum class TileColor(val wire: String) {
    PRIMARY("primary"),
    SECONDARY("secondary"),
    TERTIARY("tertiary"),
    ERROR("error"),
    PRIMARY_CONTAINER("primaryContainer"),
    SECONDARY_CONTAINER("secondaryContainer"),
    TERTIARY_CONTAINER("tertiaryContainer"),
    ERROR_CONTAINER("errorContainer"),
}

data class Tile(
    val col: Int,
    val row: Int,
    val cols: Int,
    val rows: Int,
    val type: TileType?,
    val form: ScalarForm,
    val color: TileColor?,
    val dimmed: Boolean,
    val label: String,
    val value: String,
    val note: String,
    val icon: String,
    val percent: Float?,
    val title: String,
    val subtitle: String,
    val imageUrl: String,
)

data class Card(val row: List<Tile>?, val detail: List<Tile>) {
    companion object {
        val NONE = Card(row = null, detail = emptyList())
    }
}

data class Account(val id: String, val name: String, val presence: Presence, val card: Card = Card.NONE)

// Keys must match client/internal/presence/keys.go.
private const val DEVICE_ID = "device_id"
private const val ACCOUNT_ID = "account_id"
private const val ACCOUNT_NAME = "account_name"
private const val DEVICE_NAME = "device_name"
private const val LAST_SEEN = "last_seen"
private const val OFFLINE = "_offline"
private const val INCOGNITO = "_incognito"
private const val INCOGNITO_NOTE = "_incognito_note"
private const val ACTIVE_APP = "active_app"
private const val MUSIC = "music"
private const val SPOTIFY_STATUS = "spotify_status"
private const val SPOTIFY_TRACK = "spotify_track"
private const val SPOTIFY_ARTIST = "spotify_artist"
private const val SPOTIFY_ALBUM = "spotify_album"
private const val SPOTIFY_ART_URL = "spotify_art_url"
private const val SPOTIFY_POSITION = "spotify_position"
private const val SPOTIFY_LENGTH = "spotify_length"
private const val GAME_STATUS = "game_status"
private const val GAME_NAME = "game_name"
private const val GAME_DISPLAY = "game_display"
private const val GAME_HEADER_URL = "game_header_url"
private const val GAME_HERO_URL = "game_hero_url"
private const val GAME_PLAYING = "playing"
private const val VIDEO_STATUS = "video_status"
private const val VIDEO_TITLE = "video_title"
private const val VIDEO_CHANNEL = "video_channel"
private const val VIDEO_POSITION = "video_position"
private const val VIDEO_LENGTH = "video_length"

private const val SHORT_ID_LENGTH = 8
private const val STALE_GAP_SECONDS = 45L

fun parseRoom(roomJSON: String): List<Account> {
    val room = JSONObject(roomJSON)
    val members = room.getJSONArray("members")
    val byAccount = linkedMapOf<String, MutableList<JSONObject>>()
    for (i in 0 until members.length()) {
        val member = members.getJSONObject(i)
        val id = member.text(ACCOUNT_ID) ?: member.text(DEVICE_ID) ?: continue
        byAccount.getOrPut(id) { mutableListOf() }.add(member)
    }
    val cards = room.optJSONArray("cards").objects().associate { it.optString("account_id") to cardOf(it) }
    return byAccount.map { (id, snapshots) -> accountOf(id, snapshots).copy(card = cards[id] ?: Card.NONE) }
        .sortedWith(compareBy<Account> { it.presence == Presence.Offline }.thenBy { it.name.lowercase() })
}

private fun accountOf(id: String, snapshots: List<JSONObject>): Account {
    val live = snapshots.filterNot { it.optBoolean(OFFLINE) }
    val labelDevice = live.minByOrNull { it.optString(DEVICE_ID) }
    val name = (listOfNotNull(labelDevice) + snapshots).firstNotNullOfOrNull { it.text(ACCOUNT_NAME) }
        ?: labelDevice?.text(DEVICE_NAME)
        ?: id.take(SHORT_ID_LENGTH)
    val newest = live.maxOfOrNull { it.optLong(LAST_SEEN) } ?: 0
    val devices = live.sortedWith(
        compareBy({ deviceRank(it) }, { newest - it.optLong(LAST_SEEN) > STALE_GAP_SECONDS }, { it.optString(DEVICE_ID) }),
    )
    val primary = devices.firstOrNull() ?: return Account(id, name, Presence.Offline)
    if (primary.optBoolean(INCOGNITO)) return Account(id, name, Presence.Incognito(primary.optString(INCOGNITO_NOTE)))
    val online = Presence.Online(
        app = primary.optString(ACTIVE_APP),
        music = devices.firstNotNullOfOrNull { musicOf(it) },
        video = devices.firstNotNullOfOrNull { videoOf(it) },
        game = devices.firstNotNullOfOrNull { gameOf(it) },
    )
    return Account(id, name, online)
}

private fun deviceRank(device: JSONObject): Int = when {
    device.optString(GAME_STATUS) == GAME_PLAYING -> 0
    device.optString(SPOTIFY_STATUS) == PlaybackStatus.PLAYING.wire -> 1
    device.has(SPOTIFY_STATUS) -> 2
    else -> 3
}

private fun musicOf(device: JSONObject): Music? {
    val status = PlaybackStatus.entries.find { it.wire == device.optString(SPOTIFY_STATUS) }
    val track = device.text(SPOTIFY_TRACK)
    if (status == null || track == null) {
        val title = device.text(MUSIC) ?: return null
        return Music(title, "", "", "", PlaybackStatus.PLAYING, 0, 0)
    }
    return Music(
        track = track,
        artist = device.optString(SPOTIFY_ARTIST),
        album = device.optString(SPOTIFY_ALBUM),
        artUrl = device.optString(SPOTIFY_ART_URL),
        status = status,
        positionSeconds = device.optInt(SPOTIFY_POSITION),
        lengthSeconds = device.optInt(SPOTIFY_LENGTH),
    )
}

private fun videoOf(device: JSONObject): Video? {
    val status = PlaybackStatus.entries.find { it.wire == device.optString(VIDEO_STATUS) } ?: return null
    val title = device.text(VIDEO_TITLE) ?: return null
    return Video(
        title = title,
        channel = device.optString(VIDEO_CHANNEL),
        status = status,
        positionSeconds = device.optInt(VIDEO_POSITION),
        lengthSeconds = device.optInt(VIDEO_LENGTH),
    )
}

private fun gameOf(device: JSONObject): Game? {
    if (device.text(GAME_STATUS) == null) return null
    val name = device.text(GAME_DISPLAY) ?: device.text(GAME_NAME) ?: return null
    return Game(name, device.text(GAME_HEADER_URL) ?: device.optString(GAME_HERO_URL))
}

fun parseCard(cardJSON: String): Card = cardOf(JSONObject(cardJSON))

private fun cardOf(card: JSONObject): Card = Card(
    row = card.optJSONArray("row")?.objects()?.map(::tileOf),
    detail = card.optJSONArray("detail").objects().map(::tileOf),
)

private fun tileOf(tile: JSONObject): Tile = Tile(
    col = tile.optInt("col"),
    row = tile.optInt("row"),
    cols = tile.optInt("cols", 1),
    rows = tile.optInt("rows", 1),
    type = TileType.entries.find { it.wire == tile.optString("type") },
    form = ScalarForm.entries.find { it.wire == tile.optString("form") } ?: ScalarForm.TEXT,
    color = TileColor.entries.find { it.wire == tile.optString("color") },
    dimmed = tile.optBoolean("dimmed"),
    label = tile.optString("label"),
    value = tile.optString("value"),
    note = tile.optString("note"),
    icon = tile.optString("icon"),
    percent = if (tile.has("percent")) tile.optDouble("percent").toFloat() else null,
    title = tile.optString("title"),
    subtitle = tile.optString("subtitle"),
    imageUrl = tile.optString("image_url"),
)

private fun JSONArray?.objects(): List<JSONObject> = if (this == null) emptyList() else List(length()) { getJSONObject(it) }

fun JSONArray.strings(): List<String> = List(length()) { getString(it) }

private fun JSONObject.text(key: String): String? = optString(key).ifEmpty { null }

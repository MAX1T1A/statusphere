package app.statusphere

import android.content.Context
import android.content.res.Resources
import android.os.Build
import android.text.format.DateFormat
import android.util.AtomicFile
import android.util.Log
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.util.readText
import androidx.core.util.writeText
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalContext
import androidx.glance.LocalSize
import androidx.glance.action.actionStartActivity
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.appWidgetBackground
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.appwidget.updateAll
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import java.io.File
import java.util.Date
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.collectIndexed
import kotlinx.coroutines.flow.conflate
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.mapNotNull
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject

private const val SAVED_ROOM_FILE = "friends-widget.json"
private val WIDGET_UPDATE_INTERVAL = 5.seconds
private val COMPACT = DpSize(100.dp, 100.dp)
private val WIDE = DpSize(220.dp, 100.dp)
private val DotSize = 10.dp

private enum class Dot { ONLINE, INCOGNITO, OFFLINE }

private data class Friend(val name: String, val dot: Dot, val status: String, val track: String)

private data class SavedRoom(val savedAt: Long, val friends: List<Friend>)

private val serviceStopped = PresenceService.status.map { it.room == null }.distinctUntilChanged()

suspend fun mirrorRoomToWidget(context: Context, status: Flow<PresenceStatus>) {
    status.mapNotNull { it.room }
        .map { room -> room.map { friendOf(it, context.resources) } }
        .distinctUntilChanged()
        .conflate()
        .collectIndexed { index, friends ->
            RoomMirror.save(context, SavedRoom(System.currentTimeMillis(), friends))
            FriendsWidget().updateAll(context)
            if (index == 0) publishPreview(context)
            delay(WIDGET_UPDATE_INTERVAL)
        }
}

suspend fun clearFriendsWidget(context: Context) {
    RoomMirror.clear(context)
    FriendsWidget().updateAll(context)
}

private fun friendOf(account: Account, resources: Resources): Friend {
    val presence = account.presence
    val dot = when (presence) {
        is Presence.Offline -> Dot.OFFLINE
        is Presence.Incognito -> Dot.INCOGNITO
        is Presence.Online -> Dot.ONLINE
    }
    val music = (presence as? Presence.Online)?.music
    val track = music?.let { listOf(it.track, it.artist).filter(String::isNotEmpty).joinToString(" - ") }.orEmpty()
    return Friend(account.name, dot, presence.statusLine(resources), track)
}

private suspend fun publishPreview(context: Context) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.VANILLA_ICE_CREAM) return
    runCatching { GlanceAppWidgetManager(context).setWidgetPreviews(FriendsWidgetReceiver::class) }
        .onFailure { Log.w(TAG, "widget_preview_failed detail=${it.message}") }
}

private object RoomMirror {
    private val latest = MutableStateFlow<SavedRoom?>(null)

    suspend fun load(context: Context): MutableStateFlow<SavedRoom?> {
        if (latest.value == null) {
            val saved = withContext(Dispatchers.IO) { read(context) }
            latest.compareAndSet(null, saved)
        }
        return latest
    }

    suspend fun save(context: Context, room: SavedRoom) {
        latest.value = room
        withContext(Dispatchers.IO) {
            runCatching { file(context).writeText(room.toJson()) }
                .onFailure { Log.e(TAG, "widget_room_save_failed detail=${it.message}") }
        }
    }

    suspend fun clear(context: Context) {
        latest.value = null
        withContext(Dispatchers.IO) {
            runCatching { file(context).delete() }
                .onFailure { Log.e(TAG, "widget_room_clear_failed detail=${it.message}") }
        }
    }

    private fun read(context: Context): SavedRoom? {
        val file = file(context)
        if (!file.baseFile.exists()) return null
        return runCatching { savedRoomOf(file.readText()) }
            .onFailure { Log.e(TAG, "widget_room_read_failed detail=${it.message}") }
            .getOrNull()
    }

    private fun file(context: Context) = AtomicFile(File(context.filesDir, SAVED_ROOM_FILE))
}

private fun SavedRoom.toJson(): String = JSONObject()
    .put("saved_at", savedAt)
    .put("friends", JSONArray(friends.map {
        JSONObject().put("name", it.name).put("dot", it.dot.name).put("status", it.status).put("track", it.track)
    }))
    .toString()

private fun savedRoomOf(json: String): SavedRoom {
    val root = JSONObject(json)
    val friends = root.getJSONArray("friends")
    return SavedRoom(
        savedAt = root.getLong("saved_at"),
        friends = (0 until friends.length()).map { i ->
            val friend = friends.getJSONObject(i)
            Friend(friend.getString("name"), Dot.valueOf(friend.getString("dot")), friend.getString("status"), friend.optString("track"))
        },
    )
}

class FriendsWidget : GlanceAppWidget() {
    override val sizeMode = SizeMode.Responsive(setOf(COMPACT, WIDE))
    override val previewSizeMode = sizeMode

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val room = RoomMirror.load(context)
        val joined = room.value != null || withContext(Dispatchers.IO) { PresenceService.isJoined(context) }
        provideContent {
            val saved by room.collectAsState()
            val stale by serviceStopped.collectAsState(initial = true)
            GlanceTheme { FriendsContent(saved, joined, stale) }
        }
    }

    override suspend fun providePreview(context: Context, widgetCategory: Int) {
        val room = RoomMirror.load(context).value
        provideContent { GlanceTheme { FriendsContent(room, joined = true, stale = false) } }
    }
}

class FriendsWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = FriendsWidget()
}

@Composable
private fun FriendsContent(room: SavedRoom?, joined: Boolean, stale: Boolean) {
    val context = LocalContext.current
    val colors = GlanceTheme.colors
    Column(
        GlanceModifier
            .fillMaxSize()
            .appWidgetBackground()
            .background(colors.widgetBackground)
            .padding(12.dp)
            .clickable(actionStartActivity<MainActivity>()),
    ) {
        if (room == null) {
            Box(GlanceModifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                val message = if (joined) R.string.room_connecting else R.string.widget_join
                Text(context.getString(message), style = TextStyle(color = colors.onSurfaceVariant, fontSize = 14.sp))
            }
            return@Column
        }
        val compact = LocalSize.current.width < WIDE.width
        val online = room.friends.count { it.dot != Dot.OFFLINE }
        Row(GlanceModifier.fillMaxWidth().padding(start = 4.dp, bottom = 8.dp)) {
            Text(
                context.getString(R.string.room_online, online, room.friends.size),
                style = TextStyle(color = colors.outline, fontSize = 14.sp),
                maxLines = 1,
                modifier = GlanceModifier.defaultWeight(),
            )
            if (stale && !compact) {
                Text(
                    context.getString(R.string.widget_saved_at, DateFormat.getTimeFormat(context).format(Date(room.savedAt))),
                    style = TextStyle(color = colors.outline, fontSize = 14.sp),
                    maxLines = 1,
                )
            }
        }
        LazyColumn {
            items(room.friends) { if (compact) CompactRow(it) else FriendCard(it) }
        }
    }
}

@Composable
private fun CompactRow(friend: Friend) {
    Row(GlanceModifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 3.dp), verticalAlignment = Alignment.CenterVertically) {
        StatusDot(friend.dot)
        Spacer(GlanceModifier.width(8.dp))
        Text(friend.name, style = TextStyle(color = nameColor(friend), fontSize = 14.sp), maxLines = 1)
    }
}

@Composable
private fun FriendCard(friend: Friend) {
    val colors = GlanceTheme.colors
    Box(GlanceModifier.fillMaxWidth().padding(bottom = 6.dp)) {
        Row(
            GlanceModifier.fillMaxWidth().background(colors.surface).cornerRadius(17.dp).padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            StatusDot(friend.dot)
            Spacer(GlanceModifier.width(12.dp))
            Column {
                Text(friend.name, style = TextStyle(color = nameColor(friend), fontSize = 16.sp), maxLines = 1)
                Text(friend.status, style = TextStyle(color = colors.outline, fontSize = 12.sp), maxLines = 1)
                if (friend.track.isNotEmpty()) {
                    Text(friend.track, style = TextStyle(color = colors.onSurfaceVariant, fontSize = 12.sp), maxLines = 1)
                }
            }
        }
    }
}

@Composable
private fun StatusDot(dot: Dot) {
    val colors = GlanceTheme.colors
    val color = when (dot) {
        Dot.ONLINE -> colors.primary
        Dot.INCOGNITO -> colors.secondary
        Dot.OFFLINE -> colors.surfaceVariant
    }
    Box(GlanceModifier.size(DotSize).background(color).cornerRadius(DotSize / 2)) {}
}

@Composable
private fun nameColor(friend: Friend): ColorProvider =
    if (friend.dot == Dot.OFFLINE) GlanceTheme.colors.onSurfaceVariant else GlanceTheme.colors.onSurface

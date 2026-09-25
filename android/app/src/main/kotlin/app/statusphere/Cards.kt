package app.statusphere

import android.content.res.Resources
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.SystemClock
import android.util.Log
import android.util.LruCache
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalResources
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import java.net.HttpURLConnection
import java.net.URL
import kotlin.math.PI
import kotlin.math.max
import kotlin.math.sin
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

internal val CardShape = RoundedCornerShape(17.dp)
internal val CardPadding = 12.dp
internal val CardSpacing = 12.dp
internal val ArtShape = RoundedCornerShape(12.dp)
private val AvatarSize = 40.dp
private val BadgeSize = 13.dp
private val BadgeBorder = 2.dp
private val MusicArtSize = 56.dp
private val ProgressStroke = 4.dp
private val ProgressGap = 4.dp
private val WaveAmplitude = 3.dp
private val WaveLength = 28.dp

private const val OFFLINE_ALPHA = 0.6f
private const val PAUSED_ART_ALPHA = 0.5f
private const val MIN_BANNER_ASPECT = 2f
private const val WAVE_PERIOD_MS = 7000
private const val ART_CACHE_BYTES = 24 * 1024 * 1024
private val ART_TIMEOUT = 10.seconds

@Composable
fun RoomCards(accounts: List<Account>, selfId: String?, onLeft: () -> Unit, onIncognito: (IncognitoChoice) -> Unit) {
    RoomHeader(accounts, onLeft)
    accounts.forEach { key(it.id) { AccountCard(it, pickable = it.id == selfId, onIncognito) } }
}

@Composable
private fun RoomHeader(accounts: List<Account>, onLeft: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var menuOpen by remember { mutableStateOf(false) }
    var confirming by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Box(Modifier.padding(horizontal = CardPadding)) {
        Text(
            stringResource(R.string.room_online, accounts.count { it.presence != Presence.Offline }, accounts.size),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.outline,
            modifier = Modifier.clickable { menuOpen = true },
        )
        DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
            DropdownMenuItem(
                text = { Text(stringResource(R.string.leave_room)) },
                onClick = {
                    menuOpen = false
                    confirming = true
                },
            )
        }
    }
    error?.let {
        Text(
            stringResource(R.string.leave_failed, it),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.error,
            modifier = Modifier.padding(horizontal = CardPadding),
        )
    }

    if (!confirming) return
    AlertDialog(
        onDismissRequest = { if (!busy) confirming = false },
        title = { Text(stringResource(R.string.leave_room_title)) },
        text = { Text(stringResource(R.string.leave_room_message)) },
        confirmButton = {
            TextButton(
                enabled = !busy,
                onClick = {
                    busy = true
                    error = null
                    scope.launch {
                        val result = PresenceService.leaveRoom(context)
                        busy = false
                        confirming = false
                        result.onSuccess { onLeft() }.onFailure { error = it.message ?: it.javaClass.simpleName }
                    }
                },
            ) { Text(stringResource(R.string.leave_room_confirm)) }
        },
        dismissButton = {
            TextButton(enabled = !busy, onClick = { confirming = false }) { Text(stringResource(R.string.action_cancel)) }
        },
    )
}

@Composable
private fun AccountCard(account: Account, pickable: Boolean, onIncognito: (IncognitoChoice) -> Unit) {
    val presence = account.presence
    val card = account.card
    var detailShown by rememberSaveable { mutableStateOf(false) }
    var selfMenuOpen by remember { mutableStateOf(false) }
    var renaming by rememberSaveable { mutableStateOf(false) }
    var styling by rememberSaveable { mutableStateOf(false) }
    Surface(
        onClick = { detailShown = !detailShown },
        enabled = card.detail.isNotEmpty(),
        shape = CardShape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier
            .fillMaxWidth()
            .alpha(if (presence == Presence.Offline) OFFLINE_ALPHA else 1f),
    ) {
        Column(Modifier.animateContentSize().padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            IncognitoHeader(pickable, AvatarSize, onIncognito, avatar = { Avatar(account) }) {
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Box {
                        Text(
                            account.name,
                            style = MaterialTheme.typography.bodyLarge,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = if (pickable) Modifier.clickable { selfMenuOpen = true } else Modifier,
                        )
                        DropdownMenu(expanded = selfMenuOpen, onDismissRequest = { selfMenuOpen = false }) {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.self_rename)) },
                                onClick = {
                                    selfMenuOpen = false
                                    renaming = true
                                },
                            )
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.self_appearance)) },
                                onClick = {
                                    selfMenuOpen = false
                                    styling = true
                                },
                            )
                        }
                    }
                    Text(
                        presence.statusLine(LocalResources.current),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            if (card.row != null) {
                if (card.row.isNotEmpty()) TileGrid(card.row, Modifier.padding(top = 8.dp))
            } else if (presence is Presence.Online && (presence.game != null || presence.music != null)) {
                Column(Modifier.padding(top = 8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    presence.game?.let { GameBanner(it) }
                    presence.music?.let { MusicCard(it) }
                }
            }
            if (detailShown && card.detail.isNotEmpty()) TileGrid(card.detail, Modifier.padding(top = 8.dp))
        }
    }
    if (renaming) RenameDialog(account.name) { renaming = false }
    if (styling) AppearanceSheet { styling = false }
}

@Composable
private fun RenameDialog(currentName: String, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var name by rememberSaveable { mutableStateOf(currentName) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.rename_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    enabled = !busy,
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                error?.let {
                    Text(
                        stringResource(R.string.rename_failed, it),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !busy && name.isNotBlank(),
                onClick = {
                    busy = true
                    error = null
                    scope.launch {
                        val result = PresenceService.setName(context, name.trim())
                        busy = false
                        result.onSuccess { onDismiss() }.onFailure { error = it.message ?: it.javaClass.simpleName }
                    }
                },
            ) { Text(stringResource(R.string.rename_save)) }
        },
        dismissButton = {
            TextButton(enabled = !busy, onClick = onDismiss) { Text(stringResource(R.string.action_cancel)) }
        },
    )
}

@Composable
private fun Avatar(account: Account) {
    val colors = MaterialTheme.colorScheme
    val presence = account.presence
    val offline = presence == Presence.Offline
    val badge = when (presence) {
        Presence.Offline -> colors.surfaceContainer
        is Presence.Incognito -> colors.secondary
        is Presence.Online -> colors.primary
    }
    Box(Modifier.size(AvatarSize)) {
        Box(
            Modifier
                .matchParentSize()
                .background(if (offline) colors.surfaceContainer else colors.secondaryContainer, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            if (presence is Presence.Incognito) {
                Icon(
                    painterResource(R.drawable.ic_visibility_off),
                    contentDescription = null,
                    tint = colors.onSecondaryContainer,
                    modifier = Modifier.size(20.dp),
                )
            } else {
                Text(
                    account.name.take(1).uppercase().ifEmpty { "?" },
                    style = MaterialTheme.typography.titleMedium,
                    color = if (offline) colors.outline else colors.onSecondaryContainer,
                )
            }
        }
        Box(
            Modifier
                .align(Alignment.BottomEnd)
                .size(BadgeSize)
                .background(badge, CircleShape)
                .border(BadgeBorder, colors.surfaceContainer, CircleShape),
        )
    }
}

internal fun Presence.statusLine(resources: Resources): String = when (this) {
    Presence.Offline -> resources.getString(R.string.status_offline)
    is Presence.Incognito -> note.ifEmpty { resources.getString(R.string.status_incognito) }
    is Presence.Online -> game?.let { resources.getString(R.string.status_playing_game, it.name) }
        ?: app.ifEmpty { resources.getString(R.string.status_online) }
}

@Composable
private fun GameBanner(game: Game) {
    val art = rememberArt(game.artUrl) ?: return
    Image(
        art.asImageBitmap(),
        contentDescription = game.name,
        contentScale = ContentScale.Crop,
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(max(art.width.toFloat() / art.height, MIN_BANNER_ASPECT))
            .clip(CardShape)
            .background(MaterialTheme.colorScheme.surfaceContainer),
    )
}

@Composable
private fun MusicCard(music: Music) {
    val playing = music.status == PlaybackStatus.PLAYING
    val lengthKnown = music.lengthSeconds > 0
    Surface(shape = CardShape, color = MaterialTheme.colorScheme.surfaceContainer, modifier = Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(CardPadding),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            rememberArt(music.artUrl)?.let {
                Image(
                    it.asImageBitmap(),
                    contentDescription = music.album.ifEmpty { null },
                    contentScale = ContentScale.Crop,
                    modifier = Modifier
                        .size(MusicArtSize)
                        .clip(ArtShape)
                        .alpha(if (playing) 1f else PAUSED_ART_ALPHA),
                )
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Column {
                    Text(music.track, style = MaterialTheme.typography.bodyLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (music.artist.isNotEmpty()) {
                        Text(
                            music.artist,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.outline,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                if (lengthKnown || !playing) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        val position = rememberPosition(music)
                        if (lengthKnown) {
                            WavyProgress(position.toFloat() / music.lengthSeconds, wavy = playing, Modifier.weight(1f))
                        }
                        Text(
                            if (lengthKnown) "${clock(position)} / ${clock(music.lengthSeconds)}" else stringResource(R.string.music_paused),
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.outline,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun rememberPosition(music: Music): Int {
    val position by produceState(music.positionSeconds, music) {
        if (music.status != PlaybackStatus.PLAYING) return@produceState
        val start = SystemClock.elapsedRealtime()
        while (value < music.lengthSeconds) {
            delay(1.seconds)
            value = music.positionSeconds + (SystemClock.elapsedRealtime() - start).milliseconds.inWholeSeconds.toInt()
        }
    }
    return position.coerceAtMost(music.lengthSeconds)
}

private fun clock(seconds: Int): String = "%d:%02d".format(seconds / 60, seconds % 60)

@Composable
internal fun WavyProgress(
    fraction: Float,
    wavy: Boolean,
    modifier: Modifier,
    active: Color = MaterialTheme.colorScheme.primary,
    track: Color = MaterialTheme.colorScheme.secondaryContainer,
) {
    val phase = if (wavy) wavePhase() else null
    Canvas(modifier.height(WaveAmplitude * 2 + ProgressStroke)) {
        val stroke = ProgressStroke.toPx()
        val mid = size.height / 2
        val start = stroke / 2
        val stop = size.width - stroke / 2
        val split = start + (stop - start) * fraction.coerceIn(0f, 1f)
        if (phase == null) {
            drawLine(active, Offset(start, mid), Offset(split, mid), stroke, StrokeCap.Round)
        } else {
            drawPath(wave(start, split, mid, phase()), active, style = Stroke(stroke, cap = StrokeCap.Round))
        }
        val trackStart = split + ProgressGap.toPx() + stroke
        if (trackStart < stop) drawLine(track, Offset(trackStart, mid), Offset(stop, mid), stroke, StrokeCap.Round)
        drawCircle(active, stroke / 2, Offset(stop, mid))
    }
}

@Composable
private fun wavePhase(): () -> Float {
    val phase by rememberInfiniteTransition(label = "wave").animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(tween(WAVE_PERIOD_MS, easing = LinearEasing)),
        label = "wave_phase",
    )
    return { phase }
}

private fun DrawScope.wave(start: Float, stop: Float, mid: Float, phase: Float): Path {
    val amplitude = WaveAmplitude.toPx()
    val length = WaveLength.toPx()
    val step = 1.dp.toPx()
    return Path().apply {
        var x = start
        while (x <= stop) {
            val y = mid + amplitude * sin(2 * PI * ((x - start) / length - phase)).toFloat()
            if (x == start) moveTo(x, y) else lineTo(x, y)
            x += step
        }
    }
}

private object ArtCache : LruCache<String, Bitmap>(ART_CACHE_BYTES) {
    override fun sizeOf(key: String, value: Bitmap): Int = value.byteCount
}

@Composable
internal fun rememberArt(url: String): Bitmap? {
    val art by produceState(ArtCache.get(url), url) {
        if (value != null || !url.startsWith("https://")) return@produceState
        value = withContext(Dispatchers.IO) {
            runCatching { fetchArt(url) }
                .onSuccess { ArtCache.put(url, it) }
                .onFailure { Log.w(TAG, "art_fetch_failed url=$url detail=${it.message}") }
                .getOrNull()
        }
    }
    return art
}

private fun fetchArt(url: String): Bitmap {
    val connection = URL(url).openConnection() as HttpURLConnection
    connection.connectTimeout = ART_TIMEOUT.inWholeMilliseconds.toInt()
    connection.readTimeout = ART_TIMEOUT.inWholeMilliseconds.toInt()
    try {
        return connection.inputStream.use { BitmapFactory.decodeStream(it) } ?: error("undecodable image")
    } finally {
        connection.disconnect()
    }
}

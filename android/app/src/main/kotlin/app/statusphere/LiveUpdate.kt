package app.statusphere

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.res.Resources
import android.graphics.Bitmap
import android.os.Build
import androidx.annotation.DrawableRes
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.graphics.drawable.IconCompat
import java.time.Instant
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.mapNotNull

private const val LIVE_CHANNEL_ID = "pinned_friend"
private const val LIVE_NOTIFICATION_ID = 2

fun createLiveChannel(context: Context) {
    val channel = NotificationChannel(LIVE_CHANNEL_ID, context.getString(R.string.live_channel), NotificationManager.IMPORTANCE_LOW)
    context.getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
}

fun clearLiveNotification(context: Context) {
    NotificationManagerCompat.from(context).cancel(LIVE_NOTIFICATION_ID)
}

private data class PlaybackProgress(val positionSeconds: Int, val lengthSeconds: Int, val paused: Boolean)

private data class RenderedLiveNotification(
    val name: String,
    val activityText: String,
    val activityShort: String,
    @DrawableRes val icon: Int,
    val artUrl: String?,
    val progress: PlaybackProgress?,
    val lastSeen: Instant?,
    val canPromote: Boolean,
)

suspend fun mirrorPinnedToNotification(context: Context, status: Flow<PresenceStatus>) {
    val largeIconPx = context.resources.getDimensionPixelSize(android.R.dimen.notification_large_icon_width)
    combine(PresenceService.pinned, status.mapNotNull { it.room }) { pinnedId, room ->
        pinnedId?.let { id -> room.find { it.id == id } }
    }.map { account -> account?.let { it.render(context) } }
        .distinctUntilChanged()
        .collectLatest { rendered ->
            val art = rendered?.artUrl?.let { loadArt(it, largeIconPx) }
            updateLiveNotification(context, rendered, art)
        }
}

private fun Account.render(context: Context): RenderedLiveNotification {
    val canPromote = presence is Presence.Online &&
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.BAKLAVA &&
        NotificationManagerCompat.from(context).canPostPromotedNotifications()
    val playback = presence.playback()
    return RenderedLiveNotification(
        name = name,
        activityText = activityText(presence, context.resources),
        activityShort = activityShort(presence, context.resources),
        icon = activityIcon(presence),
        artUrl = (presence as? Presence.Online)?.let { it.game?.artUrl ?: it.music?.artUrl }?.ifEmpty { null },
        progress = playback?.progress(),
        lastSeen = (presence as? Presence.Offline)?.lastSeen,
        canPromote = canPromote,
    )
}

private fun updateLiveNotification(context: Context, rendered: RenderedLiveNotification?, art: Bitmap?) {
    if (rendered == null) {
        clearLiveNotification(context)
        return
    }
    val openApp = PendingIntent.getActivity(
        context, 0, Intent(context, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
    )
    val builder = NotificationCompat.Builder(context, LIVE_CHANNEL_ID)
        .setSmallIcon(rendered.icon)
        .setLargeIcon(art)
        .setContentTitle(rendered.name)
        .setContentText(rendered.activityText)
        .setContentIntent(openApp)
        .setOngoing(true)
        .setOnlyAlertOnce(true)
        .setShowWhen(rendered.lastSeen != null)

    rendered.lastSeen?.let { builder.setWhen(it.toEpochMilli()) }
    rendered.progress?.let { builder.setStyle(progressStyle(context, it)) }
    if (rendered.canPromote) {
        builder.setRequestPromotedOngoing(true).setShortCriticalText(rendered.activityShort)
    }

    NotificationManagerCompat.from(context).notify(LIVE_NOTIFICATION_ID, builder.build())
}

private fun progressStyle(context: Context, progress: PlaybackProgress): NotificationCompat.ProgressStyle {
    val tracker = if (progress.paused) R.drawable.ic_tile_pause else R.drawable.ic_tile_play_arrow
    return NotificationCompat.ProgressStyle()
        .setProgressSegments(listOf(NotificationCompat.ProgressStyle.Segment(progress.lengthSeconds)))
        .setProgress(progress.positionSeconds)
        .setProgressTrackerIcon(IconCompat.createWithResource(context, tracker))
}

private fun Presence.playback(): Playback? {
    if (this !is Presence.Online || game != null) return null
    return video ?: music
}

private fun Playback.progress(): PlaybackProgress? {
    val (position, length, status) = when (this) {
        is Music -> Triple(positionSeconds, lengthSeconds, status)
        is Video -> Triple(positionSeconds, lengthSeconds, status)
    }
    if (length <= 0) return null
    return PlaybackProgress(position.coerceIn(0, length), length, paused = status != PlaybackStatus.PLAYING)
}

@DrawableRes
private fun activityIcon(presence: Presence): Int = when {
    presence is Presence.Incognito -> R.drawable.ic_visibility_off
    presence is Presence.Online && presence.game != null -> R.drawable.ic_tile_sports_esports
    else -> when (presence.playback()) {
        is Video -> R.drawable.ic_tile_play_arrow
        is Music -> R.drawable.ic_tile_library_music
        null -> R.drawable.ic_presence
    }
}

private fun activityText(presence: Presence, resources: Resources): String = when (val playback = presence.playback()) {
    is Video -> titledLine(playback.title, playback.channel)
    is Music -> titledLine(playback.track, playback.artist)
    null -> presence.statusLine(resources)
}

private fun activityShort(presence: Presence, resources: Resources): String = when (val playback = presence.playback()) {
    is Video -> playback.title
    is Music -> playback.track
    null -> presence.statusLine(resources)
}

private fun titledLine(title: String, subtitle: String): String =
    listOf(title, subtitle).filter(String::isNotEmpty).joinToString(" - ")

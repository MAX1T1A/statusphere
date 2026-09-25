package app.statusphere

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.res.Resources
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
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

suspend fun mirrorPinnedToNotification(context: Context, status: Flow<PresenceStatus>) {
    combine(PresenceService.pinned, status.mapNotNull { it.room }) { pinnedId, room ->
        pinnedId?.let { id -> room.find { it.id == id } }
    }.distinctUntilChanged().collect { account -> updateLiveNotification(context, account) }
}

private fun updateLiveNotification(context: Context, account: Account?) {
    if (account == null) {
        clearLiveNotification(context)
        return
    }
    val presence = account.presence
    val openApp = PendingIntent.getActivity(
        context, 0, Intent(context, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
    )
    val builder = NotificationCompat.Builder(context, LIVE_CHANNEL_ID)
        .setSmallIcon(R.drawable.ic_presence)
        .setContentTitle(account.name)
        .setContentText(activityText(presence, context.resources))
        .setContentIntent(openApp)
        .setOngoing(true)
        .setOnlyAlertOnce(true)

    val canPromote = presence is Presence.Online &&
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.BAKLAVA &&
        NotificationManagerCompat.from(context).canPostPromotedNotifications()
    if (canPromote) {
        builder.setRequestPromotedOngoing(true).setShortCriticalText(activityShort(presence, context.resources))
    }

    NotificationManagerCompat.from(context).notify(LIVE_NOTIFICATION_ID, builder.build())
}

private fun activityText(presence: Presence, resources: Resources): String {
    if (presence is Presence.Online && presence.game == null) {
        presence.video?.let { return titledLine(it.title, it.channel) }
        presence.music?.let { return titledLine(it.track, it.artist) }
    }
    return presence.statusLine(resources)
}

private fun activityShort(presence: Presence, resources: Resources): String {
    if (presence is Presence.Online && presence.game == null) {
        presence.video?.let { return it.title }
        presence.music?.let { return it.track }
    }
    return presence.statusLine(resources)
}

private fun titledLine(title: String, subtitle: String): String =
    listOf(title, subtitle).filter(String::isNotEmpty).joinToString(" - ")

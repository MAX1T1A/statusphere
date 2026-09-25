package app.statusphere

import android.content.ComponentName
import android.content.Context
import android.media.MediaMetadata
import android.media.session.MediaController
import android.media.session.MediaSessionManager
import android.media.session.PlaybackState
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.service.notification.NotificationListenerService
import android.util.Log

class MediaNotificationListener : NotificationListenerService()

class MusicTracker(context: Context, private val onChange: (Music?) -> Unit) {
    private val sessionManager = context.getSystemService(MediaSessionManager::class.java)
    private val listenerComponent = ComponentName(context, MediaNotificationListener::class.java)
    private val mainHandler = Handler(Looper.getMainLooper())
    private var controllers: List<MediaController> = emptyList()
    private var started = false

    private val sessionsChanged = MediaSessionManager.OnActiveSessionsChangedListener {
        track(it.orEmpty())
    }

    private val controllerCallback = object : MediaController.Callback() {
        override fun onPlaybackStateChanged(state: PlaybackState?) = publish()
        override fun onMetadataChanged(metadata: MediaMetadata?) = publish()
        override fun onSessionDestroyed() = publish()
    }

    fun start() {
        if (started) return
        try {
            sessionManager.addOnActiveSessionsChangedListener(sessionsChanged, listenerComponent, mainHandler)
            track(sessionManager.getActiveSessions(listenerComponent))
            started = true
        } catch (e: SecurityException) {
            Log.w(TAG, "media_sessions_denied detail=${e.message}")
        }
    }

    fun stop() {
        if (!started) return
        sessionManager.removeOnActiveSessionsChangedListener(sessionsChanged)
        track(emptyList())
        started = false
    }

    private fun track(active: List<MediaController>) {
        controllers.forEach { it.unregisterCallback(controllerCallback) }
        controllers = active
        controllers.forEach { it.registerCallback(controllerCallback, mainHandler) }
        Log.d(TAG, "media_sessions_changed count=${active.size}")
        publish()
    }

    fun current(): Music? = chosen()?.toMusic()

    // getActiveSessions returns controllers in priority order, most recently active first.
    private fun chosen(): MediaController? =
        controllers.firstOrNull { it.playbackState?.state == PlaybackState.STATE_PLAYING } ?: controllers.firstOrNull()

    private fun publish() {
        onChange(chosen()?.toMusic())
    }

    private fun MediaController.toMusic(): Music? {
        val meta = metadata ?: return null
        val track = meta.getString(MediaMetadata.METADATA_KEY_TITLE).orEmpty()
        if (track.isEmpty()) return null
        val state = playbackState
        return Music(
            track = track,
            artist = meta.getString(MediaMetadata.METADATA_KEY_ARTIST).orEmpty(),
            album = meta.getString(MediaMetadata.METADATA_KEY_ALBUM).orEmpty(),
            artUrl = meta.publicArtUrl().orEmpty(),
            status = state.status(),
            positionSeconds = state?.currentPositionMs()?.let { (it / 1000).toInt() } ?: 0,
            lengthSeconds = (meta.getLong(MediaMetadata.METADATA_KEY_DURATION) / 1000).toInt().coerceAtLeast(0),
        )
    }

    private fun MediaMetadata.publicArtUrl(): String? = ART_URI_KEYS
        .mapNotNull { getString(it) }
        .firstOrNull { it.startsWith("https://") || it.startsWith("http://") }

    private fun PlaybackState?.status(): PlaybackStatus = when (this?.state) {
        PlaybackState.STATE_PLAYING, PlaybackState.STATE_BUFFERING -> PlaybackStatus.PLAYING
        PlaybackState.STATE_PAUSED -> PlaybackStatus.PAUSED
        else -> PlaybackStatus.STOPPED
    }

    private fun PlaybackState.currentPositionMs(): Long {
        if (position < 0) return 0
        if (state != PlaybackState.STATE_PLAYING) return position
        val elapsed = SystemClock.elapsedRealtime() - lastPositionUpdateTime
        return position + (elapsed * playbackSpeed).toLong()
    }

    private companion object {
        val ART_URI_KEYS = listOf(
            MediaMetadata.METADATA_KEY_ART_URI,
            MediaMetadata.METADATA_KEY_ALBUM_ART_URI,
            MediaMetadata.METADATA_KEY_DISPLAY_ICON_URI,
        )
    }
}

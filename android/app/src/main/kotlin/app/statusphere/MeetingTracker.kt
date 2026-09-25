package app.statusphere

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.database.ContentObserver
import android.os.Handler
import android.os.Looper
import android.provider.CalendarContract
import androidx.core.content.ContextCompat
import androidx.core.content.edit
import java.time.Duration
import java.time.Instant
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

private val MEETING_LOOKAHEAD: Duration = Duration.ofHours(24)
private val ONCHANGE_DEBOUNCE: Duration = Duration.ofMillis(300)
private const val PREF_MEETING_ENABLED = "meeting_enabled"

object MeetingSettings {
    fun isEnabled(context: Context): Boolean =
        PresenceService.settings(context).getBoolean(PREF_MEETING_ENABLED, false)

    fun setEnabled(context: Context, enabled: Boolean) {
        PresenceService.settings(context).edit { putBoolean(PREF_MEETING_ENABLED, enabled) }
    }
}

class MeetingTracker(private val context: Context, private val onChange: (Instant?) -> Unit) {
    private val mainHandler = Handler(Looper.getMainLooper())
    private var started = false
    private var boundary: Runnable? = null
    private var debounced: Runnable? = null
    private var scope: CoroutineScope? = null
    private var queryJob: Job? = null

    private val observer = object : ContentObserver(mainHandler) {
        override fun onChange(selfChange: Boolean) {
            debounced?.let(mainHandler::removeCallbacks)
            val runnable = Runnable(::publish)
            debounced = runnable
            mainHandler.postDelayed(runnable, ONCHANGE_DEBOUNCE.toMillis())
        }
    }

    fun start() {
        if (started || !MeetingSettings.isEnabled(context) || !hasPermission()) return
        scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
        context.contentResolver.registerContentObserver(CalendarContract.Instances.CONTENT_URI, true, observer)
        started = true
        publish()
    }

    fun stop() {
        if (!started) return
        context.contentResolver.unregisterContentObserver(observer)
        boundary?.let(mainHandler::removeCallbacks)
        boundary = null
        debounced?.let(mainHandler::removeCallbacks)
        debounced = null
        queryJob?.cancel()
        queryJob = null
        scope?.cancel()
        scope = null
        started = false
    }

    // The toggle and the permission grant change from the UI, not from anything
    // this tracker listens for, so the screen calls this after either one.
    fun restart() {
        stop()
        start()
        if (!started) onChange(null)
    }

    private fun hasPermission() =
        ContextCompat.checkSelfPermission(context, Manifest.permission.READ_CALENDAR) == PackageManager.PERMISSION_GRANTED

    private fun publish() {
        queryJob?.cancel()
        queryJob = scope?.launch {
            val now = Instant.now()
            val instances = withContext(Dispatchers.IO) { queryInstances(now) }
            val current = instances.filter { !it.begin.isAfter(now) && it.end.isAfter(now) }
            val until = current.maxOfOrNull { it.end }
            onChange(until)
            val next = instances.filter { it.begin.isAfter(now) }.minOfOrNull { it.begin }
            scheduleBoundary(until ?: next)
        }
    }

    private fun scheduleBoundary(at: Instant?) {
        boundary?.let(mainHandler::removeCallbacks)
        boundary = null
        val delay = at?.let { it.toEpochMilli() - System.currentTimeMillis() } ?: return
        if (delay <= 0) return
        val runnable = Runnable(::publish)
        boundary = runnable
        mainHandler.postDelayed(runnable, delay)
    }

    private data class Instance(val begin: Instant, val end: Instant)

    private fun queryInstances(now: Instant): List<Instance> {
        val projection = arrayOf(
            CalendarContract.Instances.BEGIN,
            CalendarContract.Instances.END,
            CalendarContract.Instances.ALL_DAY,
            CalendarContract.Instances.AVAILABILITY,
            CalendarContract.Instances.SELF_ATTENDEE_STATUS,
        )
        val from = now.toEpochMilli()
        val to = from + MEETING_LOOKAHEAD.toMillis()
        val uri = CalendarContract.Instances.CONTENT_URI.buildUpon()
            .appendPath(from.toString())
            .appendPath(to.toString())
            .build()
        val out = mutableListOf<Instance>()
        context.contentResolver.query(uri, projection, null, null, null)?.use { cursor ->
            while (cursor.moveToNext()) {
                if (cursor.getInt(2) != 0) continue
                if (cursor.getInt(3) == CalendarContract.Events.AVAILABILITY_FREE) continue
                if (cursor.getInt(4) == CalendarContract.Attendees.ATTENDEE_STATUS_DECLINED) continue
                out += Instance(Instant.ofEpochMilli(cursor.getLong(0)), Instant.ofEpochMilli(cursor.getLong(1)))
            }
        }
        return out
    }
}

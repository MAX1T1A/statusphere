package app.statusphere

import android.app.AlarmManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Handler
import android.os.Looper
import androidx.core.content.ContextCompat
import java.time.Duration
import java.time.Instant

private val ALARM_WINDOW: Duration = Duration.ofHours(24)

class AlarmTracker(private val context: Context, private val onChange: (Instant?) -> Unit) {
    private val alarmManager = context.getSystemService(AlarmManager::class.java)
    private val mainHandler = Handler(Looper.getMainLooper())
    private var started = false
    private var windowCross: Runnable? = null

    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) = publish()
    }

    fun start() {
        if (started) return
        ContextCompat.registerReceiver(
            context, receiver, IntentFilter(AlarmManager.ACTION_NEXT_ALARM_CLOCK_CHANGED), ContextCompat.RECEIVER_NOT_EXPORTED,
        )
        started = true
        publish()
    }

    fun stop() {
        if (!started) return
        context.unregisterReceiver(receiver)
        windowCross?.let(mainHandler::removeCallbacks)
        windowCross = null
        started = false
    }

    private fun publish() {
        val trigger = alarmManager.nextAlarmClock?.triggerTime?.let(Instant::ofEpochMilli)
        onChange(trigger?.takeIf { it.isBefore(Instant.now().plus(ALARM_WINDOW)) })
        scheduleWindowCross(trigger)
    }

    // The alarm is published only once it is within ALARM_WINDOW, so a change
    // that leaves it untouched still needs a republish the moment it crosses in.
    private fun scheduleWindowCross(trigger: Instant?) {
        windowCross?.let(mainHandler::removeCallbacks)
        windowCross = null
        val crossesAt = trigger?.minus(ALARM_WINDOW) ?: return
        val delay = crossesAt.toEpochMilli() - System.currentTimeMillis()
        if (delay <= 0) return
        val runnable = Runnable(::publish)
        windowCross = runnable
        mainHandler.postDelayed(runnable, delay)
    }
}

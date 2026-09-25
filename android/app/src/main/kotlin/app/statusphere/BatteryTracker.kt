package app.statusphere

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import androidx.core.content.ContextCompat

class BatteryTracker(private val context: Context, private val onChange: (Battery?) -> Unit) {
    private var started = false
    private var last: Battery? = null

    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) = publish(intent)
    }

    fun start() {
        if (started) return
        val sticky = ContextCompat.registerReceiver(
            context, receiver, IntentFilter(Intent.ACTION_BATTERY_CHANGED), ContextCompat.RECEIVER_NOT_EXPORTED,
        )
        started = true
        sticky?.let(::publish)
    }

    fun stop() {
        if (!started) return
        context.unregisterReceiver(receiver)
        started = false
    }

    private fun publish(intent: Intent) {
        val level = intent.getIntExtra(BatteryManager.EXTRA_LEVEL, -1)
        val scale = intent.getIntExtra(BatteryManager.EXTRA_SCALE, -1)
        if (level < 0 || scale <= 0) return
        val plugged = intent.getIntExtra(BatteryManager.EXTRA_PLUGGED, 0) != 0
        val battery = Battery(percent = level * 100 / scale, charging = plugged)
        if (battery == last) return
        last = battery
        onChange(battery)
    }
}

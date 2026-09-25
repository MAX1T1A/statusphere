package app.statusphere

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.seconds

class ForegroundAppTracker(private val context: Context) {
    private val usageStats = context.getSystemService(UsageStatsManager::class.java)
    private val packageManager = context.packageManager
    private var queriedUntil = 0L
    private var latestPackage: String? = null

    fun current(): ForegroundApp? {
        val now = System.currentTimeMillis()
        val from = if (queriedUntil == 0L) now - FIRST_LOOKBACK.inWholeMilliseconds else queriedUntil - QUERY_OVERLAP.inWholeMilliseconds
        latestResumedPackage(from, now)?.let { latestPackage = it }
        queriedUntil = now
        val pkg = latestPackage ?: return null
        if (pkg == context.packageName || pkg == launcherPackage()) return null
        return ForegroundApp(label = labelOf(pkg), packageName = pkg)
    }

    private fun latestResumedPackage(from: Long, to: Long): String? {
        val events = usageStats.queryEvents(from, to) ?: return null
        val event = UsageEvents.Event()
        var latest: String? = null
        while (events.hasNextEvent()) {
            events.getNextEvent(event)
            if (event.eventType == RESUMED_EVENT) latest = event.packageName
        }
        return latest
    }

    private fun launcherPackage(): String? = packageManager
        .resolveActivity(Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_HOME), PackageManager.MATCH_DEFAULT_ONLY)
        ?.activityInfo?.packageName

    private fun labelOf(pkg: String): String = try {
        packageManager.getApplicationLabel(packageManager.getApplicationInfo(pkg, 0)).toString()
    } catch (_: PackageManager.NameNotFoundException) {
        pkg
    }

    private companion object {
        val FIRST_LOOKBACK = 24.hours
        val QUERY_OVERLAP = 5.seconds

        @Suppress("DEPRECATION")
        val RESUMED_EVENT = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            UsageEvents.Event.ACTIVITY_RESUMED
        } else {
            UsageEvents.Event.MOVE_TO_FOREGROUND
        }
    }
}

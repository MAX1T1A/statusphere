package app.statusphere

import android.Manifest
import android.app.AppOpsManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.PowerManager
import android.os.Process
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat

data class Permissions(
    val notificationAccess: Boolean,
    val usageAccess: Boolean,
    val postNotifications: Boolean,
    val batteryUnrestricted: Boolean,
) {
    val allGranted get() = notificationAccess && usageAccess && postNotifications && batteryUnrestricted

    companion object {
        fun check(context: Context) = Permissions(
            notificationAccess = hasNotificationAccess(context),
            usageAccess = hasUsageAccess(context),
            postNotifications = canPostNotifications(context),
            batteryUnrestricted = context.getSystemService(PowerManager::class.java)
                .isIgnoringBatteryOptimizations(context.packageName),
        )

        private fun hasNotificationAccess(context: Context) =
            context.packageName in NotificationManagerCompat.getEnabledListenerPackages(context)

        private fun hasUsageAccess(context: Context): Boolean {
            val appOps = context.getSystemService(AppOpsManager::class.java)
            val mode = appOps.checkOpNoThrow(AppOpsManager.OPSTR_GET_USAGE_STATS, Process.myUid(), context.packageName)
            if (mode != AppOpsManager.MODE_DEFAULT) return mode == AppOpsManager.MODE_ALLOWED
            return context.checkSelfPermission(Manifest.permission.PACKAGE_USAGE_STATS) == PackageManager.PERMISSION_GRANTED
        }

        private fun canPostNotifications(context: Context) =
            Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
                ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
                PackageManager.PERMISSION_GRANTED
    }
}

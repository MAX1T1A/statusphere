package app.statusphere

import android.Manifest
import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import androidx.annotation.StringRes
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.LifecycleResumeEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : ComponentActivity() {
    private var joined by mutableStateOf<Boolean?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        lifecycleScope.launch {
            val isJoined = withContext(Dispatchers.IO) { PresenceService.isJoined(this@MainActivity) }
            if (isJoined) PresenceService.start(this@MainActivity)
            joined = isJoined
        }
        setContent {
            StatusphereTheme {
                if (joined == false) JoinScreen(onJoined = { joined = true }) else Scaffold { padding ->
                    Column(
                        Modifier
                            .fillMaxSize()
                            .padding(padding)
                            .padding(16.dp)
                            .verticalScroll(rememberScrollState()),
                        verticalArrangement = Arrangement.spacedBy(CardSpacing),
                    ) {
                        if (joined == true) MainScreen()
                    }
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        if (joined == true) PresenceService.start(this)
    }
}

@Composable
private fun StatusphereTheme(content: @Composable () -> Unit) {
    val context = LocalContext.current
    val dark = isSystemInDarkTheme()
    val colors = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S ->
            if (dark) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        dark -> darkColorScheme()
        else -> lightColorScheme()
    }
    MaterialTheme(colorScheme = colors, content = content)
}

@Composable
private fun MainScreen() {
    val context = LocalContext.current
    val status by PresenceService.status.collectAsStateWithLifecycle()
    status.room?.let { room ->
        RoomCards(room, status.me?.accountId) { PresenceService.setIncognito(context, it.on, it.minutes) }
    } ?: Text(
        stringResource(R.string.room_connecting),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.outline,
        modifier = Modifier.padding(horizontal = CardPadding),
    )
    status.lastError?.let {
        Text(
            stringResource(R.string.room_last_error, it),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.error,
            modifier = Modifier.padding(horizontal = CardPadding),
        )
    }
    PermissionsChecklist()
}

@Composable
private fun PermissionsChecklist() {
    val context = LocalContext.current
    var permissions by remember { mutableStateOf(Permissions.check(context)) }
    LifecycleResumeEffect(Unit) {
        permissions = Permissions.check(context)
        onPauseOrDispose {}
    }
    val requestNotifications = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) {
        permissions = Permissions.check(context)
    }
    val openSettings = { action: String -> context.startActivity(Intent(action)) }
    if (permissions.allGranted) return

    Text(
        stringResource(R.string.permissions_title),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.outline,
        modifier = Modifier.padding(start = CardPadding, top = CardSpacing),
    )
    Surface(shape = CardShape, color = MaterialTheme.colorScheme.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            PermissionRow(R.string.permission_notification_access, permissions.notificationAccess) {
                openSettings(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)
            }
            PermissionRow(R.string.permission_usage_access, permissions.usageAccess) {
                openSettings(Settings.ACTION_USAGE_ACCESS_SETTINGS)
            }
            PermissionRow(R.string.permission_post_notifications, permissions.postNotifications) {
                requestNotifications.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
            PermissionRow(R.string.permission_battery, permissions.batteryUnrestricted) {
                openSettings(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS)
            }
        }
    }
}

@Composable
private fun PermissionRow(@StringRes title: Int, granted: Boolean, onOpen: () -> Unit) {
    if (granted) return
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(stringResource(title), style = MaterialTheme.typography.bodyLarge)
            Text(
                stringResource(R.string.permission_not_granted),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
        FilledTonalButton(onClick = onOpen) { Text(stringResource(R.string.permission_open)) }
    }
}

package app.statusphere

import android.Manifest
import android.content.Intent
import android.net.Uri
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.LifecycleResumeEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.lifecycleScope
import app.statusphere.mobile.Mobile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : ComponentActivity() {
    private var joined by mutableStateOf<Boolean?>(null)
    private var pendingInvite by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        pendingInvite = inviteFrom(intent)
        lifecycleScope.launch {
            val isJoined = withContext(Dispatchers.IO) { PresenceService.isJoined(this@MainActivity) }
            if (isJoined) PresenceService.start(this@MainActivity)
            joined = isJoined
        }
        setContent {
            StatusphereTheme {
                when {
                    joined == false -> JoinScreen(
                        prefillInvite = pendingInvite,
                        onJoined = {
                            pendingInvite = null
                            joined = true
                        },
                    )
                    joined == true -> Scaffold { padding ->
                        Column(
                            Modifier
                                .fillMaxSize()
                                .padding(padding)
                                .padding(16.dp)
                                .verticalScroll(rememberScrollState()),
                            verticalArrangement = Arrangement.spacedBy(CardSpacing),
                        ) {
                            MainScreen(onLeft = { joined = false })
                        }
                        pendingInvite?.let { invite ->
                            SwitchRoomDialog(
                                invite = invite,
                                onDismiss = { pendingInvite = null },
                                onSwitched = { pendingInvite = null },
                            )
                        }
                    }
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        inviteFrom(intent)?.let { pendingInvite = it }
    }

    override fun onResume() {
        super.onResume()
        if (joined == true) PresenceService.start(this)
    }

    private fun inviteFrom(intent: Intent?): String? {
        val data = intent?.data ?: return null
        if (data.scheme != "statusphere" || data.host != "join") return null
        return data.toString()
    }
}

@Composable
private fun SwitchRoomDialog(invite: String, onDismiss: () -> Unit, onSwitched: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.join_switch_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.join_switch_message))
                error?.let { Text(stringResource(R.string.join_failed, it), color = MaterialTheme.colorScheme.error) }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !busy,
                onClick = {
                    busy = true
                    error = null
                    scope.launch {
                        val result = PresenceService.leaveRoom(context).mapCatching {
                            withContext(Dispatchers.IO) { Mobile.join(PresenceService.baseDir(context), invite, "") }
                        }
                        busy = false
                        result
                            .onSuccess {
                                PresenceService.start(context)
                                onSwitched()
                            }
                            .onFailure { error = it.message ?: it.javaClass.simpleName }
                    }
                },
            ) { Text(stringResource(R.string.join_switch_confirm)) }
        },
        dismissButton = {
            TextButton(enabled = !busy, onClick = onDismiss) { Text(stringResource(R.string.action_cancel)) }
        },
    )
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
private fun MainScreen(onLeft: () -> Unit) {
    val context = LocalContext.current
    val status by PresenceService.status.collectAsStateWithLifecycle()
    status.room?.let { room ->
        RoomCards(room, status.me?.accountId, onLeft) { PresenceService.setIncognito(context, it.on, it.minutes) }
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
    MeetingToggle()
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
            if (!permissions.notificationAccess && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                RestrictedSettingsHint {
                    context.startActivity(
                        Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.fromParts("package", context.packageName, null)),
                    )
                }
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
private fun MeetingToggle() {
    val context = LocalContext.current
    var enabled by remember { mutableStateOf(MeetingSettings.isEnabled(context)) }
    val requestCalendar = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (!granted) enabled = false
        MeetingSettings.setEnabled(context, enabled)
        PresenceService.refreshMeetingTracking(context)
    }
    Surface(shape = CardShape, color = MaterialTheme.colorScheme.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(CardPadding).fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(stringResource(R.string.meeting_toggle_title), style = MaterialTheme.typography.bodyLarge)
                Text(
                    stringResource(R.string.meeting_toggle_note),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
            Switch(
                checked = enabled,
                onCheckedChange = { checked ->
                    enabled = checked
                    if (checked) {
                        requestCalendar.launch(Manifest.permission.READ_CALENDAR)
                    } else {
                        MeetingSettings.setEnabled(context, false)
                        PresenceService.refreshMeetingTracking(context)
                    }
                },
            )
        }
    }
}

@Composable
private fun RestrictedSettingsHint(onOpenAppInfo: () -> Unit) {
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            stringResource(R.string.permission_restricted_hint),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.outline,
            modifier = Modifier.weight(1f),
        )
        TextButton(onClick = onOpenAppInfo) { Text(stringResource(R.string.permission_app_info)) }
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

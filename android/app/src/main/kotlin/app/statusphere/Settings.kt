package app.statusphere

import androidx.activity.compose.BackHandler
import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeTopAppBar
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.launch

private val GroupSpacing = 16.dp
private val GroupItemGap = 2.dp
private val GroupOuterCorner = 24.dp
private val GroupInnerCorner = 6.dp
private val ProfileAvatarSize = 56.dp
private val ItemIconBox = 40.dp
private val ItemIconSize = 22.dp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(onBack: () -> Unit, onLeft: () -> Unit) {
    val context = LocalContext.current
    val colors = MaterialTheme.colorScheme
    val status by PresenceService.status.collectAsStateWithLifecycle()
    val self = status.room?.find { it.id == status.me?.accountId }
    val (meetingShown, setMeetingShown) = rememberMeetingSwitch()
    var renaming by rememberSaveable { mutableStateOf(false) }
    var styling by rememberSaveable { mutableStateOf(false) }
    var leaving by rememberSaveable { mutableStateOf(false) }
    val version = remember { context.packageManager.getPackageInfo(context.packageName, 0).versionName.orEmpty() }
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    BackHandler(onBack = onBack)

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            LargeTopAppBar(
                title = { Text(stringResource(R.string.settings)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(R.drawable.ic_arrow_back), contentDescription = stringResource(R.string.settings_back))
                    }
                },
                scrollBehavior = scrollBehavior,
            )
        },
    ) { padding ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(GroupSpacing),
        ) {
            self?.let { ProfileCard(it) { renaming = true } }
            SettingsGroup {
                SettingsItem(
                    position = GroupPosition.First,
                    icon = R.drawable.ic_palette,
                    iconColors = colors.tertiaryContainer to colors.onTertiaryContainer,
                    title = stringResource(R.string.self_appearance),
                    summary = stringResource(R.string.self_appearance_note),
                    onClick = { styling = true },
                )
                SettingsItem(
                    position = GroupPosition.Last,
                    icon = R.drawable.ic_tile_event_busy,
                    iconColors = colors.secondaryContainer to colors.onSecondaryContainer,
                    title = stringResource(R.string.meeting_toggle_title),
                    summary = stringResource(R.string.meeting_toggle_note),
                    onClick = { setMeetingShown(!meetingShown) },
                    trailing = { Switch(checked = meetingShown, onCheckedChange = null) },
                )
            }
            SettingsGroup {
                SettingsItem(
                    position = GroupPosition.Single,
                    icon = R.drawable.ic_logout,
                    iconColors = colors.errorContainer to colors.onErrorContainer,
                    title = stringResource(R.string.leave_room),
                    summary = stringResource(R.string.leave_room_message),
                    onClick = { leaving = true },
                )
            }
            Text(
                stringResource(R.string.settings_version, version),
                style = MaterialTheme.typography.bodySmall,
                color = colors.outline,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 16.dp),
            )
        }
    }
    if (renaming && self != null) RenameDialog(self.name) { renaming = false }
    if (styling) AppearanceSheet { styling = false }
    if (leaving) LeaveRoomDialog(onDismiss = { leaving = false }, onLeft = onLeft)
}

private enum class GroupPosition(val top: Dp, val bottom: Dp) {
    Single(GroupOuterCorner, GroupOuterCorner),
    First(GroupOuterCorner, GroupInnerCorner),
    Middle(GroupInnerCorner, GroupInnerCorner),
    Last(GroupInnerCorner, GroupOuterCorner),
}

@Composable
private fun ProfileCard(self: Account, onRename: () -> Unit) {
    Surface(
        onClick = onRename,
        shape = RoundedCornerShape(GroupOuterCorner),
        color = MaterialTheme.colorScheme.surfaceContainer,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            Modifier.padding(20.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Avatar(self, size = ProfileAvatarSize, initialStyle = MaterialTheme.typography.headlineSmall)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(self.name, style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(
                    stringResource(R.string.settings_name_hint),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Icon(
                painterResource(R.drawable.ic_edit),
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(20.dp),
            )
        }
    }
}

@Composable
private fun SettingsGroup(content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(GroupItemGap)) { content() }
}

@Composable
private fun SettingsItem(
    position: GroupPosition,
    @DrawableRes icon: Int,
    iconColors: Pair<Color, Color>,
    title: String,
    summary: String,
    onClick: () -> Unit,
    trailing: (@Composable () -> Unit)? = null,
) {
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(topStart = position.top, topEnd = position.top, bottomStart = position.bottom, bottomEnd = position.bottom),
        color = MaterialTheme.colorScheme.surfaceContainer,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Box(
                Modifier
                    .size(ItemIconBox)
                    .background(iconColors.first, CircleShape),
                contentAlignment = Alignment.Center,
            ) {
                Icon(painterResource(icon), contentDescription = null, tint = iconColors.second, modifier = Modifier.size(ItemIconSize))
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(title, style = MaterialTheme.typography.titleMedium)
                Text(summary, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            trailing?.invoke()
        }
    }
}

@Composable
private fun LeaveRoomDialog(onDismiss: () -> Unit, onLeft: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.leave_room_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.leave_room_message))
                error?.let {
                    Text(
                        stringResource(R.string.leave_failed, it),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !busy,
                onClick = {
                    busy = true
                    error = null
                    scope.launch {
                        val result = PresenceService.leaveRoom(context)
                        busy = false
                        result.onSuccess { onLeft() }.onFailure { error = it.message ?: it.javaClass.simpleName }
                    }
                },
            ) { Text(stringResource(R.string.leave_room_confirm)) }
        },
        dismissButton = {
            TextButton(enabled = !busy, onClick = onDismiss) { Text(stringResource(R.string.action_cancel)) }
        },
    )
}

@Composable
private fun RenameDialog(currentName: String, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var name by rememberSaveable { mutableStateOf(currentName) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.rename_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    enabled = !busy,
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                error?.let {
                    Text(
                        stringResource(R.string.rename_failed, it),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !busy && name.isNotBlank(),
                onClick = {
                    busy = true
                    error = null
                    scope.launch {
                        val result = PresenceService.setName(context, name.trim())
                        busy = false
                        result.onSuccess { onDismiss() }.onFailure { error = it.message ?: it.javaClass.simpleName }
                    }
                },
            ) { Text(stringResource(R.string.rename_save)) }
        },
        dismissButton = {
            TextButton(enabled = !busy, onClick = onDismiss) { Text(stringResource(R.string.action_cancel)) }
        },
    )
}

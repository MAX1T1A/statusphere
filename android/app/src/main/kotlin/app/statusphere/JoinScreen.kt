package app.statusphere

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboard
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import app.statusphere.mobile.Mobile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun JoinScreen(onJoined: () -> Unit) {
    val context = LocalContext.current
    val clipboard = LocalClipboard.current
    val focusManager = LocalFocusManager.current
    val scope = rememberCoroutineScope()
    var invite by rememberSaveable { mutableStateOf("") }
    var name by rememberSaveable { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by rememberSaveable { mutableStateOf<String?>(null) }
    val canJoin = !busy && invite.isNotBlank()

    fun join() {
        if (!canJoin) return
        focusManager.clearFocus()
        busy = true
        error = null
        scope.launch {
            val result = withContext(Dispatchers.IO) {
                runCatching { Mobile.join(PresenceService.baseDir(context), invite.trim(), name.trim()) }
            }
            busy = false
            result
                .onSuccess {
                    PresenceService.start(context)
                    onJoined()
                }
                .onFailure { error = it.message ?: it.javaClass.simpleName }
        }
    }

    fun pasteInvite() {
        scope.launch {
            val clip = clipboard.getClipEntry()?.clipData ?: return@launch
            if (clip.itemCount == 0) return@launch
            invite = clip.getItemAt(0).coerceToText(context).toString().trim()
            error = null
        }
    }

    Scaffold { padding ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .consumeWindowInsets(padding)
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 48.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            AppIdentity()
            Spacer(Modifier.height(16.dp))
            Surface(shape = CardShape, color = MaterialTheme.colorScheme.surfaceContainerLow) {
                Column(Modifier.padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = invite,
                        onValueChange = {
                            invite = it
                            error = null
                        },
                        label = { Text(stringResource(R.string.join_invite)) },
                        supportingText = {
                            Text(error?.let { stringResource(R.string.join_failed, it) } ?: stringResource(R.string.join_invite_hint))
                        },
                        isError = error != null,
                        trailingIcon = {
                            if (invite.isEmpty()) {
                                IconButton(onClick = ::pasteInvite, enabled = !busy) {
                                    Icon(painterResource(R.drawable.ic_paste), stringResource(R.string.join_paste))
                                }
                            }
                        },
                        enabled = !busy,
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(
                            capitalization = KeyboardCapitalization.None,
                            autoCorrectEnabled = false,
                            keyboardType = KeyboardType.Ascii,
                            imeAction = ImeAction.Next,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it },
                        label = { Text(stringResource(R.string.join_name)) },
                        supportingText = { Text(stringResource(R.string.join_name_hint)) },
                        enabled = !busy,
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(
                            capitalization = KeyboardCapitalization.Words,
                            imeAction = ImeAction.Done,
                        ),
                        keyboardActions = KeyboardActions(onDone = { join() }),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Button(onClick = ::join, enabled = canJoin, modifier = Modifier.fillMaxWidth().height(48.dp)) {
                        if (busy) {
                            CircularProgressIndicator(Modifier.size(20.dp), color = LocalContentColor.current, strokeWidth = 2.dp)
                        } else {
                            Text(stringResource(R.string.join_button))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AppIdentity() {
    Surface(shape = CircleShape, color = MaterialTheme.colorScheme.secondaryContainer) {
        Icon(
            painterResource(R.drawable.ic_presence),
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSecondaryContainer,
            modifier = Modifier.padding(24.dp).size(48.dp),
        )
    }
    Text(stringResource(R.string.app_name), style = MaterialTheme.typography.headlineMedium)
    Text(
        stringResource(R.string.app_tagline),
        style = MaterialTheme.typography.bodyLarge,
        color = MaterialTheme.colorScheme.outline,
        textAlign = TextAlign.Center,
    )
}

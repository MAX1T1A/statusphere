package app.statusphere

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageInstaller
import android.os.Build
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.statusphere.mobile.Mobile
import app.statusphere.mobile.Update
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

data class UpdateState(val update: Update? = null, val installing: Boolean = false, val error: String? = null)

object AppUpdate {
    const val ACTION_INSTALL_STATUS = "app.statusphere.action.INSTALL_STATUS"

    private val mutableState = MutableStateFlow(UpdateState())
    val state: StateFlow<UpdateState> = mutableState

    suspend fun check(context: Context) {
        if (isDebugBuild(context) || mutableState.value.update != null) return
        val current = context.packageManager.getPackageInfo(context.packageName, 0).versionName.orEmpty()
        val found = withContext(Dispatchers.IO) { runCatching { Mobile.checkUpdate(current) } }.getOrNull()
        if (found != null) mutableState.value = UpdateState(found)
    }

    suspend fun install(context: Context) {
        val update = mutableState.value.update ?: return
        mutableState.update { it.copy(installing = true, error = null) }
        runCatching { withContext(Dispatchers.IO) { commit(context, update) } }
            .onFailure { fail(it.message ?: it.javaClass.simpleName) }
    }

    fun onInstallStatus(context: Context, intent: Intent) {
        when (intent.getIntExtra(PackageInstaller.EXTRA_STATUS, PackageInstaller.STATUS_FAILURE)) {
            PackageInstaller.STATUS_PENDING_USER_ACTION -> confirmIntent(intent)?.let {
                context.startActivity(it.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
            }
            PackageInstaller.STATUS_SUCCESS -> Unit
            else -> fail(intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE) ?: "install failed")
        }
    }

    private fun fail(message: String) = mutableState.update { it.copy(installing = false, error = message) }

    private fun commit(context: Context, update: Update) {
        val apk = File(context.cacheDir, "update.apk")
        try {
            update.download(apk.path)
            val installer = context.packageManager.packageInstaller
            val params = PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL).apply {
                setAppPackageName(context.packageName)
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                    setRequireUserAction(PackageInstaller.SessionParams.USER_ACTION_NOT_REQUIRED)
                }
            }
            installer.openSession(installer.createSession(params)).use { session ->
                session.openWrite("statusphere.apk", 0, apk.length()).use { out ->
                    apk.inputStream().use { it.copyTo(out) }
                    session.fsync(out)
                }
                session.commit(statusReceiver(context).intentSender)
            }
        } finally {
            apk.delete()
        }
    }

    private fun statusReceiver(context: Context): PendingIntent {
        val intent = Intent(context, MainActivity::class.java).setAction(ACTION_INSTALL_STATUS)
        return PendingIntent.getActivity(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
        )
    }

    @Suppress("DEPRECATION")
    private fun confirmIntent(intent: Intent): Intent? =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            intent.getParcelableExtra(Intent.EXTRA_INTENT, Intent::class.java)
        } else {
            intent.getParcelableExtra(Intent.EXTRA_INTENT)
        }

    // A debug build is signed with the debug key, so a release APK cannot install over it.
    private fun isDebugBuild(context: Context) =
        context.applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE != 0
}

@Composable
fun UpdateBanner() {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val state by AppUpdate.state.collectAsStateWithLifecycle()
    LaunchedEffect(Unit) { AppUpdate.check(context) }
    val update = state.update ?: return

    Surface(shape = CardShape, color = MaterialTheme.colorScheme.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(CardPadding),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(stringResource(R.string.update_available, update.version), style = MaterialTheme.typography.bodyLarge)
                state.error?.let {
                    Text(
                        stringResource(R.string.update_failed, it),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
            FilledTonalButton(
                enabled = !state.installing,
                onClick = { scope.launch { AppUpdate.install(context.applicationContext) } },
            ) {
                Text(stringResource(if (state.installing) R.string.update_installing else R.string.update_install))
            }
        }
    }
}

package app.statusphere

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.text.format.DateFormat
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import app.statusphere.mobile.Mobile
import app.statusphere.mobile.RoomListener
import app.statusphere.mobile.Session
import java.time.Instant
import java.util.Date
import kotlin.concurrent.thread
import kotlin.time.Duration
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

const val TAG = "Statusphere"

data class Me(val accountId: String, val incognito: Boolean, val incognitoUntil: Instant?)

data class PresenceStatus(val room: List<Account>? = null, val lastError: String? = null, val me: Me? = null)

private enum class ScreenMode(val listening: Boolean, val heartbeat: Duration, val ping: Duration) {
    ON(listening = true, heartbeat = 30.seconds, ping = 20.seconds),
    OFF(listening = false, heartbeat = 180.seconds, ping = 120.seconds),
}

private val APP_POLL_INTERVAL = 3.seconds
private val PUBLISH_MIN_INTERVAL = 700.milliseconds
private const val CHANNEL_ID = "presence"
private const val NOTIFICATION_ID = 1
private const val ACTION_GO_INCOGNITO = "app.statusphere.action.GO_INCOGNITO"
private const val ACTION_GO_VISIBLE = "app.statusphere.action.GO_VISIBLE"
private const val EXTRA_MINUTES = "minutes"

class PresenceService : Service() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val sessionDispatcher = Dispatchers.IO.limitedParallelism(1)
    private val snapshot = MutableStateFlow(PhoneSnapshot())
    private val screenOn = MutableStateFlow(false)
    private lateinit var music: MusicTracker
    private lateinit var apps: ForegroundAppTracker
    private val session = MutableStateFlow<Session?>(null)

    private val screenReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            screenOn.value = intent.action != Intent.ACTION_SCREEN_OFF
        }
    }

    private val roomListener = object : RoomListener {
        override fun onRoom(roomJSON: String) {
            val room = runCatching { parseRoom(roomJSON) }
                .onFailure { Log.e(TAG, "room_parse_failed detail=${it.message}") }
                .getOrNull() ?: return
            mutableStatus.update { it.copy(room = room) }
        }

        override fun onError(event: String, detail: String) {
            Log.e(TAG, "$event detail=$detail")
            mutableStatus.update { it.copy(lastError = "$event: $detail") }
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        startInForeground()
        music = MusicTracker(this) { track -> snapshot.update { it.copy(music = track) } }
        apps = ForegroundAppTracker(this)
        screenOn.value = getSystemService(PowerManager::class.java).isInteractive
        val screenEvents = IntentFilter().apply {
            addAction(Intent.ACTION_SCREEN_ON)
            addAction(Intent.ACTION_SCREEN_OFF)
            addAction(Intent.ACTION_USER_PRESENT)
        }
        ContextCompat.registerReceiver(this, screenReceiver, screenEvents, ContextCompat.RECEIVER_NOT_EXPORTED)
        scope.launch { run() }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        music.start()
        val action = intent?.action
        if (action == ACTION_GO_INCOGNITO || action == ACTION_GO_VISIBLE) {
            val on = action == ACTION_GO_INCOGNITO
            val minutes = intent.getLongExtra(EXTRA_MINUTES, Mobile.IncognitoUntilTurnedOff)
            scope.launch { setIncognito(session.filterNotNull().first(), on, minutes) }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        scope.cancel()
        unregisterReceiver(screenReceiver)
        music.stop()
        session.value?.let { thread(name = "session_stop") { it.stop() } }
        session.value = null
        mutableStatus.update { it.copy(room = null, me = null) }
        super.onDestroy()
    }

    private suspend fun setIncognito(s: Session, on: Boolean, minutes: Long) = withContext(sessionDispatcher) {
        runCatching { s.setIncognito(on, minutes) }
            .onFailure { Log.e(TAG, "incognito_set_failed on=$on minutes=$minutes detail=${it.message}") }
        publishMe(s)
    }

    private fun publishMe(s: Session) {
        val until = s.incognitoUntilUnix().takeIf { it > 0 }?.let(Instant::ofEpochSecond)
        mutableStatus.update { it.copy(me = Me(s.accountID(), s.incognito(), until)) }
    }

    private suspend fun expireIncognito(s: Session, until: Instant) {
        delay(until.toEpochMilli() - System.currentTimeMillis())
        setIncognito(s, on = false, minutes = Mobile.IncognitoUntilTurnedOff)
    }

    private suspend fun run() {
        val opened = withContext(sessionDispatcher) { runCatching { Mobile.open(baseDir(this@PresenceService)) } }
        val s = opened.getOrElse {
            Log.e(TAG, "session_open_failed detail=${it.message}")
            stopSelf()
            return
        }
        session.value = s
        withContext(sessionDispatcher) {
            publishMe(s)
            runCatching { s.start(roomListener) }.onFailure { Log.e(TAG, "session_start_failed detail=${it.message}") }
        }
        scope.launch { publishThrottled(s) }
        scope.launch { mirrorRoomToWidget(this@PresenceService, status) }
        val me = status.map { it.me }.distinctUntilChanged()
        scope.launch { me.collect(::showNotification) }
        scope.launch { me.collectLatest { it?.incognitoUntil?.let { until -> expireIncognito(s, until) } } }
        screenOn.collectLatest { on ->
            if (on) {
                applyMode(s, ScreenMode.ON)
                pollForegroundApp()
            } else {
                snapshot.update { it.copy(app = null) }
                applyMode(s, ScreenMode.OFF)
            }
        }
    }

    private suspend fun publishThrottled(s: Session) {
        snapshot.collect { snap ->
            withContext(sessionDispatcher) {
                runCatching { s.publish(snap.toJson()) }
                    .onFailure { Log.e(TAG, "presence_publish_failed detail=${it.message}") }
            }
            delay(PUBLISH_MIN_INTERVAL)
        }
    }

    private suspend fun pollForegroundApp() {
        while (true) {
            val app = withContext(Dispatchers.IO) { apps.current() }
            snapshot.update { it.copy(app = app) }
            delay(APP_POLL_INTERVAL)
        }
    }

    private suspend fun applyMode(s: Session, mode: ScreenMode) = withContext(sessionDispatcher) {
        Log.i(TAG, "screen_mode_changed mode=$mode")
        s.setListening(mode.listening)
        runCatching {
            s.setHeartbeatSeconds(mode.heartbeat.inWholeSeconds)
            s.setPingSeconds(mode.ping.inWholeSeconds)
        }.onFailure { Log.e(TAG, "session_intervals_failed mode=$mode detail=${it.message}") }
    }

    private fun startInForeground() {
        val channel = NotificationChannel(CHANNEL_ID, getString(R.string.service_channel), NotificationManager.IMPORTANCE_LOW)
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        val notification = notification(me = null)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            startForeground(NOTIFICATION_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun showNotification(me: Me?) {
        getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, notification(me))
    }

    private fun notification(me: Me?): Notification {
        val openApp = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_presence)
            .setContentTitle(getString(R.string.service_notification_title))
            .setContentIntent(openApp)
            .setOngoing(true)
        when {
            me == null -> Unit
            me.incognito -> builder
                .setContentTitle(getString(R.string.status_incognito))
                .setContentText(me.incognitoUntil?.let { getString(R.string.incognito_until, clockTime(it)) } ?: getString(R.string.incognito_hidden))
                .addAction(R.drawable.ic_visibility, getString(R.string.incognito_go_visible), incognitoIntent(this, on = false))
            else -> builder
                .addAction(R.drawable.ic_visibility_off, getString(R.string.incognito_go), incognitoIntent(this, on = true))
        }
        return builder.build()
    }

    private fun clockTime(at: Instant): String = DateFormat.getTimeFormat(this).format(Date.from(at))

    companion object {
        private val mutableStatus = MutableStateFlow(PresenceStatus())
        val status: StateFlow<PresenceStatus> = mutableStatus.asStateFlow()

        fun baseDir(context: Context): String = context.filesDir.path

        fun isJoined(context: Context): Boolean = runCatching { Mobile.open(baseDir(context)) }.isSuccess

        fun start(context: Context) {
            ContextCompat.startForegroundService(context, Intent(context, PresenceService::class.java))
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, PresenceService::class.java))
        }

        suspend fun leaveRoom(context: Context): Result<Unit> = withContext(Dispatchers.IO) {
            runCatching {
                val left = Mobile.open(baseDir(context)).leave()
                check(left) { "you're the owner, or not a member of this room" }
            }.onSuccess {
                stop(context)
                clearFriendsWidget(context)
            }
        }

        fun setIncognito(context: Context, on: Boolean, minutes: Long) {
            ContextCompat.startForegroundService(context, incognitoCommand(context, on, minutes))
        }

        private fun incognitoCommand(context: Context, on: Boolean, minutes: Long) =
            Intent(context, PresenceService::class.java)
                .setAction(if (on) ACTION_GO_INCOGNITO else ACTION_GO_VISIBLE)
                .putExtra(EXTRA_MINUTES, minutes)

        private fun incognitoIntent(context: Context, on: Boolean): PendingIntent = PendingIntent.getForegroundService(
            context, 0, incognitoCommand(context, on, Mobile.IncognitoUntilTurnedOff), PendingIntent.FLAG_IMMUTABLE,
        )
    }
}

class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED && intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
        val pending = goAsync()
        thread(name = "boot_start") {
            try {
                if (PresenceService.isJoined(context)) {
                    runCatching { PresenceService.start(context) }
                        .onFailure { Log.e(TAG, "presence_boot_start_failed action=${intent.action} detail=${it.message}") }
                }
            } finally {
                pending.finish()
            }
        }
    }
}

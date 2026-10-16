package com.rahalgo.customer.live

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.rahalgo.customer.MainActivity
import com.rahalgo.customer.R
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.net.ApiClient
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خدمةُ مشاركة الموقع المباشر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **نقطةٌ كلَّ عشر ثوانٍ إلى المحرّك** — آخرُها وحدَها تُحفظ هناك.
 *
 * # ومتى تقف — أربعةُ أبواب
 *
 *   - **«وقّف المشاركة»** من الإشعار أو من بطاقة الطلب — **ويُمحى موقعُه في المحرّك.**
 *   - **وصل الطلبُ أو انتهى** — المحرّكُ يردّ `live_location_inactive`، **وبطاقةُ
 *     الطلب تُوقفها كذلك** (`OrdersViewModel`).
 *   - **خمسٌ وأربعون دقيقة** — **ومشاركةٌ تُنسى لا تبقى في جيبه إلى الليل.**
 *   - **سُحب الإذن** — تقف ولا تسقط.
 *
 * # وكخدمة السائق في نمطها لا في عملها
 *
 * **رفعٌ آمنٌ للخدمة الأماميّة ومرشِّحُ تردّد** (`app-driver/location/LocationService.kt`)
 * — **وما سواه يخصّ الزبونَ وحدَه**: طلبٌ واحدٌ ومهلةٌ وزرُّ إيقاف، **ولا طابورَ ولا
 * نبضة**: نقطةٌ سقطت تتبعها أخرى بعد عشر ثوانٍ، **والقديمةُ لا تفيد أحداً.**
 */
class LiveShareService : Service() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client by lazy { LocationServices.getFusedLocationProviderClient(this) }
    private val api by lazy { CustomerApi(AppCore.get().api) }

    private var orderId: String? = null

    /** **الدقيقُ وحدَه يُرسَل، وأحسنُ ما وُجد بعد ثلاثين ثانية** (قرارُ المالك ٢٠٢٦-١٠-١٦). */
    private val filter = LiveSendFilter()
    private var deadline: Job? = null
    private var stopping = false

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            val point = result.lastLocation ?: return
            send(point.latitude, point.longitude, if (point.hasAccuracy()) point.accuracy.toDouble() else null)
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            finish(clearServer = true)
            return START_NOT_STICKY
        }
        val id = intent?.getStringExtra(EXTRA_ORDER)
        // **ولا إذنَ فلا خدمة** — رفعُ خدمةٍ أماميّةٍ من نوع الموقع بلا إذنٍ يُسقط التطبيق.
        if (id.isNullOrEmpty() || !com.rahalgo.map.Here.granted(this) || !startForegroundSafely()) {
            LiveShare.activeOrderId = null
            stopSelf()
            return START_NOT_STICKY
        }
        // **وطلبٌ ثانٍ يحلّ محلّ الأوّل** — الخدمةُ واحدة، ومشاركةٌ واحدةٌ في كلّ لحظة.
        if (orderId != id) filter.reset()
        orderId = id
        stopping = false
        LiveShare.activeOrderId = id
        request()
        deadline?.cancel()
        deadline = scope.launch {
            delay(MAX_SHARE_MS)
            Log.i(TAG, "خمسٌ وأربعون دقيقة — تقف المشاركة")
            finish(clearServer = true)
        }
        // **ولا تُعاد إن قتلها النظام** — المشاركةُ بإذنِ ضغطةٍ لا تُستأنف بلاها.
        return START_NOT_STICKY
    }

    private fun request() {
        val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, SEND_EVERY_MS)
            .setMinUpdateIntervalMillis(SEND_EVERY_MS / 2)
            .build()
        try {
            client.removeLocationUpdates(callback)
            client.requestLocationUpdates(req, callback, Looper.getMainLooper())
        } catch (e: SecurityException) {
            Log.w(TAG, "سُحب إذنُ الموقع — تقف المشاركة", e)
            finish(clearServer = true)
        }
    }

    /**
     * **يرسل النقطة إن أجازها المرشّح** (`LiveSendFilter`) — الدقيقةُ فوراً، والتقريبيّةُ
     * تُحبس، **وبعد ثلاثين ثانيةً بلا دقيقةٍ يُرسَل أحسنُ ما وُجد** ومعه دقّتُه دائماً،
     * فيَسِمه المحرّكُ «تقريبيّاً». **ولا أسرعَ من مرّةٍ كلَّ ثماني ثوانٍ.**
     */
    private fun send(lat: Double, lng: Double, accuracyM: Double?) {
        val id = orderId ?: return
        if (stopping) return
        val pick = filter.offer(
            LiveSendFilter.Reading(lat, lng, accuracyM, SystemClock.elapsedRealtime()),
        ) ?: return
        scope.launch {
            try {
                // **والحدُّ يعود من اللوحة** — فيُرشَّح التالي برقمها.
                api.shareLive(id, pick.lat, pick.lng, pick.accuracyM)
                    ?.let { filter.maxAccuracyM = it.toDouble() }
            } catch (e: ApiClient.ApiException) {
                // **والطلبُ وصل أو انتهى أو ليس له** — تقف بلا محوٍ: المحرّكُ محاه أصلاً.
                if (e.body.code == "live_location_inactive" || e.body.code == "not_found") {
                    Log.i(TAG, "الطلبُ ليس في طريقه — تقف المشاركة")
                    finish(clearServer = false)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                // **والشبكةُ تسقط في الشارع** — والنقطةُ التالية بعد عشر ثوانٍ تحلّ محلّها.
                Log.d(TAG, "لم تصل نقطةُ الموقع", e)
            }
        }
    }

    /** **تقف المشاركة** — ويُمحى موقعُه في المحرّك إن طُلب، ثمّ تُغلق الخدمة. */
    private fun finish(clearServer: Boolean) {
        if (stopping) return
        stopping = true
        val id = orderId
        runCatching { client.removeLocationUpdates(callback) }
        deadline?.cancel()
        scope.launch {
            if (clearServer && id != null) {
                try {
                    api.stopLive(id)
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    Log.d(TAG, "لم يصل إيقافُ المشاركة", e)
                }
            }
            withContext(Dispatchers.Main) {
                LiveShare.activeOrderId = null
                orderId = null
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
    }

    override fun onDestroy() {
        runCatching { client.removeLocationUpdates(callback) }
        if (LiveShare.activeOrderId == orderId) LiveShare.activeOrderId = null
        scope.cancel()
        super.onDestroy()
    }

    /** **يرفع الخدمةَ أماميّةً بإشعارٍ ظاهرٍ وزرّ إيقاف** — **ويردّ `false` إن ردّها النظام.** */
    private fun startForegroundSafely(): Boolean {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(CHANNEL, getString(R.string.live_channel), NotificationManager.IMPORTANCE_LOW),
            )
        }
        val open = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
        )
        val stop = PendingIntent.getService(
            this, 1, Intent(this, LiveShareService::class.java).setAction(ACTION_STOP),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val note: Notification = NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(com.rahalgo.ui.R.drawable.ic_notification)
            .setContentTitle(getString(R.string.live_note_title))
            .setContentText(getString(R.string.live_note_text))
            .setContentIntent(open)
            .addAction(0, getString(R.string.live_stop), stop)
            .setOngoing(true)
            .build()
        return try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                startForeground(NOTE_ID, note, ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION)
            } else {
                startForeground(NOTE_ID, note)
            }
            true
        } catch (e: SecurityException) {
            Log.w(TAG, "رُدّت الخدمةُ الأماميّة", e)
            false
        } catch (e: IllegalStateException) {
            Log.w(TAG, "لا يُسمح برفع الخدمة الآن", e)
            false
        }
    }

    companion object {
        private const val TAG = "RahalGo/live"
        private const val CHANNEL = "rahalgo_live_share"
        private const val NOTE_ID = 2001
        const val EXTRA_ORDER = "order_id"
        const val ACTION_STOP = "com.rahalgo.customer.live.STOP"

        /** **كلَّ عشر ثوانٍ** — قرارُ المالك. */
        const val SEND_EVERY_MS = 10_000L

        /** **خمسٌ وأربعون دقيقةً ثمّ تقف** — قرارُ المالك. */
        const val MAX_SHARE_MS = 45 * 60 * 1000L
    }
}

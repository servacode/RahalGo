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
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
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
    private val client by lazy { com.rahalgo.map.RgLocation.client(this) }
    private val api by lazy { CustomerApi(AppCore.get().api) }

    private var orderId: String? = null

    /** **مشاركةُ الطرف الآخر** (قرارُ المالك ٢٠٢٦-١١-٢٨) — نقاطٌ إلى `my/party-orders` وتبدأ دقيقةً بلا انتظار. */
    private var party = false

    /** **الدقيقُ وحدَه يُرسَل، وأحسنُ ما وُجد بعد ثلاثين ثانية** (قرارُ المالك ٢٠٢٦-١٠-١٦). */
    private val filter = LiveSendFilter()
    private var stopping = false

    /** **ينتظر أن يُسند الطلب** — يبدأ هكذا، **وأوّلُ نقطةٍ تُقبل تُنهيه.** */
    private var waiting = true

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            val point = result.lastLocation ?: return
            send(point.latitude, point.longitude, if (point.hasAccuracy()) point.accuracy.toDouble() else null)
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            // **زرُّ الإشعار إيقافٌ بيده** — فلا تعود تلقائيّاً لهذا الطلب (٢٠٢٦-١١-٢٣).
            orderId?.let { LiveStopped.add(this, if (party) PartyPolicy.stopKey(it) else it) }
            finish(clearServer = true)
            return START_NOT_STICKY
        }
        val id = intent?.getStringExtra(EXTRA_ORDER)
        val isParty = intent?.getBooleanExtra(EXTRA_PARTY, false) == true
        // **ولا إذنَ فلا خدمة** — رفعُ خدمةٍ أماميّةٍ من نوع الموقع بلا إذنٍ يُسقط التطبيق.
        if (id.isNullOrEmpty() || !com.rahalgo.map.Here.granted(this) ||
            !startForegroundSafely(isParty, intent?.getBooleanExtra(EXTRA_PARTY_AMANAH, false) == true)
        ) {
            LiveShare.activeOrderId = null
            LiveShare.activePartyId = null
            stopSelf()
            return START_NOT_STICKY
        }
        // **وطلبٌ ثانٍ يحلّ محلّ الأوّل** — الخدمةُ واحدة، ومشاركةٌ واحدةٌ في كلّ لحظة.
        if (orderId != id || party != isParty) filter.reset()
        orderId = id
        party = isParty
        stopping = false
        if (isParty) {
            LiveShare.activeOrderId = null
            LiveShare.activePartyId = id
        } else {
            LiveShare.activePartyId = null
            LiveShare.activeOrderId = id
        }
        // **والطرفُ الآخر دقيقٌ كلَّ عشر ثوانٍ من أوّل لحظة** — لا إسنادَ ينتظره.
        waiting = false
        request()
        // **وتُعاد إن قتلها النظام** (قرارُ المالك ٢٠٢٦-١١-٢٣: «اذا فاتح او مسكر يظل
        // الموقع المباشر شغال») — **حيث يسمح أندرويد**؛ وإن ردّها وقفت بسلام
        // (`startForegroundSafely`) **وعادت حين يفتح التطبيق** (`LiveSharePolicy.autoStart`).
        return START_REDELIVER_INTENT
    }

    /**
     * **قبل الإسناد قراءةٌ أهدأ** — كلَّ ثلاثين ثانيةً بدقّةٍ متوسّطة: لا كابتنَ يراها بعد،
     * **والبطاريّةُ لا تُستنزف ساعةً والمطعمُ يحضّر.** وأوّلُ نقطةٍ تُقبل تعيدها عشراً.
     */
    private fun request() {
        val req = if (waiting) {
            LocationRequest.Builder(Priority.PRIORITY_BALANCED_POWER_ACCURACY, WAIT_EVERY_MS)
                .setMinUpdateIntervalMillis(WAIT_EVERY_MS / 2)
                .build()
        } else {
            LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, SEND_EVERY_MS)
                .setMinUpdateIntervalMillis(SEND_EVERY_MS / 2)
                .build()
        }
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
        // **والانتظارُ يسأل المحرّكَ بالقراءة كما هي** — لا مرشّحَ يحبسها: السؤالُ «أأُسند؟».
        if (waiting) {
            scope.launch { probe(id, lat, lng, accuracyM) }
            return
        }
        val pick = filter.offer(
            LiveSendFilter.Reading(lat, lng, accuracyM, SystemClock.elapsedRealtime()),
        ) ?: return
        val asParty = party
        scope.launch {
            try {
                // **والحدُّ يعود من اللوحة** — فيُرشَّح التالي برقمها.
                (if (asParty) api.shareParty(id, pick.lat, pick.lng, pick.accuracyM)
                else api.shareLive(id, pick.lat, pick.lng, pick.accuracyM))
                    ?.let { filter.maxAccuracyM = it.toDouble() }
            } catch (e: ApiClient.ApiException) {
                // **والطرفُ الآخر**: الطلبُ انتهى أو ليس له أو لم يوافق — تقف بلا محوٍ.
                if (asParty) {
                    if (PartyPolicy.shouldStop(e.body.code)) {
                        Log.i(TAG, "طلبُ الطرف الآخر ليس في طريقه — تقف المشاركة")
                        finish(clearServer = false)
                    }
                    return@launch
                }
                // **ولم يُسند بعد** (أُعيد للتوزيع مثلاً) — **تبقى وتنتظر بهدوء.**
                if (e.body.code == "live_location_not_yet") {
                    withContext(Dispatchers.Main) {
                        if (!stopping && !waiting) {
                            waiting = true
                            request()
                        }
                    }
                    return@launch
                }
                // **والزبونُ قال «مو بعنوان التوصيل»** (٢٠٢٦-١١-٢٤) — تقف ولا تعود لهالطلب.
                if (e.body.code == "live_location_elsewhere") {
                    orderId?.let { LiveStopped.add(this@LiveShareService, it) }
                    finish(clearServer = false)
                    return@launch
                }
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

    /** **أأُسند الطلب؟** — نقطةٌ تُقبل تعني نعم فتعود القراءةُ دقيقةً كلَّ عشر ثوانٍ. */
    private suspend fun probe(id: String, lat: Double, lng: Double, accuracyM: Double?) {
        try {
            api.shareLive(id, lat, lng, accuracyM)?.let { filter.maxAccuracyM = it.toDouble() }
            withContext(Dispatchers.Main) {
                if (!stopping && waiting) {
                    waiting = false
                    request()
                }
            }
        } catch (e: ApiClient.ApiException) {
            if (e.body.code == "live_location_elsewhere") LiveStopped.add(this@LiveShareService, id)
            if (e.body.code == "live_location_inactive" || e.body.code == "not_found" ||
                e.body.code == "live_location_elsewhere"
            ) {
                finish(clearServer = false)
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            Log.d(TAG, "لم يصل سؤالُ الإسناد", e)
        }
    }

    /** **تقف المشاركة** — ويُمحى موقعُه في المحرّك إن طُلب، ثمّ تُغلق الخدمة. */
    private fun finish(clearServer: Boolean) {
        if (stopping) return
        stopping = true
        val id = orderId
        val asParty = party
        runCatching { client.removeLocationUpdates(callback) }
        scope.launch {
            if (clearServer && id != null) {
                try {
                    if (asParty) api.stopParty(id) else api.stopLive(id)
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    Log.d(TAG, "لم يصل إيقافُ المشاركة", e)
                }
            }
            withContext(Dispatchers.Main) {
                LiveShare.activeOrderId = null
                LiveShare.activePartyId = null
                orderId = null
                party = false
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
    }

    override fun onDestroy() {
        runCatching { client.removeLocationUpdates(callback) }
        if (LiveShare.activeOrderId == orderId) LiveShare.activeOrderId = null
        if (LiveShare.activePartyId == orderId) LiveShare.activePartyId = null
        scope.cancel()
        super.onDestroy()
    }

    /** **يرفع الخدمةَ أماميّةً بإشعارٍ ظاهرٍ وزرّ إيقاف** — **ويردّ `false` إن ردّها النظام.** */
    private fun startForegroundSafely(forParty: Boolean = false, amanah: Boolean = false): Boolean {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(CHANNEL, getString(R.string.live_channel), NotificationManager.IMPORTANCE_MIN),
            )
        }
        val open = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
        )
        runCatching { manager.deleteNotificationChannel("rahalgo_live_share") }
        val builder = NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(com.rahalgo.ui.R.drawable.ic_notification)
            .setContentIntent(open)
            .setOngoing(true)
        if (forParty) {
            // **الطرفُ الآخر** (قرارُ المالك ٢٠٢٦-١١-٢٨): نصٌّ واحدٌ وزرُّ «إيقاف».
            val stop = PendingIntent.getService(
                this, 1,
                Intent(this, LiveShareService::class.java).setAction(ACTION_STOP),
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
            )
            builder
                .setContentTitle(getString(if (amanah) R.string.party_note_amanah else R.string.party_note_order))
                .addAction(0, getString(R.string.party_note_stop), stop)
        } else {
            builder
                .setContentTitle(getString(R.string.live_note_title))
                .setContentText(getString(R.string.live_note_text))
        }
        val note: Notification = builder.build()
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
        /**
         * **قناةٌ صامتةٌ مطويّة** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤: «اذا نقدر نخفيها لسا احسن») — `IMPORTANCE_MIN`
         * لا رمزَ لها في الشريط وتُطوى أسفلَ الإشعارات. **ولا يُخفى إشعارُ خدمةِ موقعٍ كلّيّاً** — شرطُ أندرويد.
         * **واسمٌ جديد** لأنّ أهمّيّةَ قناةٍ أُنشئت لا تتبدّل عند من ثبّت قبل.
         */
        private const val CHANNEL = "rahalgo_live_share_min"
        private const val NOTE_ID = 2001
        const val EXTRA_ORDER = "order_id"

        /** **مشاركةُ الطرف الآخر** — `EXTRA_ORDER` معرّفُ الطلب، والنقاطُ إلى `my/party-orders`. */
        const val EXTRA_PARTY = "party"
        const val EXTRA_PARTY_AMANAH = "party_amanah"
        const val ACTION_STOP = "com.rahalgo.customer.live.STOP"

        /** **كلَّ عشر ثوانٍ** — قرارُ المالك. */
        const val SEND_EVERY_MS = 10_000L

        /** **قبل الإسناد: كلَّ ثلاثين ثانية** — لا كابتنَ يراها بعد. */
        const val WAIT_EVERY_MS = 30_000L

        // **ولا سقفَ زمنيّاً** (قرارُ المالك ٢٠٢٦-١١-٢٤: «ما بدي يوقف لحاله ليخلص الطلب»):
        // **تقف بانتهاء الطلب** (`live_location_inactive`) **أو بإيقاف الزبون** وحدهما.
    }
}

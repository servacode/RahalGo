package com.rahalgo.driver.location

import com.rahalgo.ui.LastPoint
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.location.Location
import android.os.Build
import android.os.IBinder
import android.os.SystemClock
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.rahalgo.driver.MainActivity
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.shared.driver.DriverApi
import com.rahalgo.shared.model.TrackPoint
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خدمة الموقع — تعمل ما دامت الوردية مفتوحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ١ من `docs/DRIVER-APP-PLAN.md`.)
 *
 * # لماذا خدمة أماميّة لا مؤقّت في الشاشة
 *
 * **السائق يضع الجوّال في جيبه** — والشاشة تموت، وأندرويد يوقف كلّ ما
 * يعمل في الخلفيّة بعد دقائق. **فينقطع الموقع في اللحظة التي يحتاجه فيها
 * المكتب والزبون.**
 *
 * **والخدمة الأماميّة عقد مع النظام**: إشعار دائم يراه صاحبه مقابل ألّا
 * يُقتل التطبيق. **والإشعار ليس زينة** — هو ما يجعل السائق يعرف أنّ
 * ورديّته مفتوحة وأنّ موقعه يُرسل.
 *
 * # وما الذي يوفّر البطارية
 *
 * **ثلاثة أشياء معا** (وهي ما يفعله هذا الصنف من التطبيقات):
 *
 * ١ · **فترة من المحرّك لا من الشيفرة** (`drivers.location_ping_sec`) —
 *     فتُضبط للمدينة كلّها من لوحة الإدارة بلا نسخة جديدة.
 * ٢ · **فلتر مسافة أثناء التوصيل** (خمسة أمتار) — **وبلا طلبٍ لا فلتر** (٢٠٢٦-١١-١٨): نقطةٌ
 *     كلَّ فترةٍ ولو واقف، ورجفتُه يمسكها الخادم.
 * ٣ · **ولا دفعات** (٢٠٢٦-١١-١٨، `setMaxUpdateDelayMillis(0)`) — كان النظامُ يجمّع بلا طلب،
 *     **ونقطتان في دفعةٍ صارتا نداءين** بعد ما صار يرسل كلَّ فترةٍ ولو واقف.
 *
 * # ولا منطق عمل هنا
 *
 * الخدمة تقرأ الموقع وترسله. **متى يبدأ ومتى يقف يقرّره حال الوردية**
 * في `HomeViewModel`.
 */
class LocationService : Service() {

    // **والأرقامُ بالأجنبيّة** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — انظر `WesternDigits`.
    override fun attachBaseContext(base: android.content.Context) {
        super.attachBaseContext(com.rahalgo.ui.WesternDigits.wrap(base))
    }


    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client by lazy { com.rahalgo.map.RgLocation.client(this) }

    // **وتقرؤه النبضةُ من خيطٍ آخر** لتحسب موعدَها منه (A6.5) — فيُرى ما كُتب للتوّ.
    @Volatile
    private var lastSentAt = 0L

    /** **حارسُ التردّد** — على قدر الفترة أثناء التوصيل (`request`). */
    private var minSendGapMs = MIN_SEND_GAP_MS

    /**
     * **كم صمتٍ قبل نبضة الواقف** — دقيقتان أثناء التوصيل، **وضعفُ الفترة بلا طلب** (٢٠٢٦-١١-١٨):
     * الواقفُ بلا طلبٍ يرسل كلَّ فترةٍ من النظام نفسِه، والنبضةُ احتياطٌ لنظامٍ سكت. وتقرؤه حلقتُها.
     */
    @Volatile
    private var heartbeatEveryMs = Heartbeat.EVERY_MS
    // **ونبضةٌ واحدةٌ لا اثنتان** — `onStartCommand` يُنادى مرّاتٍ
    // (إعادةُ تشغيلٍ بعد قتلٍ، أو تبديلُ تردّد)، **وحلقةٌ لكلّ نداءٍ
    // تُضاعف النداءات بلا أن يظهر ذلك في شيء.**
    private var beating = false
    private val queue by lazy { PointQueue(applicationContext) }

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            // **وكلُّ نقاط الدفعة لا آخرُها** (A6.1، 0.9.0) — الأحدثُ يُرسَل حيّاً، **وما قبله يمرّ
            // بالطابور** فيصل الأثرَ مع نداء الدفعة بعده. (كانت الأحدثُ وحدَها تُستعمل والباقي
            // يُرمى — فبلا طلبٍ لا يصل إلّا موضعٌ كلَّ دقيقةٍ أو دقيقتين.)
            val (older, point) = PointQueue.splitDelivered(result.locations) { it.time }
            if (point == null) return
            val earlier = older.map { trackPointOf(it) }
            // **وأثر الوصول يُكتب** — «لا فشل» ليس دليل نجاح: قد لا
            // تصل نقطة أصلا، **والسكوت يُقرأ عملا وهو صمت.**
            Log.i(TAG, "نقطة: ${point.latitude}, ${point.longitude} دقّة ${point.accuracy}")
            // **والشاشة تقرؤه من هنا** — الخريطة تتحرّك مع صاحبها.
            // ══════════════════════════════════════════════════════════
            // **وتُحمَل السرعةُ ووقتُها معه** (قرارُ المالك ٢٠٢٦-٠٩-٢٨)
            // ══════════════════════════════════════════════════════════
            //
            // **ولوحةُ الرحلة تعرض `0 كم/س` للواقف** — وذاك لا يُعرف من
            // موضعٍ بلا وقت: **قراءةٌ شائخةٌ لا تصف وقوفاً.** انظر
            // `SpeedReadout`.
            //
            // **و`hasSpeed()` كاذبةٌ على الواقف** — فيُمرَّر فراغُها كما
            // هو، **والصفرُ يُقرّره القارئُ لا الكاتب.**
            accept(point, earlier)
        }
    }

    /**
     * **قراءةٌ حقيقيّةٌ من النظام** — تُحفظ للشاشة وتُرسَل، والإشارةُ حيّة.
     *
     * و[earlier] ما سبقها في الدفعة نفسِها — يُصَفّ ويخرج مع نداء الدفعة بعدها.
     */
    private fun accept(point: Location, earlier: List<TrackPoint> = emptyList()) {
        LastPoint.set(
            point.latitude,
            point.longitude,
            mocked = point.isMocked(),
            speedMps = if (point.hasSpeed()) point.speed else null,
            atMs = android.os.SystemClock.elapsedRealtime(),
            accuracyM = if (point.hasAccuracy()) point.accuracy else null,
        )
        GpsSignal.alive()
        send(point, earlier)
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val seconds = intent?.getLongExtra(EXTRA_PING_SEC, 0L)?.takeIf { it > 0 } ?: DEFAULT_PING_SEC
        val live = intent?.getBooleanExtra(EXTRA_LIVE, false) ?: false
        val off = intent?.getBooleanExtra(EXTRA_OFF_SHIFT, false) ?: false
        // ══════════════════════════════════════════════════════════════
        // **وإذنٌ سُحب والورديّةُ مفتوحة** (`AB-09`، قِيس ٢٠٢٦-٠٩-١٦)
        // ══════════════════════════════════════════════════════════════
        //
        // **وقِيس على المحاكي**: **سُحب إذنُ الموقع من الإعدادات
        // والسائقُ في ورديّته** — **فأعاد النظامُ تشغيلَ الخدمة
        // (`START_STICKY`)، فطلبت خدمةً أماميّةً من نوع `location` بلا
        // إذن**:
        //
        //	SecurityException: Starting FGS with type location …
        //	requires … ACCESS_FINE_LOCATION
        //
        // **فسقط التطبيقُ — ثمّ أُعيد فسقط**: **حلقةُ سقوطٍ في يد
        // سائقٍ يعمل.**
        //
        // **والحارسُ القديمُ كان حول `requestLocationUpdates`** —
        // **والسقوطُ يقع قبله**، **في `startForeground` نفسِها.**
        //
        // **ومن لا إذنَ له يقف ولا يُعاد تشغيلُه** (`START_NOT_STICKY`)
        // — **والجاهزيّةُ تقول لصاحبها ما ينقص** (`Readiness`).
        if (!LocationPermission.granted(this)) {
            Log.w(TAG, "إذنُ الموقع مسحوب — تقف الخدمةُ ولا تُعاد")
            if (intent?.getBooleanExtra(EXTRA_FROM_BOOT, false) == true) BootResume.askToOpen(this)
            stopSelf()
            return START_NOT_STICKY
        }
        // **ورفعُ الخدمة قد يُردّ من النظام** — **إذنٌ يُسحب في اللحظة
        // بين السؤال والرفع، أو حالٌ لا تسمح بخدمةٍ أماميّة.**
        if (!startForegroundSafely(off)) {
            // **وبعد الإقلاع يُطلب فتحُ التطبيق** — لا تقف الخدمةُ صامتةً والورديّةُ مفتوحة.
            if (intent?.getBooleanExtra(EXTRA_FROM_BOOT, false) == true) BootResume.askToOpen(this)
            stopSelf()
            return START_NOT_STICKY
        }
        request(seconds, live)
        heartbeat()
        appliedSec = seconds
        appliedLive = live
        appliedOff = off
        // **ويُعاد تشغيلها إن قتلها النظام تحت ضغط الذاكرة** — الوردية
        // مفتوحة ولا أحد يعرف أنّ الموقع انقطع.
        return START_STICKY
    }

    // ══════════════════════════════════════════════════════════════════
    // **نبضةُ الواقف — يبقى مرئيّاً وإن لم يتحرّك**
    // ══════════════════════════════════════════════════════════════════
    //
    // **تُعيد إرسالَ آخر موضعٍ معلومٍ حين يطول الصمت** — انظر شرحَ
    // `HEARTBEAT_SEC`. **ولا تنافس المرشِّح**: إن جاءت قراءةٌ جديدةٌ في
    // الأثناء رُفع `lastSentAt`، **فتجد النبضةُ الصمتَ قصيراً فتنام.**
    //
    // **وتُبنى على آخرِ موضعٍ قرأه التطبيقُ نفسُه** (`LastPoint`) ما دام عمرُه
    // دون خمس دقائق — **ولا تسأل النظامَ في كلّ نبضة**: سؤالٌ كلَّ دقيقتين
    // يوقظ عتادَ الموقع ويستنزف ما وُفِّر.
    //
    // **وما شاخ لا يُعاد** (٢٠٢٦-١٠-٠٢، `Heartbeat`): يُسأل النظامُ موضعاً
    // جديداً، **فإن لم يردّ فلا نبضة** — وموضعٌ ميّتٌ يُعاد «حديثاً» يُبنى عليه
    // توزيعٌ ويُقاس به قرب، **وهو أسوأُ من غيابه.**
    //
    // **وموعدُها من آخر إرسالٍ لا على دورةٍ مستقلّة** (A6.5، 0.9.0): كانت تنام دقيقتين على
    // دورتها هي، **فالواقفُ يُسمع بعد آخر نقطةٍ بدقيقتين إلى أربع**. والآن تصحو حين يتمّ الصمتُ
    // دقيقتين بالضبط (`Heartbeat.waitMs`)، **ولا تُعيد المحاولةَ قبل دقيقتين من محاولتها السابقة.**
    private fun heartbeat() {
        if (beating) return
        beating = true
        scope.launch {
            var lastBeat = SystemClock.elapsedRealtime()
            while (true) {
                val every = heartbeatEveryMs
                delay(Heartbeat.waitMs(SystemClock.elapsedRealtime(), lastSentAt, lastBeat, every))
                val now = SystemClock.elapsedRealtime()
                if (Heartbeat.due(now, lastSentAt, every)) lastBeat = now
                val p = LastPoint.value
                // **ولا يُغسَل موضعٌ مزيَّفٌ بنبضة** — **`isMocked` تُقرأ من
                // `Location` الذي سلّمه النظام**، وموضعٌ نبنيه نحن يُقرأ أصيلاً
                // دائماً. **فمن زيّف موضعَه يشيخ** — وذلك صوابٌ لا نقص.
                when (Heartbeat.decide(now, lastSentAt, p, every)) {
                    Heartbeat.Act.WAIT -> Unit
                    Heartbeat.Act.RESEND -> {
                        Log.i(TAG, "نبضةُ واقف: يُعاد آخرُ موضع")
                        send(
                            Location(HEARTBEAT_PROVIDER).apply {
                                latitude = p!!.lat
                                longitude = p.lng
                            },
                        )
                    }
                    // **وشاخ ما عندنا** — يُسأل النظامُ موضعاً جديداً (`Heartbeat`).
                    Heartbeat.Act.REFRESH -> {
                        val fix = freshFix()
                        if (fix != null) {
                            accept(fix)
                        } else {
                            // **ولا إرسال** — يشيخ عند الخادم فلا يُعرض عليه ما
                            // لا يستطيع بلوغَه، **ويُقال له في لوحته.**
                            Log.w(TAG, "لا موضعَ حيّ — لا نبضة، والإشارةُ مفقودة")
                            GpsSignal.dead()
                        }
                    }
                }
            }
        }
    }

    /**
     * **موضعٌ جديدٌ من النظام — أو لا شيء** (٢٠٢٦-١٠-٠٢).
     *
     * **ولا يُقبل ما شاخ**: `getCurrentLocation` قد تردّ مخبوءاً قديماً.
     */
    private suspend fun freshFix(): Location? = try {
        kotlinx.coroutines.withTimeoutOrNull(FIX_TIMEOUT_MS) {
            kotlinx.coroutines.suspendCancellableCoroutine<Location?> { cont ->
                val cancel = com.google.android.gms.tasks.CancellationTokenSource()
                cont.invokeOnCancellation { cancel.cancel() }
                client.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, cancel.token)
                    .addOnCompleteListener { t ->
                        if (cont.isActive) cont.resumeWith(Result.success(if (t.isSuccessful) t.result else null))
                    }
            }
        }?.takeIf {
            val ageMs = (SystemClock.elapsedRealtimeNanos() - it.elapsedRealtimeNanos) / 1_000_000
            ageMs in 0..Heartbeat.EVERY_MS
        }
    } catch (e: SecurityException) {
        Log.w(TAG, "إذن الموقع غير ممنوح", e)
        null
    }

    override fun onDestroy() {
        appliedSec = 0L
        appliedLive = false
        appliedOff = false
        client.removeLocationUpdates(callback)
        // **ولا إنذارَ «إشارةٌ مفقودة» لمن أغلق ورديّته.**
        GpsSignal.alive()
        scope.cancel()
        super.onDestroy()
    }

    private fun request(seconds: Long, live: Boolean) {
        val ms = seconds * 1000
        // ══════════════════════════════════════════════════════════════
        // **وأثناء التوصيل نبضةٌ حيّة** (قرارُ المالك ٢٠٢٦-١٠-١٩)
        // ══════════════════════════════════════════════════════════════
        //
        // **طلبٌ شغّال** (`HomeViewModel.syncService` يقولها صراحةً، `live`) — فلا
        // تجميعَ يؤخّر النقطة، **وحدُّ الحركة خمسةُ أمتار** فيُرى السائقُ يمشي لا يقفز،
        // **وحارسُ التردّد على قدر الفترة** لا خمسَ ثوانٍ ثابتة.
        //
        // ══════════════════════════════════════════════════════════════
        // **وبلا طلبٍ يرسل كلَّ فترةٍ ولو ما تحرّك** (قرارُ المالك ٢٠٢٦-١١-١٨)
        // ══════════════════════════════════════════════════════════════
        //
        // **كان حدُّ العشرين متراً يُسكت الواقف**، فلا يُسمع إلّا بنبضة الدقيقتين — فتقول الخريطةُ
        // «قبل دقيقتين» عن سائقٍ واقفٍ على الدوام. **والآن لا حدَّ مسافة**: نقطةٌ كلَّ فترة
        // (`drivers.location_ping_sec`، خمسٌ افتراضاً)، **ورجفتُه يمسكها فلترُ العرض في الخادم**
        // (`drivermonitor.NextShown`). **ولا أسرعَ من الفترة**: لا نقطةَ قبل الفترة من النظام
        // (`setMinUpdateIntervalMillis`)، والحارسُ لا ينزل تحت `MIN_SEND_GAP_MS` إلّا بهامش
        // `LIVE_SLACK_MS` — نقطةٌ تصل قبل موعدها بأجزاء ثانيةٍ لا تُرمى فتصير الفترةُ ضعفين.
        // **ولا تجميع**: نقطتان في دفعةٍ نداءان للشبكة بدل نداء.
        //
        // **ونسخةٌ قديمةٌ من `HomeViewModel` لا تقول `live`** غير ممكنة — الخدمةُ والشاشةُ في
        // التطبيق نفسِه. **والإقلاعُ اللاصقُ وبعد إقلاع الهاتف بلا طلب** حتّى تقرأ الشاشةُ الحال.
        minSendGapMs = if (live) {
            (ms - LIVE_SLACK_MS).coerceAtLeast(2_000L)
        } else {
            (ms - LIVE_SLACK_MS).coerceAtLeast(MIN_SEND_GAP_MS - LIVE_SLACK_MS)
        }
        heartbeatEveryMs = if (live) Heartbeat.EVERY_MS else (ms * 2).coerceIn(IDLE_HEARTBEAT_MIN_MS, Heartbeat.EVERY_MS)
        val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, ms)
            // **ولا نقطة أسرع من نصف الفترة أثناء التوصيل** — حتّى لا يغرقنا مزوّد
            // نشيط بتحديثات لا نحتاجها. **وبلا طلبٍ ولا أسرعَ من الفترة نفسِها.**
            .setMinUpdateIntervalMillis(if (live) ms / 2 else ms.coerceAtLeast(MIN_SEND_GAP_MS))
            // **وأثناء التوصيل خمسةُ أمتارٍ حدُّ الحركة** — **وبلا طلبٍ لا حدّ** (فوق).
            .setMinUpdateDistanceMeters(if (live) LIVE_MIN_MOVE_M else 0f)
            // **ولا تجميع** — نقطةٌ مجمّعةٌ تصل متأخّرةً لا تُري حركة، وبلا طلبٍ تضاعف النداءات.
            .setMaxUpdateDelayMillis(0L)
            .build()
        try {
            client.requestLocationUpdates(req, callback, mainLooper)
        } catch (e: SecurityException) {
            // **إذن سُحب والخدمة تعمل** — يقع حين يغيّره صاحبه من
            // الإعدادات والتطبيق مفتوح. **فتقف الخدمة ولا تسقط.**
            Log.w(TAG, "إذن الموقع غير ممنوح", e)
            stopSelf()
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يرسل النقطة — وما لم يصل يُحفظ**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والشبكة تنقطع في الشارع كثيرا**: نفق، أو حيّ بلا تغطية، أو حزمة
     * انتهت. **ومن رمى النقطة عند أوّل فشل** ترك في مسار السائق ثقبا:
     * **يظهر واقفا عشر دقائق وهو يسير.**
     *
     * **فالفاشلة تُكتب في الطابور**، وأوّل نقطة تنجح بعدها **تجرّ ما
     * تجمّع في دفعة واحدة** — لا نداء لكلّ نقطة.
     */
    private fun send(point: Location, earlier: List<TrackPoint> = emptyList()) {
        // ══════════════════════════════════════════════════════════════
        // **ومهلةٌ تُقاس بساعةٍ لا ترجع** (`AB-06`، ٢٠٢٦-٠٩-١٦)
        // ══════════════════════════════════════════════════════════════
        //
        // **وكانت `currentTimeMillis`** — **وهي ساعةُ الحائط: تقفز إلى
        // الوراء حين يضبطها النظامُ أو صاحبُ الجهاز.**
        //
        // **وقفزةٌ إلى الوراء تجعل الفرقَ سالباً** — **فيُقرأ «لم تمضِ
        // المهلة» أبداً**: **فيصمت موقعُ السائق حتّى تلحق الساعة**،
        // **ويظهر للمكتب واقفاً وهو يسير.**
        //
        // **و`elapsedRealtime` تُعدّ منذ الإقلاع ولا تُضبَط** — **وهي
        // ساعةُ المهل**، **وهي المستعملةُ في `Orderable` أصلاً.**
        val now = SystemClock.elapsedRealtime()
        // **وحارس ثانٍ على التردّد** — النظام قد يسلّم أسرع ممّا طُلب،
        // **ونداء لكلّ نقطة يستنزف البطارية والحزمة معا.**
        if (now - lastSentAt < minSendGapMs) {
            // **وما سبقها في الدفعة يُصَفّ وإن لم تُرسَل هي** — يخرج مع الإرسال التالي.
            if (earlier.isNotEmpty()) scope.launch { earlier.forEach { queue.add(it) } }
            return
        }
        lastSentAt = now

        val speed = if (point.hasSpeed()) point.speed.toDouble() else null
        val accuracy = if (point.hasAccuracy()) point.accuracy.toDouble() else null
        // ══════════════════════════════════════════════════════════════
        // **والاتّجاهُ يُسأل عنه ولا يُقرأ مباشرة**
        // ══════════════════════════════════════════════════════════════
        //
        // (المرحلة ١، تصحيحُ المالك ٢٠٢٦-٠٨-٢٠.)
        //
        // **و`getBearing()` تردّ صفراً حين لا اتّجاه** — لا فراغاً.
        // **فمن قرأها بلا `hasBearing()` وجّه كلَّ درّاجةٍ واقفةٍ
        // شمالاً**، وكتب في الأثر اتّجاهاً لم يقله الجهازُ قطّ.
        //
        // **ولا يُخترع اتّجاهٌ من نقطتين هنا** — ذاك حسابُ الملاحة
        // (`BearingTracker`)، **وأثرُ الورديّة يحفظ ما قاله الجهازُ
        // لا ما استنتجناه.**
        val bearing = if (point.hasBearing()) point.bearing.toDouble() else null
        val capturedAt = if (point.provider == HEARTBEAT_PROVIDER || point.time <= 0L) null else stamp(point.time)

        scope.launch {
            // **وما سبقها في الدفعة يُصَفّ أوّلاً** — فيخرج مع نداء الدفعة بعد نجاحها (A6.1).
            earlier.forEach { queue.add(it) }
            val api = Backend.of(applicationContext).driver
            try {
                api.sendLocation(
                    point.latitude, point.longitude, speed, accuracy, bearing,
                    point.isMocked(),
                    // **ووقتُ الالتقاط لا الوصول** (٢٠٢٦-١٠-٠٢) — والنبضةُ المعادةُ
                    // بلا وقت: تشهد «ما زلتُ هنا» والإشارةُ حيّةٌ منذ دقائق.
                    recordedAt = capturedAt,
                )
            } catch (e: CancellationException) {
                // **وتوقّفُ الخدمة ليس فشلَ إرسال** — ولو صُفَّت النقطةُ
                // هنا **لَتراكمت طوابيرُ ورديّةٍ انتهت**، وأُرسلت
                // مواضعُ سائقٍ أغلق ورديّتَه.
                throw e
            } catch (e: Exception) {
                // **ووقت الالتقاط يُحفظ معها** — لا وقت الإرسال:
                // **دفعة تصل بعد ربع ساعة بوقت الوصول** تجعل السائق
                // يقفز من حيّ إلى حيّ في لحظة.
                //
                // **والنبضةُ المعادةُ لا تُصَفّ** (0.9.0) — لا وقتَ التقاطٍ لها، وموضعُها
                // نفسُه في الطابور أو وصل من قبل.
                if (capturedAt != null) queue.add(trackPointOf(point))
                Log.w(TAG, "تعذّر الإرسال — حُفظت في الطابور", e)
                return@launch
            }

            // **والطابور يُفرَغ بعد أوّل نجاح** — الشبكة عادت.
            flush(api)
        }
    }

    /** **نداءُ دفعةٍ واحدٌ في كلّ مرّة** — نجاحان متقاربان لا يرسلان الجزءَ نفسَه مرّتين. */
    private val flushing = Mutex()

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يُفرغ الطابورَ أجزاءً — ويمحو ما أُرسل وحدَه** (A6.3، 0.9.0)
     * ══════════════════════════════════════════════════════════════════
     *
     * **كان يُرسَل الطابورُ كلُّه في نداءٍ واحدٍ ثمّ يُحذف الملفّ** — فما فوق المئتين يردّه المحرّك
     * فيعلق، **ونقطةٌ أُضيفت أثناء النداء تُمحى ولم تُرسَل** (`R19`). والآن: أقدمُ مئتين، ثمّ
     * **يُمحى ذلك الجزءُ وحدَه** حين ينجح، ثمّ التالي — **وأوّلُ فشلٍ يوقف ويترك الباقي كما هو.**
     */
    private suspend fun flush(api: DriverApi) {
        if (!flushing.tryLock()) return
        try {
            repeat(FLUSH_MAX_PARTS) {
                val part = queue.oldest(DriverApi.BATCH_MAX_POINTS)
                if (part.isEmpty()) return
                try {
                    api.sendBatch(part)
                } catch (e: CancellationException) {
                    // **والطابورُ يبقى كما هو** — يُرسَل حين تعود الورديّة.
                    throw e
                } catch (e: Exception) {
                    // **ولا تُمحى إن فشلت الدفعة** — تُترك للمحاولة
                    // التالية، **ومن مسحها قبل أن تصل فقدها.**
                    Log.w(TAG, "تعذّرت الدفعة", e)
                    return
                }
                // **وجزءٌ لم يُمحَ منه شيءٌ لا يُعاد في الحلقة نفسِها.**
                if (queue.remove(part) == 0) return
                Log.i(TAG, "أُرسلت دفعة: ${part.size} نقطة، وبقي ${queue.size()}")
            }
        } finally {
            flushing.unlock()
        }
    }

    /** **نقطةُ الطابور من قراءة النظام** — بوقت التقاطها واتّجاهها وتزييفها. */
    private fun trackPointOf(point: Location): TrackPoint = TrackPoint(
        lat = point.latitude,
        lng = point.longitude,
        at = stamp(point.time),
        speedMps = if (point.hasSpeed()) point.speed.toDouble() else null,
        accuracyM = if (point.hasAccuracy()) point.accuracy.toDouble() else null,
        // **ويُحفظ مع النقطة في الطابور** — **وإلّا
        // ضاع اتّجاهُ كلّ من انقطعت شبكتُه**، وهي
        // أطولُ المسارات وأغناها بالمنعطفات.
        bearingDeg = if (point.hasBearing()) point.bearing.toDouble() else null,
        mocked = point.isMocked(),
    )

    /** وقت الالتقاط بصيغة `ISO-8601` كما يقبلها المحرّك. */
    private fun stamp(millis: Long): String =
        java.time.Instant.ofEpochMilli(millis).toString()

    /** **إشعار الخدمة** — شرط النظام، ونافذة السائق على حاله. */
    /** **يرفع الخدمةَ أماميّةً** — **ويردّ `false` إن ردّها النظام.** */
    private fun startForegroundSafely(offShift: Boolean): Boolean {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    CHANNEL,
                    getString(R.string.loc_channel),
                    // **ومنخفض لا صامت**: يُرى في الشريط ولا يرنّ كلّ
                    // مرّة، **وإشعار يرنّ طوال الوردية يُطفئه صاحبه**
                    // فتُقتل الخدمة معه.
                    NotificationManager.IMPORTANCE_LOW,
                ),
            )
        }
        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val note: Notification = NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_orders)
            // **ودوامُه مسكّر بيقول الإشعار إنّو موقعه عم ينبعت** (٢٠٢٦-١١-٢٢) — مو «ورديّتك مفتوحة».
            .setContentTitle(getString(if (offShift) R.string.loc_off_title else R.string.loc_title))
            .setContentText(getString(if (offShift) R.string.loc_off_text else R.string.loc_text))
            .setContentIntent(open)
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
            // **وسُحب الإذنُ بين السؤال والرفع** — **أو منع النظامُ
            // خدمةً أماميّةً في هذه الحال.** **ولا يُترجَم ذلك سقوطا.**
            Log.w(TAG, "رُدّت الخدمةُ الأماميّة", e)
            false
        } catch (e: IllegalStateException) {
            // **ورفعُ خدمةٍ أماميّةٍ من الخلفيّة يُردّ كذلك** — `ForegroundServiceStartNotAllowedException`.
            Log.w(TAG, "لا يُسمح برفع الخدمة الآن", e)
            false
        }
    }

    companion object {
        private const val TAG = "RahalGo/loc"
        private const val CHANNEL = "rahalgo_shift"
        private const val NOTE_ID = 1001
        private const val EXTRA_PING_SEC = "ping_sec"
        private const val DEFAULT_PING_SEC = 20L
        private const val MIN_SEND_GAP_MS = 5_000L

        /** **أقصرُ نبضة واقفٍ بلا طلب** — ضعفُ أقصر فترة؛ النظامُ نفسُه يرسل كلَّ فترة (`request`). */
        private const val IDLE_HEARTBEAT_MIN_MS = 10_000L

        /** **أمعه طلبٌ شغّال؟** — من الشاشة صراحةً (٢٠٢٦-١١-١٨)، لا من طول الفترة. */
        private const val EXTRA_LIVE = "live"

        /** **دوامُه مسكّر؟** (٢٠٢٦-١١-٢٢) — بيضل يبعت أبطأ، والإشعارُ بنصّ تاني. */
        private const val EXTRA_OFF_SHIFT = "off_shift"

        /** **هامشُ حارس التردّد أثناء التوصيل** — نقطةٌ تصل قبل موعدها بقليلٍ لا تُرمى. */
        private const val LIVE_SLACK_MS = 800L

        /** **حدُّ الحركة أثناء التوصيل** — يُرى يمشي لا يقفز عشرين متراً. */
        private const val LIVE_MIN_MOVE_M = 5f

        // ══════════════════════════════════════════════════════════════
        // **ونبضةٌ تُبقي الواقفَ مرئيّاً** — `GAP-LOC-IDLE-01`
        // ══════════════════════════════════════════════════════════════
        //
        // **مرشِّحُ العشرين متراً يمنع الإرسالَ عن الواقف** («ومن وقف لا
        // يُرسل» أعلاه) — **ومحرّكُ التوزيع يشترط موضعاً أحدثَ من
        // `drivers.location_fresh_sec`** (٣٠٠ث افتراضاً).
        //
        // **فالسائقُ المتوقّفُ ينتظر عملاً كان يصير شائخاً بعد خمس دقائق،
        // فلا يُعرض عليه شيءٌ أبداً حتّى يتحرّك عشرين متراً.**
        //
        // **والعطبُ يضرب أسوأَ ما يمكن**: المنشغلُ يتحرّك فيبقى مرئيّاً،
        // **والفارغُ الواقفُ يختفي** — وهو بعينه من يُراد إعطاؤه طلباً.
        //
        // **وقِيس على الجهاز** (٢٠٢٦-٠٩-٢٩، والتطبيقُ مفتوحٌ على شاشة
        // الرحلة): العمرُ نما ٢٥٠ ⇐ ٣١٦ ⇐ ٣٣٦ ⇐ ٣٧٠ ⇐ ٤٠٥ ⇐ ٤٣٩،
        // **وطلبٌ في الطابور لم يُعرَض عليه دقائق.**
        //
        // (حكمُ المالك ٢٠٢٦-٠٩-٢٩: «لا يجوز ألّا يُعرض أيُّ طلبٍ على سائقٍ
        //  لأنّه متوقّف، هذا غلطٌ كبير».)
        //
        // (**وبلا طلبٍ صار بلا مرشِّح** منذ ٢٠٢٦-١١-١٨ — يرسل كلَّ فترة، والنبضةُ بضعفها احتياط؛
        //  وما يلي عن المرشِّح يبقى لأثناء التوصيل.)
        //
        // **والمرشِّحُ يبقى** — النبضةُ تُكمله لا تُلغيه: **الخادمُ يحتاج
        // «أعرف أين هو الآن» لا «تحرّك».** وإعادةُ آخرِ موضعٍ معلومٍ حزمةٌ
        // واحدةٌ كلَّ دقيقتين، **وحذفُ المرشِّح يجعل كلَّ ضجيجِ قمرٍ
        // صناعيٍّ نداءَ شبكة.**
        //
        // **ومئةٌ وعشرون دون الثلاثمئة بهامشٍ يحتمل نداءً يسقط ويُعاد** —
        // **ونبضةٌ تساوي الحدَّ تصل متأخّرةً ثانيةً فتسقط.**
        internal const val HEARTBEAT_SEC = 120L

        /** **كم يُنتظر النظامُ ليردّ موضعاً جديداً** — قبل أن تُعدّ الإشارةُ مفقودة. */
        private const val FIX_TIMEOUT_MS = 30_000L

        /** **أكثرُ أجزاء الدفعة في إفراغٍ واحد** — الطابورُ كلُّه بأقصى سقفه، وجزءٌ زائد. */
        private const val FLUSH_MAX_PARTS = PointQueue.MAX_POINTS / DriverApi.BATCH_MAX_POINTS + 1

        // **ومصدرُها يُسمّى** — من قرأ سجلّاً عرف أنّها إعادةٌ لا قراءة.
        private const val HEARTBEAT_PROVIDER = "rahalgo-heartbeat"

        /** **من الإقلاع؟** — فإن رُدّت الخدمةُ طُلب فتحُ التطبيق (`BootResume`). */
        private const val EXTRA_FROM_BOOT = "from_boot"

        /**
         * **الفترةُ النافذةُ الآن** — صفرٌ والخدمةُ واقفة.
         *
         * **وتقرؤها الشاشةُ قبل أن تُعيد التشغيل**: كلُّ نبضةِ تحديثٍ كانت
         * تنادي `start` فتُعيد رفعَ الخدمة وطلبَ الموقع وإن لم يتغيّر شيء.
         * **ومن الخدمة نفسِها لا من ذاكرة الشاشة** — فإن قتلها النظامُ صار
         * صفراً فتُرفع من جديد.
         */
        @Volatile
        var appliedSec: Long = 0L
            private set

        /** **أتعمل الآن بنبضة التوصيل؟** — مع `appliedSec`: تتبدّل فتُعاد الخدمة. */
        @Volatile
        var appliedLive: Boolean = false
            private set

        /** **أتعمل الآن ودوامُه مسكّر؟** — تتبدّل فتُعاد الخدمة (والإشعار). */
        @Volatile
        var appliedOff: Boolean = false
            private set

        /** **تبدأ مع الوردية** — والفترة من المحرّك. */
        fun start(
            context: Context,
            pingSec: Long,
            fromBoot: Boolean = false,
            live: Boolean = false,
            offShift: Boolean = false,
        ) {
            val intent = Intent(context, LocationService::class.java)
                .putExtra(EXTRA_PING_SEC, pingSec)
                .putExtra(EXTRA_FROM_BOOT, fromBoot)
                .putExtra(EXTRA_LIVE, live)
                .putExtra(EXTRA_OFF_SHIFT, offShift)
            context.startForegroundService(intent)
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, LocationService::class.java))
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقال الجهازُ هذا الموضعَ أم قاله تطبيقُ تزييف؟**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(قِيس ٢٠٢٦-٠٩-٠٢: لا سطرَ في المنصّة كلِّها يسأل هذا السؤال.)**
 *
 * **وتطبيقاتُ التزييف مجّانيّةٌ في المتجر**، ولا تحتاج جذراً ولا حيلة:
 * تُفتح خيارات المطوّر، ويُختار «تطبيق الموقع الوهميّ»، **فيصير
 * السائقُ عند بيت الزبون وهو في بيته.**
 *
 * **وأندرويد لا يخفي ذلك** — يقوله في كلّ قراءة. **فمن لم يسأل لم
 * يُخدع بذكاءٍ بل بسؤالٍ لم يسأله.**
 *
 * **والاسمُ تبدّل في أندرويد ١٢** (`isFromMockProvider` → `isMock`)،
 * **والقديمةُ باقيةٌ تعمل** — فتُستعمل ولا يُفرَّق، وإلّا لزم فرعان
 * لشيءٍ واحد.
 */
@Suppress("DEPRECATION")
internal fun android.location.Location.isMocked(): Boolean =
    if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.S) isMock else isFromMockProvider

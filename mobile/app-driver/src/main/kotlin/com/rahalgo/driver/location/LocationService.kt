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
import com.rahalgo.shared.model.TrackPoint
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

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
 * ٢ · **فلتر مسافة**: من وقف عند إشارة لا تُرسل نقطته عشرين مرّة.
 * ٣ · **دفعات**: يُسمح للنظام أن يجمّع تحديثات ويسلّمها معا
 *     (`setMaxUpdateDelayMillis`) — **فيوقظ الجهاز مرّة بدل خمس.**
 *
 * # ولا منطق عمل هنا
 *
 * الخدمة تقرأ الموقع وترسله. **متى يبدأ ومتى يقف يقرّره حال الوردية**
 * في `HomeViewModel`.
 */
class LocationService : Service() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client by lazy { LocationServices.getFusedLocationProviderClient(this) }
    private var lastSentAt = 0L
    private val queue by lazy { PointQueue(applicationContext) }

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            // **وآخر نقطة في الدفعة هي الحقّ** — ما قبلها ماضٍ.
            val point = result.lastLocation ?: return
            // **وأثر الوصول يُكتب** — «لا فشل» ليس دليل نجاح: قد لا
            // تصل نقطة أصلا، **والسكوت يُقرأ عملا وهو صمت.**
            Log.i(TAG, "نقطة: ${point.latitude}, ${point.longitude} دقّة ${point.accuracy}")
            // **والشاشة تقرؤه من هنا** — الخريطة تتحرّك مع صاحبها.
            LastPoint.set(point.latitude, point.longitude)
            send(point)
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val seconds = intent?.getLongExtra(EXTRA_PING_SEC, 0L)?.takeIf { it > 0 } ?: DEFAULT_PING_SEC
        startForegroundSafely()
        request(seconds)
        // **ويُعاد تشغيلها إن قتلها النظام تحت ضغط الذاكرة** — الوردية
        // مفتوحة ولا أحد يعرف أنّ الموقع انقطع.
        return START_STICKY
    }

    override fun onDestroy() {
        client.removeLocationUpdates(callback)
        scope.cancel()
        super.onDestroy()
    }

    private fun request(seconds: Long) {
        val ms = seconds * 1000
        val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, ms)
            // **ولا نقطة أسرع من نصف الفترة** — حتّى لا يغرقنا مزوّد
            // نشيط بتحديثات لا نحتاجها.
            .setMinUpdateIntervalMillis(ms / 2)
            // **ومن وقف لا يُرسل**: عشرون مترا حدّ الحركة الحقيقيّة،
            // **ودونها ضجيج قمر صناعيّ لا تنقّل.**
            .setMinUpdateDistanceMeters(MIN_MOVE_M)
            // **ويُسمح للنظام أن يجمّع** — يوقظ الجهاز مرّة بدل مرّات.
            .setMaxUpdateDelayMillis(ms * 2)
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
    private fun send(point: Location) {
        val now = System.currentTimeMillis()
        // **وحارس ثانٍ على التردّد** — النظام قد يسلّم أسرع ممّا طُلب،
        // **ونداء لكلّ نقطة يستنزف البطارية والحزمة معا.**
        if (now - lastSentAt < MIN_SEND_GAP_MS) return
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

        scope.launch {
            val api = Backend.of(applicationContext).driver
            try {
                api.sendLocation(point.latitude, point.longitude, speed, accuracy, bearing)
            } catch (e: CancellationException) {
                // **وتوقّفُ الخدمة ليس فشلَ إرسال** — ولو صُفَّت النقطةُ
                // هنا **لَتراكمت طوابيرُ ورديّةٍ انتهت**، وأُرسلت
                // مواضعُ سائقٍ أغلق ورديّتَه.
                throw e
            } catch (e: Exception) {
                // **ووقت الالتقاط يُحفظ معها** — لا وقت الإرسال:
                // **دفعة تصل بعد ربع ساعة بوقت الوصول** تجعل السائق
                // يقفز من حيّ إلى حيّ في لحظة.
                queue.add(
                    TrackPoint(
                        lat = point.latitude,
                        lng = point.longitude,
                        at = stamp(point.time),
                        speedMps = speed,
                        accuracyM = accuracy,
                        // **ويُحفظ مع النقطة في الطابور** — **وإلّا
                        // ضاع اتّجاهُ كلّ من انقطعت شبكتُه**، وهي
                        // أطولُ المسارات وأغناها بالمنعطفات.
                        bearingDeg = bearing,
                    ),
                )
                Log.w(TAG, "تعذّر الإرسال — حُفظت في الطابور", e)
                return@launch
            }

            // **والطابور يُفرَغ بعد أوّل نجاح** — الشبكة عادت.
            if (!queue.isEmpty()) {
                val waiting = queue.all()
                try {
                    api.sendBatch(waiting)
                    queue.clear()
                    Log.i(TAG, "أُرسلت دفعة: ${waiting.size} نقطة")
                } catch (e: CancellationException) {
                    // **والطابورُ يبقى كما هو** — يُرسَل حين تعود الورديّة.
                    throw e
                } catch (e: Exception) {
                    // **ولا تُمحى إن فشلت الدفعة** — تُترك للمحاولة
                    // التالية، **ومن مسحها قبل أن تصل فقدها.**
                    Log.w(TAG, "تعذّرت الدفعة", e)
                }
            }
        }
    }

    /** وقت الالتقاط بصيغة `ISO-8601` كما يقبلها المحرّك. */
    private fun stamp(millis: Long): String =
        java.time.Instant.ofEpochMilli(millis).toString()

    /** **إشعار الخدمة** — شرط النظام، ونافذة السائق على حاله. */
    private fun startForegroundSafely() {
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
            .setContentTitle(getString(R.string.loc_title))
            .setContentText(getString(R.string.loc_text))
            .setContentIntent(open)
            .setOngoing(true)
            .build()

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(NOTE_ID, note, ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION)
        } else {
            startForeground(NOTE_ID, note)
        }
    }

    companion object {
        private const val TAG = "RahalGo/loc"
        private const val CHANNEL = "rahalgo_shift"
        private const val NOTE_ID = 1001
        private const val EXTRA_PING_SEC = "ping_sec"
        private const val DEFAULT_PING_SEC = 20L
        private const val MIN_MOVE_M = 20f
        private const val MIN_SEND_GAP_MS = 5_000L

        /** **تبدأ مع الوردية** — والفترة من المحرّك. */
        fun start(context: Context, pingSec: Long) {
            val intent = Intent(context, LocationService::class.java)
                .putExtra(EXTRA_PING_SEC, pingSec)
            context.startForegroundService(intent)
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, LocationService::class.java))
        }
    }
}

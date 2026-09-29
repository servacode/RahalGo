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
import com.rahalgo.shared.model.TrackPoint
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
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
    // **ونبضةٌ واحدةٌ لا اثنتان** — `onStartCommand` يُنادى مرّاتٍ
    // (إعادةُ تشغيلٍ بعد قتلٍ، أو تبديلُ تردّد)، **وحلقةٌ لكلّ نداءٍ
    // تُضاعف النداءات بلا أن يظهر ذلك في شيء.**
    private var beating = false
    private val queue by lazy { PointQueue(applicationContext) }

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            // **وآخر نقطة في الدفعة هي الحقّ** — ما قبلها ماضٍ.
            val point = result.lastLocation ?: return
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
            LastPoint.set(
                point.latitude,
                point.longitude,
                mocked = point.isMocked(),
                speedMps = if (point.hasSpeed()) point.speed else null,
                atMs = android.os.SystemClock.elapsedRealtime(),
            )
            send(point)
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val seconds = intent?.getLongExtra(EXTRA_PING_SEC, 0L)?.takeIf { it > 0 } ?: DEFAULT_PING_SEC
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
            stopSelf()
            return START_NOT_STICKY
        }
        // **ورفعُ الخدمة قد يُردّ من النظام** — **إذنٌ يُسحب في اللحظة
        // بين السؤال والرفع، أو حالٌ لا تسمح بخدمةٍ أماميّة.**
        if (!startForegroundSafely()) {
            stopSelf()
            return START_NOT_STICKY
        }
        request(seconds)
        heartbeat()
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
    // **وتُبنى على آخرِ موضعٍ قرأه التطبيقُ نفسُه** (`LastPoint`) — **ولا
    // تسأل النظامَ موضعاً جديداً**: سؤالٌ كلَّ دقيقتين يوقظ عتادَ الموقع
    // ويستنزف ما وُفِّر.
    //
    // **وإن لم يُقرأ موضعٌ بعدُ لم تخترع واحداً** — **وموضعٌ مخترَعٌ أسوأُ
    // من غيابه**: يُبنى عليه توزيعٌ ويُقاس به قرب.
    private fun heartbeat() {
        if (beating) return
        beating = true
        scope.launch {
            while (true) {
                delay(HEARTBEAT_SEC * 1000)
                val silence = SystemClock.elapsedRealtime() - lastSentAt
                if (silence < HEARTBEAT_SEC * 1000) continue
                val p = LastPoint.value ?: continue
                // **ولا يُغسَل موضعٌ مزيَّفٌ بنبضة.**
                //
                // **`isMocked` تُقرأ من `Location` الذي سلّمه النظام**،
                // **وموضعٌ نبنيه نحن يُقرأ أصيلاً دائماً** — فلو نبضنا
                // بمزيَّفٍ لخرج من بابنا بلا وسمه، **ويُبنى عليه إثباتُ
                // تسليم.**
                //
                // **فمن زيّف موضعَه يشيخ** — وذلك صوابٌ لا نقص.
                if (p.mocked) continue
                Log.i(TAG, "نبضةُ واقف: صمتٌ ${silence / 1000}ث — يُعاد آخرُ موضع")
                send(
                    Location(HEARTBEAT_PROVIDER).apply {
                        latitude = p.lat
                        longitude = p.lng
                    },
                )
            }
        }
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
                api.sendLocation(
                    point.latitude, point.longitude, speed, accuracy, bearing,
                    point.isMocked(),
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
                        mocked = point.isMocked(),
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
    /** **يرفع الخدمةَ أماميّةً** — **ويردّ `false` إن ردّها النظام.** */
    private fun startForegroundSafely(): Boolean {
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
        private const val MIN_MOVE_M = 20f
        private const val MIN_SEND_GAP_MS = 5_000L

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
        // **والمرشِّحُ يبقى** — النبضةُ تُكمله لا تُلغيه: **الخادمُ يحتاج
        // «أعرف أين هو الآن» لا «تحرّك».** وإعادةُ آخرِ موضعٍ معلومٍ حزمةٌ
        // واحدةٌ كلَّ دقيقتين، **وحذفُ المرشِّح يجعل كلَّ ضجيجِ قمرٍ
        // صناعيٍّ نداءَ شبكة.**
        //
        // **ومئةٌ وعشرون دون الثلاثمئة بهامشٍ يحتمل نداءً يسقط ويُعاد** —
        // **ونبضةٌ تساوي الحدَّ تصل متأخّرةً ثانيةً فتسقط.**
        private const val HEARTBEAT_SEC = 120L

        // **ومصدرُها يُسمّى** — من قرأ سجلّاً عرف أنّها إعادةٌ لا قراءة.
        private const val HEARTBEAT_PROVIDER = "rahalgo-heartbeat"

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

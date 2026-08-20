package com.rahalgo.navigation

import android.annotation.SuppressLint
import android.content.Context
import android.location.Location
import android.util.Log
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّكُ موقع الملاحة — سريعٌ محلّيّاً، صامتٌ نحوَ الخادم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «**رفع تردّد GPS محليّاً لا يعني رفع تردّد
 *  الإرسال إلى الخادم**… لا أريد طلب شبكة كلّ ثانية».)
 *
 * # وضعان لا وضعٌ واحد
 *
 *     التتبّعُ العاديّ    خدمةُ الورديّة — كلَّ عشرين ثانيةً وفلترُ
 *                        عشرين مترا، **وترسل إلى الخادم.**
 *                        (`LocationService` — لا تُمسّ.)
 *
 *     الملاحةُ النشطة    هذا المحرّك — قراءةٌ في الثانية بلا فلترِ
 *                        مسافة، **ولا يرسل شيئاً إلى أحد.**
 *
 * **وهما مجريان مستقلّان يعملان معاً**: خدمةُ الورديّة تُبقي الخادمَ
 * يعرف أين السائق، **وهذا يُبقي الشاشةَ تتحرّك.**
 *
 * # ولماذا لا يُرفع تردّدُ الخدمة بدل مجرًى ثانٍ
 *
 * **لأنّ الخدمةَ ترسل ما تقرأ.** ورفعُ تردّدها يعني ألفَ نداءِ شبكةٍ
 * في الرحلة الواحدة — **حزمةُ السائق وبطّاريّتُه وخادمُنا.**
 *
 * **والفصلُ يجعل كلَّ واحدٍ يُضبط لغايته**: التردّدُ للعين، والإرسالُ
 * للخادم. **وهو نصُّ أمر المالك.**
 *
 * # ولا يكتب في `LastPoint`
 *
 * **`LastPoint` حاملُ موضعِ الورديّة** — يكتب فيه أربعةٌ لا خامسَ لهم،
 * **وحارسٌ يمنع الخامس** (`check-one-map-way.mjs`). **وهذا المحرّك
 * مجرًى للشاشة لا مصدرَ موضعٍ للمنصّة.**
 *
 * # وفلترُ المسافة يُلغى هنا وحدَه
 *
 * (أمرُ المالك: «لا تفرض قاعدة ٢٠ متراً في الحلقة المحلّيّة للملاحة».)
 *
 * **وعشرون متراً في خدمة الورديّة توفيرُ بطّاريّةٍ وحزمة** — **وفي
 * الملاحة تعني أيقونةً تقف ثمّ تقفز عشرين متراً دفعةً واحدة.**
 */
class LocationEngine(private val context: Context) {

    /** **يُنادى بكلّ قراءةٍ خامّة** — والترشيحُ بعده في `NavPipeline`. */
    var onFix: ((NavFix) -> Unit)? = null

    private val client by lazy {
        LocationServices.getFusedLocationProviderClient(context.applicationContext)
    }

    private var active = false

    /** **أتعمل الملاحةُ الآن؟** — تُقرأ في التقرير والتشخيص. */
    val isActive: Boolean get() = active

    /** **كم قراءةً وصلت في هذه الجلسة** — للقياس لا للمنطق. */
    var samples: Int = 0
        private set

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            // **وكلُّ قراءةٍ في الدفعة تُمرَّر** — لا آخرُها وحدَها:
            // **الملاحةُ تريد المسارَ لا الطرف**، وحذفُ ما بينهما يجعل
            // الحركةَ تقفز.
            for (loc in result.locations) {
                samples++
                onFix?.invoke(toFix(loc))
            }
        }
    }

    /**
     * **يبدأ الملاحة.**
     *
     * @param intervalMs الفاصلُ المطلوب — **والنظامُ قد يعطي أبطأ.**
     */
    @SuppressLint("MissingPermission")
    fun start(intervalMs: Long = NAV_INTERVAL_MS): Boolean {
        if (active) return true
        val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, intervalMs)
            // **ولا حدَّ أدنى للمسافة** — انظر أعلاه.
            .setMinUpdateDistanceMeters(0f)
            // **ويُقبل الأسرع** — من أعطاه النظامُ قراءةً مبكّرةً أخذها.
            .setMinUpdateIntervalMillis(intervalMs / 2)
            // **ولا تجميع** — التجميعُ يوقظ الجهازَ مرّةً بدل خمسٍ
            // وهو صوابٌ في الورديّة، **وفي الملاحة يعني خمسَ قراءاتٍ
            // تصل معاً فتقفز الأيقونةُ خمسَ مرّاتٍ في إطار.**
            .setMaxUpdateDelayMillis(0)
            .setWaitForAccurateLocation(false)
            .build()
        return try {
            client.requestLocationUpdates(req, callback, context.mainLooper)
            active = true
            samples = 0
            Log.i(TAG, "بدأت الملاحة — فاصل ${intervalMs}ملّي")
            true
        } catch (e: SecurityException) {
            // **إذنٌ سُحب والملاحةُ تبدأ** — تقف ولا تُسقط الشاشة.
            Log.w(TAG, "إذنُ الموقع غيرُ ممنوح", e)
            false
        }
    }

    /**
     * **يوقف الملاحة ويعيد الجهازَ إلى وضعه.**
     *
     * (معيارُ المالك: «إنهاء جلسة الملاحة يعيد Location Mode للوضع
     *  الطبيعيّ».)
     *
     * **وخدمةُ الورديّة لا تُمسّ** — هي مجرًى آخرُ يعمل وحدَه.
     */
    fun stop() {
        if (!active) return
        client.removeLocationUpdates(callback)
        active = false
        Log.i(TAG, "انتهت الملاحة — $samples قراءة")
    }

    companion object {
        private const val TAG = "RahalGo/nav"

        /**
         * **قراءةٌ في الثانية** — هدفُ الملاحة.
         *
         * **والنظامُ يعطي أبطأَ حين تسوء السماءُ أو يُقيَّد التطبيق**،
         * ولذلك يُقاس التردّدُ الفعليُّ ولا يُفترض. (البند ٥ من
         * التقرير.)
         */
        const val NAV_INTERVAL_MS = 1_000L

        /**
         * **يحوّل قراءةَ النظام إلى قراءتنا.**
         *
         * **و`hasX()` تُسأل قبل القراءة**: `bearing` تردّ صفراً حين
         * تغيب، **ومن قرأ الصفرَ وجّه الدرّاجةَ شمالاً وهي تسير
         * جنوبا.**
         *
         * **و`elapsedRealtimeNanos` لا `time`**: الثاني ساعةُ الجدار
         * **تقفز حين يضبطها النظامُ من الشبكة**، فينقلب ترتيبُ
         * القراءات. لكنّ الأوّل يُقاس من الإقلاع، **فيُحوَّل إلى مقياسٍ
         * موحّدٍ بجمعه على لحظةِ الإقلاع.**
         */
        fun toFix(loc: Location): NavFix = NavFix(
            lat = loc.latitude,
            lng = loc.longitude,
            accuracyM = if (loc.hasAccuracy()) loc.accuracy else Float.MAX_VALUE,
            speedMps = if (loc.hasSpeed()) loc.speed else null,
            bearingDeg = if (loc.hasBearing()) loc.bearing else null,
            atMs = monotonicMs(loc),
        )

        private fun monotonicMs(loc: Location): Long {
            val nanos = loc.elapsedRealtimeNanos
            return if (nanos > 0) nanos / 1_000_000 else loc.time
        }
    }
}

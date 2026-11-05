package com.rahalgo.driver.location

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **عقودُ طبقة الموقع — `P-8`.**
 *
 * # لماذا إثباتٌ بنيويٌّ لا اختبارُ جهاز
 *
 * **`LocationService` يعتمد `android.content.Context` و`android.util.Log`**
 * — **فلا يعمل في آلة Java وحدَها**، وتشغيلُه يحتاج جهازاً أو Robolectric.
 * (**و`PointQueue` صار يُبنى من ملفٍّ** منذ 0.9.0 — فيُختبر سلوكُه في `PointQueueTest`.)
 *
 * **ولا جهازَ متّصلٌ اليوم.**
 *
 * **فما يُثبَت هنا هو بنيةُ المسار من مصدره** — كما أُثبت ترتيبُ `D2` في
 * `P-6` بقراءة المصدر لا بتشغيله. **وهذا دليلٌ حقيقيٌّ لا ادّعاء**: يسقط
 * يومَ تتبدّل الشيفرةُ، **ويقول أين بالضبط.**
 *
 * **ولا يُسمّى إثباتاً على جهاز** — التصنيفُ في
 * `ANDROID_TEST_MATRIX.json`: `STRUCTURAL`، **لا `DEVICE`.**
 */
class LocationContractTest {

    /** توكيدٌ برسالةٍ أوّلاً — كما في JUnit4. */
    private fun assertTrue2(cond: Boolean, msg: String) = assertTrue(msg, cond)


    private fun source(rel: String): String {
        // **والمسارُ نسبيٌّ إلى وحدة القياس** — `app-driver`.
        val f = File("src/main/kotlin/com/rahalgo/driver/$rel")
        assertTrue2(f.exists(), "المصدرُ غيرُ موجود: ${f.absolutePath}")
        return f.readText()
    }

    /**
     * **`R19` — نافذةُ الطابور بين القراءة والمسح — أُصلح في 0.9.0** (A6.3، ٢٠٢٦-١١-٠٥).
     *
     * **كان**: `queue.all()` ثمّ `api.sendBatch(waiting)` ثمّ `queue.clear()` — **والمسحُ يحذف
     * الملفَّ كلَّه**، فنقطةٌ أُضيفت أثناء نداء الشبكة تُمحى ولم تُرسَل.
     *
     * **والآن**:
     * ```
     * val part = queue.oldest(...)   // ١ أقدمُ جزءٍ لا يزيد على مئتين
     * api.sendBatch(part)            // ٢ نداءُ شبكة
     * queue.remove(part)             // ٣ يمحو ذلك الجزءَ وحدَه
     * ```
     * **والسلوكُ نفسُه مُثبَتٌ على ملفٍّ حقيقيّ في `PointQueueTest`.**
     */
    @Test
    fun `R19 — يُمحى من الطابور ما أُرسل وحده`() {
        val svc = source("location/LocationService.kt")
        val queue = source("location/PointQueue.kt")

        val readAt = svc.indexOf("queue.oldest(")
        val sendAt = svc.indexOf("api.sendBatch(part)")
        val removeAt = svc.indexOf("queue.remove(part)")
        assertTrue2(readAt >= 0 && sendAt >= 0 && removeAt >= 0, "المسارُ تبدّل — يُعاد القياس")
        assertTrue2(readAt < sendAt && sendAt < removeAt, "الترتيبُ ليس قراءةً ثمّ إرسالاً ثمّ محواً للمُرسَل")

        // **ولا مسحَ للملفّ كلِّه بعد الإرسال.**
        assertTrue2(!svc.contains("queue.clear()"), "عاد مسحُ الطابور كلِّه — R19 رجع")
        assertTrue2(!queue.contains("fun clear()"), "عاد مسحُ الطابور كلِّه — R19 رجع")
        assertTrue2(queue.contains("fun remove(sent: List<TrackPoint>)"), "المحوُ لم يعد للمُرسَل وحدَه")

        println("R19 LOCATION QUEUE RACE = FIXED (STRUCTURAL + UNIT PointQueueTest)")
        println("  oldest() ثمّ sendBatch() ثمّ remove(part) — يُمحى ما أُرسل وحدَه")
        println("  DEVICE PROOF = NOT EXECUTED — لا جهازَ متّصل")
    }

    /**
     * **`R20` — نجاحُ الدفعة يمسح كلَّ شيء.**
     *
     * **`sendBatch` تنجح إن ردَّ الخادمُ `2xx`** — **ولا تُقرأ حمولةُ
     * الردّ** لتُعرَف نقطةٌ رُفضت (زمنٌ مستقبليّ مثلاً).
     * **فالمسحُ يقع على المقبولِ والمرفوضِ سواء.**
     */
    @Test
    fun `R20 — لا تُقرأ حمولةُ الردّ لتمييز المرفوض`() {
        val svc = source("location/LocationService.kt")
        val i = svc.indexOf("api.sendBatch(part)")
        assertTrue2(i >= 0, "المسارُ تبدّل")
        val after = svc.substring(i, minOf(i + 400, svc.length))

        // **ولا تُقرأ نتيجةٌ ولا يُصفّى طابور.**
        val readsResult = after.contains("val res") || after.contains("rejected") ||
            after.contains("accepted") || after.contains(".body")
        assertTrue2(!readsResult, "ثمّةَ قراءةُ نتيجةٍ — R20 قد يكون أُصلح")

        println("R20 LOCATION BATCH REJECTION = RISK CONFIRMED (STRUCTURAL)")
        println("  الدفعةُ تُمسح على نجاحِ النداء لا على قبولِ كلّ نقطة")
        println("  DEVICE PROOF = NOT EXECUTED — لا جهازَ متّصل")
    }

    /**
     * **`D16` — الطابورُ بلا صاحب.**
     *
     * **`points.jsonl` ملفٌّ واحدٌ للتطبيق** — **لا يحمل معرّفَ سائق.**
     * **وتسجيلُ خروجٍ لا يمسحه** (لا نداءَ لـ`clear()` في مسار الخروج)،
     * **فنقاطُ الأوّل تُرفَع باسم الثاني.**
     */
    @Test
    fun `D16 — لا مِلكيّةَ في طابور النقاط ولا يُمسح عند الخروج`() {
        val queue = source("location/PointQueue.kt")
        assertTrue2(queue.contains("points.jsonl"), "اسمُ الملفّ تبدّل — يُعاد القياس")

        val hasOwner = queue.contains("driverId") || queue.contains("userId") ||
            queue.contains("ownerId")
        assertTrue2(!hasOwner, "ثمّةَ مِلكيّةٌ في الطابور — D16 قد يكون أُصلح")

        // **ومسارُ الخروج لا يمسح الطابور** — يُقاس من كلّ مصادر التطبيق.
        val cleared = File("src/main/kotlin").walkTopDown()
            .filter { it.isFile && it.extension == "kt" }
            .any { f ->
                val s = f.readText()
                (s.contains("logout", ignoreCase = true) || s.contains("signOut", ignoreCase = true)) &&
                    s.contains("PointQueue")
            }
        assertTrue2(!cleared, "مسارُ الخروج يمسّ الطابور — D16 قد يكون أُصلح")

        println("D16 LOCATION QUEUE OWNER = EXPECTED FAIL (STRUCTURAL)")
        println("  points.jsonl ملفٌّ واحدٌ بلا معرّفِ سائق · ولا مسارَ خروجٍ يمسحه")
        println("  DEVICE PROOF = NOT EXECUTED — لا جهازَ متّصل")
    }

    /**
     * **`D18` — الفاصلُ يعود إلى الافتراضيّ بعد الإقلاع اللاصق.**
     *
     * ```
     * val seconds = intent?.getLongExtra(EXTRA_PING_SEC, 0L)?.takeIf { it > 0 } ?: DEFAULT_PING_SEC
     * return START_STICKY
     * ```
     *
     * **وأندرويد يُسلّم `intent = null` عند الإقلاع اللاصق** —
     * **فالفاصلُ المضبوطُ يضيع ويعود الافتراضيّ.**
     */
    @Test
    fun `D18 — الإقلاعُ اللاصق يفقد الفاصلَ المضبوط`() {
        val svc = source("location/LocationService.kt")
        assertTrue2(svc.contains("START_STICKY"), "الخدمةُ لم تعد لاصقةً — يُعاد القياس")

        // **والمرساةُ على التصريح لا على ذِكرِ الاسم** — **تعليقٌ يسمّي
        // الدالّةَ قبل تصريحِها يُزحلق النافذةَ فيسقط الاختبارُ وسلوكُ
        // الخدمة سليم.** (وقع ٢٠٢٦-٠٩-٣٠: تعليقُ «نبضةٌ واحدةٌ لا اثنتان»
        // ذكرها في السطر ٧٢ والتصريحُ في ١٠٩.)
        val i = svc.indexOf("override fun onStartCommand")
        assertTrue2(i >= 0, "تصريحُ onStartCommand غيرُ موجود")
        val body = svc.substring(i, minOf(i + 600, svc.length))

        assertTrue2(body.contains("intent?."), "الفاصلُ لا يُقرأ من intent — يُعاد القياس")
        assertTrue2(body.contains("DEFAULT_PING_SEC"), "لا افتراضيَّ — يُعاد القياس")

        // **ولا يُحفَظ الفاصلُ خارجَ النيّة** — لا تفضيلاتٍ ولا حالٍ ثابتة.
        val persisted = svc.contains("SharedPreferences") || svc.contains("DataStore") ||
            svc.contains("prefs")
        assertTrue2(!persisted, "الفاصلُ يُحفَظ — D18 قد يكون أُصلح")

        println("D18 STICKY RESTART INTERVAL = EXPECTED FAIL (STRUCTURAL)")
        println("  الفاصلُ يُقرأ من intent وحدَه · وأندرويد يُسلّم null عند الإقلاع اللاصق")
        println("  DEVICE PROOF = NOT EXECUTED — لا جهازَ متّصل")
    }

    /**
     * **`R17` — الإقلاعُ اللاصق لا يُعيد التحقّق.**
     *
     * `onStartCommand` تُقلع الواجهةَ وتطلب الموقعَ **ولا تسأل عن
     * الورديّة ولا الجلسة ولا حالِ السائق.** **فخدمةٌ تُقلع بعد إغلاقِ
     * ورديّةٍ تجمع مواقعَ لمن ليس على الدوام.**
     */
    @Test
    fun `R17 — لا إعادةَ تحقّقٍ بعد الإقلاع اللاصق`() {
        val svc = source("location/LocationService.kt")
        val i = svc.indexOf("onStartCommand")
        val body = svc.substring(i, minOf(i + 600, svc.length))

        val revalidates = body.contains("shift", ignoreCase = true) ||
            body.contains("session", ignoreCase = true) ||
            body.contains("token", ignoreCase = true) ||
            body.contains("status", ignoreCase = true)
        assertTrue2(!revalidates, "ثمّةَ إعادةُ تحقّق — R17 قد يكون أُصلح")

        println("R17 STICKY REVALIDATION = RISK CONFIRMED (STRUCTURAL)")
        println("  onStartCommand تُقلع وتطلب الموقعَ بلا سؤالٍ عن ورديّةٍ أو جلسةٍ أو حال")
        println("  DEVICE PROOF = NOT EXECUTED — لا جهازَ متّصل")
    }
}

package com.rahalgo.driver.location

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **عقودُ طبقة الموقع — `P-8`.**
 *
 * # لماذا إثباتٌ بنيويٌّ لا اختبارُ جهاز
 *
 * **`PointQueue` و`LocationService` يعتمدان `android.content.Context`
 * و`android.util.Log`** — **فلا يعملان في آلة Java وحدَها**، وتشغيلُهما
 * يحتاج جهازاً أو Robolectric.
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
     * **`R19` — نافذةُ الطابور بين القراءة والمسح.**
     *
     * الترتيبُ المقيسُ في `LocationService`:
     * ```
     * val waiting = queue.all()      // ١ يقرأ ما في الملفّ
     * api.sendBatch(waiting)         // ٢ نداءُ شبكةٍ يستغرق زمناً
     * queue.clear()                  // ٣ يحذف الملفَّ كلَّه
     * ```
     *
     * **و`clear()` تحذف الملفَّ لا الصفوفَ المقروءة** — **فنقطةٌ أُضيفت
     * أثناء نداء الشبكة تُمحى ولم تُرسَل.**
     */
    @Test
    fun `R19 — الطابور يُمحى كاملاً بعد إرسال ما قُرئ`() {
        val svc = source("location/LocationService.kt")
        val queue = source("location/PointQueue.kt")

        val readAt = svc.indexOf("queue.all()")
        val sendAt = svc.indexOf("api.sendBatch(waiting)")
        val clearAt = svc.indexOf("queue.clear()")
        assertTrue2(readAt >= 0 && sendAt >= 0 && clearAt >= 0, "المسارُ تبدّل — يُعاد القياس")
        assertTrue2(readAt < sendAt && sendAt < clearAt, "الترتيبُ ليس قراءةً ثمّ إرسالاً ثمّ مسحاً")

        // **والمسحُ يحذف الملفَّ** — لا يحذف ما أُرسل وحدَه.
        val clearsWholeFile = queue.contains("fun clear()") && queue.contains("file.delete()")
        assertTrue2(clearsWholeFile, "clear() لم تعد تحذف الملفَّ — يُعاد القياس")

        // **ولا مِلكيّةَ في الطابور** — لا `id` ولا مؤشّرَ إرسال.
        val hasCursor = queue.contains("removeFirst") || queue.contains("dropWhile") ||
            queue.contains("sentUpTo") || queue.contains("cursor")
        assertTrue2(!hasCursor, "ثمّةَ مؤشّرُ إرسالٍ — R19 قد يكون أُصلح، يُعاد القياس")

        println("R19 LOCATION QUEUE RACE = RISK CONFIRMED (STRUCTURAL)")
        println("  all() ثمّ sendBatch() ثمّ clear() — والملفُّ يُحذف كاملاً")
        println("  فنقطةٌ تُضاف أثناء نداء الشبكة تُمحى ولم تُرسَل")
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
        val i = svc.indexOf("api.sendBatch(waiting)")
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

        val i = svc.indexOf("onStartCommand")
        assertTrue2(i >= 0, "onStartCommand غيرُ موجودة")
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

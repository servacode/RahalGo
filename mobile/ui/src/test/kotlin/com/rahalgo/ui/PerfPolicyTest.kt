package com.rahalgo.ui

import java.io.File
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سياسةُ الانتظار والشبكة والبطّاريّة** (`PF`·`CP`·`PB`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا قياسٌ ثمّ حارس
 *
 * **ولا يُحسَّن ما لم يُقَس** — **والمقيسُ هنا ثلاثةٌ**:
 *
 *	دوّارةٌ بلا نهايةٍ في `LoadState`  ⇒ **حالُ «لا تحميلَ ولا خطأ» تدور أبداً**
 *	موجةُ نداءاتٍ لكلّ حدثٍ حيّ         ⇒ **١٧ سامعاً ونداءاتُ الغلاف معهم**
 *	مهلُ تسعين ثانيةً لخادمٍ مضى        ⇒ **`Render` ينام، و`CX33` لا ينام**
 *
 * **وما يُحرَس هنا عقودٌ لا أرقامُ أجهزة** — **وزمنُ الإقلاع يُقاس على
 * جهازٍ لا في فحصِ وحدة.**
 */
@OptIn(ExperimentalCoroutinesApi::class)
class PerfPolicyTest {

    private val dispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
        Refresh.resetForTest()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() &&
                File(dir, "app-customer").exists()
            ) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        // **ونهاياتُ الأسطر تُوحَّد قبل المطابقة** — **وجيتٌ على
        // ويندوز يكتب CRLF**، **فمطابقةُ نصٍّ فيه سطرٌ جديدٌ تسقط**:
        // **فيُقرأ العيبُ في المنتج وهو في الفحص.**
        return f.readText().replace("\r\n", "\n")
    }

    // ═════════════ PF-01 · لا دوّارةَ بلا نهاية ═════════════

    /**
     * **PF-01 · وأوّلُ تحميلٍ لا يبقى دائراً بعد نهايةٍ صامتة.**
     *
     * **وحالُ «لا تحميلَ ولا خطأ» كانت تدور أبداً** — **نداءٌ ابتُلع، أو
     * ردٌّ فارغ، أو رايةٌ لم تُخفَض.**
     */
    @Test
    fun `الانتظارُ محدودٌ بزمنٍ ثمّ يُقال ويُعرَض بابُ الإعادة`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/Kit.kt")
        assertTrue(
            "**عاد الدورانُ بلا حدّ**",
            src.contains("private const val STUCK_AFTER_MS"),
        )
        assertTrue("**لا تُقاس المدّة**", src.contains("delay(STUCK_AFTER_MS)"))
        assertTrue("**لا يُقال لصاحبه**", src.contains("R.string.load_stuck"))
        assertTrue("**لا بابَ للإعادة**", src.contains("R.string.act_retry"))
        // **والحدُّ معقولٌ**: **ليس أقصرَ من مهلة النداء فيكذب، ولا
        // دقيقةً فيصير صمتاً.**
        val limit = Regex("STUCK_AFTER_MS = (\\d+)_000L").find(src)?.groupValues?.get(1)?.toInt()
        assertTrue("**حدٌّ غيرُ معقول**: " + limit, limit != null && limit in 15..45)
    }

    // ═════════════ PF-10 · موجةٌ واحدةٌ لا موجات ═════════════

    /**
     * **PF-10 · والنبضاتُ المتلاحقةُ تُجمَع في واحدة.**
     *
     * **وحدثٌ حيٌّ واحدٌ كان يوقظ سبعةَ عشرَ سامعاً** — **وطلبٌ يمرّ
     * بأربع حالاتٍ في دقيقةٍ يعني أربعَ موجات.**
     */
    @Test
    fun `عشرُ نبضاتٍ متلاحقةٍ تصير نبضةً واحدة`() = runTest(dispatcher) {
        val before = Refresh.tick.value
        repeat(10) { Refresh.bump() }
        // **ولا شيءَ يقع قبل انقضاء النافذة** — **فلا يُنادى الخادمُ
        // عشراً.**
        assertEquals("**بُثّت قبل أن تُجمَع**", before, Refresh.tick.value)
        advanceTimeBy(400)
        assertEquals("**لم تُجمَع في واحدة**", before + 1, Refresh.tick.value)
    }

    /**
     * **ولا تُبتلع نبضة** — **من بثّ بعد النافذة يصله دورُه.**
     *
     * **والجمعُ تأخيرٌ قصيرٌ لا إسقاط** — **ولو أُسقطت لبقيت شاشةٌ على
     * حقيقةٍ مضت.**
     */
    @Test
    fun `الجمعُ لا يبتلع ما جاء بعده`() = runTest(dispatcher) {
        val before = Refresh.tick.value
        Refresh.bump()
        advanceTimeBy(400)
        assertEquals(before + 1, Refresh.tick.value)
        Refresh.bump()
        advanceTimeBy(400)
        assertEquals("**ابتُلعت موجةٌ تالية**", before + 2, Refresh.tick.value)
    }

    /** **وفعلُ صاحبِ الشاشة يُبَثّ الآن** — **ولا ينتظر نافذةً.** */
    @Test
    fun `بثُّ فعلِ الشاشة فوريّ`() = runTest(dispatcher) {
        val before = Refresh.tick.value
        Refresh.bumpNow()
        assertEquals("**أُخِّر فعلٌ فعله صاحبُه بيده**", before + 1, Refresh.tick.value)
    }

    // ═════════════ PF-05 · PF-06 — الإعادةُ محدودة ═════════════

    /**
     * **PF-05 · PF-06 · ولا يُعاد نداءٌ يكتب من تلقاء نفسه.**
     *
     * **ولا مُعيدَ في العميل أصلاً** — **وهذا عقدٌ يُحرَس**: **من
     * أضاف `HttpRequestRetry` أعاد إنشاءَ طلبٍ سقط ردُّه.**
     *
     * **والإعادةُ الوحيدةُ المسموحة: تجديدُ رمزٍ عند ٤٠١ مرّةً واحدة.**
     */
    @Test
    fun `لا مُعيدَ أعمى في عميل الشبكة`() {
        val src = read("shared/src/main/kotlin/com/rahalgo/shared/net/ApiClient.kt")
        assertFalse(
            "**رُكّب مُعيدٌ عامٌّ يعيد كلَّ نداءٍ ومنها ما يكتب**",
            src.contains("HttpRequestRetry"),
        )
        // **والتجديدُ مرّةً لا حلقة.**
        assertEquals(
            "**تبدّل عقدُ التجديد الواحد**",
            1,
            Regex("refresh\\(\\)\\s*\\n\\s*return raw").findAll(src).count(),
        )
    }

    // ═════════════ PB-05 · الإعادةُ متباعدةٌ ومحدودة ═════════════

    /**
     * **PB-05 · PB-06 · ووصلٌ ينقطع لا يُعاد في حلقةٍ ضيّقة.**
     *
     * **ومن أعاد الوصلَ بلا تباعدٍ أيقظ الراديو مئةَ مرّةٍ في دقيقة** —
     * **وتلك بطّاريّةُ سائقٍ في نفق.**
     */
    @Test
    fun `إعادةُ الوصل الحيّ تتباعد ولها سقف`() {
        val src = read("shared/src/main/kotlin/com/rahalgo/shared/net/LiveSocket.kt")
        assertTrue("**ذهب التباعد**", src.contains("wait = (wait * 2).coerceAtMost(maxRetryMs)"))
        assertTrue("**ذهب الانتظارُ بين محاولتين**", src.contains("delay(wait)"))
        val first = Regex("FIRST_RETRY_MS = (\\d+)_000L").find(src)?.groupValues?.get(1)?.toInt()
        assertTrue("**أوّلُ إعادةٍ أسرعُ من ثانيتين**: " + first, first != null && first >= 2)
    }

    // ═════════════ Part 18 · المهلُ من قياس ═════════════

    /**
     * **ومهلةٌ لخادمٍ مضى ليست مهلة.**
     *
     * **وقِيس ٢٠٢٦-٠٩-١٦**: **أبطأُ بابٍ ٢٦ مل.ث · وزمنُ الوصول ٢٦٥
     * مل.ث** — **وتسعون ثانيةً كانت لخطّةٍ مجّانيّةٍ تُنيم الخدمة.**
     */
    @Test
    fun `مهلُ النداء معقولةٌ ومحدودة`() {
        val src = read("shared/src/main/kotlin/com/rahalgo/shared/net/ApiClient.kt")
        fun ms(name: String): Int? =
            Regex(name + " = (\\d+)_000").find(src)?.groupValues?.get(1)?.toInt()
        val connect = ms("connectTimeoutMillis")
        val request = ms("requestTimeoutMillis")
        assertTrue("**لا مهلةَ اتّصال**", connect != null && connect in 5..20)
        assertTrue("**مهلةُ النداء غيرُ معقولة**: " + request, request != null && request in 10..45)
        // **ولا نداءَ بلا مهلة.**
        assertTrue("**رُفعت إضافةُ المهل**", src.contains("install(HttpTimeout)"))
        // **والرفعُ يأخذ مهلتَه** — **ولا يُقاس بمهلة القراءة.**
        assertTrue("**رفعٌ بمهلة قراءة**", src.contains("timeout {"))
    }

    // ═════════════ CP · حديثٌ لا يستنزف ═════════════

    /**
     * **CP-01 · CP-04 · والحديثُ يُنعَش بالنبضة لا باستجوابٍ دوريّ.**
     *
     * **ولا مؤقّتَ في شيفرة الحديث** — **ومن وضع استجواباً كلَّ ثانيةٍ
     * أيقظ الراديو ستّين مرّةً في الدقيقة لشاشةٍ صامتة.**
     */
    @Test
    fun `الحديثُ بلا استجوابٍ دوريّ`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt")
        assertTrue("**ذهب الإنعاشُ بالنبضة**", src.contains("Refresh.tick"))
        assertFalse("**استُحدث استجوابٌ بمهلة**", src.contains("delay("))
        assertFalse("**مقبسٌ حيٌّ ثانٍ**", src.contains("WebSocket"))
        // **CP-01 · ولا يُنعَش إلّا الحديثُ المفتوح.**
        assertTrue(
            "**أُنعشت أحاديثُ غيرِ المفتوح**",
            src.contains("if (current.isNotEmpty()) load(current)"),
        )
    }

    /**
     * **CP-03 · وتبديلُ أ/ب لا يراكم وظائفَ إنعاش.**
     *
     * **ونموذجٌ واحدٌ للمضيف** — **ومجمِّعٌ واحدٌ في `init`**: **فلا
     * وظيفةٌ تُضاف مع كلّ فتحة.**
     */
    @Test
    fun `تبديلُ الأحاديث لا يراكم مجمّعات`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt")
        assertEquals(
            "**أكثرُ من مجمِّعٍ للنبضة في نموذج الحديث**",
            1,
            Regex("Refresh\\.tick").findAll(src).count(),
        )
        // **والمجمِّعُ في `init` لا في `load`** — **ولو كان في `load`
        // لَتراكم مع كلّ تبديل.**
        val initAt = src.indexOf("init {")
        val loadAt = src.indexOf("fun load(")
        assertTrue("**ذهب `init`**", initAt > 0)
        assertTrue(
            "**المجمِّعُ داخلَ الجلب** — **فيتراكم مع كلّ تبديل**",
            src.indexOf("Refresh.tick") in (initAt + 1) until loadAt,
        )
    }

    // ═════════════ PB-10 · PB-11 — لا موقعَ بلا عمل ═════════════

    /**
     * **PB-10 · PB-11 · وخدمةُ الموقع تتبع الورديّة.**
     *
     * **ومن ترك الخدمةَ بعد إغلاق الورديّة ترك إشعاراً في الشريط
     * وموقعاً يُرسَل** — **وبطّاريّةً تُستنزف لعملٍ انتهى.**
     */
    @Test
    fun `خدمةُ الموقع تقف مع الورديّة ومع الخروج`() {
        val vm = read("app-driver/src/main/kotlin/com/rahalgo/driver/home/HomeViewModel.kt")
        assertTrue(
            "**لا تتبع الخدمةُ حالَ الورديّة**",
            vm.contains("if (me.onShift && LocationPermission.granted(app))"),
        )
        assertTrue("**لا تقف عند إغلاقها**", vm.contains("LocationService.stop(app)"))
        assertTrue(
            "**تبقى بعد الخروج**",
            vm.contains("LocationService.stop(getApplication())"),
        )
    }

    /**
     * **PB-01 · PB-03 · والساكنُ لا يُستجوَب كالمُلاحِ.**
     *
     * **ودرجتان مقيستان**: **الورديّةُ عشرون ثانيةً وعشرون متراً
     * وتجميعٌ مسموح** · **والملاحةُ ثانيةٌ بلا تجميعٍ ولا حدِّ مسافة.**
     *
     * **ولو استُعمل معدّلُ الملاحة في السكون لَنزفت البطّاريّةُ بلا
     * عمل.**
     */
    @Test
    fun `معدّلُ الورديّة غيرُ معدّل الملاحة`() {
        val svc = read("app-driver/src/main/kotlin/com/rahalgo/driver/location/LocationService.kt")
        assertTrue("**ذهب معدّلُ الورديّة**", svc.contains("DEFAULT_PING_SEC = 20L"))
        assertTrue("**ذهب حدُّ الحركة**", svc.contains("MIN_MOVE_M = 20f"))
        assertTrue(
            "**رُفع التجميعُ من الورديّة** — **فيُوقَظ الجهازُ لكلّ قراءة**",
            svc.contains("setMaxUpdateDelayMillis(ms * 2)"),
        )
        assertTrue(
            "**رُفع حدُّ الإرسال** — **فنداءٌ لكلّ نقطة**",
            svc.contains("MIN_SEND_GAP_MS"),
        )
        val nav = read("driver-navigation/src/main/kotlin/com/rahalgo/navigation/LocationEngine.kt")
        assertTrue("**تبدّل معدّلُ الملاحة**", nav.contains("NAV_INTERVAL_MS = 1_000L"))
        // **ومعدّلُ الملاحة لا يُستعمل في خدمة الورديّة.**
        assertFalse("**استُعمل معدّلُ الملاحة في الورديّة**", svc.contains("1_000L"))
    }
}

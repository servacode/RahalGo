package com.rahalgo.customer

import com.rahalgo.shared.model.Ordering
import java.io.File
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ الاستقبال في الجهاز — عرضٌ لا حكم** (`PH-27`، `PH-28`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والحكمُ في المحرّك ويُقاس هناك** (`internal/platform` و`internal/qa`)
 * — **وهذه تقيس ما في الجهاز**: **أنّ الحالَ تشيخ فتُجدَّد**، **وأنّ
 * الغيابَ لا يمنع أحداً**، **وأنّ السلّة تقرأ الحالَ ولا تحسبها.**
 */
class ServingPolicyTest {

    @Before fun clean() = Serving.resetForTest()

    @After fun after() = Serving.resetForTest()

    /**
     * **وقبل أوّل قراءةٍ لا يُمنَع أحد.**
     *
     * **ومنعٌ بلا علمٍ أسوأُ من ردٍّ بعلم**: **نداءٌ لم يصل بعدُ يُغلق
     * السلّةَ على صاحبها والمنصّةُ مفتوحة.**
     */
    @Test
    fun `الغيابُ لا يمنع`() {
        assertTrue("**حالٌ غائبةٌ منعت الطلب**", Serving.available)
        assertEquals("", Serving.reason)
    }

    /** **وما يقوله الخادمُ يُعرَض كما هو.** */
    @Test
    fun `السببُ والنصُّ والموعدُ تُقرأ من الخادم`() {
        Serving.put(
            Ordering(
                available = false,
                reason = "platform_closed_now",
                message = "",
                nextAvailableAt = "2026-09-14T17:00:00+03:00",
            ),
            1_000L,
        )
        assertFalse(Serving.available)
        assertEquals("platform_closed_now", Serving.reason)
        assertEquals("2026-09-14T17:00:00+03:00", Serving.nextAt)
    }

    /** **ونصُّ المالك يصل حين يضبطه.** */
    @Test
    fun `نصُّ المالك يصل`() {
        Serving.put(
            Ordering(available = false, reason = "temporarily_unavailable", message = "صيانة"),
            1_000L,
        )
        assertEquals("صيانة", Serving.message)
    }

    /**
     * **PH-28 · والحالُ تشيخ فتُجدَّد.**
     *
     * **ومن ترك التطبيقَ في الخلفيّة ساعاتٍ وعاد يرى حالاً قديمة** —
     * **فيملأ سلّةً ليُردّ في آخرها.**
     */
    @Test
    fun `الحالُ تشيخ بعد المهلة`() {
        Serving.put(Ordering(), 10_000L)
        assertFalse("**حالٌ طازجةٌ عُدّت شائخة**", Serving.stale(10_000L + 1_000L))
        assertTrue(
            "**حالٌ عمرُها أكثرُ من المهلة لم تُعَدّ شائخة** — فلا تُجدَّد أبداً",
            Serving.stale(10_000L + Serving.FRESH_MS),
        )
    }

    /** **ولا حالَ أصلاً شائخةٌ دائماً** — فأوّلُ فتحةٍ تسأل. */
    @Test
    fun `الغيابُ شائخٌ فيُسأل الخادم`() {
        assertTrue(Serving.stale(0L))
        assertTrue(Serving.stale(Long.MAX_VALUE / 2))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والسلّةُ تقرأ الحالَ ولا تحسبها** — بنيةٌ تُقاس بالنصّ لأنّها نصّ
     * ══════════════════════════════════════════════════════════════════
     *
     * **ومن أعاد الزرَّ مفتوحاً خارجَ الدوام أرسل الزبونَ ليُردَّ** —
     * **ولا فحصَ سلوكٍ في وحدةٍ يرى قشرةَ شاشةٍ في وحدةٍ أخرى.**
     *
     * **والأخطرُ منه أن يُحسَب الدوامُ هنا**: **ساعةُ الهاتف تُبدَّل
     * بإصبع** — **فيُمنَع أن يُقرأ `LocalTime.now()` في هذه الشاشة.**
     */
    @Test
    fun `السلّةُ تقرأ حالَ الاستقبال ولا تحسب وقتاً`() {
        val f = File(appRoot(), "src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt")
        assertTrue("لم أجد شاشةَ السلّة: " + f.absolutePath, f.exists())
        val text = f.readText()

        // **ولا يكفي أن يُذكَر الاسمُ في الملفّ** — **فالتنبيهُ فوقَ
        // الزرّ يذكره أيضاً**: **ومن نزعه من شرط `enabled` وحدَه ترك
        // زرّاً يعمل تحتَ تنبيهٍ يقول «مغلق».** (قِيس: أوّلُ صياغةٍ
        // لهذا الفحص مرّت على ذلك الكسر بعينه.)
        assertTrue(
            "**زرُّ الإتمام لا يقرأ حالَ الاستقبال في شرط تفعيله** — " +
                "فيُرسَل الزبونُ ليُردّ",
            text.contains("enabled = !vm.busy && address != null && Serving.available &&"),
        )
        assertTrue(
            "**لا تنبيهَ فوقَ الزرّ يقول السبب** — فزرٌّ باهتٌ بلا سببٍ يُقرأ عطباً",
            text.contains("if (!Serving.available) {"),
        )
        for (banned in listOf("LocalTime.now(", "LocalDate.now(", "Calendar.getInstance(")) {
            assertFalse(
                "**الشاشةُ تحسب وقتاً بساعة الجهاز**: " + banned +
                    " — **وساعةُ الهاتف تُبدَّل بإصبع**",
                text.contains(banned),
            )
        }
    }

    /** **وجذرُ الوحدة يُبلَغ صعوداً** — ومسارٌ مثبَّتٌ يسقط إن نُقل الفحص. */
    private fun appRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "src/main/kotlin/com/rahalgo/customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ app-customer من " + File("").absolutePath)
    }
}

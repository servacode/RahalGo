package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ الوضعين والانتقال بينهما — بلا جهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠، البند ١١: «Transition between
 *  navigation/normal mode» و**«عدم إرسال نقطة لكلّ ثانية إلى
 *  Backend»**.)
 *
 * # وكيف يُثبَت أنّ الخادمَ لا يُنادى
 *
 * **لا يُقاس بالنيّة بل بالبناء**: `NavPipeline` و`NavigationSession`
 * **لا تعرفان شبكةً أصلاً** — لا `Backend` ولا `DriverApi` ولا
 * `TrackPoint`. **وما لا يملك الباب لا يطرقه.**
 *
 * **والحارسُ هنا يقرأ المصدرَ نفسَه** — فمن أضاف نداءً يوماً يسقط
 * بناؤه. (وهو أصدقُ من عدّ النداءات في تشغيلٍ واحد.)
 */
class NavSessionTest {

    private val lat = 35.9506
    private val lng = 39.0094

    private fun fix(i: Int, speed: Float = 11f, bearing: Float? = 0f, acc: Float = 6f) =
        NavFix(lat + i * 0.00009, lng, acc, speed, bearing, 1_000L + i * 1_000L)

    /**
     * **مجرًى وهميٌّ يغذّي السلسلة** — بذرةُ `navigation-fixtures`.
     *
     * (أمرُ المالك: «إذا احتجت Fake Location stream للاختبارات، اجعله
     *  جزءاً نظيفاً من `driver-navigation`».)
     */
    private fun play(pipeline: NavPipeline, fixes: List<NavFix>): List<NavPipeline.Step> =
        fixes.map { pipeline.onFix(it) }

    // ══════════════════════════════════════════════════════════════════
    // **الانتقالُ بين الوضعين**
    // ══════════════════════════════════════════════════════════════════

    /** **وبدءُ جلسةٍ جديدةٍ ينسى ما قبلها** — وإلّا حُسبت قفزةٌ كاذبة. */
    @Test
    fun `إعادةُ الضبط تنسى كلَّ شيء`() {
        val p = NavPipeline()
        play(p, (0..4).map { fix(it) })
        assertNotNull(p.headingDeg)
        assertTrue(p.seen > 0)

        p.reset()
        assertNull(p.headingDeg)
        assertEquals(0, p.seen)
        assertEquals(0, p.rejected)
        assertNull(p.accepted)
    }

    /**
     * **ورحلةٌ ثانيةٌ تبدأ بعيداً عن الأولى لا تُرفض.**
     *
     * **ومن نسي الإعادةَ حسب الانتقالَ من الرقّة إلى دمشق قفزةً** —
     * فرُفضت كلُّ قراءاتِ الرحلة الجديدة ووقفت الأيقونةُ أبدا.
     */
    @Test
    fun `رحلةٌ جديدةٌ في مدينةٍ أخرى تعمل بعد الإعادة`() {
        val p = NavPipeline()
        p.onFix(fix(0))
        p.reset()
        val damascus = NavFix(33.5138, 36.2765, 6f, 11f, 0f, 2_000L)
        val step = p.onFix(damascus)
        assertEquals(FixGrade.ACCEPTED, step.grade)
        assertNotNull(step.targetLat)
    }

    // ══════════════════════════════════════════════════════════════════
    // **ما يُرسَم**
    // ══════════════════════════════════════════════════════════════════

    /** **والمرفوضةُ لا تُنتج شيئاً يُرسم.** */
    @Test
    fun `الخطوةُ المرفوضةُ لا تُبنى للرسم`() {
        val p = NavPipeline()
        p.onFix(fix(0))
        val jumped = NavFix(lat + 0.05, lng, 6f, 11f, 0f, 2_000L)
        val step = p.onFix(jumped)
        assertNull(NavRender.of(step, 1L))
    }

    /** **وكلُّ خطوةٍ مقبولةٍ لها معرّفٌ يزيد** — فلا تُعاد حركتُها. */
    @Test
    fun `معرّفُ الخطوة يزيد فلا تُعاد الحركة`() {
        val p = NavPipeline()
        val a = NavRender.of(p.onFix(fix(0)), 1L)!!
        val b = NavRender.of(p.onFix(fix(1)), 2L)!!
        assertTrue(b.stepId > a.stepId)
    }

    /** **والكاميرا تأخذ مدّةَ الحركة نفسَها** — فتصل معها لا قبلها. */
    @Test
    fun `مدّةُ الكاميرا توافق مدّةَ الأيقونة`() {
        val p = NavPipeline()
        p.onFix(fix(0))
        val step = p.onFix(fix(1))
        val render = NavRender.of(step, 2L)!!
        assertEquals(render.durationMs, render.camera(0f).durationMs)
        assertEquals(step.animationMs, render.durationMs)
    }

    // ══════════════════════════════════════════════════════════════════
    // **البناءُ نفسُه يمنع نداءَ الخادم**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **ولا سطرَ شبكةٍ في وحدة الملاحة.**
     *
     * **يُقرأ المصدرُ لا السلوك**: من أضاف `Backend` أو `DriverApi`
     * أو `sendLocation` يوماً **يسقط هذا الحارسُ في اللحظة**، لا بعد
     * أن يشتكي سائقٌ من حزمته.
     */
    @Test
    fun `وحدةُ الملاحة لا تعرف شبكةً أصلا`() {
        val root = java.io.File("src/main/kotlin/com/rahalgo/navigation")
        assertTrue("لا مصدرَ للفحص: ${root.absolutePath}", root.isDirectory)
        val banned = listOf("Backend", "DriverApi", "sendLocation", "sendBatch", "HttpClient", "TrackPoint")
        val offenders = mutableListOf<String>()
        root.walkTopDown().filter { it.extension == "kt" }.forEach { f ->
            // **والتعليقُ لا يُحاسَب** — حارسٌ يسقط على شرحه لا يحرس
            // شيئاً. (وقع مثلُه في حارسِ الصور ٢٠٢٦-٠٨-١٩.)
            val code = f.readText()
                .replace(Regex("(?s)/\\*.*?\\*/"), " ")
                .lines().joinToString("\n") { it.substringBefore("//") }
            for (word in banned) {
                if (code.contains(word)) offenders += "${f.name}: $word"
            }
        }
        assertTrue("وحدةُ الملاحة تطرق بابَ الشبكة: $offenders", offenders.isEmpty())
    }

    /**
     * **وتردّدُ الملاحة المحلّيُّ ليس تردّدَ الإرسال.**
     *
     * **رقمان مختلفان قصداً** — وهذا نصُّ أمر المالك.
     */
    @Test
    fun `تردّدُ الملاحة أسرعُ من تردّد الورديّة بمراتب`() {
        // **وخدمةُ الورديّة عشرون ثانية** — قِيس في `LocationService`.
        val shiftMs = 20_000L
        assertTrue(
            "الملاحةُ ليست أسرع: ${LocationEngine.NAV_INTERVAL_MS}",
            LocationEngine.NAV_INTERVAL_MS * 10 <= shiftMs,
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **رحلةٌ فيها ما طلبه المالك**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **مستقيمٌ · انعطافٌ · وقوفٌ · انطلاقٌ · قراءةٌ سيّئة.**
     *
     * **وهي بذرةُ `navigation-fixtures`** — تُعاد بلا هاتف.
     */
    @Test
    fun `رحلةٌ كاملةٌ لا ترتجف ولا تقفز`() {
        val p = NavPipeline()
        var t = 1_000L
        var la = lat
        var ln = lng

        // **مستقيمٌ شمالاً.**
        repeat(8) {
            la += 0.0001
            p.onFix(NavFix(la, ln, 6f, 12f, 0f, t)); t += 1_000
        }
        assertTrue(GpsQuality.angleClose(p.headingDeg!!, 0f, 10f))

        // **قراءةٌ سيّئةٌ واحدة** — لا تُحرّك شيئاً.
        val badStep = p.onFix(NavFix(la + 0.03, ln, 6f, 12f, 0f, t)); t += 1_000
        assertEquals(FixGrade.REJECTED, badStep.grade)
        assertNull(badStep.targetLat)

        // **انعطافٌ يمينٌ إلى الشرق.**
        repeat(8) {
            ln += 0.0001
            p.onFix(NavFix(la, ln, 6f, 12f, 90f, t)); t += 1_000
        }
        val east = p.headingDeg!!
        assertTrue("لم ينعطف: $east", GpsQuality.angleClose(east, 90f, 15f))

        // **وقوفٌ** — والجهازُ يهذي بالاتّجاهات.
        repeat(6) {
            p.onFix(NavFix(la, ln, 6f, 0.1f, (it * 60).toFloat(), t)); t += 1_000
        }
        assertEquals("دار السهمُ وهو واقف", east, p.headingDeg!!, 0.01f)

        // **انطلاقٌ جنوباً بعد الوقوف.**
        repeat(8) {
            la -= 0.0001
            p.onFix(NavFix(la, ln, 6f, 12f, 180f, t)); t += 1_000
        }
        assertTrue("لم يتّجه جنوباً: ${p.headingDeg}", GpsQuality.angleClose(p.headingDeg!!, 180f, 20f))

        assertEquals(1, p.rejected)
    }
}

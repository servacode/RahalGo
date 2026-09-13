package com.rahalgo.merchant

import com.rahalgo.map.data.MapConfig
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ عزل الإنتاج — تطبيقُ المتجر** (`P-8`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكان هذا الحارسُ في تطبيق السائق وحدَه** — **والثلاثةُ الباقيةُ بلا
 * حارسِ عنوانٍ أصلاً**، لأنّها كانت تقرأ ثابتاً مشتركاً لا يُبدَّل.
 *
 * **ويومَ فُتح مدخلُ التجاوز للتجربة** (`P-8`: أربعةُ تطبيقاتٍ على جهازٍ
 * واحدٍ تمشي الدورةَ على التجهيز) **صار للثلاثة ما للسائق**: **بابٌ
 * يلزمه حارس.**
 *
 * # وما يحرسه
 *
 * **الأوّل — الإصدارُ يبقى حرفاً بحرف على الإنتاج.** **فمن أضاف
 * تجاوزاً إلى `defaultConfig` بدل `debug` فتح البابَ للإصدار وهو يظنّه
 * للتصحيح** — **وقطعةٌ تُرفع إلى الناس وتكلّم تجهيزاً أسوأُ من عطبٍ
 * يظهر.**
 *
 * **الثاني — بناءُ القبول لا يكلّم الإنتاج.** **يُمرَّر التجاوزُ فيُخطأ
 * في اسم الخاصّيّة، فيُبنى بناءٌ يبدو بناءَ قبولٍ وهو يكتب في بيانات
 * الناس** — ولا شيءَ في الشاشة يقول ذلك.
 *
 * **الثالث — `Backend` موصولٌ بالحقل المولَّد لا بنصٍّ ثانٍ.**
 * **والاختبارُ الذي يقارن نصّاً بنصٍّ يمرّ دائماً**: يقرأ ما كتبتُه لا
 * ما بُني. **وهذا يقرأ `BuildConfig`**، فإن انقطع التوصيلُ سقط.
 */
class ProductionEndpointGuardTest {

    private companion object {
        const val PROD_API = "https://api.rahalgo.com"
        const val PROD_MAPS = "https://maps.rahalgo.com"
    }

    /** **والإصدارُ يبقى على عنوانَي الإنتاج — لا خاصّيّةَ تُغيّره.** */
    @Test
    fun `الإصدارُ يبقى على عنوانَي الإنتاج`() {
        if (BuildConfig.DEBUG) return
        assertEquals(PROD_API, BuildConfig.API_BASE_URL)
        assertEquals(PROD_MAPS, BuildConfig.MAPS_BASE_URL)
    }

    /**
     * **وبناءُ القبول لا يشير إلى الإنتاج.**
     *
     * **ويعمل في الحالين**: بُني بتجاوزٍ فيسقط إن بقي على الإنتاج،
     * **وبُني بلا تجاوزٍ فلا يُدّعى عليه شيء** — فالاختبارُ يقيس ما بين
     * يديه لا ما يتمنّاه.
     *
     * **وعلامةُ «بناءِ قبول» أن يكون العنوانُ غيرَ الإنتاج** — **ولا
     * يُشترَط أن يكون حلقيّاً**: تجهيزُ `P-8` على خادمٍ بعنوانٍ عامّ.
     */
    @Test
    fun `بناءُ القبولِ المُوجَّهُ لا يبقى على الإنتاج`() {
        if (!BuildConfig.DEBUG) return

        val api = BuildConfig.API_BASE_URL
        val maps = BuildConfig.MAPS_BASE_URL
        // **وبناءٌ تجريبيٌّ بلا تجاوزٍ يبقى على الإنتاج بحقّ** — فلا يُقاس.
        val directed = api != PROD_API || maps != PROD_MAPS
        if (!directed) return

        // **ومن وُجّه عنوانُ محرّكِه وجب أن يفترق عن الإنتاج فعلاً** —
        // **ولا يكفي أن يشبهه**: نطاقٌ ينتهي بـ`api.rahalgo.com` إنتاجٌ.
        assertFalse(
            "بناءُ قبولٍ يكلّم محرّكَ الإنتاج: $api",
            api == PROD_API || api.endsWith("//api.rahalgo.com") ||
                api.endsWith(".api.rahalgo.com"),
        )
    }

    /** **و`Backend` يقرأ الحقلَ المولَّد لا نصّاً ثانياً.** */
    @Test
    fun `Backend موصولٌ بـBuildConfig لا بنصٍّ مستقلّ`() {
        assertEquals(BuildConfig.API_BASE_URL, Backend.BASE_URL)
        assertEquals(BuildConfig.MAPS_BASE_URL, Backend.MAPS_BASE_URL)
    }

    /**
     * **ولا `http` إلى مضيفٍ ليس حلقيّاً** — ولو كان البناءُ تجريبيّاً.
     *
     * **وثقبُ الحلقيِّ ثقبٌ لا باب**: `http://127.0.0.1.attacker.com`
     * يبدأ بالعنوان الحلقيِّ نصّاً وليس حلقيّاً.
     */
    @Test
    fun `لا نصَّ صريحاً إلى مضيفٍ غيرِ حلقيّ`() {
        val api = BuildConfig.API_BASE_URL
        if (!api.startsWith("http://")) return
        assertFalse(
            "عنوانٌ غيرُ مشفَّرٍ إلى مضيفٍ ليس حلقيّاً: $api",
            !MapConfig.isLoopbackHttp(api),
        )
    }
    /**
     * ══════════════════════════════════════════════════════════════════
     * **وقفلُ الإصدار يُقاس في متغيّرٍ يعمل** (`P-8`، ٢٠٢٦-٠٩-١٤)
     * ══════════════════════════════════════════════════════════════════
     *
     * **والتأكيدُ فوقُ يبدأ بـ`if (BuildConfig.DEBUG) return`** —
     * **و`AGP 9` لا يولّد فحوصَ وحدةٍ للإصدار افتراضاً** (قِيس: لا
     * مهمّةَ `testReleaseUnitTest` في أيٍّ من الأربعة). **فذاك التأكيدُ
     * خاملٌ لا يُشغَّل** — **وحارسٌ لا يعمل ليس حارساً.**
     *
     * **فيُقاس القفلُ حيث يُكتب**: **كتلةُ `release` في بناء هذا
     * التطبيق** — **عنوانان حرفيّان، ولا ذكرَ لتجاوزٍ فيها.**
     *
     * **ومن نقل التجاوزَ إلى `defaultConfig` أو أضافه إلى `release`
     * أسقط هذا الفحصَ في متغيّرٍ يُشغَّل كلَّ مرّة.**
     */
    @Test
    fun `كتلةُ الإصدار في البناء لا تقبل تجاوزاً`() {
        val gradle = java.io.File("build.gradle.kts")
        assertTrue("لم أجد ملفَّ البناء: " + gradle.absolutePath, gradle.exists())
        val text = gradle.readText()

        val start = text.indexOf("        release {")
        assertTrue("لا كتلةَ release في البناء", start >= 0)
        val debugAt = text.indexOf("        debug {", start)
        val end = if (debugAt > start) debugAt else text.length
        val release = text.substring(start, end)

        assertTrue(
            "كتلةُ الإصدار لا تكتب عنوانَ المحرّك حرفاً",
            release.contains("quoted(\"" + PROD_API + "\")"),
        )
        assertTrue(
            "كتلةُ الإصدار لا تكتب عنوانَ الخرائط حرفاً",
            release.contains("quoted(\"" + PROD_MAPS + "\")"),
        )
        assertFalse(
            "**كتلةُ الإصدار تقرأ تجاوزاً** — فقطعةُ إصدارٍ قد تشير إلى تجهيز",
            release.contains("overrideOrNull") || release.contains("findProperty"),
        )
        // **ولا تجاوزَ في `defaultConfig`** — **فهو يسري على الإصدار.**
        val dcAt = text.indexOf("    defaultConfig {")
        if (dcAt >= 0) {
            val dcEnd = text.indexOf("\n    }", dcAt)
            val dc = text.substring(dcAt, if (dcEnd > dcAt) dcEnd else text.length)
            assertFalse(
                "**تجاوزٌ في `defaultConfig` يسري على الإصدار**",
                dc.contains("overrideOrNull") || dc.contains("API_BASE_URL"),
            )
        }
    }

}

package com.rahalgo.driver

import com.rahalgo.map.data.MapConfig
import com.rahalgo.map.data.MapUrls
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ عزل الإنتاج — البند ٨ من قرار ٢٠٢٦-٠٨-٢٢**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً**: «أريد فشل build/test إذا حصل خطأ يجعل Debug
 * acceptance build يكتب Production أثناء الاختبار».
 *
 * # **ما يحرسه، ولماذا كلُّ بندٍ فيه**
 *
 * **الأوّل — البناءُ التجريبيُّ لا يبقى على الإنتاج وقد أُعطي تجاوزاً.**
 * وهذا هو العطبُ الحقيقيُّ المخيف: **يُمرَّر `-Prahalgo.apiBaseUrl`،
 * ويُخطأ في اسم الخاصّيّة أو موضعِها، فيُبنى بناءٌ يبدو بناءَ قبولٍ
 * وهو يكلّم الزبائن.** ولا شيءَ في الشاشة يقول ذلك — **الطلبُ يُقبل،
 * والسائقُ يتحرّك، والكتابةُ تقع في الإنتاج.**
 *
 * **الثاني — الإصدارُ يبقى حرفاً بحرف.** فمن أضاف تجاوزاً إلى
 * `defaultConfig` بدل `debug` **فتح البابَ للإصدار وهو يظنّه للتصحيح**.
 *
 * **الثالث — ثقبُ الحلقيّ ثقبٌ لا باب.** `http://` إلى أيّ مضيفٍ آخرَ
 * مرفوضٌ ولو كان البناءُ تجريبيّاً.
 *
 * **ولماذا `BuildConfig` لا نصٌّ مكتوب**: **الاختبارُ الذي يقارن نصّاً
 * بنصٍّ يمرّ دائماً** — يقرأ ما كتبتُه أنا لا ما بُني فعلاً. **وهذا
 * يقرأ الحقلَ المولَّد**، فإن انقطع التوصيلُ سقط.
 */
class ProductionEndpointGuardTest {

    private companion object {
        const val PROD_API = "https://api.rahalgo.com"
        const val PROD_MAPS = "https://maps.rahalgo.com"

        /** **مضيفُ التجهيز للميدان** — **عنوانٌ واحدٌ لا ثانيَ له.** */
        const val STAGING_API = "https://staging-api.rahalgo.com"
    }

    /**
     * **بناءُ القبول لا يكلّم الإنتاج.**
     *
     * **ويعمل في الحالين**: بُني بتجاوزٍ فيسقط إن بقي على الإنتاج،
     * وبُني بلا تجاوزٍ فلا يُدّعى عليه شيء — **فالاختبارُ يقيس ما بين
     * يديه لا ما يتمنّاه.**
     */
    @Test
    fun `بناءُ القبول التجريبيُّ لا يشير إلى الإنتاج`() {
        if (!BuildConfig.DEBUG) return

        val api = BuildConfig.API_BASE_URL

        // ══════════════════════════════════════════════════════════════
        // **ولا شرطَ يُعفي بناءَ تصحيحٍ من هذا** (٢٠٢٦-٠٩-١٦)
        // ══════════════════════════════════════════════════════════════
        //
        // **وكان الحارسُ يصمت إلّا إذا رأى عنواناً حلقيّاً** — **فبناءُ
        // تصحيحٍ بلا تجاوزٍ كان يمرّ وهو على الإنتاج.**
        //
        // **والافتراضُ صار مضيفَ التجهيز** — **فلا عذرَ لصمتٍ:** **كلُّ
        // بناءِ تصحيحٍ إمّا تجهيزٌ وإمّا حلقيٌّ للتطوير، ولا ثالثَ.**
        assertFalse(
            "بناءُ تصحيحٍ يكلّم محرّكَ الإنتاج: $api",
            api.contains("//api.rahalgo.com"),
        )
        assertTrue(
            "بناءُ تصحيحٍ لا يكلّم التجهيزَ ولا الحلقيّ: $api",
            api.startsWith(STAGING_API) || MapConfig.isLoopbackHttp(api),
        )

        // **وأثرُ الميدان على HTTPS** — **ولا نصَّ صريحٌ إلّا للحلقيّ.**
        if (!MapConfig.isLoopbackHttp(api)) {
            assertTrue("أثرُ ميدانٍ بلا HTTPS: $api", api.startsWith("https://"))
        }

        // **والخرائطُ تبقى على مضيفها العامّ** — **بلاطٌ يُقرأ ولا
        // يُكتب، ولا مضيفَ تجهيزٍ له**: **فلا يُدَّعى عليه شيء.**
    }

    /** **ومضيفُ التجهيز مكتوبٌ مرّةً في البناء لا في الشيفرة.** */
    @Test
    fun `مضيفُ التجهيز من البناء لا من نصٍّ في الشيفرة`() {
        val gradle = java.io.File(appModuleDir(), "build.gradle.kts")
            .readText().replace("\r\n", "\n")
        assertTrue(
            "**ذهب افتراضُ التجهيز من كتلة التصحيح**",
            gradle.contains("?: \"$STAGING_API\""),
        )
        // **ولا مبدّلَ بيئةٍ في وقت التشغيل** — **الوجهةُ تُحقَن وقتَ
        // البناء وتُقفل.**
        assertFalse(
            "**ظهر مبدّلُ بيئةٍ في وقت التشغيل**",
            gradle.contains("BuildConfig.DEBUG ? ") || gradle.contains("if (isStaging)"),
        )
    }

    private fun appModuleDir(): java.io.File {
        var dir = java.io.File("").absoluteFile
        repeat(6) {
            if (java.io.File(dir, "build.gradle.kts").exists() &&
                java.io.File(dir, "src/main").exists()
            ) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد مجلَّد التطبيق من " + java.io.File("").absolutePath)
    }

    /** **والإصدارُ يبقى حرفاً بحرف — لا خاصّيّةَ تُغيّره.** */
    @Test
    fun `الإصدارُ يبقى على عنوانَي الإنتاج`() {
        if (BuildConfig.DEBUG) return
        assertEquals(PROD_API, BuildConfig.API_BASE_URL)
        assertEquals(PROD_MAPS, BuildConfig.MAPS_BASE_URL)
    }

    /** **و`Backend` يقرأ الحقلَ المولَّد لا نصّاً ثانياً.** */
    @Test
    fun `Backend موصولٌ بـBuildConfig لا بنصٍّ مستقلّ`() {
        assertEquals(BuildConfig.API_BASE_URL, com.rahalgo.driver.data.Backend.BASE_URL)
        assertEquals(BuildConfig.MAPS_BASE_URL, com.rahalgo.driver.data.Backend.MAPS_BASE_URL)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **وثقبُ الحلقيّ ثقبٌ — لا باب**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والخداعُ المقصودُ منعُه**: `http://127.0.0.1.attacker.com`
     * يبدأ بالعنوان الحلقيّ نصّاً **وليس حلقيّاً**. فمن قارن ببادئةٍ
     * فتح البابَ لكلّ نطاقٍ يشتري اسماً يبدأ بـ`127.0.0.1.`.
     */
    @Test
    fun `الحلقيُّ يُقبل وما سواه يُرفض`() {
        assertTrue(MapConfig.isLoopbackHttp("http://127.0.0.1:8791"))
        assertTrue(MapConfig.isLoopbackHttp("http://localhost:8791/style.json"))
        assertTrue(MapConfig.isLoopbackHttp("http://[::1]:8791"))

        assertFalse(MapConfig.isLoopbackHttp("http://127.0.0.1.attacker.com/x"))
        assertFalse(MapConfig.isLoopbackHttp("http://evil.test/127.0.0.1"))
        assertFalse(MapConfig.isLoopbackHttp("http://192.168.0.104:8791"))
        assertFalse(MapConfig.isLoopbackHttp("https://maps.rahalgo.com"))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والحارسُ الثاني — `MapUrls` — يُختبر وحدَه**
     * ══════════════════════════════════════════════════════════════════
     *
     * **قِيس ٢٠٢٦-٠٨-٢٢**: فُتح ثقبُ `MapConfig` وحدَه فمرّ الإعدادُ
     * **وسقط التنزيلُ صامتاً** — لأنّ كلَّ عنوانٍ يُبنى يمرّ من
     * `MapUrls` أيضاً، **وفيها `https` وحدَها.**
     *
     * **ولو لم يُختبر الحارسان معاً لعاد العطبُ نفسُه** عند أوّل
     * ترقيةٍ تنسى أحدَهما.
     */
    @Test
    fun `MapUrls تفتح للحلقيِّ بإذنٍ وتُغلق بلا إذن`() {
        assertEquals(
            "http://127.0.0.1:8791/map-resources/1/",
            MapUrls.resolveRemote("http://127.0.0.1:8791", "map-resources/1/", true),
        )
        assertEquals(
            "pmtiles://http://127.0.0.1:8791/base/x.pmtiles",
            MapUrls.onlinePmtiles("http://127.0.0.1:8791/base/x.pmtiles", true),
        )

        runCatching { MapUrls.resolveRemote("http://127.0.0.1:8791", "map-resources/1/") }
            .onSuccess { error("مرّ نصٌّ صريحٌ بلا إذن") }
        runCatching { MapUrls.resolveRemote("http://192.168.0.104:8791", "x/", true) }
            .onSuccess { error("مرّ مضيفٌ بعيدٌ بإذن الحلقيّ") }
        runCatching { MapUrls.onlinePmtiles("http://maps.rahalgo.com/x.pmtiles", true) }
            .onSuccess { error("مرّت بلاطاتٌ بعيدةٌ بنصٍّ صريح") }
    }

    /** **ولا يُبنى `MapConfig` بـ`http` إلّا بإذنٍ صريحٍ ومضيفٍ حلقيّ.** */
    @Test
    fun `MapConfig يرفض النصَّ الصريح بلا إذنٍ أو بمضيفٍ بعيد`() {
        MapConfig(baseUrl = "http://127.0.0.1:8791", allowLoopbackHttp = true)
        MapConfig(baseUrl = PROD_MAPS)

        runCatching { MapConfig(baseUrl = "http://127.0.0.1:8791") }
            .onSuccess { error("قُبل نصٌّ صريحٌ بلا إذن") }
        runCatching { MapConfig(baseUrl = "http://192.168.0.104:8791", allowLoopbackHttp = true) }
            .onSuccess { error("قُبل مضيفٌ بعيدٌ بإذن الحلقيّ") }
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

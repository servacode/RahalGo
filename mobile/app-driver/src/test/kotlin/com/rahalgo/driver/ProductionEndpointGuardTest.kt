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
        val maps = BuildConfig.MAPS_BASE_URL
        val acceptance = MapConfig.isLoopbackHttp(api) || MapConfig.isLoopbackHttp(maps)
        if (!acceptance) return

        assertFalse(
            "بناءُ قبولٍ يكلّم محرّكَ الإنتاج: $api",
            api == PROD_API || api.contains("api.rahalgo.com"),
        )
        assertFalse(
            "بناءُ قبولٍ يجلب خرائطَ الإنتاج: $maps",
            maps == PROD_MAPS || maps.contains("maps.rahalgo.com"),
        )
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
}

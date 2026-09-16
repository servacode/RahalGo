package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مصفوفةُ الإساءة — ما وجدناه بالمحاولة** (`AB`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ويُفترَض أنّ صاحبَ الجهاز فضوليٌّ أو عجِلٌ أو يحاول أن يكسر** —
 * **ولكلّ فعلٍ جوابٌ واحدٌ من ثلاثة**: **نجاحٌ · منعٌ صريحٌ · عطبٌ
 * يُستردّ منه.**
 *
 * **وما يُحرَس هنا ما وجدته المحاولةُ فعلاً** — **لا ما يُظنّ.**
 */
class AbuseMatrixTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText()
    }

    private val cart = "app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt"
    private val precart = "app-customer/src/main/kotlin/com/rahalgo/customer/PreCart.kt"
    private val locSvc = "app-driver/src/main/kotlin/com/rahalgo/driver/location/LocationService.kt"
    private val drvOrders = "app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt"

    // ═════════════ AB-03 · سباقُ العناوين ═════════════

    /**
     * **AB-03 · وتسعيرةُ عنوانٍ غادره لا تُكتب فوق عنوانِه.**
     *
     * **وُجد بالمحاولة** (٢٠٢٦-٠٩-١٦): **كان ردُّ التسعير يُكتب بلا
     * شرط** — **فمن اختار الرقّة ثمّ دمشق، وعاد ردُّ الرقّة متأخّراً،
     * قرأ أجرةَ مدينةٍ وعنوانُه في أخرى.**
     *
     * **والمالُ محميٌّ في المحرّك** (يُعاد الحسابُ عند الإرسال) —
     * **والمكسورُ كان ما يُقرأ**: **ورقمٌ يُعرَض ولا يصمد وعدٌ لا
     * يُوفى.**
     */
    @Test
    fun `AB-03 تسعيرةٌ متأخّرةٌ لا تكتب فوق عنوانٍ أحدث`() {
        val src = read(cart)
        assertTrue(
            "**سقط حارسُ النقطة في ردّ التسعير**",
            src.contains("if (lastPoint != point) return@onSuccess"),
        )
        assertTrue(
            "**عطبُ نداءٍ لعنوانٍ غادره يُعرَض على عنوانِه**",
            src.contains("if (lastPoint != point) return@onFailure"),
        )
        // **والنقطةُ تُلتقط قبل النداء لا بعده** — **ولو قُرئت بعده
        // لَقرأت أحدثَ ما صار.**
        val launch = src.indexOf("viewModelScope.launch {", src.indexOf("fun quote("))
        val capture = src.indexOf("val point = \"\$lat,\$lng\"")
        assertTrue("**النقطةُ تُلتقط بعد إطلاق النداء**", capture in 1 until launch)
    }

    /**
     * **AB-04 · وحالُ التوفّر محفوظةٌ بمفتاح نقطتها.**
     *
     * **وهي كانت محميّةً قبل هذه الدفعة** — **ويُحرَس ألّا تُفكّ.**
     */
    @Test
    fun `AB-04 حالُ التوفّر تُحفظ بمفتاح نقطتها`() {
        val src = read(precart)
        assertEquals(
            "**تبدّل حفظُ التوفّر بمفتاح النقطة**",
            2,
            Regex("Orderable\\.put\\(it, point").findAll(src).count(),
        )
        assertTrue("**سقط حارسُ النداء الجاري**", src.contains("if (inFlight == point) return"))
    }

    // ═════════════ AB-05 · AB-06 — ساعةُ الجهاز ═════════════

    /**
     * **AB-06 · ومهلةُ إرسال الموضع بساعةٍ لا ترجع.**
     *
     * **وُجد بالمحاولة**: **كانت `currentTimeMillis`** — **وهي ساعةُ
     * الحائط.** **ومن أرجع ساعةَ جهازه يوماً جعل الفرقَ سالباً** —
     * **فيُقرأ «لم تمضِ المهلة» أبداً، فيصمت موقعُه ويظهر واقفاً وهو
     * يسير.**
     */
    @Test
    fun `AB-06 مهلةُ إرسال الموضع لا تُقاس بساعة الحائط`() {
        val src = read(locSvc)
        assertTrue(
            "**عادت ساعةُ الحائط إلى مهلة الإرسال**",
            src.contains("val now = SystemClock.elapsedRealtime()"),
        )
        val send = src.indexOf("private fun send(point: Location)")
        val after = src.substring(send, minOf(send + 1200, src.length))
        assertFalse(
            "**ساعةُ الحائط في حارس التردّد**",
            after.contains("System.currentTimeMillis()"),
        )
    }

    /** **AB-05 · وعمرُ جلبةِ المسارات كذلك.** */
    @Test
    fun `AB-05 عمرُ جلبة المسارات لا يُقاس بساعة الحائط`() {
        val src = read(drvOrders)
        assertTrue(
            "**عادت ساعةُ الحائط إلى عمر الجلبة**",
            src.contains("choicesAtMs = android.os.SystemClock.elapsedRealtime()"),
        )
        assertFalse(
            "**عمرُ الجلبة بساعة الحائط**",
            src.contains("System.currentTimeMillis() - choicesAtMs"),
        )
    }

    /**
     * **وسلطةُ الزمن للمحرّك** — **ولا يُقرَّر بساعة الجهاز أنّ عرضاً
     * سارٍ أو أنّ المنصّة مفتوحة.**
     *
     * **وحالُ العرض تُقرأ من الردّ** (`live`) — **لا تُحسب من
     * `ends_at` وساعةِ الجهاز.**
     */
    @Test
    fun `AB-05 حالُ العرض تُقرأ من المحرّك لا من ساعة الجهاز`() {
        val model = read("shared/src/main/kotlin/com/rahalgo/shared/model/Shop.kt")
        assertTrue("**ذهب حقلُ السريان من الردّ**", model.contains("val live: Boolean = false"))
        // **ولا تُحسب الحالُ في الشاشة من الزمن.**
        val offers = read("ui/src/main/kotlin/com/rahalgo/ui/Offers.kt")
        assertFalse(
            "**حُسبت حالُ العرض من ساعة الجهاز**",
            offers.contains("System.currentTimeMillis()") || offers.contains("Instant.now()"),
        )
    }

    // ═════════════ AB-01 · AB-02 — الإرسالُ مرّةً ═════════════

    /**
     * **AB-01 · AB-02 · وضغطتان طلبٌ واحد، وردٌّ ضائعٌ يُستردّ.**
     *
     * **وثلاثةُ أقفال**: **رايةُ انشغالٍ في الشاشة · مفتاحُ محاولةٍ
     * على القرص · وحارسُ التكرار في المحرّك.**
     */
    @Test
    fun `AB-01 الإرسالُ بثلاثة أقفال`() {
        val src = read(cart)
        assertTrue("**سقطت رايةُ الانشغال**", src.contains("if (busy) return"))
        assertTrue(
            "**سقط مفتاحُ المحاولة**",
            src.contains("Attempt.key(com.rahalgo.ui.Attempt.ORDER)"),
        )
        // **والمفتاحُ يُمحى بعد النجاح وحدَه.**
        val clear = src.indexOf("Attempt.clear(com.rahalgo.ui.Attempt.ORDER)")
        assertTrue("**ذهب محوُ المفتاح**", clear > 0)
        assertTrue("**المحوُ قبل النداء**", src.indexOf("api.createOrder(") < clear)
    }

    // ═════════════ AB-35 · لا تُبنى حقيقةُ مالٍ من ذاكرة ═════════════

    /**
     * **AB-35 · ولا تتخطّى ذاكرةُ الجهاز التحقّقَ الأخير.**
     *
     * **والجسمُ المرسَلُ بلا ثمن** — **والسعرُ يُحسَب في المحرّك.**
     */
    @Test
    fun `AB-35 الطلبُ المرسَلُ بلا ثمنٍ من الجهاز`() {
        val api = read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
        val at = api.indexOf("data class NewOrder(")
        val body = api.substring(at, api.indexOf(")\n", at))
        for (banned in listOf("price", "total", "subtotal", "deliveryFee", "discount")) {
            assertFalse("**حقلُ مالٍ في جسم الطلب**: " + banned, body.contains(banned))
        }
    }

    // ═════════════ AB-36 · العودةُ لا تُضاعف ═════════════

    /**
     * **AB-36 · والعودةُ من الخلفيّة تُنعش ما شاخ وحدَه.**
     *
     * **ولا نداءَ في كلّ عودةٍ مهما قصرت** — **والمراقبُ يُنزَع عند
     * الخروج فلا يتراكم.**
     */
    @Test
    fun `AB-36 العودةُ تُنعش ما شاخ ولا تتراكم`() {
        val main = read("app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt")
        assertTrue("**صارت العودةُ تُنعش كلَّ شيء**", main.contains("Serving.refreshIfStale(scope)"))
        assertTrue("**لا يُنزَع المراقب**", main.contains("onDispose { owner.lifecycle.removeObserver(obs) }"))
    }
}

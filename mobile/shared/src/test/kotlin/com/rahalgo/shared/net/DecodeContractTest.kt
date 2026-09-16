package com.rahalgo.shared.net

import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Platform
import java.io.File
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقدُ الفكّ — ما يُقبَل وما يُردّ** (`SC`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع — دورتان
 *
 * **١ · قِيس على المحاكي**: `/public/platform` **يردّ `"logo": null`**
 * حين لا صورةَ مرفوعة — **ونوعُه في النموذج كان `String` غيرَ قابلٍ
 * للعدم** — **فسقط فكُّ الرسالة كلِّها، ومعها `otp_login`، فاختفى بابُ
 * الدخول بالرمز.**
 *
 * **٢ · وأُصلح بتساهلٍ عامّ** (`coerceInputValues`) — **فأصلح الظاهرَ
 * وأخفى ما هو أخطر**: **وأكثرُ حقولنا لها قيمٌ افتراضيّة، فكلُّ `null`
 * واصلةٍ تصير قيمةً مقبولةً صامتة:**
 *
 *	`"available": null` ⇒ **`true`** — **صنفٌ لا يُباع يُعرَض للطلب**
 *	`"price": null`     ⇒ **صفر**   — **مالٌ يُقرأ صفراً**
 *	`"otp_login": null` ⇒ **`true`** — **بابٌ يُعرَض وقد أُغلق**
 *
 * # والقاعدةُ الآن
 *
 * **ما يجوز أن يكون عدماً يُقال في نوعه** — **وما لا يجوز يُردّ ولا
 * يُخترَع له معنى.** **وردٌّ لا يُفهَم خبرُ عطبٍ بين محرّكٍ وحزمة**،
 * **وابتلاعُه يجعل العطبَ صامتاً حتّى يظهر في مالٍ أو في صنفٍ يُباع
 * وهو غيرُ متاح.**
 */
class DecodeContractTest {

    /** **نسخةُ الإعداد كما في `ApiClient` حرفاً.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    /** **ردٌّ حقيقيٌّ من التجهيز** (٢٠٢٦-٠٩-١٦) — **قُصّ ولم يُبدَّل.** */
    private val realPlatform = """
        {"address":"","app_url":"/api/v1/public/app","auth_bg":null,"auth_bg_dim":70,
         "auth_bg_mobile":null,"join_open":false,"location":"","logo":null,"name":"",
         "ordering":{"hours_enforced":false,"message":"","ordering_available":true},
         "otp_login":true,"password_min_length":8,"show_login":true,"show_shop":true,
         "signup_verify":false,"site_bg":null,"site_bg_blur":0,"site_bg_dim":0,
         "site_bg_mobile":null,"social":{},"support_phone":""}
    """.trimIndent()

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

    // ═════════════ SC-01 · SC-02 · SC-03 ═════════════

    /** **SC-01 · SC-OTP-01 · والردُّ الحقيقيُّ يُفكّ كما هو.** */
    @Test
    fun `SC-01 ردُّ المنصّة الحقيقيُّ يُفكّ`() {
        val p = json.decodeFromString<Platform>(realPlatform)
        assertEquals("", p.name)
        assertEquals(8, p.passwordMinLength)
    }

    /** **SC-02 · وشعارٌ عدمٌ يُقرأ عدماً — ولا يُسقط الفكّ.** */
    @Test
    fun `SC-02 شعارٌ عدمٌ يُقرأ عدماً`() {
        assertNull(
            "**لم يُقرأ الشعارُ الغائبُ عدماً**",
            json.decodeFromString<Platform>(realPlatform).logo,
        )
    }

    /** **SC-03 · وحقلٌ غائبٌ أصلاً يأخذ افتراضَه.** */
    @Test
    fun `SC-03 حقلٌ غائبٌ يأخذ افتراضَه`() {
        val p = json.decodeFromString<Platform>("""{"name":"س"}""")
        assertEquals("س", p.name)
        assertNull(p.logo)
        assertTrue("**تبدّل افتراضُ بابِ الرمز**", p.otpLogin)
    }

    // ═════════════ SC-OTP-02 · SC-OTP-03 · SC-OTP-04 ═════════════

    /**
     * **SC-OTP · وبابا الدخول يتبعان المحرّك لا الشعار.**
     *
     * **وهو العطبُ الذي وقع**: **شعارٌ عدمٌ أسقط الردَّ كلَّه فاختفى
     * بابُ الرمز وبقي بابُ كلمة المرور وحدَه.**
     */
    @Test
    fun `SC-OTP بابا الدخول يتبعان إعداد المحرّك`() {
        assertTrue(
            "**ضاع بابُ الرمز مع شعارٍ عدم**",
            json.decodeFromString<Platform>(realPlatform).otpLogin,
        )
        // **وإذا أطفأه المحرّكُ أُطفئ** — **ولا يُعرَض بابٌ أُغلق.**
        assertFalse(
            "**عُرض بابُ رمزٍ أطفأه المحرّك**",
            json.decodeFromString<Platform>("""{"logo":null,"otp_login":false}""").otpLogin,
        )
    }

    // ═════════════ SC-05 — المعرّفُ لا يُخترَع ═════════════

    /**
     * **SC-05 · ومعرّفٌ يصل عدماً يُردّ.**
     *
     * **ولو قُبل لصار صنفاً بلا هويّةٍ في سلّة** — **يُطلب فلا يُعرَف
     * ما هو.**
     */
    @Test
    fun `SC-05 معرّفٌ يصل عدماً يُردّ`() {
        assertThrows(SerializationException::class.java) {
            json.decodeFromString<Item>("""{"id":null,"name":"شاورما","price":5000}""")
        }
    }

    // ═════════════ SC-06 · SC-07 · SC-08 — المالُ لا يُخترَع ═════════════

    /**
     * **SC-07 · وثمنٌ يصل عدماً يُردّ ولا يُقرأ صفراً.**
     *
     * **وصفرٌ مخترَعٌ في سعرٍ أخطرُ من رسالةِ عطب**: **يُعرَض الصنفُ
     * مجّاناً فيُطلَب، ثمّ يُحاسَب أحدٌ على الفرق.**
     */
    @Test
    fun `SC-07 ثمنٌ يصل عدماً يُردّ ولا يصير صفراً`() {
        assertThrows(SerializationException::class.java) {
            json.decodeFromString<Item>("""{"id":"a","name":"شاورما","price":null}""")
        }
    }

    /** **SC-08 · ورقمٌ مشوَّهٌ يُردّ.** */
    @Test
    fun `SC-08 رقمٌ مشوَّهٌ يُردّ`() {
        assertThrows(SerializationException::class.java) {
            json.decodeFromString<Item>("""{"id":"a","name":"ش","price":"كثير"}""")
        }
    }

    // ═════════════ SC-11 — التوفّرُ لا يُخترَع متساهلاً ═════════════

    /**
     * **SC-11 · وتوفّرٌ يصل عدماً يُردّ ولا يصير «متاحاً».**
     *
     * **وافتراضُ `available` في النموذج `true`** — **وهو صوابٌ لحقلٍ
     * غائبٍ في محرّكٍ أقدم**، **وكذبٌ لحقلٍ وصل `null`.**
     */
    @Test
    fun `SC-11 توفّرٌ يصل عدماً يُردّ ولا يصير متاحاً`() {
        assertThrows(SerializationException::class.java) {
            json.decodeFromString<Item>("""{"id":"a","name":"ش","price":1,"available":null}""")
        }
        // **والغيابُ يبقى كما كان** — **عقدُ التوافق مع محرّكٍ أقدم.**
        assertTrue(json.decodeFromString<Item>("""{"id":"a","name":"ش","price":1}""").available)
    }

    // ═════════════ SC-09 · SC-10 · SC-12 — لا حالَ تُخترَع ═════════════

    /**
     * **SC-09 · SC-10 · SC-12 · ولا `enum` في عقود الشبكة.**
     *
     * **والحالُ والدورُ ووسيلةُ الدفع نصوصٌ** — **فقيمةٌ جديدةٌ من
     * محرّكٍ أحدثَ تصل كما هي، والشاشةُ تقول إنّها لا تعرفها**
     * (`CartChanges` · `OfferStatus` · `Engagement.route`).
     *
     * **ولو كانت `enum` بقيمةٍ افتراضيّةٍ لَصارت المجهولةُ حالاً
     * صالحةً صامتة** — **«مقبول» أو «مدفوع» أو «متاح».**
     */
    @Test
    fun `SC-09 لا enum في عقود الشبكة`() {
        val dir = File(mobileRoot(), "shared/src/main/kotlin/com/rahalgo/shared/model")
        val enums = dir.listFiles().orEmpty().filter {
            it.name.endsWith(".kt") &&
                Regex("@Serializable[\\s\\S]{0,40}enum class").containsMatchIn(it.readText())
        }
        assertTrue(
            "**صار في عقود الشبكة `enum`** — **وقيمةٌ جديدةٌ من المحرّك تُسقط الفكَّ " +
                "أو تُخترَع حالاً صالحة**: " + enums.joinToString { it.name },
            enums.isEmpty(),
        )
    }

    // ═════════════ ولا تساهلَ عامّ ═════════════

    /**
     * **ولا يُعاد التساهلُ العامُّ إلى مُفكِّك الشبكة.**
     *
     * **ومن أعاده أعاد ابتلاعَ كلِّ `null` بقيمةٍ معقولة** — **وهو
     * القرارُ الذي اتُّخذ بعد قياسٍ لا بذوق.**
     */
    @Test
    fun `لا تساهلَ عامٌّ في مُفكِّك الشبكة`() {
        val src = File(
            mobileRoot(),
            "shared/src/main/kotlin/com/rahalgo/shared/net/ApiClient.kt",
        ).readText()
        val live = src.lines().filterNot { it.trimStart().startsWith("//") }.joinToString("\n")
        assertFalse(
            "**عاد `coerceInputValues` إلى مُفكِّك الشبكة**",
            live.contains("coerceInputValues"),
        )
    }

    /** **والعقدُ القديمُ باقٍ**: **حقلٌ لا نعرفه يُتجاهَل.** */
    @Test
    fun `حقلٌ جديدٌ لا يُسقط تطبيقاً قديماً`() {
        val p = json.decodeFromString<Platform>(
            """{"name":"س","logo":null,"brand_new_field":{"x":1},"otp_login":false}""",
        )
        assertEquals("س", p.name)
        assertFalse(p.otpLogin)
    }
}

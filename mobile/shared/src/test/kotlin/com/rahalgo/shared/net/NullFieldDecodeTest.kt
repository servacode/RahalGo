package com.rahalgo.shared.net

import com.rahalgo.shared.model.Platform
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **و`null` صريحةٌ لا تُسقط الفكّ** (`B9-D`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما قِيس على المحاكي
 *
 * **فُتح تطبيقُ السائق على محرّك التجهيز فكُتب في سجلّه**:
 * `تعذّرت قراءة حال المنصّة` — **مع أثرِ فكٍّ كامل.**
 *
 * **والسببُ**: `/public/platform` **يردّ `"logo": null`** حين لا صورةَ
 * مرفوعة، **ونوعُ `logo` في النموذج `String` غيرُ قابلٍ للعدم.**
 *
 * **و`explicitNulls = false` تحكم الكتابةَ لا القراءة** — **فالحقلُ
 * الواصلُ `null` يُسقط فكَّ الرسالة كلِّها**، **ولو كانت له قيمةٌ
 * افتراضيّة.**
 *
 * # وما ضاع بذلك
 *
 * **حالُ المنصّة تحمل `otp_login`** — **فسقوطُها أخفى بابَ الدخول
 * برمز التحقّق**، **وبقي بابُ كلمة المرور وحدَه.** **وقِيس: عاد
 * البابان بعد الإصلاح.**
 *
 * # ولماذا فحصٌ سلوكيٌّ لا قراءةُ سطر
 *
 * **وقراءةُ `coerceInputValues = true` في المصدر تُثبت أنّ السطرَ
 * مكتوب** — **لا أنّ الردَّ يُفكّ.** **فيُفكّ ردٌّ حقيقيٌّ فيه
 * `null`.**
 */
class NullFieldDecodeTest {

    /** **نسخةُ الإعداد كما في `ApiClient` حرفاً.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    /** **ردٌّ حقيقيٌّ من التجهيز** — **قُصّ ولم يُبدَّل.** */
    private val real = """
        {"address":"","app_url":"","auth_bg":null,"auth_bg_dim":70,
         "auth_bg_mobile":null,"join_open":false,"location":"",
         "logo":null,"name":"","otp_login":true}
    """.trimIndent()

    @Test
    fun `حقلٌ يصل عدماً يأخذ قيمتَه الافتراضيّةَ ولا يُسقط الفكّ`() {
        val p = json.decodeFromString<Platform>(real)
        assertEquals("**لم تُقرأ الصورةُ الغائبةُ فراغاً**", "", p.logo)
        assertEquals("", p.name)
        // **وما يحمله الردُّ فعلاً يُقرأ كما هو** — **والتساهلُ لا
        // يبتلع قيمةً موجودة.**
        assertTrue("**ضاع بابُ الدخول برمز التحقّق**", p.otpLogin)
    }

    /**
     * **وكلُّ حقلٍ نصّيٍّ قد يصل عدماً** — **فالمنصّةُ الجديدةُ بلا
     * صورةٍ ولا عنوانٍ ولا موضع.**
     */
    @Test
    fun `منصّةٌ بلا شيءٍ مرفوعٍ تُقرأ كلُّها`() {
        val bare = """{"address":null,"app_url":null,"location":null,"logo":null,"name":null}"""
        val p = json.decodeFromString<Platform>(bare)
        assertEquals("", p.logo)
        assertEquals("", p.name)
    }

    /**
     * **والعقدُ القديمُ باقٍ**: **حقلٌ لا نعرفه يُتجاهَل** — **ومحرّكٌ
     * أحدثُ من الحزمة لا يُسقط تطبيقاً منشوراً.**
     */
    @Test
    fun `حقلٌ جديدٌ لا يُسقط تطبيقاً قديماً`() {
        val future = """{"name":"س","logo":null,"brand_new_field":{"x":1},"otp_login":false}"""
        val p = json.decodeFromString<Platform>(future)
        assertEquals("س", p.name)
        assertEquals(false, p.otpLogin)
    }
}

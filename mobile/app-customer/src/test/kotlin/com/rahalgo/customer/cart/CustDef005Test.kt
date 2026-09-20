package com.rahalgo.customer.cart

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-005 · CAF-08 — المعروضُ من تسعيرة المحرّك لا من سعرِ السلّة المخزَّن**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **السلّةُ كانت تعرض `Cart.subtotal`** — مجموعَ `unitPrice` المحفوظِ ساعةَ
 * الإضافة، **وهو يبقى بعد إعادة التشغيل.** فسعرٌ تبدّل في المحرّك بعد الإضافة
 * يجعل **المعروضَ غيرَ المحسوب**: يرى الزبونُ مبلغاً ويُحاسَب بآخر.
 *
 * **وأوّلُ تسعيرةٍ كانت بلا مقارنة** (`expected = null` حين `priced == null`)،
 * **فبوّابةُ المراجعة لا تُفتح في أوّل فتحةٍ للسلّة** — والمسارُ الشائعُ (يفتح
 * السلّةَ ثمّ يُرسل) يمرّ بلا أن يُبرَز تبدّلُ السعر.
 *
 * # الحقيقةُ من المحرّك
 *
 * **المحرّكُ يحسب سعرَ البيع عند الطلب** (`priceItems` — لا يقرأ عموداً مخزَّناً)،
 * **والتسعيرةُ والإنشاءُ يستعملانه معاً** — فمجموعُ التسعيرة (`Quote.subtotal`)
 * هو المحاسَبُ به بعينِه. **والزبونُ لا يرسل سعراً** (`CartLine`)، فلا يؤثّر في
 * المبلغ. **فالعطبُ عرضٌ وبوّابةٌ في الجهاز، لا ثغرةُ خادمٍ ولا تلاعبُ عميل.**
 *
 * # العقدُ بعد الإصلاح
 *
 * **المعروضُ من `priced` متى وُجدت** (المجموعُ والإجماليّ)، و`Cart.subtotal`
 * احتياطٌ قبل أوّل تسعيرةٍ لا غير. **وكلُّ تسعيرةٍ تُقارَن** — حتّى الأولى —
 * فسعرٌ تبدّل يدخل بوّابةَ المراجعة. **ولا يُرسَل طلبٌ قبل أن تجهز التسعيرةُ
 * الموثوقة** (`vm.priced != null`).
 *
 * # ولماذا حراسةُ مصدرٍ لا رسمُ شاشة
 *
 * **العطبُ في القرار**: أيُّ رقمٍ يُعرَض ومتى يُقارَن ومتى يُقبَل الإرسال —
 * **لا في البكسل.** والشاشةُ Compose تحتاج جهازاً، **والقرارُ نصٌّ يُقاس بلا
 * جهاز** (على منوال `CustDef004Test`/CU-CHAT-16).
 */
class CustDef005Test {

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
        return f.readText().replace("\r\n", "\n")
    }

    /** **جسدُ مقطعٍ** — من علامةٍ إلى أوّلِ نهايةٍ بعدها. */
    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val cart = "app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt"

    private fun quoteBody() = body(read(cart), "fun quote(", "\n    fun send(", "\n    fun ")

    // ══════════════════════════════════════════════════════════════════
    // **العرضُ من تسعيرة المحرّك**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-005-01 · المجموعُ المعروضُ من `priced.subtotal` متى وُجدت.** */
    @Test
    fun displayedSubtotalUsesQuote() {
        assertTrue(
            "**المجموعُ يُعرض من `Cart.subtotal` المخزَّن لا من تسعيرة المحرّك**",
            read(cart).contains("money(q?.subtotal ?: Cart.subtotal)"),
        )
    }

    /** **CUST-DEF-005-02 · الإجماليُّ من مجموع المحرّك لا من سعرِ السلّة المخزَّن.** */
    @Test
    fun displayedTotalUsesQuote() {
        val src = read(cart)
        assertTrue(
            "**الإجماليُّ لا يُشتقّ من `q.subtotal`**",
            src.contains("money(q.subtotal + fee - (cut?.discount ?: 0))"),
        )
        assertFalse(
            "**الإجماليُّ ما زال يُشتقّ من `Cart.subtotal` المخزَّن (العطب)**",
            src.contains("money(Cart.subtotal + fee"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **المقارنةُ من أوّل تسعيرة**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-005-03 · أسعارُ الأسطر تُرسَل للمقارنة دائماً — حتّى أوّلَ تسعيرة.** */
    @Test
    fun firstQuoteSendsExpectedLines() {
        val b = quoteBody()
        assertTrue(
            "**`expected` لا يُبنى دائماً — أوّلُ تسعيرةٍ بلا مقارنة**",
            b.contains("val expected: Map<String, Any> = buildMap {"),
        )
        assertTrue(
            "**أسعارُ الأسطر لا تُرسَل للمقارنة**",
            b.contains("put(\"lines\", Cart.lines.associate { it.item.id to it.unitPrice })"),
        )
        assertFalse(
            "**بقيت المقارنةُ مشروطةً بوجود تسعيرةٍ سابقة (العطب): `if (seen == null)` تُلغيها**",
            b.contains("if (seen == null) {\n            null"),
        )
    }

    /** **CUST-DEF-005-04 · ولا تُخترَع أجرةُ توصيلٍ لم تُعرَض في أوّل فتحة.** */
    @Test
    fun firstQuoteDoesNotInventDeliveryFee() {
        assertTrue(
            "**أجرةُ التوصيل تُرسَل بلا شرطٍ — تُخترَع مقارنةٌ لم تُعرَض**",
            quoteBody().contains("if (seen != null) put(\"delivery_fee\", seen.deliveryFee)"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **بوّابةُ الإرسال — لا طلبَ قبل تسعيرةٍ موثوقة**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **CUST-DEF-005-05 · لا يُرسَل قبل أن تجهز التسعيرةُ الموثوقة.**
     *
     * **بوّابةٌ قائمةٌ قبل الإصلاح** — تُثبَّت لئلّا تسقط سهواً فيصير
     * `Cart.subtotal` الاحتياطيُّ مبلغاً يُجيز الإرسال.
     */
    @Test
    fun submitBlockedUntilQuoteReady() {
        assertTrue(
            "**الإرسالُ لا يشترط تسعيرةً موثوقة (`vm.priced != null`)**",
            read(cart).contains("vm.priced != null &&"),
        )
    }
}

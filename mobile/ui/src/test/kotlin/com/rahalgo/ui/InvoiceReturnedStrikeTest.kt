package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المرتجعُ في الفاتورة مشطوبٌ باهتٌ، وكلمتُه عريضةٌ بلون الخطر — كالويب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك: ما في تطبيق الزبون والمتجر يطابق الموقع.)
 *
 * كان السطرُ الذي رجّعه الزبونُ بعد الاستلام يظهر في تطبيقات الزبون والسائق والمتجر
 * **رماديّاً بخطٍّ عاديٍّ بلا شطب**، وكلمتُه «رجّعه الزبون» بلا عَرض؛ والويبُ
 * (`web/packages/ui/src/InvoiceLines.tsx`) يشطبه ويُعرّض كلمته بلون الخطر.
 * **والمكوّنُ واحدٌ مشترك** (`InvoiceLines` في `Invoice.kt`)، فإصلاحٌ واحدٌ يغطّي الثلاثة.
 *
 * **والرسمُ نفسُه لا يُختبر في JVM بلا Robolectric**، فتُقرأ الشيفرةُ كما في بقيّة
 * اختبارات هذه الوحدة، ويُثبَّت شكلُ الويب مرجعاً.
 */
class InvoiceReturnedStrikeTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "ui").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val invoice = "ui/src/main/kotlin/com/rahalgo/ui/Invoice.kt"

    /**
     * **متنُ دالةٍ في أعلى الملفّ** — من ترويستها إلى أوّل قوسٍ مغلقٍ في أوّل السطر
     * (ما بداخلها مزاحٌ بمسافات، فلا يلتبس به).
     */
    private fun body(src: String, header: String): String {
        val start = src.indexOf(header)
        assertTrue("**الدالةُ غائبة**: " + header, start >= 0)
        val end = src.indexOf("\n}", start)
        assertTrue("**لا نهايةَ للدالة**: " + header, end > start)
        return src.substring(start, end)
    }

    /**
     * **السطرُ المرتجع يصل الصفَّ مشطوباً وباهتاً.** الباهتُ بـ`bold = !l.returned`
     * (رماديٌّ بخطٍّ عاديّ)، والشطبُ بـ`strikethrough = l.returned`.
     */
    @Test
    fun returnedLineReachesTheRowStruckAndMuted() {
        val lines = body(read(invoice), "fun InvoiceLines(")
        assertTrue(
            "**السطرُ المرتجعُ لا يُمرَّر إلى `InvoiceRow` مشطوباً** (`strikethrough = l.returned`)",
            lines.contains("strikethrough = l.returned"),
        )
        assertTrue(
            "**المرتجعُ لم يعد باهتاً** — يجب `bold = !l.returned` (رماديٌّ بخطٍّ عاديّ)",
            lines.contains("bold = !l.returned"),
        )
    }

    /**
     * **الصفُّ يشطب نصَّه وحدَه عند الطلب لا المبلغ** — كالويب: `line-through` على نصّ
     * السطر، و`Money` خارجَه. فالشطبُ قبل كتلة المبلغ، ويظهر مرّةً واحدة.
     */
    @Test
    fun rowStrikesItsLabelOnlyWhenAsked() {
        val row = body(read(invoice), "fun InvoiceRow(")
        assertTrue(
            "**`InvoiceRow` بلا خيار شطب** (`strikethrough: Boolean = false`)",
            row.contains("strikethrough: Boolean = false"),
        )
        assertTrue(
            "**النصُّ لا يُشطب حين يُطلب** (`textDecoration` بـ`LineThrough` عند `strikethrough`)",
            row.contains("textDecoration = if (strikethrough) TextDecoration.LineThrough else null"),
        )
        val strike = row.indexOf("TextDecoration.LineThrough")
        val amountBlock = row.indexOf("if (amount != null)")
        assertTrue("**كتلةُ المبلغ غائبة**", amountBlock > 0)
        assertTrue("**الشطبُ ليس على نصّ السطر** — يجب قبل كتلة المبلغ", strike in 0 until amountBlock)
        assertTrue(
            "**المبلغُ مشطوبٌ أيضاً** — الويبُ يشطب النصَّ وحدَه",
            Regex("TextDecoration\\.LineThrough").findAll(row).count() == 1,
        )
    }

    /** **كلمةُ «رجّعه الزبون» عريضةٌ بلون الخطر** — كـ`font-bold text-danger` في الويب. */
    @Test
    fun returnedWordIsBoldInTheDangerColour() {
        val lines = body(read(invoice), "fun InvoiceLines(")
        assertTrue("**كتلةُ المرتجع غائبة**", lines.contains("if (l.returned) {"))
        val word = lines.substringAfter("if (l.returned) {").substringBefore("if (l.note.isNotBlank())")
        assertTrue("**الكلمةُ لا تُكتب**", word.contains("R.string.inv_returned"))
        assertTrue("**الكلمةُ ليست بلون الخطر**", word.contains("color = Rahal.colors.danger"))
        assertTrue("**الكلمةُ ليست عريضة**", word.contains("fontWeight = FontWeight.Bold"))
    }

    /**
     * **والثلاثةُ تستعمل المكوّنَ المشترك نفسَه** — فلا نسخةَ خاصّةً تفوتها الإصلاحات:
     * الزبون (بطاقةُ الطلب) والسائق (مشوارُه) والمتجر (الطلباتُ والسجلّ).
     */
    @Test
    fun threeAppsDrawTheSharedInvoice() {
        listOf(
            "app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt",
            "app-driver/src/main/kotlin/com/rahalgo/driver/trip/TripScreen.kt",
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/orders/OrdersScreen.kt",
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/orders/HistoryScreen.kt",
        ).forEach { rel ->
            assertTrue(
                "**الشاشةُ لا تستعمل `com.rahalgo.ui.InvoiceLines` المشتركة**: " + rel,
                read(rel).contains("com.rahalgo.ui.InvoiceLines("),
            )
        }
    }

    /**
     * **مرجعُ الويب ما زال هو** — السطرُ المرتجع `text-ink-muted line-through` وكلمتُه
     * `font-bold text-danger`. فإن تبدّل هناك وجب أن يتبعه التطبيق هنا عن قصد.
     */
    @Test
    fun webStillDrawsReturnedTheSameWay() {
        val web = File(mobileRoot().parentFile, "web/packages/ui/src/InvoiceLines.tsx")
        assertTrue("**ملفُّ فاتورة الويب غائب**", web.exists())
        val tsx = web.readText().replace("\r\n", "\n")
        assertTrue(
            "**الويبُ لم يعد يشطب السطر المرتجع**",
            tsx.contains("it.returned ?") && tsx.contains("text-ink-muted line-through"),
        )
        assertTrue("**كلمةُ المرتجع في الويب ليست عريضةً بلون الخطر**", tsx.contains("font-bold text-danger"))
    }
}

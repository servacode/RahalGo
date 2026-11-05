package com.rahalgo.rep.clients

import com.rahalgo.shared.rep.RepMerchantDetail
import com.rahalgo.shared.rep.RepMerchantHead
import com.rahalgo.shared.rep.RepStoreCommission
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نسبةُ عمولة المتجر عند المندوب — الكتابةُ والفحصُ والحفظُ** (٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ثلاثةُ عيوبٍ في سطرٍ واحد:**
 *
 * **أ · الحفظُ الناجح لا يبقى** — كان يُكتب في حال السطر المحلّيّة وحدَها، فإذا فتح المندوبُ
 * «ساعات الدوام» أو «الأقسام» ورجع قرأ السطرُ التفصيلَ القديم من الـvm. **وبقي منه طريقٌ
 * ضيّق**: النداءُ نفسُه كان في نطاق السطر (`rememberCoroutineScope`) فإذا فتح المندوبُ صفحةً
 * أخرى قبل الردّ أُلغي ولم تدخل النسبةُ الـvm — فانتقل الحفظُ إلى `viewModelScope`.
 *
 * **ب · الزرُّ يُفعَّل لما يردّه المحرّك** — «7.» والأرقامُ العربيّةُ (٧٫٥)؛ والمحرّكُ
 * (`pricing.ParsePercent2`) يقبل `0`–`9` ونقطةً يتبعها رقمٌ فقط.
 *
 * **ج · الصفرُ «0.0»** على أندرويد القديم (`java.math` قبل OpenJDK) — **ولا يُقلَّد هذا على الجهاز
 * الذي يشغّل الاختبار** (OpenJDK يردّ «0»)، فيُثبَّت الاستثناءُ في المصدر أيضاً (انظر اختبار الأسلاك).
 *
 * **ولا شاشةَ هنا**: الفحصُ على دوالَّ نقيّةٍ، والأسلاكُ تُثبَّت من المصدر كما في بقيّة الحزمة.
 */
class ClientCommissionTest {

    private companion object {
        const val LO = 0.0
        const val HI = 10.0
    }

    /** يكتب نصّاً لاتينيّاً كما يكتبه كيبوردُ الأرقام العربيّة: ٠–٩ و«٫». */
    private fun arabicKeyboard(s: String): String = buildString {
        for (c in s) append(if (c in '0'..'9') '٠' + (c - '0') else if (c == '.') '٫' else c)
    }

    /** والفارسيّ: ۰–۹ و«٫». */
    private fun persianKeyboard(s: String): String = buildString {
        for (c in s) append(if (c in '0'..'9') '۰' + (c - '0') else if (c == '.') '٫' else c)
    }

    @Test
    fun `مولّدا الكيبورد يكتبان ما نظنّه`() {
        assertEquals("٧٫٥", arabicKeyboard("7.5"))
        assertEquals("۷٫۵", persianKeyboard("7.5"))
    }

    // ── ج · الصفر ─────────────────────────────────────────────────────

    @Test
    fun `الصفرُ يُكتب صفراً بلا كسر`() {
        assertEquals("0", pctText(0.0))
        assertEquals("0", pctText(-0.0))
    }

    @Test
    fun `النسبةُ بلا أصفارٍ زائدة`() {
        assertEquals("10", pctText(10.0))
        assertEquals("7.5", pctText(7.5))
        assertEquals("7.25", pctText(7.25))
        assertEquals("0.5", pctText(0.5))
        assertEquals("2", pctText(2.0))
    }

    // ── ب · ما يُكتب وما يُرسَل ───────────────────────────────────────

    @Test
    fun `الأرقامُ العربيّةُ والفارسيّةُ تصير لاتينيّةً والفاصلةُ تصير نقطة`() {
        assertEquals("7.5", normalizePctInput(arabicKeyboard("7.5")))
        assertEquals("7.5", normalizePctInput(persianKeyboard("7.5")))
        assertEquals("10", normalizePctInput(arabicKeyboard("10")))
        assertEquals("7.5", normalizePctInput("7,5"))
        assertEquals("7.5", normalizePctInput("7.5"))
    }

    @Test
    fun `ما يُرسَل لاتينيٌّ ولو كُتب بأرقام الكيبورد العربيّ`() {
        assertEquals("7.5", commissionPayload(arabicKeyboard("7.5"), LO, HI))
        assertEquals("5", commissionPayload(arabicKeyboard("5"), LO, HI))
        assertEquals("10", commissionPayload(arabicKeyboard("10"), LO, HI))
        assertEquals("7.5", commissionPayload(persianKeyboard("7.5"), LO, HI))
        // أرقامٌ عربيّةٌ ونقطةٌ لاتينيّة
        assertEquals("7.5", commissionPayload("٧.٥", LO, HI))
        assertEquals("7.5", commissionPayload("7,5", LO, HI))
        assertEquals("7.5", commissionPayload("7.5", LO, HI))
        assertEquals("0.25", commissionPayload("0.25", LO, HI))
        // المحرّك يقبل «.5» (الجزءُ الصحيحُ الفارغُ صفر)
        assertEquals(".5", commissionPayload(".5", LO, HI))
    }

    @Test
    fun `النقطةُ الأخيرةُ لا تُرسَل`() {
        assertEquals("7", commissionPayload("7.", LO, HI))
        assertEquals("7", commissionPayload(arabicKeyboard("7."), LO, HI))
        // نقطةٌ وحدَها أو نقطتان: لا رقمَ يُرسَل
        assertNull(commissionPayload(".", LO, HI))
        assertNull(commissionPayload("٫", LO, HI))
        assertNull(commissionPayload("7..", LO, HI))
        assertNull(commissionPayload("", LO, HI))
        assertNull(commissionPayload("   ", LO, HI))
    }

    @Test
    fun `ما يردّه المحرّك لا يُفعَّل له الزرّ`() {
        // ثلاثُ منازل — يُردّ ولا يُقرَّب
        assertNull(commissionPayload("7.555", LO, HI))
        assertNull(commissionPayload("1.2.3", LO, HI))
        assertNull(commissionPayload("abc", LO, HI))
        assertNull(commissionPayload("-1", LO, HI))
        assertNull(commissionPayload("7 5", LO, HI))
        assertNull(commissionPayload("1e1", LO, HI))
        // أرقامٌ يقرؤها `BigDecimal` ويرفضها المحرّك (ديفاناغاري) — الشكلُ يُفحص قبله
        assertNull(commissionPayload("१२", LO, HI))
    }

    @Test
    fun `الحدّان من المحرّك لا من التطبيق`() {
        assertEquals("0", commissionPayload("0", LO, HI))
        assertEquals("10", commissionPayload("10", LO, HI))
        assertNull(commissionPayload("10.01", LO, HI))
        assertNull(commissionPayload("11", LO, HI))
        // حدّان عشريّان
        assertNull(commissionPayload("0.4", 0.5, 7.25))
        assertEquals("0.5", commissionPayload("0.5", 0.5, 7.25))
        assertEquals("7.25", commissionPayload("7.25", 0.5, 7.25))
        assertNull(commissionPayload("7.26", 0.5, 7.25))
    }

    // ── أ · الحفظُ يبقى ───────────────────────────────────────────────

    /** تفصيلُ عميلٍ نسبتُه 10 وحدّاها 0 و10 — كما يعرضه المحرّك قبل أن يغيّرها المندوب. */
    private fun detailOf(): RepMerchantDetail = RepMerchantDetail(
        merchant = RepMerchantHead(
            name = "store",
            storeCommissionPercent = 10.0,
            storeCommissionMin = 0.0,
            storeCommissionMax = 10.0,
            ownerPhone = "0912345678",
        ),
        total = 3,
        page = 2,
    )

    @Test
    fun `النسبةُ المحفوظةُ تُكتب في التفصيل فتقرؤها الخانةُ حين تُبنى من جديد`() {
        val before = detailOf()
        val after = before.withStoreCommission(RepStoreCommission(percent = 7.5, min = 0.0, max = 8.0))

        assertEquals(7.5, after.merchant.storeCommissionPercent, 0.0)
        assertEquals(0.0, after.merchant.storeCommissionMin, 0.0)
        assertEquals(8.0, after.merchant.storeCommissionMax, 0.0)
        // ما تفتح عليه الخانةُ وما يقوله السطرُ إذا عاد المندوبُ من صفحةٍ أخرى
        assertEquals("7.5", pctText(after.merchant.storeCommissionPercent))
        // وكلُّ ما عداها كما هو
        assertEquals(before.merchant.copy(storeCommissionPercent = 7.5, storeCommissionMax = 8.0), after.merchant)
        assertEquals(before.summary, after.summary)
        assertEquals(before.orders, after.orders)
        assertEquals(3, after.total)
        assertEquals(2, after.page)
    }

    /**
     * **والحفظُ صار يعيش في الـvm فقد يصل ردُّه بعد أن غادر المندوبُ الصفحةَ** — بل بعد أن أغلق
     * العميلَ وفتح غيرَه. **فالردُّ يدخل التفصيلَ المفتوحَ لصاحبه وحدَه.**
     */
    @Test
    fun `ردُّ الحفظ يدخل التفصيلَ المفتوحَ لصاحبه وحدَه`() {
        val saved = RepStoreCommission(percent = 7.5, min = 0.0, max = 8.0)
        val open = detailOf()

        // العميلُ نفسُه ما زال مفتوحاً (ولو غاب السطرُ خلف «ساعات الدوام»): تدخل النسبةُ وحدّاها
        val updated = detailAfterCommissionSaved(open, openId = "a", savedFor = "a", saved = saved)
        assertEquals(open.withStoreCommission(saved), updated)
        assertEquals(7.5, updated!!.merchant.storeCommissionPercent, 0.0)
        assertEquals(8.0, updated.merchant.storeCommissionMax, 0.0)

        // فتح المندوبُ عميلاً آخرَ قبل الردّ: نسبةُ «أ» لا تُكتب في تفصيل «ب»
        assertSame(open, detailAfterCommissionSaved(open, openId = "b", savedFor = "a", saved = saved))
        // أغلق العميلَ ولم يفتح شيئاً: لا شيءَ يتغيّر
        assertSame(open, detailAfterCommissionSaved(open, openId = "", savedFor = "a", saved = saved))
        // أعاد فتحَه والتفصيلُ بعدُ يُجلب: يبقى فارغاً ويصل الجديدُ من المحرّك
        assertNull(detailAfterCommissionSaved(null, openId = "a", savedFor = "a", saved = saved))
    }

    /**
     * **الأسلاكُ في المصدر** — كبقيّة حزمة الوحدة: السلوكُ الحيُّ يُشهَد على الجهاز، وهذا يمنع أن
     * يُنزَع سلكٌ بإصلاحٍ لاحق فيرجع العطبُ صامتاً.
     */
    @Test
    fun `الأسلاكُ قائمةٌ في المصدر`() {
        val row = clientsSource("ClientCommission.kt")
        val vm = clientsSource("ClientsViewModel.kt")

        // الـvm يكتب النسبةَ المحفوظةَ في التفصيل (للمفتوح وحدَه)، لا السطرُ
        assertTrue(
            "**الـvm لا يكتب النسبةَ في التفصيل**",
            vm.contains("detail = detailAfterCommissionSaved(detail, openId, id, saved)"),
        )
        // وجلبُ صفحةِ طلباتٍ تالية لا يردّها إلى قديمها: رأسُ العميل من الحاليّ لا من لقطةِ ما قبل الانتظار
        assertTrue(
            "**جلبُ المزيد من الطلبات يردّ النسبةَ المحفوظةَ إلى قديمها**",
            vm.contains("merchant = detail?.merchant ?: d.merchant"),
        )

        // الزرُّ وما يُرسَل من الدالة نفسها — لا فحصَ ثانياً يتفرّق عنها
        assertTrue(row.contains("val payload = commissionPayload(text, lo, hi)"))
        assertTrue(row.contains("enabled = payload != null && !saving"))
        assertTrue(row.contains("onValueChange = { v -> text = normalizePctInput(v)"))
        assertFalse("**عاد فحصٌ يدويٌّ بلا تطبيعٍ للأرقام**", row.contains("toBigDecimalOrNull"))

        // الصفرُ مستثنى نصّاً: `java.math` القديمة لا تُقلّد OpenJDK، وهذه الآلةُ لا تُعيد ذاك العطب —
        // فيُثبَّت الاستثناءُ في المصدر لئلّا يُحذف لأنّ الاختبارَ يمرّ بدونه.
        assertTrue("**ذهب استثناءُ الصفر من pctText**", row.contains("if (v == 0.0) \"0\" else"))
    }

    /**
     * **الحفظُ في الـvm لا في نطاق السطر** — فلو عاد النداءُ إلى `rememberCoroutineScope` داخل السطر
     * رجع العطبُ: يضغط المندوبُ «حفظ» ثمّ «ساعات الدوام» قبل الردّ فيُلغى النداءُ ويُعرض له
     * «تعذّر إتمام الطلب» والمحرّكُ قد حفظ، ولا تدخل النسبةُ الـvm.
     *
     * **والحيُّ يُشهَد على الجهاز** (حفظٌ ثمّ فتحُ صفحةٍ ثمّ رجوعٌ)، وهذا يمنع أن يُنزَع السلكُ.
     */
    @Test
    fun `الحفظُ في الـvm لا في نطاق السطر`() {
        val row = clientsSource("ClientCommission.kt")
        val detail = clientsSource("ClientDetail.kt")
        val vm = clientsSource("ClientsViewModel.kt")

        // السطرُ لا يملك النداءَ: لا نطاقَ له، ولا ينادي المحرّكَ بنفسه، ولا ردَّ نجاحٍ يعود إليه من نطاقه
        assertFalse("**عاد نطاقٌ في السطر**", row.contains("rememberCoroutineScope"))
        assertFalse("**عاد إطلاقٌ في السطر**", row.contains("scope.launch"))
        assertFalse("**السطرُ ينادي المحرّكَ بنفسه**", row.contains("RepCommissionApi"))
        assertFalse("**السطرُ ينادي المحرّكَ بنفسه**", row.contains("setStoreCommission("))
        // (بحدّ الكلمة: «detailAfterCommissionSaved» فيها «onSaved» جزءَ كلمة)
        assertFalse("**عاد ردُّ النجاح من نطاق السطر**", Regex("""\bonSaved\b""").containsMatchIn(row))
        // ويقرأ نسبتَه وحدّيها من التفصيل المحفوظ وحدَه — لا نسخةَ محلّيّةً تبقى قديمةً حين يصل الجديد
        assertTrue("**نسبةُ السطر ليست من التفصيل**", row.contains("val current = head.storeCommissionPercent"))
        assertTrue("**حدُّ السطر الأدنى ليس من التفصيل**", row.contains("val lo = head.storeCommissionMin"))
        assertTrue("**حدُّ السطر الأعلى ليس من التفصيل**", row.contains("val hi = head.storeCommissionMax"))
        // ويُسلّم الـvm ما يُرسَل، وإغلاقُ الخانة بعد النجاح هو ما يُسلَّم معه
        assertTrue("**السطرُ لا يُسلّم الحفظَ للـvm**", row.contains("onSave(payload) { editing = false }"))
        // والشاشةُ تصل السطرَ بالـvm: الحفظُ منه، وحالُ «جارٍ» منه
        assertTrue(
            "**الشاشةُ لا تصل السطرَ بحفظ الـvm**",
            detail.contains("vm.saveStoreCommission(storeId, percent, done)"),
        )
        assertTrue("**حالُ «جارٍ» ليست من الـvm**", detail.contains("saving = vm.commissionSaving"))
        assertFalse("**الشاشةُ تكتب في التفصيل بنفسها**", detail.contains("applyStoreCommission"))

        // والـvm: النداءُ في نطاقه، ثمّ التفصيل، ثمّ إغلاقُ الخانة، والإلغاءُ لا يُعرض عطباً، والرايةُ تنزل دائماً
        assertTrue(
            "**رايةُ «جارٍ» تُكتب من خارج الـvm**",
            vm.contains("var commissionSaving by mutableStateOf(false)\n        private set"),
        )
        val save = functionText(vm, "fun saveStoreCommission(")
        assertTrue("**الحفظُ ليس في viewModelScope**", save.contains("viewModelScope.launch"))
        assertTrue("**الحفظُ لا ينادي المحرّكَ**", save.contains("setStoreCommission(id, percent)"))
        assertTrue("**لا يُمنع طلبان معاً**", save.contains("commissionSaving) return"))
        val raised = save.indexOf("commissionSaving = true")
        val launched = save.indexOf("viewModelScope.launch")
        val applied = save.indexOf("applyStoreCommission(id, got)")
        val done = save.indexOf("onDone()")
        assertTrue("**الرايةُ لا تُرفع قبل الإطلاق**", raised in 0 until launched)
        assertTrue("**الردُّ لا يدخل التفصيلَ قبل إغلاق الخانة**", applied in (launched + 1) until done)
        val cancelled = save.indexOf("catch (e: CancellationException)")
        val failed = save.indexOf("catch (e: Exception)")
        val last = save.indexOf("finally")
        assertTrue("**الإلغاءُ يُعرض عطباً**", cancelled in done until failed)
        assertTrue("**الإلغاءُ لا يُعاد رميُه**", save.substring(cancelled, failed).contains("throw e"))
        assertTrue("**الفشلُ لا يُعرض**", failed < last && save.substring(failed, last).contains("Flash.fail(apiError(getApplication(), e))"))
        assertTrue("**الرايةُ لا تنزل في كلّ الأحوال**", last > 0 && save.substring(last).contains("commissionSaving = false"))
    }

    /** نصُّ دالّةٍ في صنفٍ من توقيعها إلى قوسها الخاتم — بإزاحة أربع مسافات كبقيّة الملفّ. */
    private fun functionText(src: String, signature: String): String {
        val start = src.indexOf(signature)
        assertTrue("**لا أجد $signature**", start >= 0)
        val end = src.indexOf("\n    }\n", start)
        assertTrue("**لا أجد نهاية $signature**", end > start)
        return src.substring(start, end)
    }

    private fun clientsSource(file: String): String {
        var dir: File? = File("").absoluteFile
        while (dir != null) {
            val f = File(dir, "src/main/kotlin/com/rahalgo/rep/clients/$file")
            if (f.exists()) return f.readText().replace("\r\n", "\n")
            dir = dir.parentFile
        }
        throw AssertionError("لم أجد " + file + " من " + File("").absolutePath)
    }
}

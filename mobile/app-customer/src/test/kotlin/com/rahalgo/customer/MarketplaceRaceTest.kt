package com.rahalgo.customer

import com.rahalgo.customer.shop.Latest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التسابقُ في شاشة السوق** (`MR`، ٢٠٢٦-٠٩-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بنودُ المصفوفة ١٤ و١٦ و١٧** من عقد دورة حياة السوق:
 *
 * - **١٤**: **عطبُ شبكةٍ والسوقُ عامرةٌ ⇒ حالُ خطأٍ لا حالُ فراغ.**
 * - **١٦**: **تبديلُ الأقسام بسرعةٍ لا يعرض بضاعةً قديمة.**
 * - **١٧**: **تكرارُ «أعد المحاولة» على عطبٍ حقيقيٍّ آمنٌ لا يُكوّم.**
 *
 * # ولماذا لا تُختبَر الشاشةُ نفسُها هنا
 *
 * **حزمةُ الوحدة بلا Robolectric** — و`AndroidViewModel` يطلب
 * `Application`. **فيُنتزع الحكمُ إلى صنفٍ خالصٍ** ([Latest]) **يُختبَر
 * سلوكاً**، **ويُحرَس أنّ نموذجَ العرض يمرّ به فعلاً** — لا أن يُقرأ
 * المصدرُ وحدَه فيُدَّعى ما لم يُقَس.
 */
class MarketplaceRaceTest {

    private companion object {
        fun read(rel: String): String =
            java.io.File(appModuleDir(), rel).readText().replace("\r\n", "\n")

        fun appModuleDir(): java.io.File {
            var dir = java.io.File("").absoluteFile
            repeat(6) {
                if (java.io.File(dir, "build.gradle.kts").exists() &&
                    java.io.File(dir, "src/main").exists()
                ) {
                    return dir
                }
                dir = dir.parentFile ?: return@repeat
            }
            throw IllegalStateException("لم أجد مجلَّد الوحدة")
        }

        const val SCREEN = "src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt"
        const val VM = "src/main/kotlin/com/rahalgo/customer/shop/ShopViewModel.kt"
    }

    /**
     * **شريطُ أقسامٍ مصغَّرٌ يحاكي `loadSection` حرفاً** — ردٌّ بمهلةٍ
     * معلومةٍ يكتب `items`.
     *
     * **و`guard` تفصل ما قبل الإصلاح عمّا بعده**: **فالفحصُ يُري
     * العطبَ قائماً بلا حَكَمٍ وذاهباً معه** — **ولا يُدَّعى إصلاحٌ
     * لم يُرَ سقوطُه.**
     */
    private class Rail(private val scope: CoroutineScope, private val guard: Boolean) {
        val latest = Latest()
        var items = ""
        var busy = false
        private var fetch: Job? = null

        /** **فتحُ قسمٍ** — يملك الدوّارةَ ويطفئها بتذكرةٍ سارية. */
        fun open(id: String, latencyMs: Long) {
            // **والإلغاءُ من الإصلاح كالحَكَم** — **فلا يدخل نموذجَ ما
            // قبله**: **وإلّا ستر الإلغاءُ العطبَ فشهد الشاهدُ زوراً.**
            if (guard) fetch?.cancel()
            val ticket = latest.begin()
            busy = true
            fetch = scope.launch {
                delay(latencyMs)
                if (guard && !latest.isCurrent(ticket)) return@launch
                items = id
                if (!guard || latest.isCurrent(ticket)) busy = false
            }
        }

        /**
         * **سحبةُ الإنعاش** — **تُبطل تذكرةَ القسم ولا تملك الدوّارة.**
         *
         * **و`inherit` تفصل ما قبل الإصلاح عمّا بعده**: **بلا وراثةٍ
         * تبقى الدوّارةُ تدور إلى الأبد.**
         */
        fun pull(id: String, latencyMs: Long, inherit: Boolean) {
            scope.launch {
                fetch?.cancel()
                val ticket = latest.begin()
                delay(latencyMs)
                if (latest.isCurrent(ticket)) items = id
                if (inherit && fetch?.isActive != true) busy = false
            }
        }
    }

    // ── ١٦ · تبديلُ الأقسام بسرعة ────────────────────────────────────

    /** **وتذكرةٌ سبقتها أحدثُ منها لا تكتب حرفا.** */
    @Test
    fun `التذكرةُ القديمةُ لا تكتب`() {
        val latest = Latest()
        val first = latest.begin()
        val second = latest.begin()
        assertFalse("**القديمةُ كتبت**", latest.isCurrent(first))
        assertTrue("**الأحدثُ مُنعت**", latest.isCurrent(second))
    }

    /**
     * **والنقرةُ الأخيرةُ هي التي تُعرض** — **لا الأسرعُ رداً.**
     *
     * **«شاورما» بطيءٌ و«حلويات» سريع**: **ولو حكم الأسرعُ لظهرت
     * شاورما تحت عنوان حلويات.**
     */
    @Test
    fun `النقرةُ الأخيرةُ تحكم لا الأسرعُ رداً`() = runTest {
        val rail = Rail(this, guard = true)
        rail.open("شاورما", latencyMs = 800)
        rail.open("حلويات", latencyMs = 50)
        advanceUntilIdle()
        assertEquals(
            "**بضاعةُ قسمٍ تحت عنوانِ قسمٍ آخر** — **وهو ما يطلبه الزبونُ وهو يظنّ غيرَه**",
            "حلويات", rail.items,
        )
    }

    /** **وشاهدُ العزل**: **العطبُ قائمٌ بلا حَكَم.** */
    @Test
    fun `بلا حَكَمٍ يظهر العطبُ نفسُه`() = runTest {
        val rail = Rail(this, guard = false)
        rail.open("شاورما", latencyMs = 800)
        rail.open("حلويات", latencyMs = 50)
        advanceUntilIdle()
        assertEquals(
            "**لم يعد التسابقُ يقع بلا حَكَم** — **فالفحصُ لا يحرس شيئا**",
            "شاورما", rail.items,
        )
    }

    /** **ونقرُ الرقاقة نفسِها مرّتين لا يترك دوّارةً تدور.** */
    @Test
    fun `نقرتان على القسم نفسِه لا تُبقيان دوّارة`() = runTest {
        val rail = Rail(this, guard = true)
        rail.open("مشاوي", latencyMs = 300)
        rail.open("مشاوي", latencyMs = 300)
        advanceUntilIdle()
        assertEquals("مشاوي", rail.items)
        assertFalse("**دوّارةٌ تدور بلا نداء** — **وهي شاهدُ الجهاز الأوّل**", rail.busy)
    }

    /** **وثلاثُ نقراتٍ متتابعةٍ تنتهي إلى آخرِها.** */
    @Test
    fun `ثلاثُ نقراتٍ تنتهي إلى آخرِها`() = runTest {
        val rail = Rail(this, guard = true)
        rail.open("شاورما", latencyMs = 900)
        rail.open("برغر", latencyMs = 600)
        rail.open("بقالة", latencyMs = 30)
        advanceUntilIdle()
        assertEquals("بقالة", rail.items)
        assertFalse(rail.busy)
    }

    // ── والنموذجُ يمرّ بالحَكَم فعلاً ─────────────────────────────────

    /**
     * **ولا تُكتب `items` إلّا بتذكرة.**
     *
     * **وثلاثةُ أبوابٍ تكتبها**: **فتحُ قسمٍ، وبحثٌ، وسحبةُ إنعاش** —
     * **ومن نسي بابا عاد العطبُ منه.**
     */
    @Test
    fun `كلُّ كاتبٍ لأصنافِ الشاشة يمرّ بالحَكَم`() {
        val vm = read(VM)
        assertTrue("**ذهب الحَكَمُ من نموذج العرض**", vm.contains("Latest()"))
        val writes = Regex("""^\s*(?:items = got|if \(latest\.isCurrent\(ticket\)\) items = got)""",
            RegexOption.MULTILINE).findAll(vm).count()
        assertEquals(
            "**عددُ من يكتب `items` ليس ثلاثةً** — **بابٌ زِيد أو بابٌ فُقد**",
            3, writes,
        )
        assertEquals(
            "**تذاكرُ أقلُّ من الكتّاب** — **وكاتبٌ بلا تذكرةٍ يدهس الأحدث**",
            3, vm.split("latest.begin()").size - 1,
        )
        // **ولا كتابةَ مباشرةً من ردّ الشبكة** — أثرُ ما قبل الإصلاح.
        assertFalse(
            "**كتابةٌ مباشرةٌ من ردّ الشبكة** — **وهي صورةُ العطب قبل إصلاحه**",
            vm.contains("items = api.sectionItems") || vm.contains("items = api.search"),
        )
    }

    /**
     * **ولا تبقى دوّارةٌ بلا صاحب.**
     *
     * **والدوّارةُ تُطفأ بتذكرةٍ سارية** — **فمن أبطل تذكرةَ غيرِه ورث
     * دوّارتَه.** **والسحبةُ تُبطل تذاكرَ من سبقها**: **فإن لم ترثها
     * دارت إلى الأبد** — **وهو ما يمنعه شاهدُ الجهاز «لا دوّارةَ
     * دائمة».**
     */
    @Test
    fun `السحبةُ ترث دوّارةَ من أبطلته`() = runTest {
        val rail = Rail(this, guard = true)
        rail.open("مشاوي", latencyMs = 900)
        rail.pull("مشاوي", latencyMs = 40, inherit = true)
        advanceUntilIdle()
        assertEquals("مشاوي", rail.items)
        assertFalse("**دوّارةٌ بلا صاحبٍ تدور** — **ولا يخرج منها الزبونُ أبدا**", rail.busy)
    }

    /** **وشاهدُ العزل**: **بلا وراثةٍ تدور الدوّارةُ إلى الأبد.** */
    @Test
    fun `بلا وراثةٍ تبقى الدوّارةُ تدور`() = runTest {
        val rail = Rail(this, guard = true)
        rail.open("مشاوي", latencyMs = 900)
        rail.pull("مشاوي", latencyMs = 40, inherit = false)
        advanceUntilIdle()
        assertTrue(
            "**لم تعد الدوّارةُ تعلق بلا وراثة** — **فالفحصُ لا يحرس شيئا**",
            rail.busy,
        )
    }

    /** **والنموذجُ يورّث الدوّارةَ في السحبة فعلاً.** */
    @Test
    fun `السحبةُ في النموذج تُطفئ الدوّارةَ أو تورّثها`() {
        val vm = read(VM)
        val i = vm.indexOf("fun refresh()")
        val j = vm.indexOf("var error by", i)
        check(i > 0 && j > i) { "**لم أجد `refresh`**" }
        val body = vm.substring(i, j)
        assertTrue(
            "**السحبةُ تُبطل تذاكرَ غيرِها ولا ترث دوّارتَهم** — **فتدور بلا نهاية**",
            body.contains("if (fetching?.isActive != true) busy = false"),
        )
        assertTrue(
            "**السحبةُ تجاور النداءَ الجاري ولا تخلُفه** — **فيبقى يعمل وقد أُبطل**",
            body.contains("fetching?.cancel()"),
        )
    }

    // ── ١٧ · تكرارُ إعادة المحاولة ───────────────────────────────────

    /**
     * **وضغطتان على «أعد المحاولة» نداءٌ واحد.**
     *
     * **والردُّ على شبكةٍ منقطعةٍ هو الردُّ عينُه مهما تكرّر** —
     * **فخمسُ ضغطاتٍ تستنزف الحزمةَ ولا تغيّر حرفا**، **وآخرُها
     * انتهاءً يحكم الشاشةَ لا آخرُها ضغطا.**
     */
    @Test
    fun `إعادةُ المحاولة لا تُكوّم نداءات`() {
        val vm = read(VM)
        val i = vm.indexOf("fun load()")
        check(i > 0) { "**ذهبت `load`**" }
        val head = vm.substring(i, minOf(i + 700, vm.length))
        assertTrue(
            "**`load` بلا حارسِ نداءٍ جارٍ** — **وخمسُ ضغطاتٍ خمسةُ نداءات**",
            head.contains("loading?.isActive == true") && head.contains("return"),
        )
        assertTrue(
            "**النداءُ لا يُحفَظ فلا يُعرَف أجارٍ هو** ",
            head.contains("loading = viewModelScope.launch"),
        )
    }

    // ── ١٤ · عطبٌ والسوقُ عامرة ──────────────────────────────────────

    /**
     * **وحالُ الخطأ تُسأل قبل حال الفراغ.**
     *
     * **وترتيبُ الفرعين هو العقدُ كلُّه**: **فلو سُئل الفراغُ أوّلاً
     * لَرأى المنقطعُ «نعمل على إضافة المتاجر»** — **فيُقرأ انقطاعُ
     * شبكتِه سوقاً خاويةً**، **ولا زرَّ إعادةٍ يخرجه منها.**
     */
    @Test
    fun `الخطأُ يُسأل قبل الفراغ`() {
        val screen = read(SCREEN)
        val err = screen.indexOf("vm.error.isNotEmpty()")
        val empty = screen.indexOf("vm.marketEmpty")
        check(err > 0) { "**ذهب فرعُ الخطأ**" }
        check(empty > 0) { "**ذهب فرعُ الفراغ**" }
        assertTrue(
            "**فرعُ الفراغ يسبق فرعَ الخطأ** — **فيُقرأ انقطاعُ الشبكة نفادَ بضاعة**",
            err < empty,
        )
    }

    /**
     * **وعطبٌ والسوقُ عامرةٌ لا يمحو ما في اليد.**
     *
     * **و`sections` تبقى على آخرِ ما جُلب عند فشل الجلب** — **فلا
     * تصير السوقُ خاويةً لأنّ نداءً سقط.**
     */
    @Test
    fun `فشلُ الجلب لا يمحو السوقَ المجلوبة`() {
        val vm = read(VM)
        val i = vm.indexOf("fun load()")
        val j = vm.indexOf("fun openSection", i)
        check(i > 0 && j > i) { "**لم أجد `load`**" }
        val body = vm.substring(i, j)
        val catch = body.indexOf("catch (e: Exception)")
        check(catch > 0) { "**`load` بلا مصيدةٍ**" }
        val tail = body.substring(catch)
        assertFalse(
            "**فشلُ الجلب يمحو الأقسام** — **فتصير السوقُ خاويةً بعطبِ شبكة**",
            tail.contains("sections = emptyList()"),
        )
        assertTrue(
            "**الفشلُ لا يرفع رسالةَ خطأ**",
            tail.contains("error = apiError"),
        )
    }
}

package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وجهةُ الخبر تُفتَح فعلاً** (`DLINK`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطبُ المقيس
 *
 * **و`Opened.pending` لم يكن لها قارئٌ للعرض ولا للمتجر** — **تُنقّى
 * الوجهةُ وتُحفَظ ثمّ لا يقرؤها أحد.** **فبندُ الميدان `B8-13`
 * («يُنقَر ⇒ يُفتَح العرضُ المقصود») لم يكن مستوفىً من طرفٍ إلى طرف**،
 * **ولا خطأَ يظهر**: **يُفتَح البيتُ فيُقرأ «الإشعارُ لا يعمل».**
 *
 * # وما يُقاس هنا
 *
 * **تنقيةُ الوجهة تُنادى**، **والعقودُ التي لا يجوز أن تنقلب صامتةً
 * تُحرَس في مصدرها.** **ولا يُدَّعى أنّ فحصاً هنا نقرةُ إشعارٍ على
 * جهاز** — **تلك في `FIELD-TESTS`.**
 */
class DeepLinkTest {

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
        // **ونهاياتُ الأسطر تُوحَّد قبل المطابقة** — **وجيتٌ على
        // ويندوز يكتب CRLF**، **فمطابقةُ نصٍّ فيه سطرٌ جديدٌ تسقط**:
        // **فيُقرأ العيبُ في المنتج وهو في الفحص.**
        return f.readText().replace("\r\n", "\n")
    }

    private val main =
        "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
    private val mineVm =
        "app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineViewModel.kt"
    private val mineUi =
        "app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineScreens.kt"

    // ═════════════ DLINK-01 · DLINK-02 · المتجر ═════════════

    /**
     * **DLINK-01 · و«المتجر» لم تعد وجهةً** (قرارُ المالك ٢٠٢٦-٠٩-١٥).
     *
     * **ولا شاشةَ متجرٍ عند الزبون** — **قرارُه ٢٠٢٦-٠٨-٠٥**، **وقِيس
     * حيّاً على التجهيز**: **ثلاثةُ أبوابٍ عامّةٍ للمتاجر ⇒ ٤٠٤.**
     *
     * **فتسقط إلى البيت كأيّ وجهةٍ لا تُعرَف** — **ولا تُفتَح شاشةٌ
     * لا وجودَ لها.**
     */
    @Test
    fun `وجهةُ المتجر تسقط إلى البيت`() {
        val id = "6f1c8f31-0b2e-4a77-9d31-2f5e6a1c9b04"
        for (t in listOf("merchant", "MERCHANT", "merchants", " merchant ", "store", "shop")) {
            val d = Engagement.route(t, id)
            assertTrue("**فُتحت وجهةُ متجر**: " + t, d.isHome)
            assertEquals("", d.id)
        }
    }

    /**
     * **DLINK-02 · وخبرٌ قديمٌ يحملها لا يفتح متجراً سبقه ولا يُسقط.**
     *
     * **ولا فرعَ لها في المستهلِك** — **ولا معرّفَ يبقى معلّقاً.**
     */
    @Test
    fun `خبرٌ قديمٌ بوجهة متجرٍ لا يفتح شيئاً`() {
        val src = read(main)
        assertFalse(
            "**بقي فرعُ المتجر** — **ووجهةٌ رُفعت من التنقية لا تبلغه أصلاً**",
            src.contains("Engagement.DEST_MERCHANT"),
        )
        assertFalse(
            "**بقي ثابتُ المتجر في التنقية**",
            read("ui/src/main/kotlin/com/rahalgo/ui/Engagement.kt").contains("DEST_MERCHANT"),
        )
        // **والبيتُ لا معرّفَ له** — **فلا يُفتَح بمعرّفِ متجرٍ سابق.**
        assertTrue(Engagement.route("merchant", "any-old-id").isHome)
        assertEquals("", Engagement.route("merchant", "any-old-id").id)
    }

    // ═════════════ DLINK-03 · العرض ═════════════

    /** **DLINK-03 · ووجهةُ العرض تُقرأ بمعرّفها.** */
    @Test
    fun `وجهةُ العرض تُقرأ كما هي`() {
        val d = Engagement.route("offer", "8b0a8a71-d186-4647-ae3b-9cd3898508bf")
        assertEquals(Engagement.DEST_OFFER, d.type)
        assertEquals("8b0a8a71-d186-4647-ae3b-9cd3898508bf", d.id)
    }

    /** **وتُفتَح في بابها القائم، ويُشار إلى المقصود.** */
    @Test
    fun `العرضُ المقصودُ يُفتَح في قائمة العروض`() {
        val src = read(main)
        assertTrue(
            "**لا قارئَ لوجهة العرض** — **وهو العطبُ الذي قِيس**",
            src.contains("com.rahalgo.ui.Engagement.DEST_OFFER ->"),
        )
        assertTrue("**لا يُفتَح بابُ العروض**", src.contains("Overlay.Menu(CustomerItems.OFFERS)"))
        assertTrue("**لا يُشار إلى المقصود**", src.contains("mineVm.openOffer(id)"))
        val ui = read(mineUi)
        assertTrue("**لا يُرفَع المقصودُ إلى أوّل القائمة**", ui.contains("sortedByDescending { it.id == focus }"))
    }

    // ═════════════ DLINK-04 · العرضُ الذي مضى ═════════════

    /**
     * **DLINK-04 · وعرضٌ أُوقف بعد إرسال الخبر لا يُعرَض سارياً.**
     *
     * **والحقيقةُ من الخادم**: **`public/offers` لا يردّ إلّا
     * السارية** — **وخبرٌ في الجيب لا يصير حقيقةً بمرور الوقت.**
     */
    @Test
    fun `العرضُ الذي مضى يُقال ولا يُعرَض سارياً`() {
        val vm = read(mineVm)
        assertTrue(
            "**لا يُجلَب من جديدٍ فيُقرأ سعرُ أمس**",
            vm.contains("open(CustomerItems.OFFERS, force = true)"),
        )
        assertTrue(
            "**لا يُعرَف أنّ المقصودَ ذهب**",
            vm.contains("focusGone = want != null && offers.orEmpty().none { it.id == want }"),
        )
        assertTrue("**ولا يُقال لصاحبه**", read(mineUi).contains("R.string.off_gone"))
    }

    // ═════════════ DLINK-05 · DLINK-06 · DLINK-07 — السقوطُ الآمن ═════════════

    /**
     * **DLINK-05 · MD-01 · ولا «متجر» في اختيارات اللوحة ولا في المحرّك.**
     *
     * **ولوحةٌ تُنقَّى وحدَها يتخطّاها نداءٌ مصنوعٌ بيد** — **فالمنعُ
     * في الثلاثة: المحرّكُ واللوحةُ والجيب.**
     */
    @Test
    fun `لا وجهةَ متجرٍ في اللوحة ولا في المحرّك`() {
        val panel = File(mobileRoot().parentFile,
            "web/apps/rahalgo/src/components/admin/CampaignsPanel.tsx")
        assertTrue("**لوحةُ الحملات غائبة**", panel.exists())
        val src = panel.readText()
        assertTrue(
            "**عادت «المتجر» إلى اختيارات الوجهة**",
            src.contains("""(["home", "offer"] as const)"""),
        )
        assertFalse("**بقي لفظُ وجهةِ المتجر**", src.contains("destMerchant"))

        val policy = File(mobileRoot().parentFile,
            "backend/internal/campaigns/policy.go")
        assertTrue("**سياسةُ الحملات غائبة**", policy.exists())
        assertFalse(
            "**عادت `DestMerchant` إلى عقد المحرّك**",
            policy.readText().contains("DestMerchant"),
        )
    }

    /** **DLINK-06 · وعرضٌ بلا معرّفٍ يسقط إلى البيت.** */
    @Test
    fun `عرضٌ بلا معرّفٍ يسقط إلى البيت`() {
        for (bad in listOf("", "   ")) {
            assertTrue("**فُتح عرضٌ بلا معرّف**", Engagement.route("offer", bad).isHome)
        }
        assertTrue(Engagement.route("offer", null).isHome)
    }

    /**
     * **DLINK-07 · وما لا نعرفه بيتٌ — ولا مقصدَ من نصّ الشبكة.**
     *
     * **ومحرّكٌ أحدثُ من الحزمة قد يرسل وجهةً جديدة** — **فلا يُدَّعى
     * علمٌ بما لا يُعرَف.**
     */
    @Test
    fun `الوجهةُ المجهولةُ تسقط إلى البيت`() {
        val bad = listOf(
            "https://evil.example/pay" to "x",
            "intent://settings" to "y",
            "../../etc" to "z",
            "offer://evil" to "x",
            "merchant/../home" to "x",
            "ORDER_CHAT" to "x",
            "" to "x",
        )
        for ((t, i) in bad) {
            val d = Engagement.route(t, i)
            assertTrue("**فُتحت وجهةٌ لا تُعرَف**: " + t, d.isHome)
            assertEquals("", d.id)
        }
    }

    // ═════════════ DLINK-08 · DLINK-09 — تُؤخَذ مرّةً ═════════════

    /**
     * **DLINK-08 · وتُؤخَذ مرّةً واحدةً — ولا يبتلع نوعٌ وجهةَ نوع.**
     */
    @Test
    fun `الوجهةُ تُؤخَذ مرّةً واحدة`() {
        Opened.resetForTest()
        assertTrue("**بدأت بوجهةٍ بلا سبب**", Opened.pending.isHome)
        assertTrue(Opened.take().isHome)
        assertTrue(Opened.take().isHome)
        // **ومستهلِكٌ واحدٌ يفرز بالنوع** — **ولا مستهلِكان يتسابقان.**
        val src = read(main)
        // **وموضعان يأخذانها**: **حديثُ الطلب والعرض** — **ورُفع
        // ثالثُهما مع وجهة المتجر** (`MD-08`).
        assertEquals(
            "**أكثرُ من موضعٍ يأخذ الوجهةَ** — **فيبتلع أحدُهما وجهةَ الآخر**",
            2,
            Regex("Opened\\.take\\(\\)").findAll(src).count(),
        )
        assertTrue("**لا فرزَ بالنوع**", src.contains("when (waiting.type) {"))
    }

    /** **DLINK-09 · وإعادةُ بناء الشاشة لا تُعيد فتحَ ما استُهلك.** */
    @Test
    fun `إعادةُ البناء لا تُعيد الوجهة`() {
        val src = read(main)
        assertTrue(
            "**سقط قيدُ الأثر** — **فيُعاد الفتحُ مع كلّ رسمة**",
            src.contains("LaunchedEffect(waiting)"),
        )
        // **والمقصودُ يُنسى بمغادرة شاشته.**
        assertTrue(
            "**تبقى الإشارةُ بعد مغادرة العروض**",
            read(mineUi).contains("onDispose { vm.clearFocus() }"),
        )
    }

    // ═════════════ DLINK-10…13 — الباردُ والدافئ ═════════════

    /**
     * **DLINK-10 · DLINK-12 · والبادئُ من إغلاقٍ يقرأ وجهتَه.**
     *
     * **و`onCreate` يضعها** — **قبل أوّل شاشة.**
     */
    @Test
    fun `الفتحُ من إغلاقٍ يقرأ الوجهة`() {
        val src = read(main)
        val at = src.indexOf("setContent { CustomerApp() }")
        assertTrue("**ذهب بناءُ الشاشة**", at > 0)
        val before = src.substring(0, at)
        assertTrue(
            "**لا تُقرأ الوجهةُ قبل أوّل شاشة**",
            before.contains("com.rahalgo.ui.Opened.from(intent)"),
        )
    }

    /**
     * **DLINK-11 · DLINK-13 · والواصلُ والتطبيقُ مفتوحٌ كذلك.**
     *
     * **و`singleTask` لا تُعيد إنشاء النشاط** — **فبلا `onNewIntent`
     * يبقى مقصدُ الأمس.**
     */
    @Test
    fun `الفتحُ والتطبيقُ مفتوحٌ يقرأ الوجهة`() {
        val src = read(main)
        val at = src.indexOf("override fun onNewIntent")
        assertTrue("**لا مستقبِلَ لمقصدٍ جديد**", at > 0)
        assertTrue(
            "**لا تُقرأ الوجهةُ في الفتح الدافئ**",
            src.substring(at).contains("com.rahalgo.ui.Opened.from(intent)"),
        )
        // **وتُمحى من المقصد نفسِه** — **فلا تُقرأ ثانيةً بإعادة بنائه.**
        val opened = read("ui/src/main/kotlin/com/rahalgo/ui/Opened.kt")
        assertTrue(
            "**تبقى في المقصد فتُقرأ مرّةً أخرى**",
            opened.contains("intent.removeExtra(RahalPushService.EXTRA_DEST_TYPE)"),
        )
    }

    // ═════════════ DLINK-14 · DLINK-15 · حديثُ الطلب كما أُقرّ ═════════════

    /** **DLINK-14 · وحديثُ الطلب لم يتبدّل عقدُه.** */
    @Test
    fun `حديثُ الطلب كما أُقرّ`() {
        val d = Engagement.route("order_chat", "a1")
        assertEquals(Engagement.DEST_ORDER_CHAT, d.type)
        assertEquals("a1", d.id)
        assertTrue(Engagement.route("order_chat", "").isHome)
    }

    /** **DLINK-15 · ووجهةُ حديثِ أ تفتح حديثَ أ.** */
    @Test
    fun `حديثُ أ يفتح أ لا آخرَ ما فُتح`() {
        val src = read(main)
        assertTrue(
            "**لا يُفتَح الحديثُ بمعرّف الوجهة**",
            src.contains("liveChat.openId = com.rahalgo.ui.Opened.take().id"),
        )
        assertTrue(
            "**اللوحُ لا يُفتَح على طلبٍ بعينه**",
            src.contains("liveChat.openId?.let { id ->"),
        )
    }

    /**
     * **DLINK-16 · ونوعان بمعرّفٍ واحدٍ لا يختلطان.**
     *
     * **والمعرّفُ وحدَه لا يقول ما هو** — **والنوعُ هو الحَكَم.**
     */
    @Test
    fun `المعرّفُ الواحدُ لا يخلط النوعين`() {
        val id = "8b0a8a71-d186-4647-ae3b-9cd3898508bf"
        assertEquals(Engagement.DEST_OFFER, Engagement.route("offer", id).type)
        assertEquals(Engagement.DEST_ORDER_CHAT, Engagement.route("order_chat", id).type)
        // **والمرفوعةُ لا تصير نوعاً بمعرّفٍ يشبه غيرَه.**
        assertTrue(Engagement.route("merchant", id).isHome)
        // **ولا نوعَ يُشتقّ من شكل المعرّف.**
        assertTrue(Engagement.route("", id).isHome)
    }
}

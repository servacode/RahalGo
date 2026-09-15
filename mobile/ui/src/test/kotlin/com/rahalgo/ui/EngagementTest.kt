package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خبرُ التفاعل ووجهتُه في الجيب** (`AN`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا يُقاس هنا وصولُ الدفع** — **ذاك على جهازٍ حقيقيّ**: **وهذا
 * يقيس ما يفعله التطبيقُ بما يصله.**
 */
class EngagementTest {

    // ═════════════════ AN-02 ═════════════════

    /** **AN-02 · والوجهةُ المعروفةُ تُفتَح.** */
    @Test
    fun `الوجهةُ المعروفةُ تُقرأ كما هي`() {
        val offer = Engagement.route("offer", "8b0a8a71-d186-4647-ae3b-9cd3898508bf")
        assertEquals(Engagement.DEST_OFFER, offer.type)
        assertEquals("8b0a8a71-d186-4647-ae3b-9cd3898508bf", offer.id)
        assertFalse(offer.isHome)

        // **و«المتجر» رُفعت** (قرارُ المالك ٢٠٢٦-٠٩-١٥) — **ولا شاشةَ
        // لها عند الزبون**: **فتسقط إلى البيت** (`MD-08`).
        assertTrue(Engagement.route("merchant", "abc").isHome)
    }

    // ═════════════════ AN-03 ═════════════════

    /**
     * **AN-03 · وما لا نعرفه يسقط إلى البيت — ولا يسقط التطبيق.**
     *
     * **ونصٌّ يجيء من الشبكة لا يُنفَّذ مقصداً نظاميّاً** — **ومحرّكٌ
     * أحدثُ من الحزمة قد يرسل وجهةً جديدة.**
     */
    @Test
    fun `الوجهةُ المجهولةُ تسقط إلى البيت`() {
        val bad = listOf(
            "https://evil.example/pay" to "x",
            "intent://settings" to "y",
            "order" to "1",
            "../../etc" to "z",
            "" to "",
            "offer" to "",        // **ومعرّفٌ فارغٌ لوجهةٍ تحتاجه**
            "merchant" to "   ",
        )
        for ((t, i) in bad) {
            val d = Engagement.route(t, i)
            assertTrue("**فُتحت وجهةٌ لا تُعرَف**: " + t, d.isHome)
            assertEquals("", d.id)
        }
        // **وفارغٌ تماماً.**
        assertTrue(Engagement.route(null, null).isHome)
    }

    // ═════════════════ AN-06 ═════════════════

    /**
     * **AN-06 · والوجهةُ تُستهلَك مرّةً.**
     *
     * **ومن أدار جهازَه فأُعيد بناءُ الشاشة وجد نفسَه يُساق إلى العرض
     * ثانيةً** — **ثمّ كلّما أدار.**
     */
    @Test
    fun `الوجهةُ تُؤخَذ مرّةً واحدة`() {
        Opened.resetForTest()
        assertTrue("**بدأت بوجهةٍ بلا سبب**", Opened.pending.isHome)
        // **ولا `Intent` هنا** — **والحالُ تُقاس بأخذها.**
        assertTrue(Opened.take().isHome)
        assertTrue(Opened.take().isHome)
    }

    // ═════════════════ AN-01 · AN-07 ═════════════════

    /**
     * **AN-07 · ولفظُ الصنف عربيٌّ لا رمزٌ آليّ.**
     *
     * **و`promo` نصٌّ لمهندسٍ لا لزبون.**
     */
    @Test
    fun `أصنافُ الصندوق مسمّاةٌ بلا رموز`() {
        assertEquals("promo", Engagement.KIND)
        assertEquals("order", Engagement.KIND_ORDER)
    }

    // ═════════════════ AN-04 · AN-05 — حراسةُ البنية ═════════════════

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
        return f.readText()
    }

    /**
     * **AN-04 · وسلوكُ خبر الطلب لم يتبدّل.**
     *
     * **وقناةُ العجلة له وحدَه** — **ومن جعل العرضَ عاجلاً أيقظ
     * الناسَ ليلاً بخصم.**
     */
    @Test
    fun `خبرُ الطلب وحدَه عاجل`() {
        val cust = read("app-customer/src/main/kotlin/com/rahalgo/customer/push/PushService.kt")
        assertTrue(
            "**تبدّل شرطُ العجلة عند الزبون**",
            cust.contains("kind == KIND_ORDER"),
        )
        assertFalse("**صار العرضُ عاجلاً**", cust.contains("promo"))
    }

    /**
     * **AN-05 · وعقدُ جاهزيّة السائق لم يُمَسّ** (الدفعة السادسة).
     *
     * **والإشعارُ شرطُ الإعلان عن التوفّر** — **ومن رفعه أعاد سائقاً
     * يُعلَن متاحاً ولا يبلغه النداء.**
     */
    @Test
    fun `جاهزيّةُ السائق كما أُقرّت`() {
        val src = read("app-driver/src/main/kotlin/com/rahalgo/driver/location/Readiness.kt")
        assertTrue(
            "**رُفع شرطُ الإشعار عن الإعلان عن التوفّر**",
            src.contains("located && !s.blockers.contains(Blocker.NOTIFICATION_PERMISSION_REQUIRED)"),
        )
    }

    /**
     * **والوجهةُ تُنقّى في موضعٍ واحد** — **ولا يفكّها كلُّ تطبيقٍ
     * بيده.**
     */
    @Test
    fun `الوجهةُ تُفَكُّ مركزيّاً`() {
        val push = read("ui/src/main/kotlin/com/rahalgo/ui/push/RahalPushService.kt")
        assertTrue(
            "**لا تنقيةَ عند الاستقبال**",
            push.contains("Engagement.route(data[\"entity\"], data[\"entity_id\"])"),
        )
        // **ولا مقصدٌ يُبنى من نصّ الشبكة** — **ولا `Intent(action)`
        // ولا `parse`**: **البيتُ يُفتَح حاملاً وجهتَه.**
        assertFalse("**مقصدٌ من نصّ الشبكة**", push.contains("Intent.parseUri"))
        assertFalse("**مقصدٌ بفعلٍ من نصّ**", push.contains("Intent(data["))
    }
}

package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ العرض ومدّتُه — قرارٌ يُقاس** (`MO` · `RO`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا تُقاس هنا صلاحيّةُ عرضٍ** — **تلك في الخادم** (`offers.StatusAt`
 * و`LiveCond`): **وساعةُ الجهاز لا سلطانَ لها.**
 *
 * **وهذا يقيس ما تفعله الشاشةُ بما يُقال لها**: **أتترجم الحالَ أم
 * تطبع رمزاً؟ أتعرض زرَّ إيقافٍ على منتهٍ؟ أتترجم المدّةَ إلى لحظة؟**
 */
class OffersPolicyTest {

    // ═════════════════ MO-03 · MO-04 · MO-05 · MO-06 ═════════════════

    /**
     * **والحالُ الأربعُ تُقرأ بأسمائها** — **ولا يُطبَع رمزٌ آليّ.**
     *
     * **و`scheduled` نصٌّ لمهندسٍ لا لصاحب مطعم.**
     */
    @Test
    fun `الحالاتُ الأربعُ معروفةٌ ومسمّاة`() {
        val known = listOf(
            OfferStatus.SCHEDULED, OfferStatus.ACTIVE,
            OfferStatus.STOPPED, OfferStatus.EXPIRED,
        )
        assertEquals(4, known.toSet().size)
        // **وهي نصُّ المحرّك حرفاً** — **ومن بدّل حرفاً هنا قطع الوصل.**
        assertEquals("scheduled", OfferStatus.SCHEDULED)
        assertEquals("active", OfferStatus.ACTIVE)
        assertEquals("stopped", OfferStatus.STOPPED)
        assertEquals("expired", OfferStatus.EXPIRED)
    }

    /**
     * **MO-06 · والمنتهي لا زرَّ إيقافٍ له.**
     *
     * **وزرٌّ يُضغط ولا يقع شيءٌ يُقرأ معطوباً** — **والمنتهي انتهى
     * وحدَه، والمُنزَلُ أُنزل.**
     */
    @Test
    fun `زرُّ الإيقاف حيث يُفيد وحدَه`() {
        assertTrue(OfferStatus.canStop(OfferStatus.ACTIVE))
        // **والمجدولُ يُلغى قبل أن يبدأ** — **ومن جدول خطأً لا ينتظر
        // يومين ليُصلحه.**
        assertTrue(OfferStatus.canStop(OfferStatus.SCHEDULED))
        assertFalse("**عُرض إيقافٌ لمنتهٍ**", OfferStatus.canStop(OfferStatus.EXPIRED))
        assertFalse("**عُرض إيقافٌ لمُنزَل**", OfferStatus.canStop(OfferStatus.STOPPED))
    }

    /**
     * **MO-03 · والسارِي وحدَه يُنقص السعرَ الآن.**
     *
     * **والمجدولُ وعدٌ لم يحلّ** — **ومن رآه «يخصم» ظنّ زبائنَه
     * يشترون بالسعر الجديد وهم لا.**
     */
    @Test
    fun `المجدولُ لا يُعَدّ خاصماً الآن`() {
        assertTrue(OfferStatus.discounting(OfferStatus.ACTIVE))
        assertFalse(OfferStatus.discounting(OfferStatus.SCHEDULED))
        assertFalse(OfferStatus.discounting(OfferStatus.EXPIRED))
        assertFalse(OfferStatus.discounting(OfferStatus.STOPPED))
    }

    /**
     * **وحالٌ لا نعرفها لا تُعَدّ ساريةً ولا تُوقَف.**
     *
     * **ومحرّكٌ أحدثُ من الحزمة قد يردّ لفظاً جديداً** — **فلا يُدَّعى
     * علمٌ به**: **ولا يُعرَض على أنّه خصمٌ قائم.**
     */
    @Test
    fun `الحالُ المجهولةُ لا تُدَّعى`() {
        assertFalse(OfferStatus.discounting("flash_sale"))
        assertFalse(OfferStatus.canStop("flash_sale"))
        assertFalse(OfferStatus.discounting(""))
    }

    // ═════════════════ MO-02 ═════════════════

    /**
     * **MO-02 · والمدّةُ تُترجَم إلى لحظةِ نهاية.**
     *
     * **ولا تُحفَظ «يومان» ثانيةً** — **و`ends_at` هي الحقيقة في
     * المحرّك**: **ومدّتان تفترقان يومَ يُعدَّل أحدُهما.**
     */
    @Test
    fun `المدّةُ تصير لحظةَ نهاية`() {
        val now = 1_700_000_000_000L
        assertEquals(now + 3_600_000L, OfferDuration.endsAtMillis(now, 1))
        assertEquals(now + 24 * 3_600_000L, OfferDuration.endsAtMillis(now, 24))
        // **وأسبوعٌ سبعةُ أيّام** — **ولا انزلاقَ في الضرب.**
        assertEquals(now + 168 * 3_600_000L, OfferDuration.endsAtMillis(now, 168))
    }

    /**
     * **وفحصُ الشاشة يقول في اللحظة ما يُعرَف في اللحظة** — **ولا
     * يُغني عن الخادم.**
     */
    @Test
    fun `المدّةُ المستحيلةُ تُردّ قبل النداء`() {
        assertFalse("**مدّةٌ صفرٌ قُبلت**", OfferDuration.valid(0))
        assertFalse("**مدّةٌ سالبةٌ قُبلت**", OfferDuration.valid(-5))
        assertTrue(OfferDuration.valid(1))
        assertTrue(OfferDuration.valid(24))
        assertFalse("**بلا سقف**", OfferDuration.valid(OfferDuration.MAX_HOURS + 1))
    }

    /** **والمددُ المعروضةُ كلُّها مقبولة** — **ولا خيارٌ يُردّ.** */
    @Test
    fun `المددُ المعروضةُ صالحةٌ كلُّها`() {
        for (h in OfferDuration.PRESET_HOURS) {
            assertTrue("**مدّةٌ معروضةٌ تُردّ**: " + h, OfferDuration.valid(h))
        }
    }

    // ═════════════════ MO-01 · MO-07 · MO-08 · MO-10 · RO-04 ═════════

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() &&
                File(dir, "app-merchant").exists()
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

    private companion object {
        const val M_VM = "app-merchant/src/main/kotlin/com/rahalgo/merchant/offers/OffersViewModel.kt"
        const val R_VM = "app-rep/src/main/kotlin/com/rahalgo/rep/offers/RepOffersViewModel.kt"
        const val API = "shared/src/main/kotlin/com/rahalgo/shared/offers/StoreOffers.kt"
    }

    /**
     * **MO-01 · RO-03 · والبطاقةُ مركزيّةٌ في البابين.**
     *
     * **ونسختان تفترقان يوماً** — **فتقول إحداهما «ساري» وتطبع الأخرى
     * رمزاً.**
     */
    @Test
    fun `التطبيقان يرسمان البطاقةَ المركزيّة`() {
        for (rel in listOf(
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/offers/OffersScreen.kt",
            "app-rep/src/main/kotlin/com/rahalgo/rep/offers/RepOffersScreen.kt",
        )) {
            val src = read(rel)
            assertTrue("**بطاقةٌ خاصّةٌ في** " + rel, src.contains("OfferCard("))
            // **ولا حسبةَ سعرٍ في الجهاز** — **والسعران يجيئان
            // محسوبين**: **وحسبةٌ في موضعين تفترق يوماً.**
            //
            // **ويُطلَب الفعلُ نفسُه لا الكلمة** — **درسُ الدفعة
            // السادسة**: `priceBefore * …` أو قسمةٌ على مئة.
            for (bad in listOf(
                "priceBefore *", "priceAfter *", "priceBefore /", "priceAfter /",
                "/ 100", "* percent", "percent *",
            )) {
                assertFalse(
                    "**حُسب السعرُ في الجهاز في** " + rel + ": " + bad,
                    src.contains(bad),
                )
            }
        }
    }

    /**
     * **MO-08 · RO-04 · وضغطتان على «أوقف» نداءٌ واحد.**
     *
     * **ومن ضغط مرّتين لأنّ الشبكةَ تأخّرت لا يُرسِل أمرين.**
     */
    @Test
    fun `الإيقافُ لا يُفتَح مرّتين`() {
        for (rel in listOf(M_VM, R_VM)) {
            val src = read(rel)
            assertTrue(
                "**لا حارسَ لضغطةٍ ثانيةٍ في** " + rel,
                src.contains("if (stopping != null) return"),
            )
        }
    }

    /**
     * **MO-07 · والحالُ من الردّ لا من تفاؤل.**
     *
     * **ومن بدّل الصفَّ متفائلاً أرى صاحبَه «موقوف» وهو سارٍ** —
     * **فيمضي وهو يظنّ أنّه أوقفه.**
     */
    @Test
    fun `الحالُ تُكتب من ردّ الخادم`() {
        for (rel in listOf(M_VM, R_VM)) {
            val src = read(rel)
            assertTrue(
                "**بُدّلت الحالُ قبل الردّ في** " + rel,
                src.contains("val updated = offers.stop("),
            )
            assertFalse(
                "**كُتبت حالٌ بيدٍ في** " + rel,
                src.contains("status = \"stopped\""),
            )
        }
    }

    /**
     * **MO-10 · RO-02 · ولا متجرَ يُكتب معرّفُه بيد.**
     *
     * **والمتجرُ يجيء من بابٍ محروس** — **وحارسُ الخادم هو الفاصل**
     * (`ownsMerchant` و`repClient`): **وشاشةٌ تحرس وحدَها ليست حارساً.**
     */
    @Test
    fun `المتجرُ من بابه لا من حقلٍ يُكتب`() {
        val m = read(M_VM)
        assertTrue(
            "**متجرُ الشاشة لا يجيء من `stores`**",
            m.contains("merchant.stores()"),
        )
        val r = read(R_VM)
        assertTrue("**نطاقُ المندوب لا يُفتح بمتجرٍ مُمرَّر**", r.contains("fun open(id: String"))
    }

    /**
     * **ولا يُرسَل من يتحمّل الخصم من الجهاز.**
     *
     * **ولو أُرسل لم يُقرأ** (`CreateScoped`) — **والحقلُ غائبٌ أصلاً
     * من العقد**: **فلا يُقرأ غداً على أنّه خيارٌ متاح.**
     */
    @Test
    fun `العقدُ لا يحمل من يتحمّل الخصم`() {
        val api = read(API)
        assertFalse("**عاد `borne_by` إلى العقد**", api.contains("borne_by"))
        assertFalse("**سعرٌ نهائيٌّ يُرسَل من الجهاز**", api.contains("final_price"))
    }
}

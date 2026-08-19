package com.rahalgo.ui

/**
 * **اشتقاقُ نسخةِ الصورة — `MED-*`.**
 *
 * الوسم: `@unit @regression`
 *
 * # لماذا لها اختبارُها
 *
 * **BUG-004 من تدقيق ٢٠٢٦-٠٨-١٩**: لافتةٌ نُزّلت في **٤٫٩ ثانية** لأنّ
 * الأصلَ ١٦٠٠×٨٠٠ رُسم في لوحٍ عرضُه ثلاثُ مئةٍ وأربعون نقطة.
 *
 * **والاشتقاقُ نصٌّ يُبنى بيد**: `<الاسم>_<العرض><اللاحقة>`. **ونصٌّ
 * يُبنى بيدٍ يُكسر بمسارٍ لا لاحقةَ له**، أو برابطٍ خارجيّ، أو بنقطةٍ
 * في اسم المجلّد.
 *
 * **وحين يُكسر لا يظهر خطأ** — تُطلب صورةٌ لا وجودَ لها فتردّ ٤٠٤،
 * **ويُقرأ «الصورُ بطيئة» لا «الرابطُ خطأ».**
 */
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MediaSizedTest {

    /** MED-001 **المسارُ العاديُّ يأخذ لاحقةَ العرض قبل النقطة.** */
    @Test
    fun `النسخةُ تُشتقُّ قبلَ اللاحقة`() {
        val out = sizedPath("banners/abc.jpg", 960, true)
        assertTrue("لم تُشتقّ النسخة: $out", out!!.endsWith("banners/abc_960.jpg"))
    }

    /**
     * MED-002 **وبلا نسخٍ مُولَّدةٍ يُعاد الأصل.**
     *
     * **وطلبُ نسخةٍ غيرِ مولَّدةٍ يردّ ٤٠٤** — وصورةٌ غائبةٌ أسوأُ من
     * صورةٍ كبيرة.
     */
    @Test
    fun `بلا نسخٍ يُعادُ الأصل`() {
        val out = sizedPath("banners/abc.jpg", 960, false)
        assertTrue("اشتُقّت نسخةٌ لا وجودَ لها: $out", !out!!.contains("_960"))
    }

    /**
     * MED-003 **والرابطُ الخارجيُّ لا يُمسّ.**
     *
     * **ولا نملك خادمَه** — فاشتقاقُ نسخةٍ منه يبني رابطاً لا وجودَ له.
     */
    @Test
    fun `الرابطُ الخارجيُّ يُترك كما هو`() {
        val url = "https://cdn.example.com/a.png"
        assertEquals(url, sizedPath(url, 960, true))
    }

    /**
     * MED-004 **ومسارٌ بلا لاحقةٍ يُعاد كما هو.**
     *
     * **وقطعُ نصٍّ عند نقطةٍ غيرِ موجودةٍ يعطي `-1`** — ومن لم يحرسه
     * بنى رابطاً معطوباً أو أسقط الرسم.
     */
    @Test
    fun `مسارٌ بلا لاحقةٍ لا يُكسر`() {
        val out = sizedPath("banners/abc", 960, true)
        assertTrue("كُسر مسارٌ بلا لاحقة: $out", out!!.endsWith("banners/abc"))
    }

    /** MED-005 **والفارغُ فارغ** — ولا يُبنى رابطٌ من لا شيء. */
    @Test
    fun `الفارغُ يبقى فارغا`() {
        assertNull(sizedPath(null, 960, true))
        assertNull(sizedPath("", 960, true))
    }
}

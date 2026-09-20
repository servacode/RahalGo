package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سوقٌ فارغةٌ خارجَ التغطيةِ ليست «قريباً»** (`CUST-07-034`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **خارجَ التغطيةِ لا متجرَ مخزَّناً، فتفرغ `visibleSections`، فيسقط تنبيهُ
 * التغطيةِ فوقُ** (شرطُه `visibleSections.isNotEmpty()`) **ويُعرَض بدلَه
 * «نعمل على إضافة المتاجر»** — فيقرأ المقصيُّ «قريباً» لا «لم نصل إليك».
 *
 * **والفرقُ حكمٌ**: «قريباً» تقول انتظر، و«لم نصل» تقول أخبرني — ودفترُ
 * الطلب يُبنى على الثانية. **فحين تُقصى النقطةُ** يُعرَض `ServiceBlockNotice`
 * (سببٌ صريحٌ + زرُّ «أخبرني»)، **وداخلَ التغطيةِ تبقى «قريباً» صادقةً.**
 *
 * **اختبارُ مصدرٍ**: عقدُ الواجهةِ لا يُقاس بلقطةِ شاشة.
 */
class CoverageEmptyStateTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val shop = "app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt"

    /** **يُقتطع جسمُ فرعِ السوقِ الفارغة** من `when`. */
    private fun emptyBranch(): String {
        val s = read(shop)
        val start = s.indexOf("(vm.marketEmpty || vm.sections.isEmpty()) && !vm.searching ->")
        assertTrue("**فرعُ السوقِ الفارغةِ غائب**", start >= 0)
        // نأخذ حتّى الفرعِ التالي `vm.items.isEmpty()`
        val end = s.indexOf("vm.items.isEmpty() ->", start)
        assertTrue("**لم أجد نهايةَ الفرع**", end > start)
        return s.substring(start, end)
    }

    /**
     * **CUST-07-034 · خارجَ التغطيةِ يُفرَّق عن الفراغِ داخلَها** —
     * الفرعُ يسأل عن الإتاحة قبل أن يقول «قريباً».
     */
    @Test
    fun emptyMarketDistinguishesOutOfCoverage() {
        val b = emptyBranch()
        assertTrue(
            "**لا يُفرَّق المقصيُّ** — يجب `val blocked0 = availability?.takeIf { !it.available }`",
            b.contains("val blocked0 = availability?.takeIf { !it.available }"),
        )
    }

    /**
     * **CUST-07-034 · المقصيُّ يرى تنبيهَ الخدمةِ لا «قريباً»** —
     * سببٌ صريحٌ وزرُّ «أخبرني» عبر `ServiceBlockNotice`.
     */
    @Test
    fun outOfCoverageShowsServiceNoticeNotComingSoon() {
        val b = emptyBranch()
        val notice = b.indexOf("ServiceBlockNotice(")
        val empty = b.indexOf("R.string.shop_market_empty")
        assertTrue("**لا `ServiceBlockNotice` في الفرع**", notice >= 0)
        assertTrue("**لا حالةَ «قريباً» للداخل**", empty >= 0)
        // التنبيهُ داخلَ فرعِ `blocked0 != null`، و«قريباً» في `else`
        assertTrue(
            "**التنبيهُ لا يُشترط بالإقصاء** — يجب `if (blocked0 != null)` قبله",
            b.contains("if (blocked0 != null)") && b.indexOf("if (blocked0 != null)") < notice,
        )
        assertTrue(
            "**«قريباً» ليست في `else`** — قد تُعرَض للمقصيّ",
            b.contains("} else {") && b.indexOf("} else {") < empty,
        )
    }

    /**
     * **CUST-07-034 · وعلمُ الاستكشافِ يُمرَّر** — فلا زرُّ طلبٍ لنقطةٍ
     * لم تُؤكَّد (عقدُ `CUST-07-030`).
     */
    @Test
    fun passesDiscoveryFlagToNotice() {
        val b = emptyBranch()
        assertTrue(
            "**لا يُمرَّر علمُ الاستكشافِ للتنبيه**",
            b.contains("discovery = ctx0.source == com.rahalgo.customer.PointSource.DISCOVERY"),
        )
    }

    /**
     * **CUST-07-034 · وداخلَ التغطيةِ تبقى «قريباً»** — الفراغُ المشروعُ
     * (منصّةٌ لم تمتلئ) لا يُمسّ.
     */
    @Test
    fun inCoverageStillShowsComingSoon() {
        val b = emptyBranch()
        assertTrue(
            "**حُذفت حالةُ «قريباً» المشروعة**",
            b.contains("R.string.shop_market_empty") && b.contains("R.string.shop_market_empty_hint"),
        )
    }
}

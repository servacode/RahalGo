package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ الاتّصال — عقدٌ لا تفصيلُ تنفيذ** (`P8-DEF-001`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العيب
 *
 * **تطبيقٌ مفتوحٌ يسقط عنه الواي فاي والبياناتُ معاً — ولا يقول شيئاً.**
 * الأصنافُ باقيةٌ وشريطُ «لا يوجد اتصال بالإنترنت» لا ينزل، **ومن سحب
 * للتحديث لا يرى جواباً.** (قِيس على SM-A525F مرّتين.)
 *
 * # والعقدُ الذي تحرسه هذه الفحوص
 *
 * **`online` يقول الحقيقةَ بعد كلّ حدث** — **ولا يعلق على «متّصل» بعد
 * أن ذهبت الشبكاتُ كلُّها.**
 *
 * # والنظامُ المزيَّف يحاكي ما يُحذّر منه أندرويد نصّاً
 *
 * `NetworkCallback`: «لا تنادِ `getNetworkCapabilities` ولا غيرَها من
 * أساليب `ConnectivityManager` المتزامنة داخلَ هذا النداء — **فلا ضمانَ
 * أن تكون الكائناتُ المردودةُ حاليّة**».
 *
 * **فـ`FakeSystem` يُبقي حالَه القديمةَ لحظةَ الحدث ثمّ يتحدّث بعده** —
 * **وهو ما يقع على الجهاز حين يسقط الواي فاي والبياناتُ معاً.**
 */
class NetTrackerTest {

    /** **نظامٌ متأخّرٌ بخطوة** — يقول الحالَ القديمةَ لحظةَ النداء. */
    private class FakeSystem {
        val validated = linkedSetOf<String>()
        var lagging = false
        private var stale: Boolean = false

        fun snapshot(): Boolean = if (lagging) stale else validated.isNotEmpty()

        /** يلتقط الحالَ قبل الحدث ليُعيدها أثناءه. */
        fun freeze() {
            stale = validated.isNotEmpty()
        }
    }

    /**
     * **والقراءةُ الأولى من النظام — خارجَ أيّ نداء** كما في `Net.install`،
     * **فلا سباق.** **وما بعدها من الأحداث وحدَها** — **والنظامُ المتأخّرُ
     * لا يُسأل أثناءها.**
     */
    private fun tracker(sys: FakeSystem): NetTracker<String> {
        val t = NetTracker<String>(initialOnline = sys.snapshot())
        // **والتسجيلُ يُعيد الشبكاتِ القائمة** — أندرويد يُرسل لكلّ شبكةٍ
        // حيّةٍ `onAvailable` ثمّ `onCapabilitiesChanged` فورَ التسجيل،
        // **وقبل أيّ حدثٍ لاحق.** ومن لم يحاكِه أخبر المتتبِّعَ بشبكاتٍ
        // لم يرَها قطّ.
        for (n in sys.validated.toList()) {
            t.onAvailable(n)
            t.onCapabilitiesChanged(n, internet = true, validated = true)
        }
        return t
    }

    private fun NetTracker<String>.up(sys: FakeSystem, n: String) {
        sys.freeze()
        sys.validated += n
        onAvailable(n)
        onCapabilitiesChanged(n, internet = true, validated = true)
    }

    private fun NetTracker<String>.down(sys: FakeSystem, n: String) {
        sys.freeze()
        sys.validated -= n
        onLost(n)
    }

    // ── ١ · يبدأ متّصلاً ─────────────────────────────────────────────────
    @Test
    fun startsOnline() {
        val sys = FakeSystem().apply { validated += "wifi" }
        assertTrue("**بدأ والشبكةُ حيّة فقال: مقطوع**", tracker(sys).online)
    }

    // ── ٢ · متّصلٌ ثمّ مقطوع ─────────────────────────────────────────────
    @Test
    fun onlineToOffline() {
        val sys = FakeSystem().apply { validated += "wifi" }
        val t = tracker(sys)
        sys.lagging = true
        t.down(sys, "wifi")
        assertFalse("**ذهبت الشبكةُ الوحيدةُ وبقي «متّصل»**", t.online)
    }

    // ── ٣ · مقطوعٌ ثمّ متّصل ─────────────────────────────────────────────
    @Test
    fun offlineToOnline() {
        val sys = FakeSystem()
        val t = tracker(sys)
        assertFalse(t.online)
        sys.lagging = true
        t.up(sys, "wifi")
        assertTrue("**عادت الشبكةُ وبقي «مقطوع»**", t.online)
    }

    // ── ٤ · سقوطُ الواي فاي والبياناتِ معاً — **وهو ما وقع على الجهاز** ──
    @Test
    fun simultaneousWifiAndCellularLoss() {
        val sys = FakeSystem().apply { validated += listOf("wifi", "cell") }
        val t = tracker(sys)
        sys.lagging = true
        t.down(sys, "wifi")
        t.down(sys, "cell")
        assertFalse(
            "**سقطت الشبكتان معاً وبقي «متّصل»** — وهذا `P8-DEF-001` بعينه",
            t.online,
        )
    }

    // ── ٤ب · وسقوطُ إحداهما يُبقي الأخرى ─────────────────────────────────
    @Test
    fun losingOneOfTwoKeepsOnline() {
        val sys = FakeSystem().apply { validated += listOf("wifi", "cell") }
        val t = tracker(sys)
        sys.lagging = true
        t.down(sys, "wifi")
        assertTrue("**بقيت البياناتُ حيّةً فقال: مقطوع**", t.online)
    }

    // ── ٥ · لا حالَ شائخةً من أحداثٍ مكرّرةٍ أو غريبة ────────────────────
    @Test
    fun noStaleStateFromDuplicateOrUnknownEvents() {
        val sys = FakeSystem().apply { validated += "wifi" }
        val t = tracker(sys)
        sys.lagging = true
        t.down(sys, "wifi")
        t.onLost("wifi")        // **حدثٌ مكرّر**
        t.onLost("ghost")       // **شبكةٌ لم تُعرف قطّ**
        assertFalse("**أحداثٌ مكرّرةٌ أعادت «متّصل»**", t.online)
    }

    // ── ٥ب · شبكةٌ موصولةٌ لا تُوصِّل ليست اتّصالاً ─────────────────────
    @Test
    fun unvalidatedNetworkIsNotOnline() {
        val sys = FakeSystem()
        val t = tracker(sys)
        t.onAvailable("captive")
        t.onCapabilitiesChanged("captive", internet = true, validated = false)
        assertFalse("**واي فاي بلا إنترنت قُرئ «متّصل»**", t.online)
    }

    // ── ٦ · الحالُ صحيحةٌ بعد تتابعِ أحداثٍ طويل (خلفيّةٌ ثمّ مقدّمة) ────
    @Test
    fun stateHoldsAcrossLongEventSequence() {
        val sys = FakeSystem().apply { validated += "wifi" }
        val t = tracker(sys)
        sys.lagging = true
        repeat(3) {
            t.down(sys, "wifi")
            assertFalse("**الدورة ${it + 1}: بقي «متّصل» بعد القطع**", t.online)
            t.up(sys, "wifi")
            assertTrue("**الدورة ${it + 1}: بقي «مقطوع» بعد العودة**", t.online)
        }
    }

    // ── ٧ · الفتحُ البارد المقطوعُ يبقى صحيحاً ───────────────────────────
    @Test
    fun freshStartOfflineStaysOffline() {
        val sys = FakeSystem()
        val t = tracker(sys)
        assertFalse("**فُتح مقطوعاً فقال: متّصل**", t.online)
    }

    // ── ٩ · حارسُ المصدر: لا سؤالَ للنظام داخلَ نداءِ شبكة ─────────────
    //
    // **والفحوصُ أعلاه تحرس `NetTracker`** — **وهذا يحرس أن يبقى `Net`
    // يمرّر إليه ما جاء به الحدث ولا يسأل النظام.** **ومن أعاد
    // `cm.activeNetwork` داخلَ نداءٍ أعاد العيبَ بعينه.**
    @Test
    fun callbacksNeverQueryTheSystem() {
        var dir = File("").absoluteFile
        while (!File(dir, "ui/src/main/kotlin/com/rahalgo/ui/Connectivity.kt").exists()) {
            dir = dir.parentFile ?: throw AssertionError("لم أجد Connectivity.kt")
        }
        val src = File(dir, "ui/src/main/kotlin/com/rahalgo/ui/Connectivity.kt").readText()
        val start = src.indexOf("object : ConnectivityManager.NetworkCallback()")
        assertTrue("**لم أجد نداءَ الشبكة في Net**", start >= 0)
        // **جسمُ النداء حتّى إغلاقه** — عدٌّ للأقواس من أوّل `{`.
        var i = src.indexOf('{', start)
        var depth = 0
        val begin = i
        while (i < src.length) {
            if (src[i] == '{') depth++
            if (src[i] == '}') { depth--; if (depth == 0) break }
            i++
        }
        val body = src.substring(begin, i)
        for (bad in listOf("activeNetwork", "getNetworkCapabilities", "has(cm)")) {
            assertFalse(
                "**نداءُ الشبكة يسأل النظامَ (`$bad`)** — وهذا `P8-DEF-001`: " +
                    "الجوابُ لا يُضمَن حاليّاً داخلَ النداء",
                body.contains(bad),
            )
        }
    }

    // ── ٨ · التعافي بعد عودة الاتّصال ────────────────────────────────────
    @Test
    fun recoveryAfterTotalLoss() {
        val sys = FakeSystem().apply { validated += listOf("wifi", "cell") }
        val t = tracker(sys)
        sys.lagging = true
        t.down(sys, "wifi")
        t.down(sys, "cell")
        assertFalse(t.online)
        // **والعودةُ على مرحلتين كما يفعل أندرويد**: موصولةٌ ثمّ مُتحقَّقة.
        sys.freeze()
        sys.validated += "wifi2"
        t.onAvailable("wifi2")
        t.onCapabilitiesChanged("wifi2", internet = true, validated = false)
        t.onCapabilitiesChanged("wifi2", internet = true, validated = true)
        assertTrue("**عاد الاتّصالُ وبقي «مقطوع»**", t.online)
    }
}

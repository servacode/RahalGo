package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-004 · CAF-09 + PC-2 — الخروجُ يحدّ الجلسة، فلا يرثها التالي**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **الخروجُ كان يمسح الجلسةَ وحدَها** — التوكن. **وثلاثةُ أشياء تبقى في
 * الجهاز بعده**، فيرثها الحسابُ التالي على الجهاز نفسِه:
 *
 *   `PC-2`     **السلّةُ** — كائنٌ عامٌّ (`Cart`) بلا صاحب، **فمن خرج
 *              تركَ أصنافَه لمن يليه**، ودفعَها الوارثُ باسمه.
 *   `CAF-09`   **الوصلةُ الحيّةُ** — `LiveSocket` مفتوحةٌ لا تُغلق،
 *              **يرثها التالي فتُنعَشه بأحداثِ من سبقه** (المحرّكُ
 *              يفحص الجلسةَ عند المصافحة وحدَها — `ws.go`/`R14`).
 *   —          **حالُ الغلاف** — الرصيدُ والبريدُ والملخّصُ في
 *              `ShellViewModel` يبقى معروضاً لثانٍ حتّى يُجلب غيرُه.
 *
 * # العقدُ بعد الإصلاح
 *
 * **حدُّ الجلسة يُطبَّق في موضعٍ مركزيّ** — خطّافُ `AppCore.afterLogout`
 * يُنادى في `logout()` **مباشرةً ومتزامناً** (قبل نداء الشبكة، فلا يتأخّر
 * بتعذّرها). **ويسجّله تطبيقُ الزبون** فيمحو السلّةَ ويوقف الوصلةَ ويبثّ
 * نبضةَ إنعاش. **وغلافُ الزبون يُحَدّ كذلك** عند تحوّله ضيفاً (`reset()`).
 *
 * **والمحرّكُ يفرض الملكيّةَ أصلاً** — لا يقرأ زبونٌ طلبَ غيرِه بمعرّفه
 * ولا يمسّه (`customer_isolation_test.go`: قراءةٌ وإلغاءٌ وحذفٌ ومحفظة،
 * ٤٠٤ لا ٤٠٣). **وهذا الحدُّ الثاني**: ما يبقى في الجهاز بعد الخروج.
 *
 * # ولماذا حراسةُ مصدرٍ لا رسمُ شاشة
 *
 * **`AppCore` و`AuthViewModel` و`ShellViewModel` تحتاج `Context`/`Application`**
 * — واختبارُ خطّافٍ يُركَّب بـ`install(context, …)` يحتاج جهازاً أو
 * `Robolectric`، **ولا نستعملها** (`CartTest`: «المنطقُ في الذاكرة يعمل
 * بلا `Context`»). **فالمقيسُ هنا القرارُ لا الرسم**: أنّ الخروجَ يستدعي
 * الحدَّ، وأنّ الحدَّ يمحو الثلاثةَ — **وأنّ سطراً منها لا يسقط صامتاً.**
 *
 * **والآليّاتُ التي يستدعيها الحدُّ مقيسةٌ سلوكيّاً في مواضعها**:
 * `Cart.clear()` يُفرِغ (`CartTest` CART-020)، و`LiveSocket.stop()` ثمّ
 * بدءٌ يحمل الرمزَ الجديدَ لا الملتقَطَ القديم (`D19-T3`)، وبدءٌ لا يفتح
 * وصلتين (`D19-T9`). **وهنا يُقاس أنّ الخروجَ يشبكها.**
 */
class CustDef004Test {

    // ── قراءةُ المصدر ─────────────────────────────────────────────────
    // **ونهاياتُ الأسطر تُوحَّد** — جيتٌ على ويندوز يكتب CRLF، فمطابقةُ
    // نصٍّ متعدّدِ الأسطر تسقط ويُقرأ العيبُ في المنتج وهو في الفحص.

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

    /** **جسدُ مقطعٍ** — من علامةِ بدايةٍ إلى أوّلِ نهايةٍ تُصادَف بعدها. */
    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val core = "ui/src/main/kotlin/com/rahalgo/ui/Core.kt"
    private val auth = "ui/src/main/kotlin/com/rahalgo/ui/AuthViewModel.kt"
    private val shell = "ui/src/main/kotlin/com/rahalgo/ui/ShellViewModel.kt"
    private val backend = "app-customer/src/main/kotlin/com/rahalgo/customer/Backend.kt"
    private val mainActivity = "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
    private val isolation = "../backend/internal/server/customer_isolation_test.go"

    private fun logoutBody() = body(read(auth), "fun logout()", "\n    private fun ", "\n    fun ", "\n}")
    // **حدُّ الجلسة الواحد** (`CUST-DEF-009`) — الخروجُ يمرّ به الآن، وفيه الخطّاف.
    private fun detachBody() = body(read(auth), "private fun detachSession()", "\n    }")
    private fun resetBody() = body(read(shell), "fun reset()", "\n}")
    private fun afterLogoutReg() = body(read(backend), "afterLogout = {", "},")

    // ══════════════════════════════════════════════════════════════════
    // **العقدُ المركزيّ — خطّافٌ يُنادى عند الخروج**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-004-01 · `AppCore` يعرض خطّافَ الخروج، و`install` يضبطه.** */
    @Test
    fun appCoreExposesLogoutHook() {
        val src = read(core)
        assertTrue(
            "**لا خطّافَ خروجٍ في `AppCore`** — فلا موضعَ مركزيّ يُحَدّ فيه",
            src.contains("var afterLogout: () -> Unit"),
        )
        assertTrue(
            "**`install` لا تقبل خطّافَ الخروج**",
            src.contains("afterLogout: () -> Unit = {}"),
        )
        assertTrue(
            "**`install` لا تضبط الخطّافَ المُمرَّر** — فيبقى فارغاً لا يفعل شيئاً",
            src.contains("this.afterLogout = afterLogout"),
        )
    }

    /**
     * **CUST-DEF-004-02 · والخروجُ يستدعي الخطّاف — عبر الحدّ الواحد.**
     *
     * **بعد `CUST-DEF-009`** انتقل الخطّافُ إلى `detachSession()` المركزيّ
     * (يمرّ به الخروجُ ورفضُ الجلسةِ معا)، **فالخروجُ يمرّ بالحدّ، والحدُّ
     * يستدعي الخطّاف** — والعقدُ قائمٌ لا مبدَّل.
     */
    @Test
    fun logoutInvokesBoundaryHook() {
        assertTrue(
            "**`logout()` لا تمرّ بالحدّ `detachSession()`** — فالجلسةُ تُمحى والباقي يُورَّث",
            logoutBody().contains("detachSession()"),
        )
        assertTrue(
            "**الحدُّ لا يستدعي `AppCore.afterLogout()`** — فالسلّةُ والوصلةُ تبقيان",
            detachBody().contains("AppCore.afterLogout()"),
        )
    }

    /**
     * **CUST-DEF-004-03 · والحدُّ متزامنٌ قبل الشبكة، بعد مسحِ الجلسة.**
     *
     * **ولو أُجِّل إلى `viewModelScope.launch` لَتأخّر بتعذّرِ الشبكة** —
     * وخروجٌ لا يُفرِغ السلّةَ إن غابت الشبكةُ خروجٌ ناقص. **والحدُّ
     * (`detachSession()`) يُنادى متزامناً قبل نداءِ الشبكة، وفيه مسحُ
     * الجلسةِ يسبق الخطّاف.**
     */
    @Test
    fun boundaryRunsSynchronouslyBeforeNetwork() {
        val lb = logoutBody()
        val boundary = lb.indexOf("detachSession()")
        val network = lb.indexOf("viewModelScope.launch")
        assertTrue("**غاب الحدُّ في الخروج**", boundary >= 0)
        assertTrue("**غاب نداءُ الشبكة**", network >= 0)
        assertTrue(
            "**الحدُّ داخلَ/بعد نداءِ الشبكة** — فيتأخّر بتعذّرها",
            boundary < network,
        )
        val d = detachBody()
        val cleared = d.indexOf("session.clear()")
        val hook = d.indexOf("AppCore.afterLogout()")
        assertTrue("**غاب مسحُ الجلسة في الحدّ**", cleared >= 0)
        assertTrue("**غاب الخطّاف في الحدّ**", hook >= 0)
        assertTrue("**الخطّافُ قبل مسحِ الجلسة**", cleared < hook)
    }

    // ══════════════════════════════════════════════════════════════════
    // **`PC-2` · السلّةُ لا تُورَّث**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-004-04 · تطبيقُ الزبون يمحو السلّةَ في خطّاف الخروج.** */
    @Test
    fun logoutClearsCart() {
        assertTrue(
            "**خطّافُ الخروج لا يمحو السلّة** — فيرثها الحسابُ التالي (`PC-2`)",
            afterLogoutReg().contains("Cart.clear()"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **`CAF-09` · الوصلةُ الحيّةُ لا تُورَّث**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-004-05 · وخطّافُ الخروج يوقف الوصلةَ الحيّة.** */
    @Test
    fun logoutStopsLiveSocket() {
        val reg = afterLogoutReg()
        assertTrue(
            "**خطّافُ الخروج لا يوقف الوصلة** — يرثها التالي فتُنعَشه بأحداثِ من سبقه (`CAF-09`)",
            reg.contains("live.stop()"),
        )
    }

    /** **CUST-DEF-004-06 · وغلافُ الزبون يوقفها كذلك عند حدّه.** */
    @Test
    fun shellResetStopsLiveSocket() {
        assertTrue(
            "**`reset()` لا توقف الوصلة**",
            resetBody().contains("backend.live.stop()"),
        )
    }

    /**
     * **CUST-DEF-004-07 · والتحوّلُ إلى ضيفٍ يَحُدّ الغلافَ.**
     *
     * **الحدُّ الأوّلُ خطّافٌ متزامن، والثاني هنا داخلَ الغلاف نفسِه** —
     * فمن خرج والغلافُ مركّبٌ لا يبقى رصيدُه معروضاً حتّى النبضةِ التالية.
     */
    @Test
    fun guestTransitionResetsShell() {
        assertTrue(
            "**التحوّلُ إلى ضيفٍ لا يَحُدّ الغلاف** — فيبقى حالُ الحساب معروضاً لضيف",
            read(mainActivity).contains("if (!guest) shell.wake() else shell.reset()"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **حالُ الغلاف — الرصيدُ والبريدُ والملخّصُ لا تُعرض لثانٍ**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-004-08 · `reset()` تمحو حالَ الحساب كلَّها.** */
    @Test
    fun shellResetClearsAccountState() {
        val b = resetBody()
        assertTrue("**بقي الملخّص**", b.contains("me = null"))
        assertTrue("**بقي الرصيد**", b.contains("balance = 0"))
        assertTrue("**بقي عدّادُ البريد**", b.contains("unread = 0"))
        assertTrue("**بقي الصندوق**", b.contains("inbox = null"))
    }

    // ══════════════════════════════════════════════════════════════════
    // **نبضةُ الإنعاش — تعود الشاشاتُ ضيفاً**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-004-09 · وخطّافُ الخروج يبثّ نبضةَ إنعاش.** */
    @Test
    fun logoutBumpsRefresh() {
        assertTrue(
            "**لا نبضةَ إنعاشٍ عند الخروج** — فتبقى شاشاتٌ على حالِ من سبق",
            afterLogoutReg().contains("Refresh.bump()"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **العقدُ الخلفيُّ محروسٌ — الحدُّ في الجهاز لا يُغني عنه**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **CUST-DEF-004-10 · والمحرّكُ يفرض الملكيّةَ بحسابين، ٤٠٤ لا ٤٠٣.**
     *
     * **حدُّ الجهاز لا يحمي وحدَه** — من نادى المحرّكَ بمعرّفِ غيرِه يجب
     * أن يُردّ. **وهذا يثبت أنّ حارسَ الخادم مُختبَرٌ** فلا يسقط سطرُ
     * الملكيّة سهواً بلا صراخ.
     */
    @Test
    fun backendOwnershipRegressionExists() {
        val g = read(isolation)
        assertTrue("**غاب اختبارُ قراءةِ الدخيل**", g.contains("TestOrder_IntruderCannotRead"))
        assertTrue("**غاب اختبارُ إلغاءِ الدخيل**", g.contains("TestOrder_IntruderCannotCancel"))
        assertTrue("**غاب اختبارُ حذفِ العنوان**", g.contains("TestAddress_IntruderCannotDelete"))
        assertTrue("**غاب اختبارُ المحفظة**", g.contains("TestWallet_ShowsOnlyOwnBalance"))
        assertTrue("**غابت قاعدةُ ٤٠٤ لا ٤٠٣**", g.contains("StatusNotFound"))
    }
}

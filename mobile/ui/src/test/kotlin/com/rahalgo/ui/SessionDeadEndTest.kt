package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **جلسةٌ ماتت ولا بابَ للخروج منها** — `GAP-DRV-SESSION-01`.
 *
 * # ما وقع على الجهاز
 *
 * **شُهد على `SM-A525F` ٢٠٢٦-٠٩-٢٩ والطلبُ في يد السائق عند باب الزبون**:
 * ضُغط «سلمت الطلب»، فالتُقطت صورةُ الإثبات، **ثمّ سقط النداءُ وظهرت «انتهت
 * جلستك — ادخل من جديد» مرّتين في الشاشة** — **ولا طريقَ إلى الدخول**: لا
 * ينتقل إلى شاشة الدخول، **وإعادةُ تشغيل التطبيق لا تنقله**، **ولا زرَّ
 * خروجٍ في «حسابي» أصلاً.**
 *
 * # والجذرُ سطرٌ واحد
 *
 * في `ApiClient.call`:
 *
 * ```
 * if (e.status != 401 || session.refreshToken().isEmpty()) throw e
 * ```
 *
 * **فإن وصلت ٤٠١ ولا توكنَ تجديدٍ في اليد، يُرمى الخطأُ بلا أن يُنادى
 * `onSessionRejected`** — **والخطّافُ موجودٌ وموصولٌ بخروجٍ قسريّ، لكنّه لا
 * يُطلَق في هذا الطريق.** فتُعرَض الرسالةُ نصّاً في موضعها **وصاحبُها
 * «داخلٌ» ظاهريّاً**، وهو عينُ ما تمنعه `Obs 3` في الطريق الآخر.
 *
 * **والفرقُ بين الطريقين لا يُقرأ من الشاشة**: كلاهما ٤٠١ وكلاهما جلسةٌ
 * ماتت — **والمستخدمُ يرى الرسالةَ نفسَها ويَعلق في أحدهما فقط.**
 */
class SessionDeadEndTest {

    private fun mobileRoot(): File {
        var d: File = File("").absoluteFile
        while (d.parentFile != null && !File(d, "settings.gradle.kts").exists()) {
            d = d.parentFile!!
        }
        return d
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val client = "shared/src/main/kotlin/com/rahalgo/shared/net/ApiClient.kt"

    /**
     * **DEAD-01 · ٤٠١ بلا توكنِ تجديدٍ جلسةٌ ماتت كذلك — فتُساق مثلَها.**
     *
     * **ولا يُقاس بوجود النصّ في الملفّ** — الخطّافُ منادى في فرع فشل التجديد
     * أصلاً. **يُقاس في الفرع الذي يرمي مباشرةً**: من كتب `throw e` بلا نداءٍ
     * قبله ترك البابَ مغلقاً.
     */
    @Test
    fun deadSessionWithoutRefreshTokenIsAlsoRouted() {
        val src = read(client)
        val call = body(src, "suspend inline fun <reified T> call(", "\n    /** نداء بلا تجديد")

        // **المقطعُ من أوّل `catch` إلى أوّل `refresh()`** — وهو الطريقُ الذي
        // كان يرمي صامتاً.
        val early = body(call, "} catch (e: ApiException) {", "refresh()")
        assertTrue(
            "**٤٠١ بلا توكنِ تجديدٍ تُرمى بلا `onSessionRejected`** — " +
                "فيرى السائقُ «انتهت جلستك» **ولا بابَ يخرج منه**، والطلبُ في يده. " +
                "(شُهد على الجهاز ٢٠٢٦-٠٩-٢٩ — GAP-DRV-SESSION-01.)",
            early.contains("onSessionRejected?.invoke("),
        )
    }

    /**
     * **DEAD-02 · ولا يُطلَق الخطّافُ على خطأٍ ليس ٤٠١.**
     *
     * **وخروجٌ قسريٌّ على خطأِ شبكةٍ أسوأُ من رسالةٍ**: من سقط نداؤه لانقطاعٍ
     * لحظيٍّ يجد نفسَه خارجاً، **ويعيد الدخولَ والطلبُ في يده.**
     */
    @Test
    fun nonAuthErrorsDoNotForceLogout() {
        val src = read(client)
        val call = body(src, "suspend inline fun <reified T> call(", "\n    /** نداء بلا تجديد")
        val early = body(call, "} catch (e: ApiException) {", "refresh()")
        assertTrue(
            "**الخطّافُ يُطلَق بلا شرطِ ٤٠١** — فيخرج من سقط نداؤه لانقطاعٍ لحظيّ",
            early.contains("e.status == 401"),
        )
    }
}

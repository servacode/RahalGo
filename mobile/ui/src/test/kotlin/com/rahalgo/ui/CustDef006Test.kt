package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-006 — الرفضُ النهائيُّ للموقع يُميَّز، وله علاجٌ لا حلقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **ردُّ نتيجةِ طلبِ الإذن كان يكتب `PERMISSION_DENIED` دائماً** — **فمن
 * رُفض نهائيّاً («لا تسأل ثانيةً») يُعاد طلبُه فيرفضه النظامُ بلا نافذةٍ**،
 * حلقةٌ صامتة. و`PERMISSION_PERMANENT` وعلاجُه (صفحةُ الإعدادات) شيفرةٌ
 * ميّتةٌ لا مُنتِجَ لها.
 *
 * # العقدُ بعد الإصلاح
 *
 * **محرّكٌ مركزيٌّ واحد** — `Here.deniedProblem(activity)`: يقرأ العقدَ
 * الرسميَّ لأندرويد (`shouldShowRequestPermissionRationale`) في ردّ
 * النتيجة: صحيحٌ ⇒ `PERMISSION_DENIED` (يُطلَب ثانيةً)؛ خطأٌ ⇒
 * `PERMISSION_PERMANENT` (لا نافذةَ بعدها — الإعدادات). **والمنتجُ في
 * الزبون يستدعيه** بدل الكتابة العمياء. **والعلاجُ قائمٌ**: الرفضُ
 * النهائيُّ يفتح `openAppSettings`. **والعودةُ من الإعدادات بالإذن تُنعش
 * الحالَ** في `ON_RESUME`. **ولا حيلةَ صانع.**
 *
 * # لماذا حراسةُ مصدرٍ لا رسمُ شاشة
 *
 * الإذنُ ونتيجتُه يحتاجان جهازاً/`Robolectric` لا نستعملها؛ **فالمقيسُ
 * القرارُ**: أنّ المنتجَ يمرّ بالمحرّك الواحد، وأنّ المحرّك يقرأ العقدَ
 * الرسميّ، وأنّ العلاجَ والإنعاشَ موصولان.
 */
class CustDef006Test {

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

    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val hereMap = "map/src/main/kotlin/com/rahalgo/map/Here.kt"
    private val hereCust = "app-customer/src/main/kotlin/com/rahalgo/customer/Here.kt"
    private val main = "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
    private val locating = "ui/src/main/kotlin/com/rahalgo/ui/Locating.kt"

    /** **CUST-DEF-006-01 · المحرّكُ المركزيُّ موجودٌ ويقرأ العقدَ الرسميّ.** */
    @Test
    fun centralDeniedProblemReadsRationale() {
        val b = body(read(hereMap), "fun deniedProblem(", "\n    @", "\n    fun ", "\n    const ", "\n}")
        assertTrue(
            "**`deniedProblem` لا يقرأ `shouldShowRequestPermissionRationale`** — فلا يُميّز النهائيّ",
            b.contains("shouldShowRequestPermissionRationale"),
        )
        assertTrue(
            "**العقدُ لا يُنتج `PERMISSION_PERMANENT`**",
            b.contains("PERMISSION_PERMANENT"),
        )
        assertTrue(
            "**العقدُ لا يُبقي `PERMISSION_DENIED` للحالة القابلة للطلب**",
            b.contains("PERMISSION_DENIED"),
        )
    }

    /**
     * **CUST-DEF-006-02 · الرفضُ النهائيُّ حين لا تُعرَض النافذةُ (rationale=false).**
     *
     * **فمن قلبَ الفرعين أسقط هذا** — والنهائيُّ يُقرأ عاديّاً فيُعاد طلبُه.
     */
    @Test
    fun permanentWhenRationaleFalse() {
        val b = body(read(hereMap), "fun deniedProblem(", "\n    @", "\n    fun ", "\n    const ", "\n}")
        val ifTrue = b.indexOf("PERMISSION_DENIED")
        val elseArm = b.indexOf("PERMISSION_PERMANENT")
        // rationale صحيح (يُطلَب ثانيةً) يسبق، ثمّ else النهائيّ
        assertTrue("**غاب أحدُ الفرعين**", ifTrue >= 0 && elseArm >= 0)
        assertTrue(
            "**النهائيُّ ليس في فرع `else`** — قد يُقرأ الرفضُ الأوّلُ نهائيّاً",
            ifTrue < elseArm,
        )
    }

    /**
     * **CUST-DEF-006-03 · المنتجُ في الزبون يمرّ بالمحرّك — لا كتابةٌ عمياء.**
     * (الشاهدُ السالبُ: إرجاعُه إلى `PERMISSION_DENIED` ثابتاً يُسقط هذا.)
     */
    @Test
    fun customerProducerRoutesThroughEngine() {
        val src = read(main)
        val cb = body(src, "ActivityResultContracts.RequestPermission()", "\n    val mapPicker")
        assertTrue(
            "**ردُّ الإذن لا يستدعي `Here.deniedProblem`** — عاد يكتب الرفضَ أعمى",
            cb.contains("Here.deniedProblem"),
        )
    }

    /** **CUST-DEF-006-04 · والعلاجُ قائمٌ: النهائيُّ يفتح الإعدادات.** */
    @Test
    fun permanentIsHealedByAppSettings() {
        val b = body(read(locating), "fun fixProblem(", "\n}")
        val perm = b.indexOf("PERMISSION_PERMANENT")
        val open = b.indexOf("openAppSettings", perm.coerceAtLeast(0))
        assertTrue(
            "**النهائيُّ لا يُفتح له `openAppSettings`** — لا مخرجَ للزبون",
            perm >= 0 && open > perm,
        )
    }

    /** **CUST-DEF-006-05 · والعودةُ من الإعدادات بالإذن تُنعش الحال (ON_RESUME).** */
    @Test
    fun resumeRefreshesWhenGranted() {
        val src = read(main)
        val resume = body(src, "Lifecycle.Event.ON_RESUME", "onDispose")
        assertTrue(
            "**لا إنعاشَ عند العودة** — من مُنح الإذنَ في الإعدادات يبقى أمام لافتةِ الرفض",
            resume.contains("PERMISSION_PERMANENT") &&
                resume.contains("Here.granted") &&
                resume.contains("Here.refresh"),
        )
    }
}

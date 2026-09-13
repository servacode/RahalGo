package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **DWR-10 · الأربعةُ على السياسة الواحدة** (`DWR`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وسياسةُ الإيماءة تُقاس بالنداء** (`DrawerGesturePolicyTest`)
 * **وسلوكُها على الجهاز** (`DrawerGestureUiTest`) — **وهذا يقيس ما لا
 * يُقاس بواحدٍ منهما**: **أنّ التطبيقات الأربعة تقرأ السياسةَ نفسَها.**
 *
 * **والبنيةُ تُقاس بالنصّ لأنّها نصّ**: **من أعاد `gesturesEnabled =
 * false` في تطبيقٍ عطّل الإيماءةَ فيه وحدَه** — **ولا فحصَ سلوكٍ في
 * `ui` يرى ذلك**، لأنّ قشرةَ كلّ تطبيقٍ في وحدته.
 *
 * **وهو الخطرُ الواقعيّ**: **التعطيلُ العامُّ كان الحلَّ القديمَ
 * المكتوبَ في الأربعة** (٢٠٢٦-٠٩-٠١)، **فعودتُه سطرٌ واحدٌ ينساه
 * مراجع.**
 */
class DrawerCentralPolicyTest {

    private companion object {
        val SHELLS = listOf(
            "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt",
            "app-driver/src/main/kotlin/com/rahalgo/driver/MainActivity.kt",
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/MainActivity.kt",
            "app-rep/src/main/kotlin/com/rahalgo/rep/MainActivity.kt",
        )
        val MAPS = listOf(
            "map/src/main/kotlin/com/rahalgo/map/MapCanvas.kt",
            "map/src/main/kotlin/com/rahalgo/map/PickPoint.kt",
        )
    }

    /**
     * **وجذرُ المستودع يُبلَغ صعوداً** — **ومجلَّدُ تشغيل الفحص وحدةٌ لا
     * جذرٌ**، **ومسارٌ مثبَّتٌ يسقط إن نُقل الفحص.**
     */
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

    @Test
    fun `القشورُ الأربعُ تقرأ السياسةَ المركزيّة`() {
        val root = mobileRoot()
        for (rel in SHELLS) {
            val f = File(root, rel)
            assertTrue("لم أجد قشرةً: " + rel, f.exists())
            val text = f.readText()
            assertTrue(
                "**قشرةٌ لا تقرأ السياسةَ المركزيّة**: " + rel,
                text.contains("DrawerGestures.enabledFor(drawer)"),
            )
            assertFalse(
                "**عاد التعطيلُ العامُّ** في " + rel +
                    " — **فالسحبُ لا يعمل وقد طُلب أن يعمل**",
                text.contains("gesturesEnabled = false"),
            )
        }
    }

    /**
     * **وسطحا الخريطة المشتركان يُعلنان القفلَ بنفسهما.**
     *
     * **ومن نزع النداءَ من أحدهما أعاد العطبَ القديم**: **سحبُ الخريطة
     * يفتح القائمة** — **ولا يظهر ذلك إلّا بيدٍ على جهاز.**
     */
    @Test
    fun `سطحا الخريطة يُعلنان القفل`() {
        val root = mobileRoot()
        for (rel in MAPS) {
            val f = File(root, rel)
            assertTrue("لم أجد سطحَ خريطة: " + rel, f.exists())
            assertTrue(
                "**سطحُ خريطةٍ لا يقفل إيماءةَ الدرج**: " + rel,
                f.readText().contains("MapGestureLock()"),
            )
        }
    }
}

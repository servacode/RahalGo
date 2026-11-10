package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ نظام التصميم الموحَّد** (٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مبدأُ المالك: «التصميم لازم يكون موحّد بكلّ التطبيقات، مشان إذا قرّرت يوماً طوّر التصميم
 *  يتطوّر معي بسهولة».)
 *
 * **والتوحيدُ لا يبقى إلّا محروساً**: شاشةٌ واحدةٌ تكتب `Card(` من Material أو `fontSize = 14.sp`
 * أو `Color(0xFF…)` تخرج من النظام، **ويومَ يتغيّر الشكلُ في ملفٍّ واحدٍ تبقى هي على القديم.**
 *
 * **ويُقاس بالنصّ لأنّه نصّ** — كحارس الدرج (`DrawerCentralPolicyTest`).
 *
 * # ما يُفحص (في `src/main` للتطبيقات الأربعة)
 * 1. **لا قطعَ Material الخامّ** — `Card`/`ElevatedCard`/`OutlinedCard`/`AlertDialog`/`Button`/
 *    `TextButton`/`OutlinedButton`/`FilledTonalButton`/`ElevatedButton`: بدلُها `RahalCard`
 *    و`RahalDialog` وأزرارُ `Buttons.kt`.
 * 2. **لا `fontSize = N.sp`** — دورٌ من `Rahal.type`.
 * 3. **لا `Color(0x…)`** — لونٌ من `Rahal.colors`.
 * 4. **و`ui` نفسُها على أدوار الخطّ** — لا `MaterialTheme.typography` فيها.
 *
 * # الاستثناءات — قصيرةٌ ومشروحة
 * كلُّ استثناءٍ مسارُ ملفٍّ وسببُه. **ومن أضاف واحداً كتب لماذا**، ومن أزال سببَه أزاله.
 */
class DesignSystemGuardTest {

    private companion object {
        val APPS = listOf("app-customer", "app-driver", "app-merchant", "app-rep")

        /** **قطعُ Material التي لها بديلٌ في `ui`.** */
        val RAW_MATERIAL = listOf(
            "Card", "ElevatedCard", "OutlinedCard", "AlertDialog",
            "Button", "TextButton", "OutlinedButton", "FilledTonalButton", "ElevatedButton",
        )

        /**
         * **استثناءاتُ `fontSize`** — مسارٌ نسبيٌّ إلى `mobile` ← سببُه.
         */
        val FONT_SIZE_ALLOWED = mapOf(
            // اسمُ الصنف في بطاقة المنتج ١٤ ثخين بقرار المالك (٢٠٢٦-١١-١١، D7) — لا دورَ بـ١٤ في السلّم.
            "app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt" to "D7: owner decision 14sp bold",
        )

        /** **استثناءاتُ `Color(0x…)`** — فارغةٌ اليوم. */
        val COLOR_ALLOWED = emptyMap<String, String>()

        val FONT_SIZE = Regex("""fontSize\s*=\s*\d+(\.\d+)?\.sp\b""")
        val RAW_COLOR = Regex("""\bColor\(\s*0x""")
    }

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun sources(module: String): List<File> {
        val main = File(mobileRoot(), "$module/src/main")
        assertTrue("لم أجد " + main.path, main.exists())
        return main.walkTopDown().filter { it.isFile && it.extension == "kt" }.toList()
    }

    private fun rel(f: File): String = f.relativeTo(mobileRoot()).invariantSeparatorsPath

    /** **الشيفرةُ بلا تعليقات** — التعليقُ يذكر `Color(0x…)` شرحاً ولا يرسمه. */
    private fun codeLines(f: File): List<Pair<Int, String>> {
        var inBlock = false
        return f.readLines().mapIndexedNotNull { i, raw ->
            var line = raw
            if (inBlock) {
                val end = line.indexOf("*/")
                if (end < 0) return@mapIndexedNotNull null
                line = line.substring(end + 2)
                inBlock = false
            }
            val start = line.indexOf("/*")
            if (start >= 0 && line.indexOf("*/", start) < 0) {
                inBlock = true
                line = line.substring(0, start)
            }
            val sl = line.indexOf("//")
            if (sl >= 0) line = line.substring(0, sl)
            (i + 1) to line
        }
    }

    @Test
    fun `التطبيقات لا تستعمل قطع Material الخامّ`() {
        val bad = mutableListOf<String>()
        val imports = RAW_MATERIAL.map { Regex("""^import androidx\.compose\.material3\.$it\s*$""") }
        val qualified = RAW_MATERIAL.map { Regex("""androidx\.compose\.material3\.$it\s*\(""") }
        for (app in APPS) for (f in sources(app)) {
            for ((n, line) in codeLines(f)) {
                val t = line.trim()
                if (t == "import androidx.compose.material3.*" ||
                    imports.any { it.matches(t) } ||
                    qualified.any { it.containsMatchIn(line) }
                ) {
                    bad += rel(f) + ":" + n + "  " + t
                }
            }
        }
        assertTrue(
            "**قطعةُ Material خامّة في تطبيق** — البديل RahalCard · RahalDialog · RahalButton/" +
                "RahalOutlineButton/RahalTextButton/RahalSmallButton:\n" + bad.joinToString("\n"),
            bad.isEmpty(),
        )
    }

    @Test
    fun `التطبيقات لا تكتب حجم خطّ برقم`() {
        val bad = mutableListOf<String>()
        for (app in APPS) for (f in sources(app)) {
            if (rel(f) in FONT_SIZE_ALLOWED) continue
            for ((n, line) in codeLines(f)) {
                if (FONT_SIZE.containsMatchIn(line)) bad += rel(f) + ":" + n + "  " + line.trim()
            }
        }
        assertTrue("**حجمُ خطٍّ برقم** — البديل دورٌ من Rahal.type:\n" + bad.joinToString("\n"), bad.isEmpty())
    }

    @Test
    fun `التطبيقات لا تكتب لوناً برقم`() {
        val bad = mutableListOf<String>()
        for (app in APPS) for (f in sources(app)) {
            if (rel(f) in COLOR_ALLOWED) continue
            for ((n, line) in codeLines(f)) {
                if (RAW_COLOR.containsMatchIn(line)) bad += rel(f) + ":" + n + "  " + line.trim()
            }
        }
        assertTrue("**لونٌ برقم** — البديل Rahal.colors:\n" + bad.joinToString("\n"), bad.isEmpty())
    }

    @Test
    fun `العدّة المشتركة على أدوار الخطّ`() {
        val bad = mutableListOf<String>()
        for (f in sources("ui")) for ((n, line) in codeLines(f)) {
            if (line.contains("MaterialTheme.typography.")) bad += rel(f) + ":" + n + "  " + line.trim()
        }
        assertTrue("**`MaterialTheme.typography` في ui** — البديل Rahal.type:\n" + bad.joinToString("\n"), bad.isEmpty())
    }

    /** **والاستثناءُ الذي زال سببُه يُزال** — وإلّا صار بابا خلفيّاً لمن بعده. */
    @Test
    fun `لا استثناء بلا موضع`() {
        val root = mobileRoot()
        for (path in FONT_SIZE_ALLOWED.keys) {
            val f = File(root, path)
            assertTrue("استثناءٌ لملفٍّ غيرِ موجود: $path", f.exists())
            assertTrue(
                "استثناءُ fontSize لم يعد يلزم — أزله: $path",
                codeLines(f).any { FONT_SIZE.containsMatchIn(it.second) },
            )
        }
    }
}

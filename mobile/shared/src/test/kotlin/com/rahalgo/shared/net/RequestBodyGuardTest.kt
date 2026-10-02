package com.rahalgo.shared.net

import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جسمُ النداء صنفٌ موسوم — لا `JsonObject` ولا خريطةٌ مختلطة** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بلاغُ المالك على جهازه**: «لدي مشكلة» في الطريق إلى المتجر — «لم يصل البلاغُ للعمليات».
 * **قِيس في سجلّ الجهاز**: `SerializationException: Serializer for class '…' is not found`.
 *
 * **ونسخةُ التجهيز مضغوطةٌ منذ ٢٠٢٦-١٠-٠١** — والمكتبةُ تخمّن نوعَ الجسم في زمن التشغيل:
 * `buildJsonObject` يسقط (`JsonLiteral` بلا مُسلسِل، قِيس ٢٠٢٦-٠٩-٢٩)، **وكذلك خريطةٌ قيمُها
 * قوائمُ أصناف أو أنواعٌ مختلطة.** **واختبارُ الوحدة لا يُضغَط فلا يرى شيئاً من هذا** — فهذا
 * الحارسُ يقرأ النصَّ ويمنع الصيغتين في أجسام النداءات.
 */
class RequestBodyGuardTest {

    private fun apiSources(): List<File> {
        val roots = listOf(File("src/main/kotlin"), File("shared/src/main/kotlin"))
        val root = roots.firstOrNull { it.isDirectory } ?: error("لا مجلّدَ مصدر")
        return root.walkTopDown().filter { it.isFile && it.name.endsWith("Api.kt") }.toList()
    }

    @Test
    fun `لا buildJsonObject في أجسام النداءات`() {
        // **الاستعمالُ لا الذكر** — `buildJsonObject {` (والتعليقُ يذكره ليشرح).
        val use = Regex("""buildJsonObject\s*\{""")
        val bad = apiSources().filter { use.containsMatchIn(it.readText()) }.map { it.name }
        assertTrue("**JsonObject يسقط في النسخة المضغوطة**: $bad", bad.isEmpty())
    }

    @Test
    fun `لا قائمة أصناف داخل خريطة جسم`() {
        // **الصيغةُ التي سقطت**: mapOf("points" to points) · mapOf("route_id" to …, "fixes" to fixes)
        val pattern = Regex("""mapOf\([^)]*"(points|fixes)"\s+to""")
        val bad = apiSources().filter { pattern.containsMatchIn(it.readText()) }.map { it.name }
        assertTrue("**قائمةٌ داخلَ خريطةٍ تُخمَّن في زمن التشغيل**: $bad", bad.isEmpty())
    }
}

package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديثٌ أُغلق لوحُه لا يُجلب** (فحصُ ما قبل الإصدار ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما كان يقع
 *
 * **نموذجُ الحديث واحدٌ للنشاط، و`current` لا يُمحى** — فمن فتح حديثَ طلبٍ مرّةً بقي كلُّ خبرٍ جديدٍ
 * على ذاك الطلب يُجلَب في الخلفية. **والجلبُ نفسُه وسمُ القراءة في المحرّك**: يرى السائقُ «قُرئت»
 * والزبونُ لم يفتح شيئاً، وتسقط شارةُ القرص قبل أن يراها.
 *
 * # وما يُقاس هنا
 *
 * **العدّادُ بلا جهاز** (`ShownCount` منطقٌ خالص)، **وتوصيلُه** (حراسةُ مصدر): النموذجُ لا يُنادى
 * إلّا من اللوح، واللوحُ يُعلن ظهورَه قبل الجلب وزوالَه حين يخرج من التركيب، **وقارئُ الحديث يُخبر
 * صاحبَ الشارة من اللوح نفسِه** فلا يُنسى بابٌ.
 *
 * **ولا يُقاس هنا تركيبُ Compose** — ذاك على جهاز.
 */
class ShownCountTest {

    // ═════════════ العدّاد ═════════════

    @Test
    fun `the chat is forgotten only when the last sheet goes`() {
        var forgotten = 0
        val shown = ShownCount { forgotten++ }

        shown.attach()
        shown.attach()
        assertEquals(2, shown.count)

        // **زال لوحٌ وبقي آخر** — حديثُ الباقي حاضرٌ فلا يُنسى.
        shown.detach()
        assertEquals("a sheet is still on screen, the chat must stay", 0, forgotten)
        assertEquals(1, shown.count)

        shown.detach()
        assertEquals("the last sheet went, the chat is forgotten once", 1, forgotten)
        assertEquals(0, shown.count)
    }

    @Test
    fun `an extra detach neither goes below zero nor forgets twice`() {
        var forgotten = 0
        val shown = ShownCount { forgotten++ }

        shown.attach()
        shown.detach()
        shown.detach()
        shown.detach()
        assertEquals(0, shown.count)
        assertEquals(1, forgotten)

        // **وبعد الفكّ الزائد يظهر لوحٌ واحد فيُحسب واحداً** — لا صفراً ناقصاً.
        shown.attach()
        assertEquals(1, shown.count)
        shown.detach()
        assertEquals(2, forgotten)
    }

    /**
     * **لوحٌ يحلّ محلَّ لوحٍ في الإطار نفسِه** — والتركيبُ لا يضمن أيَّهما أوّلاً.
     *
     * **إن ركّب الجديدُ قبل أن يُفكَّ القديمُ** بقي الحديث، **وإن فُكَّ القديمُ أوّلاً** نُسي ثمّ جلبه الجديدُ
     * من الصفر — **وفي الحالين لا يبقى حديثٌ قديمٌ تجلبه نبضةٌ بعد أن يزول الجميع.**
     */
    @Test
    fun `a sheet replacing another keeps working in either order`() {
        var forgotten = 0
        val shown = ShownCount { forgotten++ }

        // **الجديدُ أوّلاً**
        shown.attach()
        shown.attach()
        shown.detach()
        assertEquals(0, forgotten)
        assertEquals(1, shown.count)

        // **ثمّ يغلق الأخيرُ فيُنسى**
        shown.detach()
        assertEquals(1, forgotten)

        // **القديمُ أوّلاً**
        shown.attach()
        shown.detach()
        shown.attach()
        assertEquals(2, forgotten)
        assertEquals(1, shown.count)
    }

    // ═════════════ والتوصيل ═════════════

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    /** **ونهاياتُ الأسطر تُوحَّد قبل المطابقة** — جيتٌ على ويندوز يكتب CRLF. */
    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("ملفٌّ غائب: $rel", f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val chatFile = "ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt"
    private val chatsFile = "ui/src/main/kotlin/com/rahalgo/ui/Chats.kt"

    /**
     * **ما يوقف النبضةَ هو محوُ `current`** — شرطُ المجمِّع (`CP-01`) هو هو لم يتبدّل، **ونسيانُ الحديث
     * يُفرغ ما يقرؤه.** وإن بقي `current` بعد آخرِ لوحٍ عاد العطب.
     */
    @Test
    fun `forgetting the chat empties what the pulse reads`() {
        val src = read(chatFile)
        val forget = src.substringAfter("private fun forget() {").substringBefore("\n    }")
        assertTrue("forget لا تمحو الطلب الجاري", forget.contains("current = \"\""))
        assertTrue("forget لا تمحو الرأس", forget.contains("head = null"))
        assertTrue("forget لا تمحو الرسائل", forget.contains("messages = emptyList()"))
        assertTrue(
            "تبدّل شرطُ النبضة",
            src.contains("Refresh.tick.drop(1).collect { if (current.isNotEmpty()) load(current) }"),
        )
    }

    @Test
    fun `the model only forgets through the counter`() {
        val src = read(chatFile)
        assertTrue("العدّادُ لا يستدعي النسيان", src.contains("private val shown = ShownCount { forget() }"))
        assertTrue("attach لا تمرّ بالعدّاد", src.contains("fun attach() = shown.attach()"))
        assertTrue("detach لا تمرّ بالعدّاد", src.contains("fun detach() = shown.detach()"))
    }

    /**
     * **اللوحُ يُعلن ظهورَه قبل أن يجلب ويُعلن زوالَه حين يخرج من التركيب** — ومفتاحُ الأثر النموذجُ
     * لا الطلب، فالتبديلُ بين الطلبات لا يكون زوالاً وظهوراً.
     */
    @Test
    fun `the sheet attaches before it loads and detaches when it leaves`() {
        val sheet = read(chatFile).substringAfter("fun OrderChatSheet(")
        val effect = sheet.indexOf("DisposableEffect(vm) {")
        val attach = sheet.indexOf("vm.attach()")
        val detach = sheet.indexOf("onDispose { vm.detach() }")
        val load = sheet.indexOf("LaunchedEffect(orderId) { vm.load(orderId) }")
        assertTrue("اللوحُ لا يُعلن ظهورَه", effect >= 0 && attach > effect)
        assertTrue("اللوحُ لا يُعلن زوالَه", detach > attach)
        assertTrue("الجلبُ يسبق إعلانَ الظهور", load > detach)
    }

    /**
     * **قارئُ الحديث يُخبر صاحبَ الشارة من اللوح نفسِه** — عند الفتح والتبديل (`onOpened`)، وكلَّما وصل
     * الحديثُ (`onLoaded`: وصولُه هو وسمُ القراءة في المحرّك).
     */
    @Test
    fun `the sheet tells the badge owner when it opens and on every arrival`() {
        val sheet = read(chatFile).substringAfter("fun OrderChatSheet(")
        assertTrue("onOpened غائب", sheet.contains("onOpened: (orderId: String) -> Unit = {}"))
        assertTrue("onLoaded غائب", sheet.contains("onLoaded: (orderId: String) -> Unit = {}"))
        assertTrue("الفتحُ والتبديلُ لا يُخبران", sheet.contains("LaunchedEffect(orderId) { opened(orderId) }"))
        assertTrue(
            "وصولُ الحديث لا يُخبر",
            sheet.contains("LaunchedEffect(head) { if (head != null) loaded(orderId) }"),
        )
        // **والمنادى يُقرأ حيّاً** — لا نسخةً قديمةً التقطها الأثرُ في أوّل تركيب.
        assertTrue(sheet.contains("val opened by rememberUpdatedState(onOpened)"))
        assertTrue(sheet.contains("val loaded by rememberUpdatedState(onLoaded)"))
    }

    /** **و«دردشاتي» بابٌ ثانٍ لما يفتحه القرصُ** — تمرّر المنادِيَين نفسَيهما إلى اللوح نفسِه. */
    @Test
    fun `the chats list forwards the two hooks to the same sheet`() {
        val chats = read(chatsFile)
        assertTrue("onOpened غائب عن ChatsScreen", chats.contains("onOpened: (orderId: String) -> Unit = {}"))
        assertTrue("onLoaded غائب عن ChatsScreen", chats.contains("onLoaded: (orderId: String) -> Unit = {}"))
        assertTrue(
            "ChatsScreen لا تمرّر المنادِيَين إلى اللوح",
            chats.contains("OrderChatSheet(vm = chatVm, orderId = writing, onOpened = onOpened, onLoaded = onLoaded)"),
        )
    }
}

package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديثُ الزبون حين تكون له طلباتٌ كثيرة** (`CU-CHAT`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا حراسةُ مصدرٍ لا رسمُ شاشة
 *
 * **والعطبُ المقيسُ ليس في الرسم** — **هو في القرار**: أيَّ طلبٍ يفتح
 * القرصُ، وما الذي تقوله الشارة، **ومتى يُمحى ما كان قبل أن يُرسَم.**
 *
 * **وما يُقاس هنا ثلاثةٌ**: تنقيةُ الوجهة (منطقٌ خالصٌ يُنادى)،
 * **والعقودُ التي لا يجوز أن تنقلب صامتةً** (حراسةُ مصدر)،
 * **وأنّ ما أُصلح لا يعود.**
 *
 * **ولا يُقاس هنا وصولُ الدفع** — **ذاك على جهازٍ حقيقيّ**
 * (`FIELD-TESTS`).
 */
class ChatMultiOrderTest {

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
        return f.readText()
    }

    private val liveChat =
        "app-customer/src/main/kotlin/com/rahalgo/customer/chat/LiveChat.kt"
    private val mainActivity =
        "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
    private val chats = "ui/src/main/kotlin/com/rahalgo/ui/Chats.kt"
    private val sheet = "ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt"

    // ═════════════ CU-CHAT-01 · قائمةٌ لا صفٌّ واحد ═════════════

    /** **CU-CHAT-01 · والقرصُ يعرف كلَّ أحاديثه الجارية لا أوّلَها.** */
    @Test
    fun `القرصُ يحمل قائمةَ الأحاديث لا معرّفاً واحداً`() {
        val src = read(liveChat)
        assertTrue(
            "**عاد القرصُ إلى معرّفٍ واحد**",
            src.contains("var chats by mutableStateOf<List<Live>>"),
        )
        assertFalse(
            "**عاد اختيارُ أوّلِ طلبٍ صامتاً**",
            src.contains("orders.firstOrNull { !it.driverName.isNullOrEmpty() }?.id"),
        )
    }

    /** **CU-CHAT-02 · وكلُّ حديثٍ يحمل عدَّه هو.** */
    @Test
    fun `لكلِّ حديثٍ عدُّ ما لم يُقرأ فيه`() {
        val src = read(liveChat)
        assertTrue(
            "**ضاع العدُّ لكلّ طلب**",
            src.contains("data class Live(val orderId: String, val unread: Int)"),
        )
        assertTrue("**لا يُربَط العدُّ بطلبِه**", src.contains("byOrder[it.id] ?: 0"))
    }

    // ═════════════ CU-CHAT-03 · ما يفتحه القرص ═════════════

    /**
     * **CU-CHAT-03 · وواحدٌ يُفتَح، وأكثرُ يُعرَض ليختار.**
     *
     * **ولا يُختار عن صاحبِه ما لا يعلمه** — **فيكتب ردَّه في الطلب
     * الخطأ ولا يدري.**
     */
    @Test
    fun `واحدٌ يُفتَح وأكثرُ يُعرَض`() {
        assertTrue(
            "**سقط شرطُ الواحد**",
            read(liveChat).contains("if (chats.size == 1) chats[0].orderId else null"),
        )
        val main = read(mainActivity)
        assertTrue(
            "**لا بابَ إلى قائمة الأحاديث**",
            main.contains("overlay.show(Overlay.Menu(CustomerItems.CHATS))"),
        )
        assertTrue("**لا يُسأل عن الواحد قبل الفتح**", main.contains("val one = liveChat.single"))
    }

    // ═════════════ CU-CHAT-04 · الشارة ═════════════

    /**
     * **CU-CHAT-04 · والشارةُ تقول ما يفتحه القرص.**
     *
     * **ومجموعٌ على زرٍّ يفتح واحداً كذبٌ**: يرى «٣» فيجد واحدةً.
     */
    @Test
    fun `الشارةُ لا تجمع ما لا يُفتَح`() {
        assertTrue(
            "**تبدّل عقدُ الشارة**",
            read(liveChat).contains("if (chats.size == 1) chats[0].unread else unread"),
        )
        assertTrue(
            "**عاد المجموعُ إلى القرص**",
            read(mainActivity).contains("ChatFab(unread = liveChat.badge)"),
        )
    }

    // ═════════════ CU-CHAT-05 · القائمةُ الواحدة ═════════════

    /**
     * **CU-CHAT-05 · والجاري في القائمة لا المنتهي وحدَه.**
     *
     * **وشاشةٌ يفتحها القرصُ ثمّ لا يجد فيها ما جاء يقرؤه.**
     */
    @Test
    fun `القائمةُ تعرض الجاريَ أوّلاً ثمّ المنتهي`() {
        val src = read(chats)
        assertFalse("**عادت تصفيةُ الجاري**", src.contains(".filter { !it.open }"))
        assertTrue(
            "**سقط ترتيبُ الجاري أوّلاً**",
            src.contains("compareByDescending<ChatThreadRow> { it.open }"),
        )
    }

    /** **CU-CHAT-06 · والصفُّ يقول حالَه وما ينتظره.** */
    @Test
    fun `صفُّ القائمة يقول الحالَ والعدّ`() {
        val src = read(chats)
        assertTrue("**لا شارةَ في الصفّ**", src.contains("R.string.chat_unread_badge"))
        assertTrue("**لا لفظَ للمنتهي**", src.contains("R.string.chat_closed"))
    }

    /**
     * **CU-CHAT-07 · ولوحُ الحديث واحدٌ في كلّ باب.**
     *
     * **ولوحان يفترقان يجعلان الحديثَ حديثين**: يُصلَح النزولُ إلى آخر
     * سطرٍ في أحدهما ويبقى الآخرُ كما كان.
     */
    @Test
    fun `الجاري يُفتَح للكتابة في اللوح نفسِه`() {
        assertTrue(
            "**لا كتابةَ من القائمة**",
            read(chats).contains("OrderChatSheet(vm = chatVm, orderId = writing)"),
        )
    }

    // ═════════════ CU-CHAT-08 · عزلُ الحال ═════════════

    /** **CU-CHAT-08 · ولا إطارٌ واحدٌ يُرسَم فيه حديثُ طلبٍ آخر.** */
    @Test
    fun `تبديلُ الطلب يمحو ما كان قبل النداء`() {
        val src = read(sheet)
        assertTrue("**سقط المحوُ عند تبديل الطلب**", src.contains("if (current != orderId) {"))
        assertTrue("**لا يُمحى المحتوى**", src.contains("messages = emptyList()"))
        assertTrue("**لا يُمحى الرأس**", src.contains("head = null"))
    }

    /** **وما كُتب لطلبٍ لا ينتقل إلى طلبٍ آخر.** */
    @Test
    fun `المسوّدةُ مقيّدةٌ بطلبِها`() {
        assertTrue(
            "**عادت المسوّدةُ تعبر بين الأحاديث**",
            read(sheet).contains("remember(orderId) { mutableStateOf(\"\") }"),
        )
    }

    /** **وردُّ نداءٍ متأخّرٍ لا يُكتب في شاشةِ طلبٍ آخر.** */
    @Test
    fun `الردُّ المتأخّرُ لا يُكتب في غير طلبِه`() {
        assertEquals(
            "**سقط حارسُ السباق في أحد المسارين**",
            2,
            Regex("if \\(current == orderId\\)").findAll(read(sheet)).count(),
        )
    }

    // ═════════════ CU-CHAT-09 · الرأسُ والبابُ إلى الطلب ═════════════

    /** **CU-CHAT-09 · ورأسُ الحديث يقول أيَّ طلبٍ هو — برقمِه.** */
    @Test
    fun `رأسُ الحديث يقول رقمَ الطلب`() {
        val src = read(sheet)
        assertTrue("**لا رقمَ في الرأس**", src.contains("R.string.chat_of_order"))
        assertTrue("**لا يُقرأ الرقمُ من الردّ**", src.contains("h.orderNumber"))
    }

    /** **ومن الحديث إلى طلبِه بابٌ — ولا يُبحث عنه في شاشةٍ أخرى.** */
    @Test
    fun `من الحديث بابٌ إلى الطلب`() {
        assertTrue(
            "**سقط البابُ إلى الطلب**",
            read(sheet).contains("onOpenOrder: (() -> Unit)? = null"),
        )
        assertTrue(
            "**لا يُوصَل البابُ في تطبيق الزبون**",
            read(mainActivity).contains("onOpenOrder = {"),
        )
    }

    // ═════════════ CU-CHAT-10 · الوجهة ═════════════

    /** **CU-CHAT-10 · وحديثُ الطلب وجهةٌ تُفتَح بمعرّف الطلب.** */
    @Test
    fun `وجهةُ حديثِ الطلب تُقرأ كما هي`() {
        val d = Engagement.route("order_chat", "8b0a8a71-d186-4647-ae3b-9cd3898508bf")
        assertEquals(Engagement.DEST_ORDER_CHAT, d.type)
        assertEquals("8b0a8a71-d186-4647-ae3b-9cd3898508bf", d.id)
        assertFalse(d.isHome)
    }

    /**
     * **CU-CHAT-11 · ووجهةٌ بلا طلبٍ تسقط إلى البيت.**
     *
     * **ولوحُ حديثٍ بلا طلبٍ يُفتَح على نداءٍ فاشلٍ يُقرأ عطبا.**
     */
    @Test
    fun `وجهةُ حديثٍ بلا معرّفٍ تسقط إلى البيت`() {
        for (bad in listOf("", "   ")) {
            assertTrue("**فُتح حديثٌ بلا طلب**", Engagement.route("order_chat", bad).isHome)
        }
        // **ولا رابطٌ حرٌّ ولا مقصدٌ نظاميّ.**
        assertTrue(Engagement.route("order_chat://evil", "x").isHome)
        assertTrue(Engagement.route("https://evil.example", "x").isHome)
    }

    /**
     * **CU-CHAT-12 · ورسالةُ الحديث صنفٌ يوقظ.**
     *
     * **وسؤالٌ في طريقٍ ينتظر جواباً الآن** — **يُقرأ بعد التسليم لا
     * ينفع.**
     */
    @Test
    fun `رسالةُ الحديث توقظ الجهاز في الطرفين`() {
        assertEquals("chat", Engagement.KIND_CHAT)
        assertTrue(
            "**لا توقظ رسالةُ السائق الزبونَ**",
            read("app-customer/src/main/kotlin/com/rahalgo/customer/push/PushService.kt")
                .contains("kind == KIND_ORDER || kind == KIND_CHAT"),
        )
        assertTrue(
            "**لا توقظ رسالةُ الزبون السائقَ**",
            read("app-driver/src/main/kotlin/com/rahalgo/driver/push/PushService.kt")
                .contains("kind == KIND_OFFER || kind == KIND_CHAT"),
        )
    }

    /**
     * **CU-CHAT-13 · ووجهةُ الحديث تُؤخَذ ولا يُبتلع سواها.**
     *
     * **ومن أخذ كلَّ ما ينتظر ابتلع وجهةَ عرضٍ لا يعرف كيف يفتحها** —
     * **فلا تُفتَح ولا تبقى.**
     */
    @Test
    fun `لا تُؤخَذ إلّا وجهةُ الحديث`() {
        val main = read(mainActivity)
        assertTrue(
            "**سقط شرطُ النوع قبل الأخذ**",
            main.contains("if (waiting.type == com.rahalgo.ui.Engagement.DEST_ORDER_CHAT)"),
        )
        assertTrue(
            "**لا يُفتَح الحديثُ بالوجهة**",
            main.contains("liveChat.openId = com.rahalgo.ui.Opened.take().id"),
        )
    }

    /** **CU-CHAT-14 · واللوحُ يُفتَح على الطلب المطلوب لا على آخرِ ما حُمِّل.** */
    @Test
    fun `اللوحُ يُفتَح على طلبٍ بعينه`() {
        assertTrue(
            "**عاد اللوحُ علماً بلا طلب**",
            read(mainActivity).contains("liveChat.openId?.let { id ->"),
        )
        assertTrue(
            "**عاد العلمُ نعم/لا**",
            read(liveChat).contains("var openId by mutableStateOf<String?>(null)"),
        )
    }

    /**
     * **CU-CHAT-15 · ولا قرصَ بلا حديثٍ جارٍ.**
     *
     * **وقرصٌ يُضغط فيفتح فراغاً عطبٌ يُقرأ.**
     */
    @Test
    fun `لا قرصَ حين لا حديثَ جارياً`() {
        assertTrue(
            "**سقط شرطُ وجودِ حديث**",
            read(mainActivity).contains("if (liveChat.chats.isNotEmpty()) {"),
        )
    }

    /**
     * **CU-CHAT-16 · والقرصُ يُطفأ عند الخروج.**
     *
     * **ولوحُ حسابٍ مضى فوق شاشةِ ضيفٍ يعرض حديثَ غيرِه.**
     */
    @Test
    fun `الخروجُ يُطفئ القرصَ واللوح`() {
        val src = read(liveChat)
        val i = src.indexOf("fun clear()")
        assertTrue("**ذهب الإطفاء**", i > 0)
        val body = src.substring(i)
        assertTrue("**بقيت الأحاديث**", body.contains("chats = emptyList()"))
        assertTrue("**بقي اللوحُ مفتوحاً**", body.contains("openId = null"))
        assertTrue("**بقي العدّاد**", body.contains("unread = 0"))
    }

    /**
     * **CU-CHAT-17 · والدفعُ لا يُنادى من شيفرة الحديث.**
     *
     * **ومحرّكُ دفعٍ ثانٍ يعني عقدَ إعادةِ محاولةٍ ثانياً** — **وصفَّ
     * خروجٍ لا يراه أحد.**
     */
    @Test
    fun `لا محرّكَ دفعٍ ثانٍ في الحديث`() {
        val src = read(sheet) + read(chats) + read(liveChat)
        assertFalse("**نداءُ فايربيس من شيفرة الحديث**", src.contains("FirebaseMessaging"))
        assertFalse("**نداءُ دفعٍ مباشر**", src.contains("fcm"))
    }

    /**
     * **CU-CHAT-18 · ولا استجوابَ دوريٌّ يُضاف للحديث.**
     *
     * **والنبضةُ توقظه** — **واستجوابٌ كلَّ ثانيةٍ يستنزف بطّاريّةَ من
     * ينتظر ردّا**، **ولا مقبسَ حيٌّ جديدٌ في هذه الدفعة.**
     */
    @Test
    fun `الحديثُ يُنعَش بالنبضة لا باستجواب`() {
        val src = read(sheet)
        assertTrue("**ذهب الإنعاشُ بالنبضة**", src.contains("Refresh.tick"))
        assertFalse("**مقبسٌ حيٌّ جديد**", src.contains("WebSocket"))
        assertFalse("**استجوابٌ بمهلة**", src.contains("delay("))
    }
}

package com.rahalgo.customer.chat

import com.rahalgo.customer.chat.LiveChatViewModel.Live
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class LiveChatBadgeTest {
    @Test
    fun `badge is the sum of unread across listed open orders`() {
        val chats = listOf(Live("a", 2, 1181), Live("b", 1, 1182))
        assertEquals(3, LiveChatViewModel.totalUnread(chats))
    }

    @Test
    fun `single order and empty list`() {
        assertEquals(4, LiveChatViewModel.totalUnread(listOf(Live("a", 4, 1))))
        assertEquals(0, LiveChatViewModel.totalUnread(emptyList()))
    }

    /**
     * **الحالةُ المقيسةُ في الفحص**: طلبان، لأحدهما رسالتان جديدتان وللآخر واحدة.
     * الزبون يفتح الأوّل ثمّ يبدّل إلى الثاني — **وكلُّ مربّعٍ قُرئ يسقط عدُّه فوراً**
     * (لا ينتظر نبضةً)، **وشارةُ القرص تنزل معه.**
     */
    @Test
    fun `opening then switching zeroes each order at once and the badge follows`() {
        val chats = listOf(Live("a", 2, 1181), Live("b", 1, 1182))

        val afterA = LiveChatViewModel.afterRead(chats, "a")
        assertEquals(listOf(Live("a", 0, 1181), Live("b", 1, 1182)), afterA)
        // **مربّعُ الأوّل لا يحمل «2» بعد أن قُرئ**، والقرصُ يقول ما بقي فقط.
        assertEquals(1, LiveChatViewModel.totalUnread(afterA))

        val afterB = LiveChatViewModel.afterRead(afterA, "b")
        assertEquals(listOf(0, 0), afterB.map { it.unread })
        assertEquals(0, LiveChatViewModel.totalUnread(afterB))
        // **ولا يُمسّ غيرُ العدّ**: المعرّفُ والرقمُ كما كانا.
        assertEquals(listOf("a" to 1181L, "b" to 1182L), afterB.map { it.orderId to it.number })
    }

    /** **وطلبٌ غيرُ موجود، أو لا شيء فيه يُقرأ، لا يُنشئ قائمةً جديدة** — فلا إعادةَ رسمٍ بلا سبب. */
    @Test
    fun `nothing to zero returns the very same list`() {
        val chats = listOf(Live("a", 0, 1181), Live("b", 1, 1182))
        assertSame(chats, LiveChatViewModel.afterRead(chats, "a"))
        assertSame(chats, LiveChatViewModel.afterRead(chats, "no-such-order"))
        val none = emptyList<Live>()
        assertSame(none, LiveChatViewModel.afterRead(none, "a"))
    }

    // ═════════════ والتوصيلُ في الشاشة ═════════════

    /** **ونهاياتُ الأسطر تُوحَّد** — جيتٌ على ويندوز يكتب CRLF. */
    private fun src(path: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            val f = File(dir, "app-customer/src/main/kotlin/com/rahalgo/customer/$path")
            if (f.exists()) return f.readText().replace("\r\n", "\n")
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد $path")
    }

    /** **نافذةٌ من النصّ بعد أوّل ظهورٍ للعلامة** — بلا ارتهانٍ لمسافات التنسيق. */
    private fun window(text: String, marker: String, width: Int): String {
        val i = text.indexOf(marker)
        assertTrue("العلامةُ غائبة: $marker", i >= 0)
        return text.substring(i, minOf(text.length, i + width))
    }

    /**
     * **نافذةُ الحديث تُصفِّر عدَّ الطلب المفتوح عند الفتح والتبديل**، **ثمّ تقرأ الحقيقةَ
     * من المحرّك بعد وصول الحديث** — وإلّا بقي «2» على مربّعٍ قُرئ وبقيت شارةُ القرص 3.
     *
     * **واللوحُ هو من يُنادي** (`OrderChatSheet`: `onOpened` عند الفتح والتبديل و`onLoaded` عند كلّ وصول)،
     * **والقرصُ يمرّر المنادِيَين**: `markRead` تُصفّر محلّياً، و`load` بلا سياقٍ تقرأ الحقيقةَ بلا رنّة.
     */
    @Test
    fun `chat dialog zeroes the opened order then reloads the server truth`() {
        val call = window(src("MainActivity.kt"), "com.rahalgo.ui.OrderChatSheet(", 2500)
        assertTrue("الفتح والتبديل لا يصفّران عدَّ الطلب", call.contains("onOpened = liveChat::markRead"))
        assertTrue("لا تُقرأ الحقيقةُ بعد وصول الحديث", call.contains("onLoaded = { liveChat.load() }"))
        assertTrue("markRead غائبة عن LiveChat", src("chat/LiveChat.kt").contains("fun markRead(orderId: String)"))
    }

    /**
     * **وكلُّ بابٍ آخرَ يفتح اللوحَ نفسَه يُخبر القرصَ كذلك** — «دردشاتي» وبطاقةُ الطلب.
     *
     * **وإلّا بقيت شارةُ القرص على الرقم القديم حتّى نبضةٍ أخرى** وقد قُرئ الحديثُ من هناك،
     * ثمّ يفتح القرصُ طلباً قُرئ.
     */
    @Test
    fun `the other doors to the chat sheet tell the badge too`() {
        val chats = window(src("MainActivity.kt"), "ChatsScreen(", 1200)
        assertTrue("«دردشاتي» لا تصفّر عدَّ الطلب", chats.contains("onOpened = liveChat::markRead"))
        assertTrue("«دردشاتي» لا تقرأ الحقيقةَ بعد الوصول", chats.contains("onLoaded = { liveChat.load() }"))

        val orders = window(src("orders/OrdersScreen.kt"), "OrderChatSheet(", 600)
        assertTrue("بطاقةُ الطلب لا تصفّر عدَّ الطلب", orders.contains("onOpened = liveChat::markRead"))
        assertTrue("بطاقةُ الطلب لا تقرأ الحقيقةَ بعد الوصول", orders.contains("onLoaded = { liveChat.load() }"))
    }
}

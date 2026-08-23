package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطبقةُ الأولى — خادمٌ مزيَّفٌ يعدّ النداءات بترتيبها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (منظومةُ اختبار تطبيق السائق، بأمر المالك ٢٠٢٦-٠٨-٢٣.)
 *
 * # وهذه هي الطبقةُ التي أفلت منها عطبُ اليوم
 *
 * **الخطُّ المستقيم** — نداءُ المسار كان يسبق وصولَ الطلبات، فيسأل عن
 * طلبٍ لم يصل بعد **فيخرج بلا نداءٍ واحد.** ولم يمسكه مترجمٌ ولا
 * حارس: **أمسكته عينُ المالك في الشارع، مرّتين.**
 *
 * **واختبارٌ يعدّ النداءات بترتيبها يمسكه في ملّي ثانية.**
 */
class OrdersLoaderTest {

    /**
     * **خادمٌ مزيَّفٌ يسجّل ما نودي وبأيّ ترتيب.**
     *
     * **ولا يزيّف الشبكةَ بل العقد** — **ومزيِّفٌ على مستوى HTTP يمسك
     * الترميزَ ولا يمسك الترتيب**، وهو ما نحرسه هنا.
     */
    private class Feed(
        val mine: List<DriverOrder> = emptyList(),
        val failList: Boolean = false,
        val failRoute: Boolean = false,
        val routeAvailable: Boolean = true,
    ) : OrdersFeed {
        val calls = mutableListOf<String>()
        var routedId: String? = null

        override suspend fun queue(): List<DriverOrder> {
            calls += "queue"
            if (failList) throw RuntimeException("الشبكة")
            return emptyList()
        }

        override suspend fun orders(): List<DriverOrder> {
            calls += "orders"
            if (failList) throw RuntimeException("الشبكة")
            return mine
        }

        override suspend fun me(): DriverMe {
            calls += "me"
            return DriverMe()
        }

        override suspend fun route(orderId: String): OrderRoute {
            calls += "route"
            routedId = orderId
            if (failRoute) throw RuntimeException("المحرّك نائم")
            return OrderRoute(available = routeAvailable, distanceM = 2863.0)
        }
    }

    private fun order(id: String) = DriverOrder(id = id)

    // ══════════════════════════════════════════════════════════════════
    // **١ · العطبُ نفسُه — والحارسُ عليه**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **المسارُ يُطلب بعد الطلبات لا قبلها.**
     *
     * **وهذا هو الترتيبُ الذي انكسر** — وقِيس أثرُه على الطلب ١٠٣٠:
     * **المستقيمُ ١٫٨ كم والطريقُ ٢٫٩ كم**، ستّون بالمئة زيادة.
     */
    @Test
    fun `the route is requested after the orders arrive`() = runBlocking {
        val f = Feed(mine = listOf(order("o-1")))
        loadOnce(f, openId = null)
        assertTrue(
            "الترتيبُ الواصل: ${f.calls}",
            f.calls.indexOf("orders") < f.calls.indexOf("route"),
        )
    }

    /** **وأوّلُ فتحةٍ تردّ مساراً** — لا فراغاً كما كانت. */
    @Test
    fun `the very first cycle already yields a route`() = runBlocking {
        val out = loadOnce(Feed(mine = listOf(order("o-1"))), openId = null)
        assertNotNull("أوّلُ دورةٍ بلا مسار — وهو العطبُ بعينه", out.route)
        assertEquals(2863.0, out.route!!.distanceM, 0.01)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · وللطلب المفتوح لا لأوّل ما في اليد**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **ومن فتح طلبَه الثانيَ فرأى مسارَ الأوّل قاد إلى غير وجهته.**
     */
    @Test
    fun `the open order wins over the first in hand`() = runBlocking {
        val f = Feed(mine = listOf(order("o-1"), order("o-2")))
        loadOnce(f, openId = "o-2")
        assertEquals("o-2", f.routedId)
    }

    @Test
    fun `with nothing open the first in hand is used`() = runBlocking {
        val f = Feed(mine = listOf(order("o-1"), order("o-2")))
        loadOnce(f, openId = null)
        assertEquals("o-1", f.routedId)
    }

    /** **ولا يُطلب مسارٌ لمن لا طلبَ في يده** — نداءٌ يُردّ بلا معنى. */
    @Test
    fun `no order in hand means no route call`() = runBlocking {
        val f = Feed(mine = emptyList())
        val out = loadOnce(f, openId = null)
        assertTrue("نودي المسارُ بلا طلب: ${f.calls}", "route" !in f.calls)
        assertNull(out.route)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · وسقوطُ نداءٍ لا يُسقط ما لا يعتمد عليه**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **والمسارُ زينةٌ حول الطلب لا شرطٌ له.**
     *
     * **وشاشةٌ تنهار لأنّ خدمةَ مساراتٍ نامت تمنع السائقَ من العمل** —
     * وهو يعرف طريقَه إلى المتجر بلا خريطة.
     */
    @Test
    fun `a failing route does not sink the cycle`() = runBlocking {
        val out = loadOnce(Feed(mine = listOf(order("o-1")), failRoute = true), null)
        assertNull(out.route)
        assertNull("سقطت الدورةُ كلُّها لأجل مسار", out.error)
        assertEquals(1, out.mine.size)
    }

    /** **و`available:false` تُقرأ «لا مسار» لا مساراً فارغاً.** */
    @Test
    fun `an unavailable route is treated as none`() = runBlocking {
        val out = loadOnce(Feed(mine = listOf(order("o-1")), routeAvailable = false), null)
        assertNull(out.route)
    }

    /**
     * **وسقوطُ القائمة يُسقط الدورةَ ويُقال.**
     *
     * **وشاشةُ طلباتٍ فارغةٌ بلا سببٍ أسوأُ من رسالةِ عطب** — يظنّ
     * السائقُ أنّه لا عمل، **والصمتُ رسوبٌ** (البند ٩ من الميدان).
     */
    @Test
    fun `a failing list surfaces the error`() = runBlocking {
        val out = loadOnce(Feed(failList = true), null)
        assertNotNull("سقطت القائمةُ بلا سببٍ يُقال", out.error)
        assertTrue(out.mine.isEmpty())
    }
}

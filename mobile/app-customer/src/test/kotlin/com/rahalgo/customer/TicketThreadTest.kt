package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خيطُ الشكوى في تطبيق الزبون — عقدُ الواجهة** (`SUP-013`/`SUP-014`، PRQ-2)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان الزبونُ يفتح شكوى ثمّ لا يرى جوابَ المنصّة** — الردودُ تحت
 * `‎/admin` وحدَها. فأُكملت الواجهة: قائمةٌ ⇒ خيطُ ردودٍ ⇒ ردّ.
 *
 * # ولمَ يُقرأ المصدرُ نصّاً لا تُشغَّل الشاشة
 *
 * **وشاشةُ Compose تحتاج جهازاً أو مُحاكياً** — **وهذه فحوصُ وحدة.**
 * فيُثبَت أنّ عقدَ الشاشة قائمٌ في المصدر (كما في `PreLaunchScreenTest`)،
 * **وسلوكُ الجهاز يبقى بندَ ميدان** (شهادةُ التجهيز الحيّة). والعزلُ
 * والتمييزُ في الخادم يشهدهما `TestPRQ2_CustomerTicketRepliesAndIsolation`.
 */
class TicketThreadTest {

    private companion object {
        fun mobileRoot(): File {
            var dir = File("").absoluteFile
            repeat(8) {
                if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
                dir = dir.parentFile ?: return@repeat
            }
            throw IllegalStateException("لم أجد جذرَ mobile")
        }

        fun read(rel: String): String =
            File(mobileRoot(), rel).readText().replace("\r\n", "\n")

        val THREAD by lazy { read("ui/src/main/kotlin/com/rahalgo/ui/TicketThreadScreen.kt") }
        val LIST by lazy { read("ui/src/main/kotlin/com/rahalgo/ui/TicketsScreen.kt") }
        val SCREENS by lazy { read("app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineScreens.kt") }
        val VM by lazy { read("app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineViewModel.kt") }
    }

    /** **١ · القائمةُ تفتح التفصيل** — صفٌّ يُنقَر فيُفتح خيطُه. */
    @Test
    fun listRowOpensDetail() {
        assertTrue("**قائمةُ الشكاوى فقدت `onOpen`/النقر**",
            LIST.contains("onOpen") && LIST.contains("clickable"))
        assertTrue("**القائمةُ لا تمرّر مُعرّفَ الصفّ للفتح**",
            SCREENS.contains("onOpen = { row -> vm.openTicket(row.id) }") &&
                SCREENS.contains("id = t.id"))
        assertTrue("**لا موزّعَ يعرض التفصيلَ عند فتح شكوى**",
            SCREENS.contains("vm.openTicketId != null") && SCREENS.contains("TicketDetail(vm)"))
    }

    /** **٢ · التفصيلُ يرسم الموضوعَ والحالَ ووقتَ الفتح.** */
    @Test
    fun detailRendersHeader() {
        assertTrue("**التفصيلُ لا يعرض الموضوع**", THREAD.contains("subject"))
        assertTrue("**التفصيلُ لا يعرض الحال**", THREAD.contains("ticketStatusText(status)"))
        assertTrue("**التفصيلُ لا يعرض وقتَ الفتح**",
            THREAD.contains("R.string.tik_created") && THREAD.contains("whenText(createdAt)"))
    }

    /** **٣ · خيطُ الردود بترتيبٍ زمنيّ** — يُبنى من `replies` كما جاءت (المحرّكُ يرتّبها). */
    @Test
    fun repliesRenderInOrder() {
        assertTrue("**التفصيلُ لا يبني الخيطَ من الردود**",
            SCREENS.contains("t.replies.map") && SCREENS.contains("TicketMessage("))
        assertTrue("**الخيطُ لا يُعرَض رسالةً رسالةً**",
            THREAD.contains("messages.forEach"))
    }

    /** **٤+٥ · ردُّ المنصّة وردُّ الزبون يتمايزان** — جهةً ولوناً ولافتةً بحسب `mine`. */
    @Test
    fun customerAndSupportRepliesAreDistinct() {
        assertTrue("**لا تمايزَ بالجهة (يجب Start مقابل End تتبعان الاتجاه)**",
            THREAD.contains("if (m.mine) Alignment.CenterEnd else Alignment.CenterStart"))
        assertTrue("**لا تمايزَ باللون (brand مقابل bubble)**",
            THREAD.contains("if (m.mine) Rahal.colors.brand else Rahal.colors.bubble"))
        assertTrue("**لا لافتةَ باسم الكاتب (أنت مقابل فريق رحّال غو)**",
            THREAD.contains("if (m.mine) R.string.tik_you else R.string.tik_support"))
    }

    /** **٦ · إرسالُ الردّ** — زرٌّ يستدعي المحرّك. */
    @Test
    fun replySubmitWired() {
        assertTrue("**لا زرَّ إرسالٍ في التفصيل**",
            THREAD.contains("R.string.tik_send") && THREAD.contains("onSend"))
        assertTrue("**التفصيلُ لا يربط الإرسالَ بالنموذج**", SCREENS.contains("onSend = vm::sendReply"))
        assertTrue("**النموذجُ لا يرسل الردَّ للخادم**", VM.contains("api.replyTicket("))
    }

    /** **٧ · المغلقةُ تمنع الردَّ وتقول لماذا** — لا حقلَ باهتٌ صامت. */
    @Test
    fun closedTicketBlocksReplyWithReason() {
        assertTrue("**لا بوّابةَ للردّ بحسب الحال**",
            THREAD.contains("if (canReply)") && THREAD.contains("R.string.tik_closed"))
        assertTrue("**التفصيلُ لا يحسب قبولَ الردّ من الحال**",
            SCREENS.contains("ticketCanReply(t.status)"))
    }

    /** **٨ · حالُ الفشلِ بإعادة** — `LoadState` بزرِّ إعادةٍ يُعيد الجلب. */
    @Test
    fun detailHasErrorRetry() {
        assertTrue("**لا حالَ تحميلٍ/فشلٍ للتفصيل**",
            SCREENS.contains("LoadState(vm.detailBusy, vm.detailError)"))
        assertTrue("**الإعادةُ لا تُعيد جلبَ الخيط**",
            SCREENS.contains("vm.openTicket(it, force = true)"))
    }

    /** **٩ · لا تكرارَ بإعادة الإرسال/القراءة** — الخيطُ يُستبدَل بردّ الخادم لا يُضاف إليه. */
    @Test
    fun noClientSideAppendOnReply() {
        assertTrue("**الردُّ لا يُستبدَل بجواب الخادم**", VM.contains("ticketDetail = api.replyTicket("))
        // **ولا يُضاف محلّيّاً** — إضافةٌ محلّيّةٌ فوق جوابِ الخادم تُكرّر الردّ.
        assertFalse("**إضافةٌ محلّيّةٌ للردود تُكرّرها**",
            VM.contains("replies +") || VM.contains("replies.plus") || VM.contains("+ TicketReply"))
    }

    /** **١٠ · الاتجاهُ العربيُّ صحيح** — `Start`/`End` لا `Left`/`Right`. */
    @Test
    fun layoutIsRtlSafe() {
        assertTrue("**لا محاذاةَ نسبيّةً تتبع الاتجاه**",
            THREAD.contains("Alignment.CenterEnd") && THREAD.contains("Alignment.CenterStart"))
        assertFalse("**محاذاةٌ مطلقةٌ تكسر العربيّة**",
            THREAD.contains("Alignment.CenterLeft") || THREAD.contains("Alignment.CenterRight"))
    }

    /** **١١ · الملكيّةُ في الخادم لا في العميل** — الشاشةُ تعرض ما يردّه الخادم ولا تفلتر. */
    @Test
    fun ownershipIsServerEnforced() {
        assertTrue("**التفصيلُ لا يُجلب من الخادم بمعرّفه**", VM.contains("api.myTicket("))
        // **ولا فلترةَ ملكيّةٍ في العميل** — العزلُ يشهده اختبارُ الخادم (404 للغريب).
        assertFalse("**فلترةُ ملكيّةٍ في العميل بدل الخادم**",
            VM.contains("customerId ==") || VM.contains("== customerId"))
    }

    /** **وليست محادثةً حيّة** — لا مؤشّرَ كتابةٍ ولا حضورٍ ولا إيصالَ قراءة. */
    @Test
    fun notAChatProduct() {
        for (banned in listOf("✓✓", "readAt", "typing", "isTyping", "presence", "onlineNow")) {
            assertFalse("**سمةُ محادثةٍ حيّةٍ في خيط الشكوى: $banned**", THREAD.contains(banned))
        }
    }
}

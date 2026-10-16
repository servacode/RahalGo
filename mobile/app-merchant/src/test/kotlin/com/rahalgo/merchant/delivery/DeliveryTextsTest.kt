package com.rahalgo.merchant.delivery

import com.rahalgo.merchant.R
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«إحضار» مع «يقبض مبلغ»: السائقُ يدفع بمكان الاستلام ويأخذه منك** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت الشاشةُ تقول «يقبض» في الوضعين** والخطواتُ «إليك ثمّ إلى المستلم» — وفي «إحضار»
 * يقع العكس. **فيُقاس الاختيارُ، ونصُّ المالك حرفاً من الموارد.**
 */
class DeliveryTextsTest {

    private val strings: String by lazy {
        val f = listOf(
            File("src/main/res/values/strings.xml"),
            File("app-merchant/src/main/res/values/strings.xml"),
        ).first { it.exists() }
        f.readText(Charsets.UTF_8)
    }

    private fun text(name: String): String {
        val m = Regex("""<string name="$name">(.*?)</string>""").find(strings)
        return m?.groupValues?.get(1) ?: error("لا نصَّ باسم $name")
    }

    @Test
    fun `fetch picks the pay-at-pickup texts — send keeps the collect texts`() {
        assertEquals(R.string.md_collect_q_fetch, DeliveryTexts.collectQuestion(true))
        assertEquals(R.string.md_collect_q, DeliveryTexts.collectQuestion(false))
        assertEquals(R.string.md_collect_sub_fetch, DeliveryTexts.collectHint(true))
        assertEquals(R.string.md_collect_amount_fetch, DeliveryTexts.collectAmountLabel(true))
        assertEquals(R.string.md_collect_amount, DeliveryTexts.collectAmountLabel(false))
        assertEquals(R.string.md_row_collect_fetch, DeliveryTexts.collectRow(true))
        assertEquals(R.string.md_row_cash_fetch, DeliveryTexts.feeRow("recipient", true))
        assertEquals(R.string.md_row_cash, DeliveryTexts.feeRow("recipient", false))
    }

    @Test
    fun `the owner's words, letter for letter`() {
        assertEquals("بدك السائق يدفع مبلغ بمكان الاستلام؟", text("md_collect_q_fetch"))
        assertEquals(
            "السائق بيدفع المبلغ من جيبه بمكان الاستلام، وبياخده منك لما يوصل. ما بيدخل حساب المنصة",
            text("md_collect_sub_fetch"),
        )
        assertEquals("المبلغ يلي بيدفعه (ل.س)", text("md_collect_amount_fetch"))
        // **ولا «يقبض» في نصوص الإحضار** — هو يدفع.
        for (n in listOf("md_collect_q_fetch", "md_collect_sub_fetch", "md_collect_amount_fetch", "md_row_collect_fetch")) {
            assertFalse("«$n» يقول يقبض", text(n).contains("يقبض"))
        }
    }

    @Test
    fun `fetch track goes to the pickup place then to you`() {
        val fetch = DeliveryTexts.trackSteps(true)
        val send = DeliveryTexts.trackSteps(false)
        assertEquals(6, fetch.size)
        assertEquals(send.size, fetch.size)
        assertEquals(R.string.md_step_to_pickup_fetch, fetch[2])
        assertEquals(R.string.md_step_to_you_fetch, fetch[4])
        assertFalse(R.string.md_step_to_recipient in fetch)
        assertTrue(R.string.md_step_to_recipient in send)
        assertEquals("في الطريق إلى مكان الاستلام", text("md_step_to_pickup_fetch"))
        assertEquals("في الطريق إليك", text("md_step_to_you_fetch"))
    }
}

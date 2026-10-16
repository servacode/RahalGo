package com.rahalgo.merchant.delivery

import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.Delivery

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نصوصُ «توصيل» و«إحضار» من موضعٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **في «إحضار» المبلغُ يُدفع لا يُقبض**: السائقُ يدفعه من جيبه بمكان الاستلام **ويأخذه من
 * المتجر لما يوصل** — والمالُ خارجَ المنصّة، عرضٌ وحدَه. **وكانت الشاشةُ تقول «يقبض» في
 * الوضعين** فيُقرأ عكسُ ما يقع.
 *
 * **والخطواتُ كذلك**: في «إحضار» يذهب السائقُ أوّلاً إلى مكان الاستلام **ثمّ إليك** — لا
 * «إليك ثمّ إلى المستلم». **فالاختيارُ هنا والشاشتان تقرآنه** (`DeliveryScreen` · `DeliveryTrack`).
 */
internal object DeliveryTexts {

    /** **سؤالُ المبلغ** — «يقبض من المستلم» أو «يدفع بمكان الاستلام». */
    fun collectQuestion(fetch: Boolean): Int =
        if (fetch) R.string.md_collect_q_fetch else R.string.md_collect_q

    /** **عنوانُ الحقل** — «يلي بيقبضه» أو «يلي بيدفعه». */
    fun collectAmountLabel(fetch: Boolean): Int =
        if (fetch) R.string.md_collect_amount_fetch else R.string.md_collect_amount

    /** **اسمُه في «ما ينقص».** */
    fun collectMissing(fetch: Boolean): Int =
        if (fetch) R.string.md_collect_missing_fetch else R.string.md_collect_missing

    /** **سطرُه في تفاصيل التوصيلة.** */
    fun collectRow(fetch: Boolean): Int =
        if (fetch) R.string.md_row_collect_fetch else R.string.md_row_collect

    /** **سطرُ الأجرة بدافعها** — والمستلِمُ في «إحضار» هو الشخصُ بمكان الاستلام. */
    fun feeRow(payer: String, fetch: Boolean): Int = when (payer) {
        "recipient" -> if (fetch) R.string.md_row_cash_fetch else R.string.md_row_cash
        "merchant_cash" -> R.string.md_row_me_cash
        else -> R.string.md_row_wallet
    }

    // ══════════════════════════════════════════════════════════════════
    // **«إحضار»: المبلغُ الحقيقيُّ وموافقتُك** (قرارُ المالك ٢٠٢٦-١٠-١٦)
    // ══════════════════════════════════════════════════════════════════

    /** **أينتظر هذا المشوارُ موافقتي على مبلغ؟** — «إحضار» والسائقُ بمكان الاستلام وكتب مبلغاً. */
    fun awaitsAmount(d: Delivery): Boolean =
        d.deliveryMode == "fetch" && d.status == "at_pickup" && d.fetchAmountStatus == "pending" && d.fetchAmount != null

    /** **أوّلُ مشوارٍ ينتظرني** — تظهر نافذتُه فوق أيّ شاشة، وفارغٌ: لا شيء. */
    fun pendingAmount(list: List<Delivery>?): Delivery? = list?.firstOrNull { awaitsAmount(it) }

    /** **سطرُ المبلغ في التفاصيل بحاله** — وفارغٌ: لم يُكتب. */
    fun amountRow(status: String): Int? = when (status) {
        "pending" -> R.string.md_amount_row_pending
        "approved" -> R.string.md_amount_row_approved
        "rejected" -> R.string.md_amount_row_rejected
        else -> null
    }

    /**
     * **خطواتُ المتابعة الستّ** — بترتيب الحال نفسِه (`stepOf`): المنصّة · السائق ·
     * الذهاب إلى الاستلام · عنده · الذهاب إلى التسليم · سُلّمت.
     */
    fun trackSteps(fetch: Boolean): List<Int> = if (fetch) {
        listOf(
            R.string.md_step_platform,
            R.string.md_step_waiting,
            R.string.md_step_to_pickup_fetch,
            R.string.md_step_at_pickup_fetch,
            R.string.md_step_to_you_fetch,
            R.string.md_step_delivered,
        )
    } else {
        listOf(
            R.string.md_step_platform,
            R.string.md_step_waiting,
            R.string.md_step_to_you,
            R.string.md_step_at_you,
            R.string.md_step_to_recipient,
            R.string.md_step_delivered,
        )
    }
}

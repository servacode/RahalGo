package com.rahalgo.ui

/**
 * **أسماءُ حركات المحفظة وحالاتِ السحب — مصدرٌ واحدٌ للشاشة والكشف المطبوع.**
 *
 * كانت الشاشةُ تعرف خمسةَ عشرَ نوعاً والكشفُ المطبوعُ تسعة، **والمحرّكُ
 * يكتب أكثرَ من الاثنين** (`backend/internal/fininv/kinds.go`): فكان
 * `payout_reversal` و`merchant_cash_accrued` و`merchant_cash_paid` تُعرض
 * بمفاتيحها الإنكليزيّة في كشف صاحب المتجر والسائق.
 *
 * **وحالاتُ السحب صارت ستّاً** (`0370_payouts_section.sql`): `processing`
 * و`failed` و`reversed` كانت تُقرأ كلُّها «بانتظار القرار».
 */
object WalletLabels {

    /** **اسمُ نوع الحركة** — `null` لنوعٍ لا نعرفه فيُعرض بمفتاحه ليُبلَّغ عنه. */
    fun kindRes(kind: String): Int? = when (kind) {
        "payout" -> R.string.wal_k_payout
        "payout_reversal" -> R.string.wal_k_payout_reversal
        "driver_earning" -> R.string.wal_k_driver_earning
        "commission" -> R.string.wal_k_commission
        "compensation" -> R.string.wal_k_compensation
        "penalty" -> R.string.wal_k_penalty
        "refund" -> R.string.wal_k_refund
        "adjustment" -> R.string.wal_k_adjustment
        "reward" -> R.string.wal_k_reward
        "settlement" -> R.string.wal_k_settlement
        "topup" -> R.string.wal_k_topup
        "order_payment" -> R.string.wal_k_order_payment
        "merchant_earning" -> R.string.wal_k_merchant_earning
        "merchant_cash_accrued" -> R.string.wal_k_merchant_cash_accrued
        "merchant_cash_paid" -> R.string.wal_k_merchant_cash_paid
        "platform_profit" -> R.string.wal_k_platform_profit
        "platform_expense" -> R.string.wal_k_platform_expense
        "operating_expense" -> R.string.wal_k_operating_expense
        else -> null
    }

    /** **اسمُ حالة طلب السحب** — والمجهولُ «بانتظار القرار» كما كان. */
    fun payoutStatusRes(status: String): Int = when (status) {
        "approved" -> R.string.wal_st_approved
        "processing" -> R.string.wal_st_processing
        "paid" -> R.string.wal_st_paid
        "rejected" -> R.string.wal_st_rejected
        "failed" -> R.string.wal_st_failed
        "reversed" -> R.string.wal_st_reversed
        else -> R.string.wal_st_pending
    }

    /** **حالةٌ انتهت بلا مال** — تُلوَّن بالخطر. */
    fun payoutFailed(status: String): Boolean = status in setOf("rejected", "failed", "reversed")

    /** **حالةٌ وصل فيها المالُ أو بدأ صرفُه** — تُلوَّن بالنجاح. */
    fun payoutOk(status: String): Boolean = status in setOf("approved", "processing", "paid")
}

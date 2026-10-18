package com.rahalgo.ui

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سببُ الشكوى أو البلاغ بالعربيّة — ولا رمزَ خامٌّ أبداً** (٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك: شكاوى المتجر تُعرض «late» بالإنكليزيّ.)
 *
 * **والرموزُ ثلاثُ قوائمَ في المحرّك**: شكوى الزبون (`support/complaints.go`)،
 * وبلاغُ السائق (`support/driver_report.go`)، وبلاغُ المتجر
 * (`support/merchant_report.go`) — **وكلُّها تقع على طلبات المتجر.** فالاسمُ
 * هنا بلفظ طرفٍ ثالث يصلح لمن يقرأ عن غيره.
 *
 * **والمجهولُ «سبب آخر» لا رمزُه** — حارسٌ (`TicketReasonsTest`) يقرأ القوائمَ
 * من المحرّك فيكشف سبباً زِيد بلا اسم، **فلا حاجةَ لعرض الرمز ليُبلَّغ عنه.**
 */
object TicketReasons {

    /** **نصُّ السبب** — والمجهولُ والفارغُ «سبب آخر». */
    fun res(code: String): Int = known(code) ?: R.string.rs_other

    /** **اسمُ رمزٍ نعرفه** — و`null` لمجهول (للحارس وحدَه). */
    fun known(code: String): Int? = when (code) {
        // شكوى الزبون
        "not_received" -> R.string.tr_not_received
        "missing_items" -> R.string.tr_missing_items
        "wrong_items" -> R.string.tr_wrong_items
        "quality" -> R.string.tr_quality
        "late" -> R.string.tr_late
        "driver_conduct" -> R.string.rs_driver_conduct
        "money" -> R.string.tr_money
        // بلاغُ السائق
        "merchant_slow" -> R.string.tr_merchant_slow
        "merchant_refused" -> R.string.tr_merchant_refused
        "merchant_wrong_goods" -> R.string.tr_merchant_wrong_goods
        "merchant_conduct" -> R.string.tr_merchant_conduct
        "customer_absent" -> R.string.tr_customer_absent
        "customer_address" -> R.string.tr_customer_address
        "customer_refused" -> R.string.tr_customer_refused
        "customer_conduct" -> R.string.tr_customer_conduct
        // بلاغُ المتجر
        "driver_late_pickup" -> R.string.tr_driver_late_pickup
        "driver_refused" -> R.string.tr_driver_refused
        "other" -> R.string.rs_other
        else -> null
    }

    fun text(ctx: Context, code: String): String = ctx.getString(res(code))

    @Composable
    fun text(code: String): String = stringResource(res(code))
}

/**
 * **سببُ الإنذار بالعربيّة** (`support/warn_reasons.go`) — **والمحرّكُ يخزّن
 * الرمزَ** في `warnings.reason`، فكان يُعرض «fraud_attempt» في شاشة المتجر.
 */
object WarnReasons {
    fun res(code: String): Int = known(code) ?: R.string.rs_other

    fun known(code: String): Int? = when (code) {
        "abuse_driver" -> R.string.wr_abuse_driver
        "repeated_cancel" -> R.string.wr_repeated_cancel
        "refused_delivery" -> R.string.wr_refused_delivery
        "wrong_address" -> R.string.wr_wrong_address
        "unreachable" -> R.string.wr_unreachable
        "fraud_attempt" -> R.string.wr_fraud_attempt
        "terms_violation" -> R.string.wr_terms_violation
        "other" -> R.string.rs_other
        else -> null
    }
}

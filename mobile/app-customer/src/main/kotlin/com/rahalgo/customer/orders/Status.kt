package com.rahalgo.customer.orders

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اسمُ الحال ولونُها — في موضعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وحالٌ تُترجَم في بطاقتين تفترق** — تُصحَّح في إحداهما ويبقى القديمُ
 * في الأخرى، **فيقرأ الزبونُ طلبَه باسمين.**
 *
 * **وما لا ترجمةَ له يُعرض رمزَه** — لا يُبتلع في «قيد المعالجة»:
 * **من رآه أبلغ عنه، ومن ابتلعه ترك حالاً مجهولةً تُعرض على الناس.**
 */
@Composable
fun statusText(status: String): String = when (status) {
    "pending" -> stringResource(R.string.stt_pending)
    "accepted", "confirmed" -> stringResource(R.string.stt_accepted)
    "preparing" -> stringResource(R.string.stt_preparing)
    "dispatching" -> stringResource(R.string.ord_st_dispatching)
    "assigned", "at_pickup", "picked_up" -> stringResource(R.string.stt_assigned)
    "on_the_way", "at_dropoff" -> stringResource(R.string.ord_st_onway)
    "delivered" -> stringResource(R.string.ord_st_delivered)
    "cancelled", "rejected" -> stringResource(R.string.stt_cancelled)
    "failed" -> stringResource(R.string.ord_st_failed)
    else -> status
}

/**
 * **ولونُ الشارة يقول الحالَ قبل أن تُقرأ.**
 *
 * **والأخضرُ للنهاية السعيدة وحدَها** — ومن لوّن «قُبل» أخضرَ جعل
 * الزبونَ يظنّ أنّ طلبَه وصل.
 */
@Composable
fun statusColor(status: String): Color = when (status) {
    "delivered" -> Rahal.colors.success
    "cancelled", "rejected", "failed" -> Rahal.colors.danger
    "on_the_way", "at_dropoff" -> Rahal.colors.brand
    else -> Rahal.colors.inkMuted
}

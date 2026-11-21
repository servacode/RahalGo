package com.rahalgo.driver.trip

import android.content.Context
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.StorePay
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نافذةُ العمولة** (قرارُ المالك ٢٠٢٦-١١-٢٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «وقت يكون في عمولة لازم يجي اشعار للسائق نافذة منبثقة تقله يوجد عمولة لرحال غو على هذا المتجر
 * وقيمة العمولة ونسبتها.»
 *
 * **مرّة لكلّ متجرٍ بكلّ طلب** — والمفتاحُ فيه المبلغُ والنسبة، فإن تغيّرا (عمولةٌ عُدّلت قبل
 * التسليم) **رجعت تطلع بالأرقام الجديدة**. والسطرُ الصغير تحت «ادفع» (`CutLine`) يضلّ متل ما هو.
 * **والأرقامُ من الخادم** (`pay_merchants`) — لا حسبةَ هنا.
 */
internal object CommissionSeen {
    private const val PREFS = "driver_commission_seen"
    private const val KEEP = 300

    fun key(orderId: String, p: StorePay): String =
        "$orderId|${p.merchantId}|${p.cut}|${p.cutPercent ?: ""}|${p.currency}"

    fun seen(context: Context, key: String): Boolean =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).contains(key)

    fun mark(context: Context, key: String) {
        val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val edit = prefs.edit().putLong(key, System.currentTimeMillis())
        // **ولا تكبر للأبد** — يُشال الأقدم فوق الحدّ.
        val all = prefs.all
        if (all.size >= KEEP) {
            all.entries.sortedBy { (it.value as? Long) ?: 0L }.take(all.size - KEEP + 1).forEach { edit.remove(it.key) }
        }
        edit.apply()
    }

    /** **المتاجر اللي عليها عمولة بهالطلب** — ما بيدفعه الكابتن وفيه خصمٌ لرحّال غو. */
    fun pending(order: DriverOrder): List<StorePay> = order.payMerchants.filter { it.cut > 0 && it.amount > 0 }
}

@Composable
internal fun CommissionNotice(order: DriverOrder) {
    val ctx = LocalContext.current
    // **ويُعاد الحساب لما يضغط «تمام»** — فتطلع نافذةُ المتجر التاني إن وُجد.
    var tick by remember { mutableIntStateOf(0) }
    val next = remember(order.id, order.payMerchants, tick) {
        CommissionSeen.pending(order).firstOrNull { !CommissionSeen.seen(ctx, CommissionSeen.key(order.id, it)) }
    } ?: return
    val ok: () -> Unit = {
        CommissionSeen.mark(ctx, CommissionSeen.key(order.id, next))
        tick++
    }
    val fmt = { v: Long -> storePayText(next.copy(amount = v)) }
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = ok,
        title = { Text(stringResource(R.string.commission_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Rahal.space.xs)) {
                Text(stringResource(R.string.commission_store, next.merchantName), style = Rahal.type.body)
                next.cutPercent?.let {
                    Text(stringResource(R.string.commission_percent, commissionPercentText(it)), style = Rahal.type.body)
                }
                Text(
                    if (next.gross > 0) stringResource(R.string.commission_amount_of, fmt(next.cut), fmt(next.gross))
                    else stringResource(R.string.commission_amount, fmt(next.cut)),
                    style = Rahal.type.body,
                )
                Text(
                    stringResource(R.string.commission_pay, fmt(next.amount)),
                    style = Rahal.type.body,
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.brand,
                )
            }
        },
        confirmButton = { RahalButton(onClick = ok) { Text(stringResource(R.string.commission_ok)) } },
    )
}

/** **«10» لا «10.0»، و«1.5» كما هي.** */
internal fun commissionPercentText(v: Double): String =
    if (v == Math.floor(v)) v.toLong().toString() else (Math.round(v * 100) / 100.0).toString()

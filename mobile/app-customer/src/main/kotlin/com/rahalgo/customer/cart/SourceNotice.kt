package com.rahalgo.customer.cart

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.stringResource
import com.rahalgo.customer.R
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Address
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.money
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صنفٌ من متجرٍ ثانٍ يُقال لحظةَ إضافته** (قرارُ المالك ٢٠٢٦-١٠-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «لازم يعطي للزبون تنبيه: انت ضفت صنف تابع لمتجر ثاني، وراح يكون في مبلغ
 * إضافي حسب مسافة المتجر الثاني.» **والسوقُ لا يعرّف الصنفَ بمتجره** — فالحكمُ
 * من التسعيرة نفسِها: **كم متجراً صار في السلّة وكم زِيد**، بحسبة المحرّك التي
 * يُحاسَب بها. **وما تجاوز السقفَ يُرفع من السلّة ويُقال لماذا** — لا يُكتشف
 * عند الإرسال.
 */
object SourceNotice {
    data class Notice(val extra: Long, val tooMany: Boolean, val max: Int, val key: String)

    var pending by mutableStateOf<Notice?>(null)
        private set

    /** **كم متجراً كان في السلّة عند آخر سؤال** — فيُعرف أنّ متجراً جديداً دخل. */
    private var known = 1

    /** يُنادى بعد كلِّ إضافةٍ لسطرٍ جديد — `increased` يعني كميّةً زادت لا صنفاً دخل. */
    fun afterAdd(scope: CoroutineScope, api: CustomerApi, address: Address?, increased: Boolean) {
        if (Cart.lines.size <= 1) {
            known = 1
            return
        }
        if (increased || address == null) return
        val key = Cart.lines.last().key
        scope.launch {
            val q = runCatching {
                api.quote(Cart.lines.map { it.toPayload() }, address.lat, address.lng)
            }.getOrNull() ?: return@launch
            when {
                q.tooManySources -> {
                    Cart.setQty(key, 0)
                    pending = Notice(0, true, q.maxSources, key)
                }
                q.sources > known -> {
                    known = q.sources
                    pending = Notice(q.sourcesFee, false, q.maxSources, key)
                }
                else -> known = q.sources
            }
        }
    }

    fun dismiss() {
        pending = null
    }

    /** **«شيله»** — يُرفع الصنفُ الذي جاء من المتجر الثاني. */
    fun undo() {
        pending?.let { Cart.setQty(it.key, 0) }
        known = 1
        pending = null
    }
}

@Composable
fun SourceNoticeDialog() {
    val n = SourceNotice.pending ?: return
    AlertDialog(
        onDismissRequest = { SourceNotice.dismiss() },
        title = {
            Text(stringResource(if (n.tooMany) R.string.src_too_many_title else R.string.src_other_title))
        },
        text = {
            Text(
                when {
                    n.tooMany && n.max <= 1 -> stringResource(R.string.src_only_one_text)
                    n.tooMany -> stringResource(R.string.src_too_many_text, n.max.toString())
                    n.extra > 0 -> stringResource(R.string.src_other_text_fee, money(n.extra))
                    else -> stringResource(R.string.src_other_text_free)
                },
            )
        },
        confirmButton = {
            RahalButton(onClick = { SourceNotice.dismiss() }) { Text(stringResource(R.string.src_ok)) }
        },
        dismissButton = if (n.tooMany) null else {
            { RahalOutlineButton(onClick = { SourceNotice.undo() }) { Text(stringResource(R.string.src_undo)) } }
        },
    )
}

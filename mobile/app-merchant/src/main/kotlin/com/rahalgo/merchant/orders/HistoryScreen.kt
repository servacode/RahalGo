package com.rahalgo.merchant.orders

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.merchant.drawer.reasonAr
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.MerchantOrder
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.Empty
import com.rahalgo.ui.Flash
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سجلُّ الطلبات — ما انتهى أمرُه، وما يُبلَّغ عنه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «سجلُّ الطلبات قسمٌ منفصلٌ كما بالويب ويكون
 *  داخل القائمة الجانبيّة»، ثمّ: «بسجلّ الطلبات يجب أن يكون هناك زرُّ
 *  إبلاغٍ عن السائق».)
 *
 * # ولماذا الإبلاغُ هنا لا في شاشةٍ مستقلّة
 *
 * **البلاغُ على طلبٍ بعينه لا على سائقٍ بعينه** — المحرّكُ يشتقّ
 * المشكوَّ عليه من سائق الطلب (`support/merchant_report.go`)، **ولا
 * يقبل بلاغاً بلا طلب.**
 *
 * **وصاحبُ المطعم يتذكّر «الطلبَ الذي تأخّر» لا اسمَ السائق** — الذي
 * لا يراه أصلاً.
 *
 * # ولا مالَ فيه
 *
 * **المحرّكُ يصفّر المبالغَ للمتجر** (`merchant_privacy.go`): «المال —
 * يراه في محفظته وتقاريره». **وصفرٌ في موضع مالٍ يُقرأ عطباً**، فلا
 * يُعرض أصلاً.
 */
class HistoryViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var orders by mutableStateOf<List<MerchantOrder>>(emptyList())
        private set

    /**
     * **ما يملك الإبلاغَ عنه — يقوله المحرّك.**
     *
     * **ولا يُشتقّ هنا**: شروطُ القبول ثلاثة، **وأحدُها محجوبٌ عن هذا
     * التطبيق عمداً** — `redactForMerchant` يمحو `driver_id`.
     */
    var reportable by mutableStateOf<Set<String>>(emptySet())
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    var sending by mutableStateOf(false)
        private set

    /** أسبابُه — تُجلب مرّةً عند أوّل فتحٍ للنافذة. */
    var reasons by mutableStateOf<List<String>>(emptyList())
        private set

    private var storeId: String = ""

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            runCatching {
                if (storeId.isEmpty()) {
                    storeId = api.stores().stores.firstOrNull()?.id ?: ""
                }
                if (storeId.isEmpty()) {
                    error = "لا متجر مرتبط بحسابك"
                    loading = false
                    return@launch
                }
                val page = api.history(storeId)
                orders = page.orders
                reportable = page.reportable.toSet()
                error = ""
            }.onFailure { error = it.message ?: "تعذّر جلب السجل" }
            loading = false
        }
    }

    fun loadReasons() {
        if (reasons.isNotEmpty()) return
        viewModelScope.launch {
            runCatching { reasons = api.reportReasons().reasons }
        }
    }

    /**
     * **يرسل بلاغَه** — وبنجاحه **يخرج الطلبُ من قائمة ما يُبلَّغ عنه**:
     * **وبلاغان على طلبٍ واحدٍ يُقرآن في المكتب شكويين فيُفصل في
     * إحداهما.**
     */
    fun report(orderId: String, reason: String, note: String, done: () -> Unit) {
        sending = true
        viewModelScope.launch {
            runCatching { api.reportDriver(orderId, reason, note) }
                .onSuccess {
                    reportable = reportable - orderId
                    Flash.ok("وصل بلاغُك — والإدارة تنظر فيه")
                    done()
                }
                .onFailure { Flash.fail(it.message ?: "تعذّر إرسال البلاغ") }
            sending = false
        }
    }
}

@Composable
fun HistoryScreen(vm: HistoryViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.orders.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(R.string.menu_history), stringResource(R.string.history_hint))
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    // **والنافذةُ لطلبٍ واحدٍ في المرّة** — رقمُه هو الحال.
    var reporting by remember { mutableStateOf<MerchantOrder?>(null) }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(stringResource(R.string.menu_history), stringResource(R.string.history_hint))

            if (vm.orders.isEmpty()) {
                Empty(stringResource(R.string.history_empty))
                return@Screen
            }

            Spacer(Modifier.height(10.dp))
            Card {
                vm.orders.forEachIndexed { i, o ->
                    if (i > 0) HorizontalDivider()
                    Row(
                        Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                "#" + o.number,
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            // **وما فيه يُقال في سطر** — من يسأل عن طلبٍ
                            // مضى يسأل عن بضاعته لا عن رقمه وحدَه.
                            if (o.items.isNotEmpty()) {
                                Text(
                                    o.items.joinToString("، ") { "${it.qty}× ${it.name}" },
                                    color = Rahal.colors.inkMuted,
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                        // ══════════════════════════════════════════════
                        // **وزرُّ البلاغ لمن يملكه وحدَه**
                        // ══════════════════════════════════════════════
                        //
                        // **وزرٌّ يُعرض ثمّ يردّ خطأً أسوأُ من زرٍّ غائب**:
                        // صاحبُ المطعم يضغطه فيُقال له «سببٌ غير صالح»
                        // وهو لم يختر سبباً بعد.
                        if (vm.reportable.contains(o.id)) {
                            RahalTextButton(onClick = {
                                reporting = o
                                vm.loadReasons()
                            }) {
                                Icon(
                                    painter = painterResource(com.rahalgo.ui.R.drawable.ic_warning),
                                    contentDescription = null,
                                    tint = Rahal.colors.danger,
                                    modifier = Modifier.size(16.dp),
                                )
                                Spacer(Modifier.width(4.dp))
                                Text(
                                    stringResource(R.string.report_driver),
                                    color = Rahal.colors.danger,
                                    style = MaterialTheme.typography.labelMedium,
                                )
                            }
                        }
                        Spacer(Modifier.width(6.dp))
                        Text(
                            statusAr(o.status),
                            color = if (o.status == "delivered") Rahal.colors.brand
                            else Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.labelMedium,
                        )
                    }
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }

    reporting?.let { o ->
        ReportSheet(
            vm = vm,
            order = o,
            onClose = { reporting = null },
        )
    }
}

/**
 * **نافذةُ البلاغ** — سببٌ من أربعةٍ وملاحظةٌ اختياريّة.
 *
 * **والأسبابُ من المحرّك لا مكتوبةً هنا** (`support.MerchantReportReasons`)
 * — **وقائمةٌ منسوخةٌ في الجهاز تشيخ يومَ يُضاف سبب**، فيُرسل رمزٌ
 * يرفضه المحرّك.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ReportSheet(vm: HistoryViewModel, order: MerchantOrder, onClose: () -> Unit) {
    var reason by remember { mutableStateOf("") }
    var note by remember { mutableStateOf("") }

    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
        Text(
            stringResource(R.string.report_title, order.number),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            stringResource(R.string.report_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(10.dp))

        vm.reasons.forEach { code ->
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                androidx.compose.material3.RadioButton(
                    selected = reason == code,
                    onClick = { reason = code },
                )
                Text(reasonAr(code), Modifier.weight(1f))
            }
        }

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = note,
            onValueChange = { note = it },
            label = { Text(stringResource(R.string.report_note)) },
            modifier = Modifier.fillMaxWidth(),
            minLines = 2,
        )
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(
                onClick = { vm.report(order.id, reason, note) { onClose() } },
                enabled = !vm.sending && reason.isNotEmpty(),
            ) { Text(stringResource(R.string.report_send)) }
            RahalTextButton(onClick = onClose) { Text(stringResource(R.string.cancel)) }
        }
        Spacer(Modifier.height(24.dp))
        }
    }
}

/**
 * **حالُ الطلب بالعربيّة** — **ورمزٌ إنكليزيٌّ في شاشةٍ عربيّةٍ لا يُقرأ.**
 *
 * (وقعت مثلُها في محفظة المالك ٢٠٢٦-٠٨-٠٧: «استرجاع طلب (cancelled)».)
 */
private fun statusAr(status: String): String = when (status) {
    "delivered" -> "سُلّم"
    "cancelled" -> "أُلغي"
    "rejected" -> "اعتُذر عنه"
    "failed" -> "تعذّر تسليمه"
    "refunded" -> "استُرجع"
    else -> status
}

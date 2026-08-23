package com.rahalgo.merchant.drawer

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.MerchantReport
import com.rahalgo.shared.merchant.ReportSummary
import com.rahalgo.shared.merchant.Warning
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.Empty
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ثلاثةُ أقسامٍ في الدرج — كلٌّ يُفتح عند حادثة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «الإنذارات احذفها من هنا وستكون بقسمٍ
 *  مستقلٍّ بالقائمة الجانبيّة · وبالقائمة الجانبيّة يجب أن يكون قسمُ
 *  الشكاوي والبلاغات · وقسمُ التقارير أيضاً».)
 *
 * # ولماذا خرجت من «متجري»
 *
 * **«متجري» شاشةٌ تُفتح صباحاً ليقول: أنا مفتوح** — حالُه وساعاتُه
 * وأرقامُ يومه. **وثلاثةُ أقسامٍ تُقرأ عند حادثةٍ لا مكانَ لها فيها**:
 * يطول السَّرْدُ فيُمرَّر ما يهمّ.
 *
 * # وواحدٌ منها ليس كالآخرَين
 *
 * **الإنذارُ ما عليه — والبلاغُ ما له.** وخلطُهما في شاشةٍ واحدةٍ
 * (كما كانا في «متجري») **يجعل صاحبَ المتجر يقرأ إنذاراً فيظنّه جواباً
 * على بلاغه.**
 */

// ══════════════════════════════════════════════════════════════════════
// **١ · الإنذارات — ما عليه**
// ══════════════════════════════════════════════════════════════════════

class WarningsViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var items by mutableStateOf<List<Warning>>(emptyList())
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            runCatching { items = api.warnings().warnings }
                .onFailure { error = it.message ?: "تعذّر جلب الإنذارات" }
                .onSuccess { error = "" }
            loading = false
        }
    }
}

@Composable
fun WarningsScreen(vm: WarningsViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.items.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(R.string.menu_warnings), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(R.string.menu_warnings),
                stringResource(R.string.warnings_hint),
            )
            Spacer(Modifier.height(10.dp))

            // **ولا إنذارَ خبرٌ سارّ** — فيُقال بلون العلامة لا بسطرٍ رماديّ.
            if (vm.items.isEmpty()) {
                Note(stringResource(R.string.store_no_warnings), Rahal.colors.brand)
                return@Screen
            }
            Card {
                vm.items.forEachIndexed { i, w ->
                    if (i > 0) HorizontalDivider()
                    Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
                        Text(w.reason, fontWeight = FontWeight.Bold)
                        if (w.note.isNotBlank()) {
                            Text(
                                w.note,
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Text(
                            day(w.createdAt),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · الشكاوي والبلاغات — ما أرسله هو**
// ══════════════════════════════════════════════════════════════════════

class MyReportsViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var items by mutableStateOf<List<MerchantReport>>(emptyList())
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            runCatching { items = api.myReports().reports }
                .onFailure { error = it.message ?: "تعذّر جلب البلاغات" }
                .onSuccess { error = "" }
            loading = false
        }
    }
}

@Composable
fun MyReportsScreen(vm: MyReportsViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.items.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(R.string.menu_reports), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(R.string.menu_reports),
                stringResource(R.string.reports_list_hint),
            )
            Spacer(Modifier.height(10.dp))

            if (vm.items.isEmpty()) {
                Empty(stringResource(R.string.reports_list_empty))
                return@Screen
            }
            Card {
                vm.items.forEachIndexed { i, t ->
                    if (i > 0) HorizontalDivider()
                    Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                reasonAr(t.reason),
                                modifier = Modifier.weight(1f),
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                ticketStatusAr(t.status),
                                color = when (t.status) {
                                    "resolved" -> Rahal.colors.brand
                                    "rejected" -> Rahal.colors.danger
                                    else -> Rahal.colors.inkMuted
                                },
                                style = MaterialTheme.typography.labelMedium,
                            )
                        }
                        t.orderNumber?.let {
                            Text(
                                stringResource(R.string.reports_on_order, it),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        // **وبلاغٌ يُغلق بلا كلمةٍ يُقرأ تجاهلاً** — ولو كان
                        // القرارُ في صالحه.
                        if (t.resolution.isNotBlank()) {
                            Spacer(Modifier.height(4.dp))
                            Text(
                                t.resolution,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Text(
                            day(t.createdAt),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · التقارير — مدًى يختاره لا يومَه وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// **و«متجري» يقول يومَه** — وهذا يقول أسبوعَه وشهرَه. **ولولا المدى
// لكان القسمان واحداً.**

class SalesViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var summary by mutableStateOf(ReportSummary())
        private set

    /** ٠ اليوم · ١ سبعة أيّام · ٢ ثلاثون يوماً. */
    var range by mutableStateOf(1)
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    private var storeId: String = ""

    init {
        load()
    }

    fun pick(r: Int) {
        range = r
        loading = true
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
                val to = java.time.LocalDate.now()
                val from = when (range) {
                    0 -> to
                    2 -> to.minusDays(29)
                    else -> to.minusDays(6)
                }
                summary = api.reports(storeId, from.toString(), to.toString()).summary
                error = ""
            }.onFailure { error = it.message ?: "تعذّر جلب التقرير" }
            loading = false
        }
    }
}

@Composable
fun SalesScreen(vm: SalesViewModel) {
    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(R.string.menu_sales),
                stringResource(R.string.sales_hint),
            )
            Spacer(Modifier.height(10.dp))

            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                listOf(
                    R.string.sales_today,
                    R.string.sales_week,
                    R.string.sales_month,
                ).forEachIndexed { i, label ->
                    FilterChip(
                        selected = vm.range == i,
                        onClick = { vm.pick(i) },
                        label = { Text(stringResource(label)) },
                    )
                }
            }
            Spacer(Modifier.height(12.dp))

            if (vm.loading || vm.error.isNotEmpty()) {
                LoadState(vm.loading, vm.error) { vm.load() }
                return@Screen
            }

            Card {
                KeyValue(stringResource(R.string.reports_orders_all), vm.summary.orders.toString())
                KeyValue(stringResource(R.string.reports_delivered), vm.summary.delivered.toString())
                KeyValue(stringResource(R.string.reports_cancelled), vm.summary.cancelled.toString())
                HorizontalDivider()
                Spacer(Modifier.height(6.dp))
                // ══════════════════════════════════════════════════════
                // **ومبيعاتُه بسعره هو لا بما دفعه الزبون**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٠.) **ومتجرٌ يقرأ مبيعاتٍ فيها
                // هامشُ المنصّة يحسب أرباحاً ليست له**، ثمّ يجدها ناقصةً
                // في محفظته فيظنّ المنصّةَ اقتطعت.
                KeyValue(stringResource(R.string.reports_sales), money(vm.summary.sales))
                KeyValue(stringResource(R.string.reports_commission), money(vm.summary.commission))
                KeyValue(
                    stringResource(R.string.reports_due),
                    money(vm.summary.due),
                    valueColor = Rahal.colors.brand,
                )
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **ألفاظٌ يقرؤها صاحبُ المتجر**
// ══════════════════════════════════════════════════════════════════════
//
// **ورمزٌ إنكليزيٌّ في شاشةٍ عربيّةٍ لا يُقرأ** — وقعت مثلُها في محفظة
// المالك ٢٠٢٦-٠٨-٠٧: «استرجاع طلب (cancelled)».

internal fun reasonAr(code: String): String = when (code) {
    "driver_late_pickup" -> "السائق تأخّر بالاستلام"
    "driver_refused" -> "السائق رفض أخذ الطلب"
    "driver_conduct" -> "سلوك السائق"
    "other" -> "سبب آخر"
    else -> code
}

private fun ticketStatusAr(status: String): String = when (status) {
    "open" -> "قيد النظر"
    "in_progress" -> "قيد النظر"
    "resolved" -> "فُصل فيه"
    "rejected" -> "رُدّ"
    "closed" -> "أُغلق"
    else -> status
}

/** **واليومُ يُقتطع من الطابع** — ولا ساعةَ فيه: **الحادثةُ يومٌ لا لحظة.** */
private fun day(iso: String): String =
    if (iso.length >= 10) iso.substring(0, 10) else iso

package com.rahalgo.merchant.drawer

import com.rahalgo.merchant.noStoreMsg
import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
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
import com.rahalgo.ui.err
import com.rahalgo.ui.RahalButton
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
                .onFailure { error = err(it) }
                .onSuccess { error = "" }
            loading = false
        }
    }
}

@Composable
fun WarningsScreen(vm: WarningsViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.items.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(com.rahalgo.merchant.R.string.menu_warnings), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(com.rahalgo.merchant.R.string.menu_warnings),
                stringResource(com.rahalgo.merchant.R.string.warnings_hint),
            )
            Spacer(Modifier.height(10.dp))

            // **ولا إنذارَ خبرٌ سارّ** — فيُقال بلون العلامة لا بسطرٍ رماديّ.
            if (vm.items.isEmpty()) {
                Note(stringResource(com.rahalgo.merchant.R.string.store_no_warnings), Rahal.colors.brand)
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

    /**
     * ══════════════════════════════════════════════════════════════════
     * **اتّجاهان — ما رفعه وما رُفع عليه**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «الشكاوي يلي عليه ويلي اله».)
     *
     * **وكان يرى ما رفعه هو وحدَه** — فلا يعلم أنّ زبوناً شكا منه
     * **حتّى يصله إنذارُ الإدارة.** وشكوى تُعالَج قبل أن تصير إنذاراً
     * خيرٌ للطرفين.
     */
    var against by mutableStateOf<List<MerchantReport>>(emptyList())
        private set

    /** `false` = ما رفعتُه · `true` = ما رُفع عليّ. */
    var showAgainst by mutableStateOf(false)

    fun switch(v: Boolean) {
        showAgainst = v
    }

    fun load() {
        viewModelScope.launch {
            runCatching { items = api.myReports().reports }
                .onFailure { error = err(it) }
                .onSuccess { error = "" }
            // **وما رُفع عليه زينةٌ حول الحال** — وعطبُه لا يحرمه من
            // رؤية شكاواه هو.
            runCatching {
                val id = api.stores().stores.firstOrNull()?.id.orEmpty()
                if (id.isNotEmpty()) against = api.reportsAgainst(id).reports
            }
            loading = false
        }
    }
}

@Composable
fun MyReportsScreen(vm: MyReportsViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.items.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(com.rahalgo.merchant.R.string.menu_reports), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(com.rahalgo.merchant.R.string.menu_reports),
                stringResource(com.rahalgo.merchant.R.string.reports_list_hint),
            )
            Spacer(Modifier.height(10.dp))

            // **ومبدّلٌ لا شاشتان** — الشكوى شكوى، والفرقُ من رفعها.
            SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                SegmentedButton(
                    selected = !vm.showAgainst,
                    onClick = { vm.switch(false) },
                    shape = SegmentedButtonDefaults.itemShape(0, 2),
                ) { Text(stringResource(com.rahalgo.merchant.R.string.reports_mine)) }
                SegmentedButton(
                    selected = vm.showAgainst,
                    onClick = { vm.switch(true) },
                    shape = SegmentedButtonDefaults.itemShape(1, 2),
                ) { Text(stringResource(com.rahalgo.merchant.R.string.reports_against)) }
            }
            Spacer(Modifier.height(10.dp))

            val list = if (vm.showAgainst) vm.against else vm.items
            if (list.isEmpty()) {
                Empty(
                    stringResource(
                        if (vm.showAgainst) com.rahalgo.merchant.R.string.reports_against_empty
                        else com.rahalgo.merchant.R.string.reports_list_empty,
                    ),
                )
                return@Screen
            }
            Card {
                list.forEachIndexed { i, t ->
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
                                stringResource(com.rahalgo.merchant.R.string.reports_on_order, it),
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

    /** **بدايةُ المدى المخصّص** — تُختار من تقويم النظام. */
    var customFrom by mutableStateOf(java.time.LocalDate.now().withDayOfMonth(1))

    /** **نهايتُه** — واليومُ افتراضاً. */
    var customTo by mutableStateOf(java.time.LocalDate.now())

    /** **المدى المعروض فعلاً** — يُطبع في الورقة فلا تُقرأ بلا تاريخ. */
    var fromShown by mutableStateOf("")
        private set
    var toShown by mutableStateOf("")
        private set

    fun setCustom(f: java.time.LocalDate, t: java.time.LocalDate) {
        // **ولا مدًى مقلوب** — من اختار البدايةَ بعد النهاية يردّ
        // المحرّكُ فراغاً، **فيظنّ متجرَه بلا مبيعات.**
        customFrom = if (f.isAfter(t)) t else f
        customTo = t
        range = 3
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
                    error = noStoreMsg()
                    loading = false
                    return@launch
                }
                // ══════════════════════════════════════════════════
                // **ومدًى يختاره لا ثلاثةَ أزرارٍ فقط**
                // ══════════════════════════════════════════════════
                //
                // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «أضيف أيضاً تاريخاً محدّداً
                //  بحيث يبقى التطبيقُ محتفظاً بالسجلّ كاملاً من أوّل
                //  لحظةٍ لآخر لحظة».)
                //
                // **وثلاثةُ مدىً محسوبةٍ في الشيفرة تكفي للنظرة
                // اليوميّة ولا تكفي للمحاسبة** — ومن أراد شهرَ آبَ
                // كاملاً بعد أن مضى لم يجد له باباً.
                //
                // **والمحرّكُ يقبل أيَّ مدىً أصلاً** (`from`/`to`) —
                // **والقيدُ كان في التطبيق وحدَه.**
                val today = java.time.LocalDate.now()
                val to = if (range == 3) customTo else today
                val from = when (range) {
                    0 -> today
                    2 -> today.minusDays(29)
                    3 -> customFrom
                    else -> today.minusDays(6)
                }
                fromShown = from.toString()
                toShown = to.toString()
                summary = api.reports(storeId, from.toString(), to.toString()).summary
                error = ""
            }.onFailure { error = err(it) }
            loading = false
        }
    }
}

@Composable
fun SalesScreen(vm: SalesViewModel) {
    val context = LocalContext.current
    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            ScreenTitle(
                stringResource(com.rahalgo.merchant.R.string.menu_sales),
                stringResource(com.rahalgo.merchant.R.string.sales_hint),
            )
            Spacer(Modifier.height(10.dp))

            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                listOf(
                    com.rahalgo.merchant.R.string.act_today,
                    com.rahalgo.merchant.R.string.sales_week,
                    com.rahalgo.merchant.R.string.sales_month,
                ).forEachIndexed { i, label ->
                    FilterChip(
                        selected = vm.range == i,
                        onClick = { vm.pick(i) },
                        label = { Text(stringResource(label)) },
                    )
                }
                // ══════════════════════════════════════════════════════
                // **ومدًى يختاره من تقويم النظام**
                // ══════════════════════════════════════════════════════
                //
                // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «أضيف أيضاً تاريخاً محدّداً
                //  بحيث يبقى التطبيقُ محتفظاً بالسجلّ كاملاً من أوّل
                //  لحظةٍ لآخر لحظة».)
                //
                // **وتقويمُ النظام لا تقويمٌ نصنعه** — رآه في المنبّه
                // والتقويم، **وواحدٌ نصنعه غريبٌ عليه** مهما أتقنّاه.
                FilterChip(
                    selected = vm.range == 3,
                    onClick = { pickRange(context, vm) },
                    label = { Text(stringResource(com.rahalgo.merchant.R.string.sales_custom)) },
                )
            }
            Spacer(Modifier.height(12.dp))

            if (vm.loading || vm.error.isNotEmpty()) {
                LoadState(vm.loading, vm.error) { vm.load() }
                return@Screen
            }

            Card {
                KeyValue(stringResource(com.rahalgo.merchant.R.string.nav_orders_all), vm.summary.orders.toString())
                KeyValue(stringResource(com.rahalgo.merchant.R.string.reports_delivered), vm.summary.delivered.toString())
                KeyValue(stringResource(com.rahalgo.merchant.R.string.reports_cancelled), vm.summary.cancelled.toString())
                HorizontalDivider()
                Spacer(Modifier.height(6.dp))
                // ══════════════════════════════════════════════════════
                // **ومبيعاتُه بسعره هو لا بما دفعه الزبون**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٠.) **ومتجرٌ يقرأ مبيعاتٍ فيها
                // هامشُ المنصّة يحسب أرباحاً ليست له**، ثمّ يجدها ناقصةً
                // في محفظته فيظنّ المنصّةَ اقتطعت.
                KeyValue(stringResource(com.rahalgo.merchant.R.string.reports_sales), money(vm.summary.sales))
                KeyValue(stringResource(com.rahalgo.merchant.R.string.reports_commission), money(vm.summary.commission))
                KeyValue(
                    stringResource(com.rahalgo.merchant.R.string.reports_due),
                    money(vm.summary.due),
                    valueColor = Rahal.colors.brand,
                )
            }

            // ══════════════════════════════════════════════════════════
            // **وورقةٌ يراجعها بيده**
            // ══════════════════════════════════════════════════════════
            //
            // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «التقارير لازم يكون فيها طباعة
            //  مشان يراجع كلشي طلبات من متجره».)
            //
            // **ونافذةُ الطباعة في أندرويد فيها «حفظ بصيغة PDF»** —
            // فمن لا طابعةَ عنده يحفظ ويرسل، **وهو ما يفعله أكثرُهم.**
            Spacer(Modifier.height(12.dp))
            val lblOrders = stringResource(com.rahalgo.merchant.R.string.nav_orders_all)
            val lblDone = stringResource(com.rahalgo.merchant.R.string.reports_delivered)
            val lblCancel = stringResource(com.rahalgo.merchant.R.string.reports_cancelled)
            val lblSales = stringResource(com.rahalgo.merchant.R.string.reports_sales)
            val lblComm = stringResource(com.rahalgo.merchant.R.string.reports_commission)
            val lblDue = stringResource(com.rahalgo.merchant.R.string.reports_due)
            val title = stringResource(com.rahalgo.merchant.R.string.menu_sales)
            RahalButton(
                onClick = {
                    printReport(
                        context, title, vm.fromShown, vm.toShown,
                        listOf(
                            lblOrders to vm.summary.orders.toString(),
                            lblDone to vm.summary.delivered.toString(),
                            lblCancel to vm.summary.cancelled.toString(),
                            lblSales to money(vm.summary.sales),
                            lblComm to money(vm.summary.commission),
                            lblDue to money(vm.summary.due),
                        ),
                    )
                },
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(com.rahalgo.merchant.R.string.sales_print)) }

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
    "driver_late_pickup" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.rs_driver_late)
    "driver_refused" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.rs_driver_refused)
    "driver_conduct" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.rs_driver_conduct)
    "other" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.rs_other)
    else -> code
}

private fun ticketStatusAr(status: String): String = when (status) {
    "open" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.tk_open)
    "in_progress" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.tk_open)
    "resolved" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.tk_resolved)
    "rejected" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.tk_rejected)
    "closed" -> com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.tk_closed)
    else -> status
}

/** **واليومُ يُقتطع من الطابع** — ولا ساعةَ فيه: **الحادثةُ يومٌ لا لحظة.** */
private fun day(iso: String): String =
    if (iso.length >= 10) iso.substring(0, 10) else iso

/**
 * ══════════════════════════════════════════════════════════════════════
 * **يفتح تقويمين: من ثمّ إلى**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦.)
 *
 * **والثاني يُفتح من داخل الأوّل** — فلا يُنسى، **ومن اختار بدايةً
 * وأغلق بقي التقريرُ على مدًى نصفِه قديم.**
 *
 * **ولا يُسمح بيومٍ بعد اليوم** — تقريرٌ لمستقبلٍ فارغٌ دائماً،
 * **ومن رآه فارغاً ظنّ العطبَ في التطبيق.**
 */
private fun pickRange(context: android.content.Context, vm: SalesViewModel) {
    val today = java.time.LocalDate.now()
    val f = vm.customFrom
    android.app.DatePickerDialog(
        context,
        { _, y1, m1, d1 ->
            val from = java.time.LocalDate.of(y1, m1 + 1, d1)
            val t = vm.customTo
            android.app.DatePickerDialog(
                context,
                { _, y2, m2, d2 ->
                    vm.setCustom(from, java.time.LocalDate.of(y2, m2 + 1, d2))
                },
                t.year, t.monthValue - 1, t.dayOfMonth,
            ).apply {
                datePicker.maxDate = System.currentTimeMillis()
                // **ولا نهايةَ قبل البداية** — انظر `setCustom`.
                show()
            }
        },
        f.year, f.monthValue - 1, f.dayOfMonth,
    ).apply {
        datePicker.maxDate = System.currentTimeMillis()
        show()
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطباعة — ورقةٌ يراجعها بيده**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «التقارير لازم يكون فيها طباعة مشان يراجع
 *  كلشي طلبات من متجره».)
 *
 * # ولماذا HTML لا رسمٌ بيدنا
 *
 * **وطابعةُ أندرويد تقبل `PrintDocumentAdapter`** — و`WebView` تصنعه
 * من HTML مجّاناً. **ورسمُ صفحةٍ بأيدينا يعني حسابَ الأسطر والصفحات
 * والهوامش**، وثلاثمئة سطرٍ لِما يفعله المتصفّح.
 *
 * # وتُحفظ PDF كما تُطبع
 *
 * **ونافذةُ الطباعة في أندرويد فيها «حفظ بصيغة PDF»** — فمن لا طابعةَ
 * عنده يحفظ ويرسل. **وهو ما يفعله أكثرُهم.**
 *
 * # والاتّجاه من اليمين
 *
 * **و`dir="rtl"` في الورقة نفسِها** — وإلّا خرجت أرقامٌ عربيّةٌ في
 * صفحةٍ إنكليزيّة الاتّجاه فتُقرأ مقلوبة.
 */
private fun printReport(
    context: android.content.Context,
    title: String,
    from: String,
    to: String,
    rows: List<Pair<String, String>>,
) {
    val body = rows.joinToString("") {
        "<tr><td>${it.first}</td><td><b>${it.second}</b></td></tr>"
    }
    val html = """
        <html dir="rtl"><head><meta charset="utf-8">
        <style>
          body{font-family:sans-serif;padding:24px;color:#07283a}
          h1{font-size:20px;margin:0 0 4px}
          .r{color:#5a6b75;font-size:13px;margin-bottom:16px}
          table{width:100%;border-collapse:collapse}
          td{padding:10px 4px;border-bottom:1px solid #e3e8eb;font-size:15px}
          .f{margin-top:20px;color:#5a6b75;font-size:12px}
        </style></head><body>
        <h1>$title</h1>
        <div class="r">من $from إلى $to</div>
        <table>$body</table>
        <div class="f">رحال غو — الرقة</div>
        </body></html>
    """.trimIndent()

    val web = android.webkit.WebView(context)
    web.webViewClient = object : android.webkit.WebViewClient() {
        override fun onPageFinished(view: android.webkit.WebView, url: String) {
            val pm = context.getSystemService(android.content.Context.PRINT_SERVICE)
                as android.print.PrintManager
            pm.print(
                title,
                view.createPrintDocumentAdapter(title),
                android.print.PrintAttributes.Builder().build(),
            )
        }
    }
    web.loadDataWithBaseURL(null, html, "text/HTML", "UTF-8", null)
}

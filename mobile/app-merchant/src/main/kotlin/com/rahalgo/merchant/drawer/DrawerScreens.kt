package com.rahalgo.merchant.drawer

import com.rahalgo.ui.DocPrint
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
import com.rahalgo.ui.TicketRow
import com.rahalgo.ui.TicketsScreen
import com.rahalgo.ui.Card
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.StatBox
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

    // **وأيُّ التبويبين معروضٌ صار في الشاشة المركزيّة** — والقائمتان
    // تُحمَّلان معاً هنا، **فحالُ العرض شأنُ الشاشة لا شأنُ النداء.**

    fun load() {
        viewModelScope.launch {
            runCatching { items = api.myReports().reports }
                .onFailure { error = err(it) }
                .onSuccess { error = "" }
            // **وما رُفع عليه زينةٌ حول الحال** — وعطبُه لا يحرمه من
            // رؤية شكاواه هو.
            // **وفشلُ «التي عليّ» يُقال** — كان يُبتلع صامتاً، **فتُعرض
            // «لا شكاوى عليك» والنداءُ لم يصل أصلاً.** ومن رآها اطمأنّ
            // إلى شيءٍ لم يُقرأ.
            runCatching {
                val id = api.stores().stores.firstOrNull()?.id.orEmpty()
                if (id.isNotEmpty()) against = api.reportsAgainst(id).reports
            }.onFailure { if (error.isEmpty()) error = err(it) }
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

    // ══════════════════════════════════════════════════════════════════
    // **والشاشةُ من `:ui` — وشكلُها هو الذي عمّ الثلاثة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٣١: «أفضلُ شكلٍ هو شكلُ الشكاوى في المتجر ·
    // وأساساً يجب أن يكون بشكلٍ مركزيّ».)
    //
    // **والسببُ يُترجَم هنا لا هناك**: رموزُ المتجر عن السائقين،
    // **ورموزُ السائق عن المتاجر والزبائن** — **وشاشةٌ تترجم للجميع
    // تحتاج أن تعرف الأدوار كلَّها.**
    val toRow: (com.rahalgo.shared.merchant.MerchantReport) -> TicketRow = { t ->
        TicketRow(
            key = "#" + t.number,
            title = reasonAr(t.reason),
            status = t.status,
            orderNumber = t.orderNumber?.toString().orEmpty(),
            resolution = t.resolution,
            date = day(t.createdAt),
        )
    }
    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        TicketsScreen(
            title = stringResource(com.rahalgo.merchant.R.string.menu_reports),
            hint = stringResource(com.rahalgo.merchant.R.string.reports_list_hint),
            mine = vm.items.map(toRow),
            mineEmpty = stringResource(com.rahalgo.merchant.R.string.reports_list_empty),
            againstMe = vm.against.map(toRow),
            againstMeEmpty = stringResource(com.rahalgo.merchant.R.string.reports_against_empty),
        )
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

    // ══════════════════════════════════════════════════════════════════
    // **وهويّةُ المنصّة للورقة المطبوعة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وورقةٌ بلا علامةٍ ولا رقمِ دعمٍ لا تُقدَّم إلى أحد** — وكشفُ
    // الحساب يحملهما منذ ٢٠٢٦-٠٨-١٣، **والتقريرُ كان بلاهما.**
    //
    // **وتُقرأ من المحرّك لا تُكتب هنا** — الاسمُ يتبدّل والرقمُ ينتقل.
    var platform by mutableStateOf("")
        private set
    var support by mutableStateOf("")
        private set
    var storeName by mutableStateOf("")
        private set

    /** **لوغو المنصّة بايتاتٍ** — انظر `DocPrint.dataUri`. */
    var logo by mutableStateOf("")
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
                    val page = api.stores()
                    val mine = page.stores.firstOrNull()
                    storeId = mine?.id ?: ""
                    storeName = mine?.name.orEmpty()
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
                if (platform.isEmpty()) {
                    // **مرّةً واحدةً** — الاسمُ والرقمُ لا يتبدّلان في جلسة.
                    runCatching { AppCore.get().auth.platform() }.getOrNull()?.let {
                        platform = it.name
                        support = it.supportPhone
                        // **واللوغو بايتاتٍ لا عنواناً** — صفحةُ الطباعة
                        // بلا أصلٍ تُنسَب إليه العناوين، **فعنوانٌ لا
                        // يُجلَب** وتسقط الورقةُ إلى حرفِ العلامة.
                        logo = DocPrint.dataUri(
                            AppCore.get().api,
                            AppCore.get().media(it.logo).orEmpty(),
                        )
                    }
                }
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

            // **ومربّعاتٌ لا أسطر** — (طلبُ المالك ٢٠٢٦-٠٨-٣١: «بالتقارير
            // يجب أن تكون مربّعاتٍ: الطلبات · تم التسليم · ملغي ·
            // المبيعات · عمولة المنصّة · مستحقاتك»).
            //
            // **وصفٌّ للعدد وصفٌّ للمال** — ولا يُخلطان: **ثلاثةٌ تُعدّ
            // وثلاثةٌ تُقبض.**
            Card {
                StatRow {
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.nav_orders_all),
                        value = vm.summary.orders.toString(),
                        modifier = Modifier.weight(1f),
                    )
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.reports_delivered),
                        value = vm.summary.delivered.toString(),
                        modifier = Modifier.weight(1f),
                    )
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.reports_cancelled),
                        value = vm.summary.cancelled.toString(),
                        modifier = Modifier.weight(1f),
                    )
                }
                Spacer(Modifier.height(8.dp))
                // ══════════════════════════════════════════════════════
                // **ومبيعاتُه بسعره هو لا بما دفعه الزبون**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٠.) **ومتجرٌ يقرأ مبيعاتٍ فيها
                // هامشُ المنصّة يحسب أرباحاً ليست له**، ثمّ يجدها ناقصةً
                // في محفظته فيظنّ المنصّةَ اقتطعت.
                StatRow {
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.reports_sales),
                        value = money(vm.summary.sales),
                        modifier = Modifier.weight(1f),
                    )
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.reports_commission),
                        value = money(vm.summary.commission),
                        modifier = Modifier.weight(1f),
                    )
                    StatBox(
                        label = stringResource(com.rahalgo.merchant.R.string.reports_due),
                        value = money(vm.summary.due),
                        modifier = Modifier.weight(1f),
                        color = Rahal.colors.brand,
                    )
                }
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
                        vm.storeName, vm.platform, vm.support, vm.logo,
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
 * **تقريرُ المبيعات — بورق المنصّة نفسِه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٩: «القالب مثل الطباعة بالمحفظة أو كشف
 *  الحساب».)
 *
 * **وكان `‎<h1>` وجدولاً بعمودين** — لا علامةَ ولا تاريخَ طباعةٍ ولا
 * رقمَ دعم. **وورقةٌ تخرج من هاتفٍ بلا اسمٍ عليها لا تُقدَّم إلى أحد.**
 *
 * **والهيكلُ في `‎ui.DocPrint`** — أنماطُ كشف الحساب وترويستُه وذيلُه.
 * **فما أُتقن على ثلاث شكاوى يُنال بلا أن يُعاد.**
 */
private fun printReport(
    context: android.content.Context,
    title: String,
    from: String,
    to: String,
    rows: List<Pair<String, String>>,
    storeName: String,
    platform: String,
    support: String,
    logo: String,
) {
    val body = rows.joinToString("") {
        """<tr><td>${DocPrint.esc(it.first)}</td>""" +
            """<td class="cell"><b>${DocPrint.esc(it.second)}</b></td></tr>"""
    }
    val html = DocPrint.page(
        DocPrint.head(context, platform, logo) +
            """
<p class="title">${DocPrint.esc(title)}</p>
<div class="meta">
  <span><b>${DocPrint.esc(storeName)}</b></span>
  <span>${DocPrint.esc(context.getString(com.rahalgo.ui.R.string.sheet_period))} """ +
            """${DocPrint.esc(from)} — ${DocPrint.esc(to)}</span>
</div>
<table><tbody>$body</tbody></table>""" +
            DocPrint.foot(context, platform, support),
    )
    DocPrint.print(context, title, html)
}

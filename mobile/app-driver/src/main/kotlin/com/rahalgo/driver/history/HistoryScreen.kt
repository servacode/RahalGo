package com.rahalgo.driver.history

import android.app.Application
import com.rahalgo.driver.ui.ScreenTitle
import com.rahalgo.design.Rahal
import android.util.Log
import androidx.compose.foundation.clickable
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.RadioButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.TextButton
import androidx.compose.runtime.remember
import androidx.compose.ui.res.painterResource
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.HistoryOrder
import com.rahalgo.shared.model.ReportReason
import com.rahalgo.shared.net.ApiClient
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سجلُّ الطلبات — جوابٌ لا قائمةٌ تُتصفَّح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن نبني سجلّ الطلبات… يجب أن يعرف
 *  السائقُ ماذا عمل بالفترات السابقة» · «واسم سجلّ الطلبات أفضل».)
 *
 * # ولماذا وُجد أصلا
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٥، وهو مكتوبٌ في `driver_history.go`:)
 *
 * **حسابٌ يُقفل بلا سجلٍّ يُقرأ ظلمٌ صامت** — يُخصم منه في آخر الشهر عن
 * طلبٍ **لا يستطيع أن يراه.**
 *
 * # ويجيب سؤالين لا ثالثَ لهما
 *
 * **أ · «أيُّ طلبٍ هذا؟»** — محفظتُه تقول «‎+٣٠٠ · الطلب ‎#١٠٠٩»،
 * **ومن أراد أن يتحقّق لا يجده.** وشكوى تجيء بعد يومين عن طلبٍ لا
 * يتذكّره **تجعله يوقّع على ما لا يعرف.**
 *
 * **ب · «كم عملتُ؟»** — وهو يُحاسَب على الرقم ولا يراه.
 *
 * # فبُني جوابا
 *
 * **سطرٌ فوق يقول كم سلّم وكم تعذّر**، ثمّ القائمةُ بالأحدث، **وبحثٌ
 * بالرقم** — وهو ما يُسأل عنه فعلاً. **والسائقُ لا يجلس يقرأ طلباتِه**:
 * يفتح السجلَّ بسؤالٍ في رأسه.
 */
data class HistoryState(
    val orders: List<HistoryOrder> = emptyList(),
    val loading: Boolean = true,
    val error: String = "",
    val done: String = "",
    val busy: Boolean = false,
    /**
     * **أسبابُ البلاغ من الخادم** — تُجلب مرّةً عند فتح النافذة.
     *
     * **ولا تُكتب في التطبيق**: قائمةٌ في مكانين تفترق حين يُضاف سببٌ
     * في أحدهما، **فيرسل التطبيقُ رمزاً لا يعرفه الخادم.**
     */
    val reasons: List<ReportReason> = emptyList(),
)

class HistoryViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(HistoryState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        load()
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    /** **يجلب أسبابَ البلاغ** — مرّةً وتبقى. */
    fun loadReasons(orderId: String) {
        if (state.reasons.isNotEmpty()) return
        viewModelScope.launch {
            state = try {
                state.copy(reasons = backend.driver.reportReasons(orderId).reasons)
            } catch (e: Exception) {
                Log.e("RahalGo/سجل", "فشل جلب الأسباب", e)
                state.copy(
                    error = getApplication<Application>().getString(R.string.hist_reasons_error),
                )
            }
        }
    }

    /** **يرفع بلاغاً** ثمّ يُنعش — البلاغُ لا يُعاد مرّتين. */
    fun report(orderId: String, reason: String, note: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            state = try {
                backend.driver.report(orderId, reason, note.trim())
                state.copy(
                    busy = false,
                    done = getApplication<Application>().getString(R.string.hist_report_sent),
                )
            } catch (e: Exception) {
                Log.e("RahalGo/سجل", "فشل البلاغ", e)
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    /** **يقيّم المتجر** ثمّ يُنعش — فيختفي الزرُّ عن الطلب. */
    fun rate(orderId: String, speed: Int, conduct: Int, comment: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            try {
                backend.driver.rateMerchant(orderId, speed, conduct, comment.trim())
                val fresh = backend.driver.history().orders
                state = state.copy(
                    orders = fresh,
                    busy = false,
                    done = getApplication<Application>().getString(R.string.hist_rate_sent),
                )
            } catch (e: Exception) {
                Log.e("RahalGo/سجل", "فشل التقييم", e)
                state = state.copy(busy = false, error = describe(e))
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **والسببُ يُقال كما ردّه المحرّك**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٣: «البلاغات والتقييمات التي أرسلتُها لم
    //  تصل للإدارة، ما هو السبب؟».)
    //
    // **وقد وصلت**: المحرّكُ ردّ `complaint_already_open` — أي أنّ
    // بلاغَه الأوّلَ وصل، **والثاني رُدّ لأنّ الأوّلَ ما زال مفتوحا.**
    //
    // **والشاشةُ كانت تقول «تعذّر الاتّصال»** لأنّ كلَّ خطأٍ يُترجَم
    // ترجمةً واحدة. **ورسالةٌ تكذب أسوأُ من رسالةٍ غامضة**: ظنّ أنّ
    // شيئاً لم يصل، **وأعاد الإرسالَ مراراً وهو واصل.**
    /**
     * **يسجّل أنّه أعاد البضاعةَ إلى متجرها.**
     *
     * **وفعلٌ يمسّ المال لا يُعاد صامتاً**: يُقال إنّه تمّ، **ويُعاد
     * جلبُ السجلّ** فيختفي الزرُّ من البطاقة — ولا يُضغط مرّتين.
     */
    fun returnGoods(orderId: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            state = try {
                backend.driver.returnGoods(orderId)
                // **والمحفظةُ والصندوقُ يتبدّلان** — فتُبطَل نسخُهما.
                Refresh.bump()
                val page = backend.driver.history()
                state.copy(
                    orders = page.orders,
                    busy = false,
                    done = getApplication<Application>().getString(R.string.hist_returned_done),
                )
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    private fun describe(e: Exception): String {
        val app = getApplication<Application>()
        return when {
            e is ApiClient.ApiException -> when (e.body.code) {
                "complaint_already_open" -> app.getString(R.string.hist_already_open)
                "already_rated" -> app.getString(R.string.hist_already_rated)
                "not_your_order", "forbidden" -> app.getString(R.string.hist_not_yours)
                "unauthorized", "invalid_refresh" -> app.getString(R.string.err_invalid_refresh)
                "" -> app.getString(R.string.err_internal)
                // **ورمزٌ لم يُترجَم يُعرض كما هو** — من رآه أبلغ عنه،
                // **ومن ابتلعه ترك صاحبَه يظنّ العطبَ في يده.**
                else -> e.body.code
            }
            else -> app.getString(R.string.err_network)
        }
    }

    fun load() {
        viewModelScope.launch {
            state = try {
                state.copy(orders = backend.driver.history().orders, loading = false, error = "")
            } catch (e: Exception) {
                Log.e("RahalGo/سجل", "فشل نداء السجلّ", e)
                state.copy(
                    loading = false,
                    error = getApplication<Application>().getString(R.string.err_internal),
                )
            }
        }
    }
}

@Composable
fun HistoryScreen(vm: HistoryViewModel) {
    val s = vm.state
    if (s.loading) {
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) { CircularProgressIndicator() }
        return
    }

    var query by rememberSaveable { mutableStateOf("") }
    // **والنافذةُ تحمل طلبَها** — لا رقماً يُبحث عنه في القائمة.
    // ══════════════════════════════════════════════════════════════════
    // **والنافذةُ تبقى بعد دوران الجهاز**
    // ══════════════════════════════════════════════════════════════════
    //
    // (كشفه فحصٌ شاملٌ ٢٠٢٦-٠٨-١٣: أُديرت الشاشةُ ونافذةُ البلاغ مفتوحةٌ
    //  **فانغلقت.**)
    //
    // **ومن كتب سببَ بلاغه ثمّ أمال هاتفَه فقد ما كتب** — والسائقُ
    // يكتب واقفاً بيدٍ واحدة، **والإمالةُ تقع بلا قصد.**
    //
    // **ولا يُحفظ الطلبُ نفسُه إنّما رقمُه**: `HistoryOrder` صنفٌ لا
    // يُسلسَل، **وحفظُ الكائن يحتاج مُسلسِلاً لا يستحقّه معرّف.**
    var reportId by rememberSaveable { mutableStateOf("") }
    var rateId by rememberSaveable { mutableStateOf("") }
    val reportFor = s.orders.firstOrNull { it.id == reportId }
    val rateFor = s.orders.firstOrNull { it.id == rateId }

    reportFor?.let { o ->
        ReportDialog(vm, s, o, onClose = { reportId = "" })
    }
    rateFor?.let { o ->
        RateDialog(vm, s, o, onClose = { rateId = "" })
    }
    // **والبحثُ بالرقم واسم المتجر** — وهما ما يتذكّره: «طلب المطعم
    // الفلانيّ» أو رقمٌ قرأه في محفظته.
    val shown = if (query.isBlank()) {
        s.orders
    } else {
        val q = query.trim()
        s.orders.filter { it.number.toString().contains(q) || it.merchantName.contains(q) }
    }

    val done = s.orders.count { it.status == "delivered" }
    val failed = s.orders.size - done

    LazyColumn(Modifier.fillMaxSize().padding(horizontal = 16.dp)) {
        item {
            Spacer(Modifier.height(12.dp))
            // ══════════════════════════════════════════════════════════
            // **وللسجلّ عنوانٌ كسائر الأقسام**
            // ══════════════════════════════════════════════════════════
            //
            // (كشفته التجربةُ البشريّة ٢٠٢٦-٠٨-١٣: **تسعةُ أقسامٍ تقول
            //  اسمَها حين تُفتح، وهذا وحدَه يفتح على أرقامٍ بلا اسم.**)
            //
            // **ومن فتحه من القائمة يعرف أين هو**، ومن فتحه بضغطةٍ
            // خاطئةٍ **لا يعرف ما يقرأ.**
            ScreenTitle(
                stringResource(R.string.menu_history),
                stringResource(R.string.hist_hint),
            )
            if (s.error.isNotEmpty()) {
                Text(s.error, color = Rahal.colors.danger, textAlign = TextAlign.Center)
                Spacer(Modifier.height(10.dp))
            }
            // **وما تمّ يُقال** — فعلٌ يمسّ المال لا يقع صامتاً.
            if (s.done.isNotEmpty()) {
                Text(s.done, color = Rahal.colors.success, textAlign = TextAlign.Center)
                Spacer(Modifier.height(10.dp))
            }
            // ══════════════════════════════════════════════════════════
            // **وسطرُ الحصيلة قبل القائمة**
            // ══════════════════════════════════════════════════════════
            //
            // **«كم عملتُ؟» سؤالٌ يُجاب برقمين لا بعدِّ أسطر** — ومن
            // فتح السجلَّ ليعرف كم سلّم **لا يعدّ ثلاثين بطاقةً بإصبعه.**
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Tally(
                    label = stringResource(R.string.hist_done),
                    value = done.toString(),
                    color = Rahal.colors.success,
                    modifier = Modifier.weight(1f),
                )
                Tally(
                    label = stringResource(R.string.hist_failed),
                    value = failed.toString(),
                    color = if (failed > 0) Rahal.colors.danger else Rahal.colors.inkMuted,
                    modifier = Modifier.weight(1f),
                )
            }
            Spacer(Modifier.height(12.dp))
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                label = { Text(stringResource(R.string.hist_search)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))
        }

        if (shown.isEmpty()) {
            item {
                Spacer(Modifier.height(24.dp))
                Text(
                    stringResource(
                        if (s.orders.isEmpty()) R.string.hist_empty else R.string.hist_no_match,
                    ),
                    color = Rahal.colors.inkMuted,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        items(shown, key = { it.id }) { o ->
            Row(
                o = o,
                onReport = { reportId = o.id; vm.loadReasons(o.id) },
                onRate = { rateId = o.id },
                onReturn = { vm.returnGoods(o.id) },
            )
        }
        item { Spacer(Modifier.height(24.dp)) }
    }
}

@Composable
private fun Tally(
    label: String,
    value: String,
    color: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(Rahal.colors.inkMuted.copy(alpha = 0.07f))
            .padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.height(4.dp))
        Text(value, color = color, style = MaterialTheme.typography.headlineSmall)
    }
}

@Composable
private fun Row(
    o: HistoryOrder,
    onReport: () -> Unit,
    onRate: () -> Unit,
    onReturn: () -> Unit,
) {
    val delivered = o.status == "delivered"
    Column(Modifier.fillMaxWidth().padding(vertical = 9.dp)) {
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("#${o.number}", style = MaterialTheme.typography.titleSmall, color = Rahal.colors.brand)
            // **والحالُ كلمةٌ ملوّنة** — تُقرأ قبل أن يُقرأ السطر.
            Text(
                text = stringResource(
                    if (delivered) R.string.hist_st_delivered else R.string.hist_st_failed,
                ),
                color = if (delivered) Rahal.colors.success else Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // **واسمٌ فارغٌ يعني طلباً خاصّاً** — وسطرٌ يُحذف يترك بطاقةً
        // برقمٍ وحده: **من فتح سجلَّه ليتذكّر طلباً لا يتذكّره برقمه.**
        Text(
            text = o.merchantName.ifBlank { stringResource(R.string.card_custom) },
            style = MaterialTheme.typography.bodyMedium,
        )
        // **وسببُ التعذّر يُقال** — من فشل طلبُه يُسأل عنه بعد أيّام،
        // **وسجلٌّ يقول «تعذّر» بلا سببٍ لا يُدافَع به.**
        if (!delivered && o.failReason.isNotEmpty()) {
            Text(o.failReason, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
        }
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = o.createdAt.take(10),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            // **وقيمةُ الطلب لا أجرتُه** — الأجرةُ في المحفظة مجموعةً،
            // **وهذه تقول ما حمله** فيتذكّره.
            // **والخاصُّ قيمتُه ما اتُّفق عليه** — لا عمودُ المال:
            // **المنصّةُ توثّق ولا تحاسب فيبقى صفراً**، وهو صحيحٌ في
            // الدفتر **وكاذبٌ في عين صاحبه.**
            val worth = if (o.total > 0) {
                o.total
            } else {
                (o.customGoods ?: 0) + (o.customFee ?: 0)
            }
            Text(
                text = money(worth),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // ══════════════════════════════════════════════════════════════
        // **وزرّان تحت السطر — بلاغٌ وتقييم**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم يجب أن يكون هناك تقييمٌ للمتجر
        //  من قِبل السائق وبلاغٌ على المتجر… الزبون لا يحتاج إلى
        //  تقييم».)
        //
        // **ولا تقييمَ للزبون** — بأمر المالك. **والبلاغُ يبقى على
        // الاثنين**: عنوانٌ وهميٌّ أو زبونٌ لا يردّ **هو ما بُني السجلُّ
        // لأجله** (٢٠٢٦-٠٨-٠٥)، **والأسبابُ تجيء من الخادم** فلا يقرّر
        // التطبيقُ على من يُشتكى.
        //
        // **والتقييمُ لمن وقف عند بابه فقط** (`can_rate_merchant`):
        // **ومن لم يقف لا رأيَ له فيه.**
        // ══════════════════════════════════════════════════════════════
        // **وبضاعةٌ تعذّر تسليمُها تبقى في يده حتّى تُسجَّل إعادتُها**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ بإرجاع البضاعة».)
        //
        // **والبابُ في المحرّك منذ زمنٍ بلا زرٍّ يفتحه** — لا في الويب
        // ولا في التطبيق. **فمن تعذّر تسليمُه رجع بالبضاعة إلى المتجر
        // ولا شيءَ يسجّل ذلك**، ويبقى مستحقُّ المتجر مدفوعاً على شيءٍ
        // رجع إليه.
        //
        // **ولا يُعرض إلّا حيث يصحّ**: طلبٌ تعذّر · ومتجرٌ يقبل الإرجاع ·
        // **ولم تُسجَّل إعادتُه بعد.** وزرٌّ يُعرض حيث لا يصحّ يُضغط
        // فيُردّ، **ورفضٌ بعد ضغطةٍ يُقرأ عطباً لا قاعدة.**
        if (!delivered && o.acceptsReturns && o.returnedAt == null) {
            Spacer(Modifier.height(4.dp))
            OutlinedButton(onClick = onReturn, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.hist_return))
            }
        }
        // **ومن سجّلها يُقال له ذلك** — لا يختفي الزرُّ صامتاً فيظنّ
        // أنّه لم يُضغط.
        if (o.returnedAt != null) {
            Text(
                text = stringResource(R.string.hist_returned),
                color = Rahal.colors.success,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        androidx.compose.foundation.layout.Row(
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            TextButton(onClick = onReport) {
                Text(stringResource(R.string.hist_report), color = Rahal.colors.danger)
            }
            if (o.canRateMerchant) {
                if (o.merchantRated) {
                    // **ومن قيّم لا يُعرض عليه الزرُّ ثانيةً** — نصٌّ
                    // يقول إنّه فعل، **وزرٌّ يُضغط مرّتين يُقرأ عطبا.**
                    Text(
                        stringResource(R.string.hist_rated),
                        color = Rahal.colors.success,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.padding(top = 14.dp),
                    )
                } else {
                    TextButton(onClick = onRate) {
                        Text(stringResource(R.string.hist_rate), color = Rahal.colors.brand)
                    }
                }
            }
        }
        HorizontalDivider()
    }
}

/**
 * **نافذةُ البلاغ** — سببٌ من الخادم وتفصيلٌ اختياريّ.
 *
 * **ولا يُرسَل بلاغٌ بلا سبب**: «شيءٌ ما حدث» لا تُحقَّق، **والمكتبُ
 * يفتحها ليجد نصّاً حرّاً لا يعرف على من هو.**
 */
@Composable
private fun ReportDialog(
    vm: HistoryViewModel,
    s: HistoryState,
    o: HistoryOrder,
    onClose: () -> Unit,
) {
    var reason by remember { mutableStateOf("") }
    var note by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onClose,
        title = { Text(stringResource(R.string.hist_report_title) + " #" + o.number) },
        text = {
            Column {
                Text(
                    stringResource(R.string.hist_report_hint),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.height(10.dp))
                for (r in s.reasons) {
                    androidx.compose.foundation.layout.Row(
                        Modifier
                            .fillMaxWidth()
                            .clickable { reason = r.code }
                            .padding(vertical = 7.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(selected = reason == r.code, onClick = { reason = r.code })
                        Spacer(Modifier.height(0.dp))
                        Text(reasonLabel(r.code))
                    }
                }
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    label = { Text(stringResource(R.string.hist_report_note)) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = { vm.report(o.id, reason, note); onClose() },
                enabled = !s.busy && reason.isNotEmpty(),
            ) { Text(stringResource(R.string.hist_report_send)) }
        },
        dismissButton = {
            TextButton(onClick = onClose) { Text(stringResource(R.string.hist_cancel)) }
        },
    )
}

/**
 * **نافذةُ تقييم المتجر — محوران لا نجمةٌ واحدة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣.)
 *
 * **«المتجر سيّئ» لا تُصلح شيئا**: أبطيءٌ في التجهيز أم سيّئُ التعامل؟
 * **والمكتبُ يعالج الاثنين بطريقتين** — يكلّم صاحبَه في الأولى ويُنذره
 * في الثانية.
 */
@Composable
private fun RateDialog(
    vm: HistoryViewModel,
    s: HistoryState,
    o: HistoryOrder,
    onClose: () -> Unit,
) {
    var speed by remember { mutableStateOf(0) }
    var conduct by remember { mutableStateOf(0) }

    AlertDialog(
        onDismissRequest = onClose,
        title = { Text(stringResource(R.string.hist_rate_title)) },
        text = {
            Column {
                Text(
                    text = o.merchantName.ifBlank { stringResource(R.string.card_custom) },
                    style = MaterialTheme.typography.bodyMedium,
                )
                Spacer(Modifier.height(12.dp))
                StarPick(stringResource(R.string.hist_rate_speed), speed) { speed = it }
                Spacer(Modifier.height(10.dp))
                StarPick(stringResource(R.string.hist_rate_conduct), conduct) { conduct = it }
            }
        },
        confirmButton = {
            TextButton(
                // ══════════════════════════════════════════════════════
                // **ولا تعليقَ مع التقييم — نجومٌ وفقط**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نحن اتّفقنا بلا تعليقٍ أو
                //  ملاحظات، فقط تقييم».)
                //
                // **ومحوران بنجومٍ يقولان ما يلزم**: أبطأ أم أساء —
                // **ونصٌّ حرٌّ يفتح خصومةً تُقرأ ولا تُحقَّق.**
                //
                // **والحقلُ باقٍ في النداء فارغا** — المحرّكُ يقبله
                // اختياريّاً، **ونداءٌ يُبدَّل شكلُه لأجل حقلٍ لا يُملأ**
                // يكسر شاشةَ الويب التي تملؤه.
                onClick = { vm.rate(o.id, speed, conduct, ""); onClose() },
                // **ولا يُرسَل نصفُ تقييم** — المحرّكُ يشترط الاثنين بين
                // واحدٍ وخمسة، **ونداءٌ يُردّ بأربعمئة لا يُفهم سببُه.**
                enabled = !s.busy && speed in 1..5 && conduct in 1..5,
            ) { Text(stringResource(R.string.hist_rate_send)) }
        },
        dismissButton = {
            TextButton(onClick = onClose) { Text(stringResource(R.string.hist_cancel)) }
        },
    )
}

/** **صفُّ نجومٍ يُضغط** — والنجومُ تُلمس لا تُكتب. */
@Composable
private fun StarPick(label: String, value: Int, onPick: (Int) -> Unit) {
    Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
    Spacer(Modifier.height(4.dp))
    androidx.compose.foundation.layout.Row {
        repeat(5) { i ->
            Icon(
                painter = painterResource(R.drawable.ic_star),
                contentDescription = null,
                tint = if (i < value) Rahal.colors.accent else Rahal.colors.inkMuted.copy(alpha = 0.30f),
                modifier = Modifier
                    .padding(end = 4.dp)
                    .clickable { onPick(i + 1) },
            )
        }
    }
}

/** **اسمُ السبب بالعربيّة** — والمجهولُ برمزه ليُبلَّغ عنه. */
@Composable
private fun reasonLabel(code: String): String = when (code) {
    "merchant_slow" -> stringResource(R.string.reason_merchant_slow)
    "merchant_refused" -> stringResource(R.string.reason_merchant_refused)
    "merchant_wrong_goods" -> stringResource(R.string.reason_merchant_wrong_goods)
    "merchant_conduct" -> stringResource(R.string.reason_merchant_conduct)
    "customer_absent" -> stringResource(R.string.reason_customer_absent)
    "customer_address" -> stringResource(R.string.reason_customer_address)
    "customer_refused" -> stringResource(R.string.reason_customer_refused)
    "customer_conduct" -> stringResource(R.string.reason_customer_conduct)
    "other" -> stringResource(R.string.reason_other)
    else -> code
}

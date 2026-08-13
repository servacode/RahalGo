package com.rahalgo.driver.wallet

import android.app.Application
import android.util.Log
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
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
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateGreen
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.Payout
import com.rahalgo.shared.model.WalletStatement
import com.rahalgo.shared.model.WalletTx
import com.rahalgo.shared.net.ApiClient
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محفظتُه — رصيدُه وحركاتُه وطلباتُ سحبه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن نبني المحفظة بنفس الويب، بنفس
 *  المحتويات وبنفس الموجود».)
 *
 * # ورقمٌ بلا حركاتٍ لا يُصدَّق
 *
 * **كان الرصيدُ رقماً في الشريط وسطراً في اللوحة** — ومن رآه نقص **لا
 * يعرف أين ذهب.** فيسأل المكتب، **والمكتبُ يفتح لوحتَه ليقرأ له ما كان
 * يستطيع قراءتَه بنفسه.**
 *
 * # والإشارةُ هي الخبر
 *
 * **موجبٌ له وسالبٌ عليه** — ولونان يقولانها قبل أن يُقرأ الرقم.
 *
 * # والسحبُ يُطلب ولا يقع
 *
 * **المالُ لا يخرج بضغطة**: يُطلب، **وتقرّره المالية.** ومن ظنّ أنّه
 * صُرف فذهب يشتري به **يقع في ما لا يُستدرَك.** فيُقال حالُ كلّ طلب
 * وقرارُه.
 */
data class WalletState(
    val statement: WalletStatement? = null,
    /** **اسمُ صاحب الكشف ورقمُه** — ورقةٌ بلا اسمٍ لا تُقدَّم حجّة. */
    val name: String = "",
    val phone: String = "",
    /** **اسمُ المنصّة ورقمُ دعمها** — لترويسة الورقة وذيلها. */
    val platform: String = "",
    val support: String = "",
    /**
     * **شعارُ المنصّة مدسوساً في الصفحة** (`data:` URI) — أو فارغ.
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٣: «كأنّه ما جاب لوغو المنصّة، جاب لوغو فيه
     *  حرف ر».)
     *
     * **ولا يُترك للعارض أن يُنزّله**: الطباعةُ تبدأ حين تنتهي الصفحة،
     * **والصورةُ قد تصل بعدها** فتُطبع ورقةٌ بلا علامة.
     */
    val logo: String = "",
    val payouts: List<Payout> = emptyList(),
    val busy: Boolean = false,
    val error: String = "",
    val done: String = "",
)

class WalletViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(WalletState())
        private set

    /** **مدى الكشف** — و«الكلّ» يعني بلا حدّين. */
    var range by mutableStateOf(Range.Month)
        private set

    private val backend = Backend.of(getApplication())

    /** **يبدّل المدى ويُعيد الجلب** — واسمٌ لا يصطدم بواضع الخاصّيّة. */
    fun pickRange(r: Range) {
        range = r
        load()
    }

    init {
        load()
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        viewModelScope.launch {
            state = try {
                val st = backend.me.wallet(rangeQuery())
                val po = runCatching { backend.me.payouts() }.getOrDefault(state.payouts)
                val me = runCatching { backend.account.summary() }.getOrNull()
                // **واسمُ المنصّة من إعداداتها لا من الشيفرة** — يبدّله
                // المالكُ من لوحته فتتبدّل الورقة.
                val plat = runCatching { backend.auth.platform() }.getOrNull()
                // **والشعارُ يُجلَب مرّةً ويبقى** — لا مع كلّ فتحة.
                val logo = if (state.logo.isEmpty() && !plat?.logo.isNullOrEmpty()) {
                    dataUri(Backend.media(plat.logo).orEmpty())
                } else {
                    state.logo
                }
                state.copy(
                    statement = st,
                    payouts = po,
                    name = me?.fullName ?: state.name,
                    phone = me?.phone ?: state.phone,
                    platform = plat?.name ?: state.platform,
                    support = plat?.supportPhone ?: state.support,
                    logo = logo,
                    error = "",
                )
            } catch (e: Exception) {
                state.copy(error = describe(e))
            }
        }
    }

    /**
     * **يطلب سحباً** — بمفتاحِ منع تكرارٍ يُولَّد مرّةً لكلّ محاولة.
     *
     * **وشبكةٌ تنقطع بعد الإرسال وقبل الردّ تجعل الإصبعَ يعيد الضغط** —
     * فيُحجَز المبلغُ مرّتين. **والمفتاحُ يجعل الثانيةَ لا شيء.**
     */
    fun requestPayout(amount: Long, note: String) {
        if (state.busy || amount <= 0) return
        state = state.copy(busy = true, error = "", done = "")
        val key = java.util.UUID.randomUUID().toString()
        viewModelScope.launch {
            state = try {
                backend.me.requestPayout(amount, note.trim(), key)
                val st = backend.me.wallet(rangeQuery())
                val po = runCatching { backend.me.payouts() }.getOrDefault(state.payouts)
                Refresh.bump()
                state.copy(
                    statement = st,
                    payouts = po,
                    busy = false,
                    done = getApplication<Application>().getString(R.string.wal_requested),
                )
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    /**
     * **حدّا المدى كما يقرؤهما المحرّك** (`?from=&to=`).
     *
     * **و«الكلّ» بلا حدّين** — والمحرّكُ يفتحهما فيردّ ما عنده حتّى
     * سقفه، **ويقول إن قصّ.**
     *
     * **والتقويمُ محلّيٌّ لا عالميّ**: من يسأل عن «هذا الشهر» يعني شهرَه
     * هو، **وشهرٌ يُحسب بتوقيت غرينتش يبدأ عنده قبل منتصف الليل بساعتين
     * أو بعده** — فتغيب حركةُ ليلةٍ أو تُزاد.
     */
    private fun rangeQuery(): String {
        if (range == Range.All) return ""
        val cal = java.util.Calendar.getInstance()
        if (range == Range.Prev) cal.add(java.util.Calendar.MONTH, -1)
        val f = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US)
        cal.set(java.util.Calendar.DAY_OF_MONTH, 1)
        val from = f.format(cal.time)
        cal.set(java.util.Calendar.DAY_OF_MONTH, cal.getActualMaximum(java.util.Calendar.DAY_OF_MONTH))
        return "?from=" + from + "&to=" + f.format(cal.time)
    }

    /**
     * **يجلب الشعارَ ويصيّره نصّاً يُدسّ في الصفحة.**
     *
     * **والنوعُ يُستنتَج من الامتداد** — `image/png` لصورةٍ شفّافةٍ
     * و`svg+xml` لرسمٍ متّجه. **ونوعٌ خاطئٌ لا يُعرض** في `WebView`.
     */
    private suspend fun dataUri(url: String): String {
        if (url.isEmpty()) return ""
        val raw = backend.api.bytes(url)
        if (raw.isEmpty()) return ""
        val mime = when {
            url.endsWith(".svg", true) -> "image/svg+xml"
            url.endsWith(".png", true) -> "image/png"
            url.endsWith(".webp", true) -> "image/webp"
            else -> "image/jpeg"
        }
        val b64 = android.util.Base64.encodeToString(raw, android.util.Base64.NO_WRAP)
        return "data:" + mime + ";base64," + b64
    }

    private fun describe(e: Exception): String {
        Log.e("RahalGo/محفظة", "فشل نداء المحفظة", e)
        val app = getApplication<Application>()
        return when {
            e is ApiClient.ApiException -> when (e.body.code) {
                "unauthorized", "invalid_refresh" -> app.getString(R.string.err_invalid_refresh)
                "insufficient_balance" -> app.getString(R.string.wal_not_enough)
                "payout_pending" -> app.getString(R.string.wal_pending_exists)
                "validation" -> app.getString(R.string.wal_bad_amount)
                "" -> app.getString(R.string.err_internal)
                else -> e.body.code
            }
            else -> app.getString(R.string.err_network)
        }
    }
}

@Composable
fun WalletScreen(vm: WalletViewModel) {
    val s = vm.state
    val st = s.statement
    if (st == null) {
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            if (s.error.isEmpty()) {
                CircularProgressIndicator()
            } else {
                Text(s.error, color = StateRed, textAlign = TextAlign.Center)
            }
        }
        return
    }

    var asking by rememberSaveable { mutableStateOf(false) }
    var statement by rememberSaveable { mutableStateOf(false) }

    if (statement) {
        StatementView(vm, st, onBack = { statement = false })
        return
    }

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        if (s.error.isNotEmpty()) Notice(s.error, StateRed)
        if (s.done.isNotEmpty()) Notice(s.done, StateGreen)

        // ══════════════════════════════════════════════════════════════
        // **الرصيدُ كرتٌ في وسط الشاشة — وزرّان تحته**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «المحفظة كرتٌ بنصّ الشاشة وتحتها
        //  الزرّان — طلب سحبٍ وكشف حساب».)
        //
        // **والرقمُ وحدَه في سطرٍ يُقرأ سطراً بين سطور** — والكرتُ يعزله
        // فيقع عليه البصرُ أوّلا، **وهو ما يفتح الشاشةَ لأجله.**
        Column(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(18.dp))
                .background(BrandTeal.copy(alpha = 0.08f))
                .padding(vertical = 22.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(stringResource(R.string.wal_balance), color = InkMuted)
            Spacer(Modifier.height(6.dp))
            Text(
                text = money(st.balance),
                color = BrandTeal,
                style = MaterialTheme.typography.headlineLarge,
            )
        }

        Spacer(Modifier.height(14.dp))
        if (asking) {
            PayoutForm(vm, s, max = st.balance, onDone = { asking = false })
        } else {
            // **والزرّان في صفٍّ متساويين** — لا واحدٌ فوق واحد: **فعلان
            // متكافئان يُقرآن متكافئين.**
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Button(
                    onClick = { asking = true },
                    enabled = !s.busy && st.balance > 0,
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.wal_ask_payout)) }
                OutlinedButton(
                    onClick = { statement = true },
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.wal_statement)) }
            }
        }

        if (s.payouts.isNotEmpty()) {
            Spacer(Modifier.height(20.dp))
            HorizontalDivider()
            Spacer(Modifier.height(14.dp))
            Text(
                stringResource(R.string.wal_payouts),
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.height(8.dp))
            s.payouts.forEach { PayoutRow(it) }
        }

        Spacer(Modifier.height(20.dp))
        HorizontalDivider()
        Spacer(Modifier.height(14.dp))
        Text(stringResource(R.string.wal_txs), style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(8.dp))
        if (st.transactions.isEmpty()) {
            Text(stringResource(R.string.wal_no_txs), color = InkMuted)
        }
        st.transactions.forEach { TxRow(it) }

        // **وناقصٌ يقول إنّه ناقص** — كشفٌ قُصّ عند السقف ولا يقول
        // **يُقرأ كاملا**، فيُجمع فلا يساوي الرصيد.
        if (st.truncated) {
            Spacer(Modifier.height(10.dp))
            Text(
                stringResource(R.string.wal_truncated),
                color = InkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(32.dp))
    }
}

@Composable
private fun PayoutForm(vm: WalletViewModel, s: WalletState, max: Long, onDone: () -> Unit) {
    var amount by remember { mutableStateOf("") }
    var note by remember { mutableStateOf("") }
    val value = amount.trim().toLongOrNull() ?: 0

    OutlinedTextField(
        value = amount,
        onValueChange = { amount = it.filter(Char::isDigit) },
        label = { Text(stringResource(R.string.wal_amount)) },
        singleLine = true,
        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
            keyboardType = androidx.compose.ui.text.input.KeyboardType.Number,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(6.dp))
    // **وسقفُه رصيدُه** — ويُقال قبل الإرسال: نداءٌ يذهب ليعود بخطأٍ
    // **يعرفه الجهازُ نفسُه** يُضيّع ثانيتين ويستهلك حزمة.
    Text(
        text = stringResource(R.string.wal_max, money(max)),
        color = if (value > max) StateRed else InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(8.dp))
    OutlinedTextField(
        value = note,
        onValueChange = { note = it },
        label = { Text(stringResource(R.string.wal_note)) },
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(10.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = { vm.requestPayout(value, note); onDone() },
            enabled = !s.busy && value in 1..max,
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.wal_send)) }
        OutlinedButton(onClick = onDone, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

@Composable
private fun PayoutRow(p: Payout) {
    val color = when (p.status) {
        "approved", "paid" -> StateGreen
        "rejected" -> StateRed
        else -> InkMuted
    }
    Column(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(money(p.amount), style = MaterialTheme.typography.titleSmall)
            Text(payoutStatus(p.status), color = color)
        }
        // **وقرارُ المالية يُقال** — من رُدّ طلبُه يسأل لماذا، **وصمتٌ
        // هنا يُرسله إلى المكتب ليسأل عمّا هو مكتوبٌ عندهم.**
        val why = p.decision.ifEmpty { p.note }
        if (why.isNotEmpty()) {
            Text(why, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
    }
}

@Composable
private fun payoutStatus(status: String): String = stringResource(
    when (status) {
        "approved" -> R.string.wal_st_approved
        "paid" -> R.string.wal_st_paid
        "rejected" -> R.string.wal_st_rejected
        else -> R.string.wal_st_pending
    },
)

@Composable
private fun TxRow(t: WalletTx) {
    val positive = t.amount >= 0
    Column(Modifier.fillMaxWidth().padding(vertical = 7.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            // **والإشارةُ قبل الرقم** — «‎+٣٠٠» تُقرأ ربحاً بنظرة.
            Text(
                text = (if (positive) "+" else "−") + money(kotlin.math.abs(t.amount)),
                color = if (positive) StateGreen else StateRed,
                style = MaterialTheme.typography.titleSmall,
            )
            t.orderNumber?.let {
                Text("#$it", color = InkMuted, style = MaterialTheme.typography.bodySmall)
            }
        }
        // ══════════════════════════════════════════════════════════════
        // **واسمُ الحركة بالعربيّة — لا مفتاحُ آلة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «والسحبُ يكون مكتوباً: تمّ سحب
        //  الرصيد، والتاريخ أيضا».)
        //
        // **والمحرّكُ يرسل `payout` و`driver_earning`** — وهي مفاتيحُ
        // آلةٍ لا تُعرض لعربيّ. **ومن قرأها ظنّ العطبَ في التطبيق.**
        //
        // **والملاحظةُ تحتها إن وُجدت** — الاسمُ يقول ما هي، **والملاحظةُ
        // تقول لماذا**: «تسوية من الإدارة» ثمّ «تصحيح نقص يوم الثلاثاء».
        Text(kindLabel(t.kind), style = MaterialTheme.typography.bodyMedium)
        if (t.note.isNotEmpty()) {
            Text(t.note, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
        // **والتاريخُ مع كلّ حركة** — بأمر المالك. **ومن رأى «‎−٥٠٠» لا
        // يعرف أهي اليومَ أم الشهرَ الماضي.**
        Text(
            text = fmtWhen(t.createdAt),
            color = InkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
    }
}

@Composable
private fun Notice(text: String, color: androidx.compose.ui.graphics.Color) {
    Text(
        text = text,
        color = color,
        textAlign = TextAlign.Center,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(color.copy(alpha = 0.08f))
            .padding(10.dp),
    )
    Spacer(Modifier.height(10.dp))
}

/** **اسمُ نوع الحركة بالعربيّة** — والمجهولُ يُعرض بمفتاحه ليُعرف. */
@Composable
private fun kindLabel(kind: String): String = when (kind) {
    "payout" -> stringResource(R.string.wal_k_payout)
    "driver_earning" -> stringResource(R.string.wal_k_driver_earning)
    "commission" -> stringResource(R.string.wal_k_commission)
    "compensation" -> stringResource(R.string.wal_k_compensation)
    "penalty" -> stringResource(R.string.wal_k_penalty)
    "refund" -> stringResource(R.string.wal_k_refund)
    "adjustment" -> stringResource(R.string.wal_k_adjustment)
    "reward" -> stringResource(R.string.wal_k_reward)
    "settlement" -> stringResource(R.string.wal_k_settlement)
    // **ونوعٌ لم يُترجَم يُعرض بمفتاحه** — لا يُبتلع: **من رآه أبلغ
    // عنه**، ومن ابتلعه ترك سطراً بلا اسمٍ في كشف مال.
    else -> kind
}

/**
 * **تاريخُ الحركة ووقتُها** — كما يقرؤه صاحبُها.
 *
 * **والمحرّكُ يرسله بصيغة ISO** (`2026-08-13T03:12:44+03:00`) — وهي
 * صيغةُ آلةٍ لا تُعرض. **وتُقصّ بلا تحويل مناطق**: الطابعُ يحمل إزاحةَ
 * دمشقَ أصلا، **وتحويلٌ ثانٍ يزيحها ساعتين.**
 */
private fun fmtWhen(iso: String): String {
    if (iso.length < 16) return iso
    val date = iso.substring(0, 10)
    val time = iso.substring(11, 16)
    return date + " · " + time
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كشفُ الحساب — بمدىً ورصيدَي طرفيه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «طلب سحبٍ · كشف حساب».)
 *
 * **ولا معنى لكشفٍ بلا افتتاحيٍّ وختاميّ**: من يبدأ من منتصف التاريخ
 * **يجمع الأسطر فلا تساوي رصيدَه** فيظنّ الخللَ في المنصّة.
 *
 * **والمدى ثلاثةُ أزرارٍ لا منتقي تاريخٍ** — سائقٌ يقف في الشارع لا
 * يفتح تقويماً، **وأكثرُ ما يُسأل عنه شهرٌ مضى أو هذا الشهر.**
 */
@Composable
private fun StatementView(vm: WalletViewModel, st: WalletStatement, onBack: () -> Unit) {
    val context = androidx.compose.ui.platform.LocalContext.current
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                stringResource(R.string.wal_statement),
                style = MaterialTheme.typography.titleMedium,
            )
            val period = periodLabel(vm.range)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // **والطباعةُ هي ما يجعل الكشفَ حجّة** — (سأل المالك
                // ٢٠٢٦-٠٨-١٣: «شو استفدنا من الكشف ما فيه طباعة؟»).
                // **ونظامُ أندرويد يعطي الطابعةَ و«حفظ كـPDF» معا.**
                Button(onClick = {
                    StatementPrint.print(
                        context,
                        vm.state.name,
                        vm.state.phone,
                        st,
                        period,
                        vm.state.platform,
                        vm.state.support,
                        vm.state.logo,
                    )
                }) { Text(stringResource(R.string.wal_print)) }
                OutlinedButton(onClick = onBack) { Text(stringResource(R.string.wal_back)) }
            }
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            for (r in Range.entries) {
                val on = vm.range == r
                OutlinedButton(
                    onClick = { vm.pickRange(r) },
                    modifier = Modifier.weight(1f),
                ) {
                    Text(
                        stringResource(r.label),
                        color = if (on) BrandTeal else InkMuted,
                    )
                }
            }
        }

        Spacer(Modifier.height(14.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.wal_opening), color = InkMuted)
            Text(money(st.opening))
        }
        Spacer(Modifier.height(6.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.wal_closing), color = InkMuted)
            Text(money(st.closing), color = BrandTeal)
        }

        Spacer(Modifier.height(14.dp))
        HorizontalDivider()
        Spacer(Modifier.height(10.dp))
        if (st.transactions.isEmpty()) {
            Text(stringResource(R.string.wal_no_txs), color = InkMuted)
        }
        st.transactions.forEach { TxRow(it) }
        if (st.truncated) {
            Spacer(Modifier.height(10.dp))
            Text(
                stringResource(R.string.wal_truncated),
                color = InkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(32.dp))
    }
}

/** **مدياتُ الكشف** — ثلاثةٌ تكفي من يقف في الشارع. */
enum class Range(val label: Int) {
    Month(R.string.wal_this_month),
    Prev(R.string.wal_prev_month),
    All(R.string.wal_all),
}

/** **اسمُ المدى كما يُكتب في الورقة.** */
@Composable
private fun periodLabel(r: Range): String = stringResource(r.label)

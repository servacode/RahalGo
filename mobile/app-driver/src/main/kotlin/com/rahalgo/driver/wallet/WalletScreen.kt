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
    val payouts: List<Payout> = emptyList(),
    val busy: Boolean = false,
    val error: String = "",
    val done: String = "",
)

class WalletViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(WalletState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        load()
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        viewModelScope.launch {
            state = try {
                val st = backend.me.wallet()
                val po = runCatching { backend.me.payouts() }.getOrDefault(state.payouts)
                state.copy(statement = st, payouts = po, error = "")
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
                val st = backend.me.wallet()
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

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        if (s.error.isNotEmpty()) Notice(s.error, StateRed)
        if (s.done.isNotEmpty()) Notice(s.done, StateGreen)

        // ══════════════════════════════════════════════════════════════
        // **الرصيدُ أوّلا — وهو ما يفتح الشاشةَ لأجله**
        // ══════════════════════════════════════════════════════════════
        Text(stringResource(R.string.wal_balance), color = InkMuted)
        Spacer(Modifier.height(4.dp))
        Text(
            text = money(st.balance),
            color = BrandTeal,
            style = MaterialTheme.typography.headlineMedium,
        )

        Spacer(Modifier.height(16.dp))
        if (!asking) {
            OutlinedButton(
                onClick = { asking = true },
                enabled = !s.busy && st.balance > 0,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.wal_ask_payout)) }
        } else {
            PayoutForm(vm, s, max = st.balance, onDone = { asking = false })
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
        // **وسببُها** — «‎+٣٠٠» وحدَها لا تقول شيئا.
        val why = t.note.ifEmpty { t.kind }
        if (why.isNotEmpty()) {
            Text(why, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
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

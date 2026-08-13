package com.rahalgo.driver.history

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
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateGreen
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.HistoryOrder
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
)

class HistoryViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(HistoryState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        load()
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
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
            if (s.error.isNotEmpty()) {
                Text(s.error, color = StateRed, textAlign = TextAlign.Center)
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
                    color = StateGreen,
                    modifier = Modifier.weight(1f),
                )
                Tally(
                    label = stringResource(R.string.hist_failed),
                    value = failed.toString(),
                    color = if (failed > 0) StateRed else InkMuted,
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
                    color = InkMuted,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        items(shown, key = { it.id }) { o -> Row(o) }
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
            .background(InkMuted.copy(alpha = 0.07f))
            .padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.height(4.dp))
        Text(value, color = color, style = MaterialTheme.typography.headlineSmall)
    }
}

@Composable
private fun Row(o: HistoryOrder) {
    val delivered = o.status == "delivered"
    Column(Modifier.fillMaxWidth().padding(vertical = 9.dp)) {
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("#${o.number}", style = MaterialTheme.typography.titleSmall, color = BrandTeal)
            // **والحالُ كلمةٌ ملوّنة** — تُقرأ قبل أن يُقرأ السطر.
            Text(
                text = stringResource(
                    if (delivered) R.string.hist_st_delivered else R.string.hist_st_failed,
                ),
                color = if (delivered) StateGreen else StateRed,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        if (o.merchantName.isNotEmpty()) {
            Text(o.merchantName, style = MaterialTheme.typography.bodyMedium)
        }
        // **وسببُ التعذّر يُقال** — من فشل طلبُه يُسأل عنه بعد أيّام،
        // **وسجلٌّ يقول «تعذّر» بلا سببٍ لا يُدافَع به.**
        if (!delivered && o.failReason.isNotEmpty()) {
            Text(o.failReason, color = StateRed, style = MaterialTheme.typography.bodySmall)
        }
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = o.createdAt.take(10),
                color = InkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            // **وقيمةُ الطلب لا أجرتُه** — الأجرةُ في المحفظة مجموعةً،
            // **وهذه تقول ما حمله** فيتذكّره.
            Text(
                text = money(o.total),
                color = InkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(9.dp))
        HorizontalDivider()
    }
}

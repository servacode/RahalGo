package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.shared.model.IncentiveEntry

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أهدافي والمكافآت**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «ليس الهدايا والمكافآت إنّما أهدافي
 *  والمكافآت، لأنّ كلَّ سائقٍ لديه أهدافٌ إذا حقّقها يأخذ مكافأة».)
 *
 * # وحافزٌ لا يُرى لا يحفّز
 *
 * **من لا يعرف أنّه على بُعد ثلاثةِ طلباتٍ من مكافأةٍ لا يسعى إليها.**
 * والمحفظةُ دفترٌ يقول «‎+٥٠٬٠٠٠ في الثالث من الشهر» — **ولا تقول لماذا،
 * ولا كم بقي، ولا أنّ ثمّة هدفاً أصلا.**
 *
 * # والعقوبةُ تُعرض كما تُعرض المكافأة
 *
 * **ومن عوقب ولا يعلم لا يُصلح شيئاً** — يُخصم منه فيُفاجأ، **ويظنّ
 * الظلمَ حيث كان خبر.**
 *
 * # وشريطٌ لا يظهر بلا هدف
 *
 * **صفرٌ يعني «لا هدفَ مضبوط»** — وشريطٌ ممتلئٌ على هدفٍ صفرٍ يُقرأ
 * إنجازاً، **فيُهنّئ من لم يُطلب منه شيء.**
 */
@Composable
fun IncentivesScreen(vm: IncentivesViewModel) {
    val data = vm.data
    if (data == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }
    val st = data.standing

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_rewards),
            stringResource(R.string.inc_hint),
        )

        if (st.target > 0) {
            Card {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        stringResource(R.string.inc_target),
                        fontWeight = FontWeight.Bold,
                    )
                    if (st.reached) {
                        Chip(stringResource(R.string.inc_reached), Rahal.colors.success)
                    } else {
                        Text(
                            text = stringResource(R.string.inc_left, (st.target - st.done).toString()),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
                Spacer(Modifier.height(10.dp))
                Bar(
                    ratio = st.done.toFloat() / st.target,
                    color = if (st.reached) Rahal.colors.success else Rahal.colors.brand,
                )
                Spacer(Modifier.height(6.dp))
                Text(
                    text = st.done.toString() + " / " + st.target.toString(),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )

                // **والجائزةُ تُقال قبل أن تُنال** — (شكوى المالك
                // ٢٠٢٦-٠٨-٠٩: «لازم يعرف شو المكافأة الي رح يحصل عليها
                // وقت يحقّق هدفه»).
                //
                // **وشاشةٌ تقول «٣ من ٥٠» ولا تقول ماذا بعدها تطلب جهداً
                // بلا وعد.** وصفرٌ يُخفيها — **لا يُعرَض وعدٌ بلا مبلغ.**
                if (data.targetReward > 0) {
                    Spacer(Modifier.height(10.dp))
                    Text(
                        text = if (st.reached) {
                            stringResource(R.string.inc_reward_got, money(data.targetReward))
                        } else {
                            stringResource(R.string.inc_reward_promise, money(data.targetReward))
                        },
                        color = if (st.reached) Rahal.colors.success else Rahal.colors.accent,
                        fontWeight = FontWeight.Medium,
                    )
                }
            }
        }

        Spacer(Modifier.height(14.dp))
        Card(tone = Rahal.colors.inkMuted) {
            KeyValue(stringResource(R.string.inc_done), st.done.toString())
            KeyValue(
                stringResource(R.string.inc_rewarded),
                money(st.rewarded),
                valueColor = Rahal.colors.success,
            )
            KeyValue(
                stringResource(R.string.inc_penalized),
                money(st.penalized),
                valueColor = if (st.penalized > 0) Rahal.colors.danger else Rahal.colors.inkMuted,
            )
        }

        SectionTitle(stringResource(R.string.inc_entries))
        if (data.entries.isEmpty()) {
            Empty(stringResource(R.string.inc_empty))
            return@Screen
        }
        data.entries.forEach { EntryRow(it) }
    }
}

@Composable
private fun EntryRow(e: IncentiveEntry) {
    val reward = e.kind == "reward"
    val color = if (reward) Rahal.colors.success else Rahal.colors.danger
    Column(Modifier.fillMaxWidth().padding(vertical = 7.dp)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Chip(
                text = stringResource(if (reward) R.string.inc_reward else R.string.inc_penalty),
                color = color,
            )
            Text(
                text = (if (reward) "+" else "−") + money(kotlin.math.abs(e.amount)),
                color = color,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
        }
        // **والسببُ إلزاميٌّ في المحرّك** — فلا سطرَ هنا بلا كلمة.
        if (e.reason.isNotEmpty()) {
            Spacer(Modifier.height(4.dp))
            Text(e.reason, style = MaterialTheme.typography.bodyMedium)
        }
        Text(whenText(e.createdAt), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
    }
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ الأهداف — والبابُ يُعطى من خارج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السائقُ يقرأ `me/incentives` والمندوبُ `rep/incentives`** — **وشاشةٌ
 * تعرف بابَها لا تُستعمل في تطبيقين.**
 *
 * **ولا يُنادى إلّا حين يُفتح القسم** — **ونداءٌ عند الإقلاع لقسمٍ لا
 * يُفتح يستنزف حزمةَ من في الشارع.**
 */
class IncentivesViewModel(
    app: android.app.Application,
    private val fetch: suspend () -> com.rahalgo.shared.model.IncentivesPayload,
) : androidx.lifecycle.AndroidViewModel(app) {

    var data by androidx.compose.runtime.mutableStateOf<com.rahalgo.shared.model.IncentivesPayload?>(null)
        private set

    var busy by androidx.compose.runtime.mutableStateOf(false)
        private set

    var error by androidx.compose.runtime.mutableStateOf("")
        private set

    fun load(force: Boolean = false) {
        if (!force && data != null) return
        busy = true
        viewModelScope.launch {
            try {
                data = fetch()
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}

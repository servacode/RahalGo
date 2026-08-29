package com.rahalgo.rep.board

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
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
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.shared.rep.RepMe
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Bar
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.apiError
import com.rahalgo.ui.money
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ المندوب — الشهريُّ أوّلاً ثمّ التراكميّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا الشهريُّ يسبق
 *
 * **رقمٌ إجماليٌّ وحدَه لا يقول أيَعمل هذا الشهرَ أم يعيش على ما مضى** —
 * **ومن جمع مئةَ متجرٍ في سنةٍ ولم يُسجّل واحداً هذا الشهرَ يرى «١٠٠»
 * فيطمئنّ وهو لا يعمل.**
 *
 * **وما يُسأل عنه يوميّاً يسبق ما يُسأل عنه مرّة** (وهو ترتيبُ الويب).
 *
 * # والهدفُ بشريطٍ لا برقم
 *
 * **«٣ من ٥» رقمان يُقرآن** — **والشريطُ يُقرأ بلمحةٍ قبل أن يُقرأ
 * الرقم.** ومن يفتح لوحتَه صباحاً يريد جواباً في ثانية.
 *
 * # وما ينتظر البتّ يُقال هنا
 *
 * **ومن سجّل ثلاثةً ولم يُقبلوا يظنّ أنّ عملَه ضاع** — **والرقمُ يقول
 * إنّه محفوظٌ ينتظر.**
 */
@Composable
fun BoardScreen(vm: BoardViewModel) {
    val me = vm.me
    if (me == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.act_my_board),
            stringResource(R.string.soon_board),
        )

        // ══════════════════════════════════════════════════════════════
        // **هدفُ الشهر — أوّلُ ما يُرى**
        // ══════════════════════════════════════════════════════════════
        val target = maxOf(1, me.monthlyTarget)
        val done = me.monthMerchants
        val reached = done >= target
        Spacer(Modifier.height(12.dp))
        Card {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    stringResource(R.string.act_month_target),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    text = "$done / $target",
                    color = if (reached) Rahal.colors.success else Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
            }
            Spacer(Modifier.height(8.dp))
            Bar(
                ratio = done.toFloat() / target.toFloat(),
                color = if (reached) Rahal.colors.success else Rahal.colors.brand,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = if (reached) {
                    stringResource(R.string.bd_target_done)
                } else {
                    stringResource(R.string.bd_target_left, (target - done).toString())
                },
                color = if (reached) Rahal.colors.success else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **وشهرُه بعده** — ما عمله هذا الشهر
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(14.dp))
        SectionTitle(stringResource(R.string.bd_this_month))
        Card {
            KeyValue(stringResource(R.string.bd_month_clients), me.monthMerchants.toString())
            KeyValue(stringResource(R.string.bd_month_delivered), me.monthDelivered.toString())
            KeyValue(
                stringResource(R.string.bd_month_commissions),
                money(me.monthCommissions),
                valueColor = Rahal.colors.brand,
            )
        }

        // **وما ينتظر البتّ** — ولا يُعرض صفراً: **رقمٌ صفريٌّ بجانب
        // «بانتظار الموافقة» يُقرأ رفضا.**
        if (me.pendingLeads > 0) {
            Spacer(Modifier.height(10.dp))
            Card {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        stringResource(R.string.bd_pending),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Chip(me.pendingLeads.toString(), Rahal.colors.accent)
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والتراكميُّ آخرا** — ما جمعه منذ بدأ
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(14.dp))
        SectionTitle(stringResource(R.string.bd_all_time))
        Card {
            KeyValue(stringResource(R.string.bd_clients), me.merchants.toString())
            KeyValue(stringResource(R.string.bd_delivered), me.deliveredOrders.toString())
            KeyValue(stringResource(R.string.bd_commissions), money(me.totalCommissions))
            Spacer(Modifier.height(6.dp))
            HorizontalDivider()
            Spacer(Modifier.height(6.dp))
            KeyValue(
                stringResource(R.string.bd_balance),
                money(me.balance),
                valueColor = Rahal.colors.brand,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **ورمزُه مقفلٌ حتّى يوثّق واتساب**
        // ══════════════════════════════════════════════════════════════
        //
        // **والمتجرُ الذي يُسجّل برمزه يُنسب إليه** — **ومن لا قناةَ
        // تواصلٍ موثّقةً له لا يُتحقّق أنّه هو.**
        Spacer(Modifier.height(14.dp))
        Card {
            Text(
                stringResource(R.string.bd_code),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(Modifier.height(4.dp))
            if (me.whatsappVerified && !me.inviteCode.isNullOrEmpty()) {
                Text(
                    text = me.inviteCode.orEmpty(),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineSmall,
                )
            } else {
                Text(
                    text = stringResource(R.string.bd_code_locked),
                    color = Rahal.colors.danger,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
        Spacer(Modifier.height(28.dp))
    }
}

class BoardViewModel(app: Application) : AndroidViewModel(app) {

    private val api = RepApi(AppCore.get().api)

    var me by mutableStateOf<RepMe?>(null)
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    init {
        load()
        // **وما يتبدّل يُقرأ** — عميلٌ يُقبل فيتحرّك الهدف.
        viewModelScope.launch { Refresh.tick.drop(1).collect { load(force = true) } }
    }

    fun load(force: Boolean = false) {
        if (!force && me != null) return
        busy = true
        viewModelScope.launch {
            try {
                me = api.me()
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}

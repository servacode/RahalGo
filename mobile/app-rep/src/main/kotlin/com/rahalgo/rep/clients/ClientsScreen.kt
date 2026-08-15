package com.rahalgo.rep.clients

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.Lead
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.ui.Bar
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عملائي — من سجّلهم وما ناله عنهم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «ما في شي اسمه متاجري، اسمه عملائي».)
 *
 * # والمعلَّقون فوق
 *
 * **هم ما ينتظر عملا** — **والمقبولون يعملون وحدَهم.** ومن سجّل عميلاً
 * صباحاً يفتح الشاشةَ ليرى أقُبل، **لا ليقرأ عمولةَ متجرٍ سجّله الشهرَ
 * الماضي.**
 *
 * # والتفعيلُ يُقال بعددين لا بكلمة
 *
 * **«قيد التفعيل» وحدَها لا تقول كم بقي** — **ومن لا يعرف كم بقي لا
 * يعرف أيلاحق صاحبَ المتجر أم ينتظر.**
 *
 * **والعمولةُ لا تجري حتّى يبلغ العددَ** (`sales.activation_orders`) —
 * **فرقمٌ ظاهرٌ خيرٌ من وعدٍ مبهم.**
 */
@Composable
fun ClientsScreen(vm: ClientsViewModel) {
    // **والتفصيلُ يغطّي القائمةَ** — **ولا صفحةٌ ثانيةٌ يخرج إليها
    // فيعود فلا يجد موضعَه.**
    if (vm.openId.isNotEmpty()) {
        BackHandler { vm.closeDetail() }
        ClientDetail(vm)
        return
    }

    val list = vm.merchants
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load() }
        return
    }
    val context = LocalContext.current

    Screen {
        ScreenTitle(
            stringResource(R.string.nav_clients),
            stringResource(R.string.soon_clients),
        )

        // ══════════════════════════════════════════════════════════════
        // **والمعلَّقون أوّلا** — ما ينتظر عملا
        // ══════════════════════════════════════════════════════════════
        if (vm.leads.isNotEmpty()) {
            SectionTitle(stringResource(R.string.cl_pending, vm.leads.size.toString()))
            vm.leads.forEach { LeadCard(it) }
            Spacer(Modifier.height(14.dp))
            HorizontalDivider()
        }

        if (list.isEmpty() && vm.leads.isEmpty()) {
            Empty(stringResource(R.string.cl_none))
            return@Screen
        }

        if (list.isNotEmpty()) {
            SectionTitle(stringResource(R.string.cl_active, list.size.toString()))
            list.forEach { m ->
                MerchantCard(
                    m = m,
                    media = { path -> Backend.of(context).media(path) },
                    onOpen = { vm.openDetail(m.id) },
                )
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **عميلٌ ينتظر البتّ.**
 *
 * **ولا عمولةَ فيه ولا طلبات** — **ورقمٌ صفريٌّ بجانب اسمٍ يُقرأ فشلا**،
 * وهو لم يبدأ بعد.
 */
@Composable
private fun LeadCard(lead: Lead) {
    Spacer(Modifier.height(8.dp))
    Card {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = lead.storeName,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            Chip(
                stringResource(
                    if (lead.status == "rejected") R.string.cl_rejected else R.string.cl_waiting,
                ),
                if (lead.status == "rejected") Rahal.colors.danger else Rahal.colors.accent,
            )
        }
        if (lead.ownerName.isNotEmpty() || lead.phone.isNotEmpty()) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = listOf(lead.ownerName, lead.phone, lead.area)
                    .filter { it.isNotEmpty() }
                    .joinToString(" · "),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // **وسببُ الرفض يُقال** — **ومن رُفض عميلُه بلا سببٍ يُعيد
        // تسجيلَه بالخطأ نفسِه.**
        if (lead.note.isNotEmpty()) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = lead.note,
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/** **عميلٌ مقبولٌ يعمل** — بعمولته وحال تفعيله. */
@Composable
private fun MerchantCard(
    m: RepMerchant,
    media: (String?) -> String?,
    onOpen: () -> Unit,
) {
    Spacer(Modifier.height(10.dp))
    // **وضغطُ البطاقة يفتح تفصيلَه** — (طلبُ المالك
    // ٢٠٢٦-٠٨-١٤: «إذا ضغطتُ على اسم العميل يجب أن أعرف كلَّ
    // التفاصيل التي تخصّني كمندوب»).
    Card(modifier = Modifier.clickable(onClick = onOpen)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(m.logoThumbUrl),
                name = m.name,
                modifier = Modifier
                    .size(46.dp)
                    .clip(RoundedCornerShape(12.dp)),
            )
            Spacer(Modifier.size(10.dp))
            Column(Modifier.fillMaxWidth(0.72f)) {
                Text(
                    text = m.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (m.categoryName.isNotEmpty()) {
                    Text(
                        text = m.categoryName,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
            Spacer(Modifier.fillMaxWidth(1f))
        }

        Spacer(Modifier.height(8.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Chip(
                stringResource(
                    if (m.status == "active") R.string.cl_working else R.string.cl_stopped,
                ),
                if (m.status == "active") Rahal.colors.success else Rahal.colors.inkMuted,
            )
        }

        Spacer(Modifier.height(8.dp))
        HorizontalDivider()
        Spacer(Modifier.height(8.dp))
        KeyValue(stringResource(R.string.cl_delivered), m.deliveredOrders.toString())
        KeyValue(
            stringResource(R.string.cl_commission),
            money(m.myCommission),
            valueColor = Rahal.colors.brand,
        )

        // ══════════════════════════════════════════════════════════════
        // **وشريطُ التفعيل حين لا يزال دونه**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يُعرض لمن فعّل** — شريطٌ ممتلئٌ دائماً يُقرأ زينةً لا خبرا.
        val needed = m.activationNeeded.toInt()
        if (needed > 1 && m.activationDone < needed) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(
                    R.string.cl_activation,
                    m.activationDone.toString(),
                    needed.toString(),
                ),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            Spacer(Modifier.height(4.dp))
            Bar(
                ratio = m.activationDone.toFloat() / needed.toFloat(),
                color = Rahal.colors.accent,
            )
        }
    }
}

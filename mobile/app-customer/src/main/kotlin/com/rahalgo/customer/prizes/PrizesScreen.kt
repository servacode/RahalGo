package com.rahalgo.customer.prizes

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.customer.Backend
import com.rahalgo.customer.InviteNudge
import com.rahalgo.customer.R
import com.rahalgo.customer.cart.Cart
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.PrizeItem
import com.rahalgo.shared.customer.PrizesMe
import com.rahalgo.shared.model.Item
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.EmptyState
import com.rahalgo.ui.ErrorState
import com.rahalgo.ui.Flash
import com.rahalgo.ui.IconBox
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.RahalSmallButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.SectionHeader
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.Tone
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائز رحّال غو** (قرارُ المالك ٢٠٢٦-١١-٢٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **قبل الافتتاح لافتةٌ** («انتظرونا ١/١٢/٢٠٢٦») تشرح الفكرة. **وبعده** تقدّمُه ودعوتُه
 * وسلّمُ الجوائز ورصيدُه وأصنافُ الجوائز: **«على قدّ جائزتك» ثمّ «بتدفع الفرق»**، **وبلا رصيدٍ
 * تُعرض الأصنافُ فقط** ولا إضافة. **والطلبُ من السلّة العاديّة بمعرّف متجر الجوائز** — والمحرّكُ
 * يطبّق الرصيدَ والتوصيلَ المجانيّ.
 */
@Composable
fun PrizesScreen(vm: PrizesViewModel) {
    LaunchedEffect(Unit) { vm.load() }
    val me = vm.me
    Refreshable(refreshing = vm.loading && me != null, onRefresh = vm::load) {
        when {
            me == null && vm.error.isNotEmpty() ->
                ErrorState(
                    title = stringResource(R.string.prizes_load_failed),
                    line = vm.error,
                    onRetry = vm::load,
                )
            me == null -> LoadState(vm.loading, vm.error) { vm.load() }
            vm.teaser -> Teaser(me)
            else -> Active(vm, me)
        }
    }
}

@Composable
private fun Teaser(me: PrizesMe) {
    LazyColumn(
        Modifier.fillMaxWidth(),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(Rahal.space.lg),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        item {
            Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                IconBox(R.drawable.ic_gift, Rahal.colors.accent, size = Rahal.size.avatarLg)
                Spacer(Modifier.height(Rahal.space.md))
                Text(
                    stringResource(R.string.prizes_title),
                    color = Rahal.colors.ink,
                    style = Rahal.type.headline,
                    textAlign = TextAlign.Center,
                )
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    stringResource(
                        R.string.prizes_teaser_wait,
                        PrizesLogic.launchLabel(me.launchOn.ifBlank { "2026-12-01" }),
                    ),
                    color = Rahal.colors.accentInk,
                    style = Rahal.type.sectionTitle,
                    textAlign = TextAlign.Center,
                )
            }
        }
        item {
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Warning, modifier = Modifier.fillMaxWidth()) {
                for (line in listOf(
                    R.string.prizes_teaser_how,
                    R.string.prizes_teaser_winners,
                    R.string.prizes_teaser_use,
                    R.string.prizes_teaser_free,
                )) {
                    Text(
                        stringResource(line),
                        color = Rahal.colors.ink,
                        style = Rahal.type.body,
                        modifier = Modifier.padding(vertical = Rahal.space.xs),
                    )
                }
            }
        }
        item { Ladder(me.amounts) }
    }
}

@Composable
private fun Ladder(amounts: List<Long>) {
    if (amounts.isEmpty()) return
    RahalCard(modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.prizes_ladder_title), color = Rahal.colors.ink, style = Rahal.type.sectionTitle)
        Spacer(Modifier.height(Rahal.space.sm))
        for ((rank, amount) in PrizesLogic.ladder(amounts)) {
            Text(
                stringResource(R.string.prizes_ladder_row, rank, money(amount)),
                color = Rahal.colors.ink,
                style = Rahal.type.body,
                modifier = Modifier.padding(vertical = Rahal.space.xxs),
            )
        }
    }
}

@Composable
private fun Active(vm: PrizesViewModel, me: PrizesMe) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    // **رمزُ الدعوة نفسُه المستعمَل في «ادعُ صديقاً»** — يُجلب إن لم يكن قد جاء.
    LaunchedEffect(Unit) { if (InviteNudge.referral == null) InviteNudge.refresh(scope) }

    val progress = PrizesLogic.progress(me)
    val split = PrizesLogic.split(vm.items, me.prizeBalance)
    val media = { path: String? -> Backend.of(context).media(path) }
    val storeId = vm.storeId
    val title = stringResource(R.string.prizes_title)
    val mixed = stringResource(R.string.prizes_cart_mixed)
    val noStore = stringResource(R.string.prizes_no_store)

    val order: (PrizeItem) -> Unit = add@{ p ->
        if (storeId.isNullOrBlank()) {
            Flash.ok(noStore)
            return@add
        }
        // **ولا يُخلط صنفُ جائزةٍ بمتجرٍ آخر** — الرصيدُ والتوصيلُ المجانيّ لمتجر الجوائز وحدَه.
        if (Cart.lines.any { it.item.merchantId != storeId }) {
            Flash.ok(mixed)
            return@add
        }
        val inc = Cart.add(p.toItem(storeId, title))
        Flash.ok(
            context.getString(
                if (inc) com.rahalgo.ui.R.string.cart_qty_updated else com.rahalgo.ui.R.string.cart_item_added,
            ),
        )
    }

    LazyColumn(
        Modifier.fillMaxWidth(),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(Rahal.space.lg),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        item {
            if (vm.error.isNotEmpty()) {
                com.rahalgo.ui.Note(vm.error, Rahal.colors.danger)
            }
        }
        item { ProgressCard(me, progress) }
        item {
            val r = InviteNudge.referral
            RahalButton(
                onClick = { if (r != null) InviteNudge.share(context, r) },
                enabled = r != null && r.code.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.prizes_share)) }
        }
        item { Ladder(me.amounts) }
        if (me.prizeBalance > 0) item { BalanceCard(me) }

        when {
            vm.items.isEmpty() -> item {
                EmptyState(
                    title = stringResource(R.string.prizes_items_empty),
                    line = stringResource(R.string.prizes_items_empty_hint),
                    icon = R.drawable.ic_gift,
                )
            }
            split.showcase -> {
                item { SectionHeader(stringResource(R.string.prizes_items_showcase)) }
                item {
                    Text(
                        stringResource(R.string.prizes_showcase_hint),
                        color = Rahal.colors.accentInk,
                        style = Rahal.type.bodyStrong,
                    )
                }
                items(split.over, key = { "s-" + it.id }) { p ->
                    PrizeRow(p, me.prizeBalance, media, enabled = false, covered = false, onAdd = {})
                }
            }
            else -> {
                if (split.within.isNotEmpty()) {
                    item { SectionHeader(stringResource(R.string.prizes_items_within)) }
                    items(split.within, key = { "w-" + it.id }) { p ->
                        PrizeRow(p, me.prizeBalance, media, enabled = true, covered = true, onAdd = { order(p) })
                    }
                }
                if (split.over.isNotEmpty()) {
                    item { SectionHeader(stringResource(R.string.prizes_items_over)) }
                    items(split.over, key = { "o-" + it.id }) { p ->
                        PrizeRow(p, me.prizeBalance, media, enabled = true, covered = false, onAdd = { order(p) })
                    }
                }
            }
        }

        // **ورابحو الشهر الماضي — تُخفى إن لم يكن فيها أحد.**
        if (vm.winners.isNotEmpty()) {
            item { SectionHeader(stringResource(R.string.prizes_winners_title)) }
            item {
                RahalCard(modifier = Modifier.fillMaxWidth()) {
                    for (w in vm.winners.sortedBy { it.rank }) {
                        Text(
                            stringResource(R.string.prizes_winner_row, w.rank, w.name, money(w.amount)),
                            color = Rahal.colors.ink,
                            style = Rahal.type.body,
                            modifier = Modifier.padding(vertical = Rahal.space.xxs),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ProgressCard(me: PrizesMe, p: PrizesLogic.Progress) {
    RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Warning, modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.prizes_progress_title), color = Rahal.colors.ink, style = Rahal.type.sectionTitle)
        Spacer(Modifier.height(Rahal.space.sm))
        LinearProgressIndicator(
            progress = { p.current.toFloat() / p.perEntry },
            color = Rahal.colors.accent,
            trackColor = Rahal.colors.line,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Rahal.space.sm))
        Text(
            stringResource(R.string.prizes_progress_line, p.current, p.perEntry),
            color = Rahal.colors.ink,
            style = Rahal.type.body,
        )
        Text(
            stringResource(R.string.prizes_entries, me.entries),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.caption,
        )
        Text(
            stringResource(R.string.prizes_to_next, p.toNext),
            color = Rahal.colors.accentInk,
            style = Rahal.type.bodyStrong,
        )
        val draw = PrizesLogic.drawLabel(me.drawAt)
        if (draw.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(R.string.prizes_draw, draw),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
    }
}

@Composable
private fun BalanceCard(me: PrizesMe) {
    RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success, modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.prizes_balance_title), color = Rahal.colors.inkMuted, style = Rahal.type.label)
        Text(money(me.prizeBalance), color = Rahal.colors.ink, style = Rahal.type.headline)
        val until = PrizesLogic.expiryLabel(me.prizeExpiresAt)
        if (until.isNotEmpty()) {
            Text(
                stringResource(R.string.prizes_balance_expiry, until),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
    }
}

@Composable
private fun PrizeRow(
    item: PrizeItem,
    balance: Long,
    media: (String?) -> String?,
    enabled: Boolean,
    covered: Boolean,
    onAdd: () -> Unit,
) {
    RahalCard(modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(item.thumbUrl ?: item.imageUrl),
                name = item.name,
                modifier = Modifier.size(Rahal.size.thumb),
            )
            Spacer(Modifier.size(Rahal.space.md))
            Column(Modifier.weight(1f)) {
                Text(
                    item.name,
                    color = Rahal.colors.ink,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.body,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(money(item.price), color = Rahal.colors.inkMuted, style = Rahal.type.price)
                val diff = PrizesLogic.difference(item.price, balance)
                // **وفي العرض بلا رصيدٍ لا يُقال «بتدفع»** — لا رصيدَ يُقارَن به.
                if (enabled && covered) {
                    Text(
                        stringResource(R.string.prizes_covered),
                        color = Rahal.colors.success,
                        style = Rahal.type.caption,
                    )
                } else if (enabled && diff > 0) {
                    Text(
                        stringResource(R.string.prizes_diff, money(diff)),
                        color = Rahal.colors.accentInk,
                        style = Rahal.type.caption,
                    )
                }
            }
            Spacer(Modifier.size(Rahal.space.sm))
            RahalSmallButton(
                onClick = onAdd,
                enabled = enabled && item.available,
                tone = Tone.Accent,
            ) { Text(stringResource(R.string.prizes_add), maxLines = 1, softWrap = false) }
        }
    }
}

/** **صنفُ جائزةٍ كصنفِ سلّةٍ عاديّ** — بمعرّف متجر الجوائز، فيطبّق المحرّكُ الرصيدَ والتوصيلَ المجانيّ. */
private fun PrizeItem.toItem(storeId: String, storeName: String) = Item(
    id = id,
    name = name,
    description = description.orEmpty(),
    price = price,
    imageUrl = imageUrl,
    imageThumbUrl = thumbUrl,
    available = available,
    merchantId = storeId,
    merchantName = storeName,
)

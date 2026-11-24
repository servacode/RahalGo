package com.rahalgo.customer.prizes

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.border
import androidx.compose.foundation.background
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
            vm.teaser -> Teaser(vm, me)
            else -> Active(vm, me)
        }
    }
}

@Composable
private fun Teaser(vm: PrizesViewModel, me: PrizesMe) {
    val context = LocalContext.current
    val media = { path: String? -> Backend.of(context).media(path) }
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
        // **«لست ضمن المسابقة بعد» قبل الافتتاح كمان** (٢٠٢٦-١١-٢٤) — ومتى تبلش.
        item { EntryStatus(me, PrizesLogic.launchLabel(me.launchOn.ifBlank { "2026-12-01" })) }
        item { Podium(me.amounts) }
        item { PrizeRules(me.invitesPerEntry, me.winnersCount.takeIf { it > 0 } ?: me.amounts.size) }
        // **والهدايا تحتها** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — عرضٌ فقط قبل الافتتاح.
        if (vm.items.isNotEmpty()) {
            item { SectionHeader(stringResource(R.string.prizes_items_showcase)) }
            item {
                Text(
                    stringResource(R.string.prizes_showcase_hint),
                    color = Rahal.colors.accentInk,
                    style = Rahal.type.bodyStrong,
                )
            }
            prizeGrid(vm.items, "t-") { p ->
                PrizeRow(p, 0L, media, enabled = false, covered = false, onAdd = {})
            }
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
        item { EntryStatus(me) }
        item { ProgressCard(me, progress) }
        item {
            val r = InviteNudge.referral
            RahalButton(
                onClick = { if (r != null) InviteNudge.share(context, r) },
                enabled = r != null && r.code.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.prizes_share)) }
        }
        item { Podium(me.amounts) }
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
                prizeGrid(split.over, "s-") { p ->
                    PrizeRow(p, me.prizeBalance, media, enabled = false, covered = false, onAdd = {})
                }
            }
            else -> {
                if (split.within.isNotEmpty()) {
                    item { SectionHeader(stringResource(R.string.prizes_items_within)) }
                    prizeGrid(split.within, "w-") { p ->
                        PrizeRow(p, me.prizeBalance, media, enabled = true, covered = true, onAdd = { order(p) })
                    }
                }
                if (split.over.isNotEmpty()) {
                    item { SectionHeader(stringResource(R.string.prizes_items_over)) }
                    prizeGrid(split.over, "o-") { p ->
                        PrizeRow(p, me.prizeBalance, media, enabled = true, covered = false, onAdd = { order(p) })
                    }
                }
            }
        }

        item { PrizeRules(me.invitesPerEntry, me.winnersCount.takeIf { it > 0 } ?: me.amounts.size) }
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

/**
 * **«إنت ضمن المسابقة» أو «لست ضمن المسابقة»** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — جوابٌ واحدٌ واضح فوق
 * التفاصيل: **فرصةٌ وحدة على الأقلّ = داخل سحب هالشهر**، وإلّا كم دعوة باقية ليدخل.
 */
@Composable
private fun EntryStatus(me: PrizesMe, startsOn: String? = null) {
    val inside = startsOn == null && me.entries > 0
    RahalCard(
        variant = CardVariant.Tinted,
        tone = if (inside) StatusTone.Success else StatusTone.Warning,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            stringResource(if (inside) R.string.prizes_in_title else R.string.prizes_out_title),
            color = Rahal.colors.ink,
            style = Rahal.type.sectionTitle,
        )
        Text(
            if (inside) {
                stringResource(R.string.prizes_in_body, me.entries)
            } else if (startsOn != null) {
                stringResource(R.string.prizes_out_before, startsOn)
            } else {
                stringResource(R.string.prizes_out_body, me.invitesToNext.coerceAtLeast(1))
            },
            color = Rahal.colors.ink,
            style = Rahal.type.body,
        )
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

/**
 * **هديّتان بكلّ سطر متل كروت السوق** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤: «كل عنصرين بسطر… صورة وسعر من تحت
 * اطلب») — صفٌّ بنصفين، والأخيرُ الوحيدُ نصفُه فاضي فلا يتمدّد على العرض كلّه.
 */
private fun androidx.compose.foundation.lazy.LazyListScope.prizeGrid(
    list: List<PrizeItem>,
    keyPrefix: String,
    cell: @Composable (PrizeItem) -> Unit,
) {
    items(list.chunked(2), key = { keyPrefix + it.first().id }) { pair ->
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
            for (p in pair) Box(Modifier.weight(1f)) { cell(p) }
            if (pair.size == 1) Spacer(Modifier.weight(1f))
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
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .background(Rahal.colors.surface),
    ) {
        RemoteImage(
            url = media(item.thumbUrl ?: item.imageUrl),
            name = item.name,
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(1f),
        )
        Column(Modifier.padding(Rahal.space.sm)) {
            Text(
                item.name,
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = Rahal.type.body,
                maxLines = 2,
                minLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(money(item.price), color = Rahal.colors.ink, style = Rahal.type.price, maxLines = 1)
            val diff = PrizesLogic.difference(item.price, balance)
            // **وفي العرض بلا رصيدٍ لا يُقال «بتدفع»** — لا رصيدَ يُقارَن به. وسطرٌ ثابتٌ فلا تختلف الكروت طولاً.
            Text(
                when {
                    enabled && covered -> stringResource(R.string.prizes_covered)
                    enabled && diff > 0 -> stringResource(R.string.prizes_diff, money(diff))
                    else -> ""
                },
                color = if (enabled && covered) Rahal.colors.success else Rahal.colors.accentInk,
                style = Rahal.type.caption,
                maxLines = 1,
            )
            Spacer(Modifier.height(Rahal.space.xs))
            // **زرُّ «اطلب» متل كرت السوق** — ممتلئٌ بلون العلامة، وباهتٌ لما ما بينطلب (عرضٌ قبل الربح).
            val can = enabled && item.available
            Box(
                Modifier
                    .fillMaxWidth()
                    .height(Rahal.size.count * 2)
                    .clip(Rahal.shape.sm)
                    .background(if (can) Rahal.colors.brand else Rahal.colors.line)
                    .then(if (can) Modifier.clickable(onClick = onAdd) else Modifier),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    stringResource(R.string.prizes_add),
                    color = if (can) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.meta,
                    maxLines = 1,
                )
            }
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

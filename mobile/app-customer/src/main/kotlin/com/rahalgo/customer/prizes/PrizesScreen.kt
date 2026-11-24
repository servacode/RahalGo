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
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
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
 * **مسابقةُ نقاطٍ شهريّة** (قرارُ المالك ٢٠٢٦-١١-٢٥). **قبل الافتتاح لافتةٌ** («المسابقة بتبلش ١/١٢»).
 * **وبعده** نقاطُه وترتيبُه وما بقي له وشروطُه وزرُّ «سجّل حضوري» — **ولا شيءَ عن زبونٍ آخر** —
 * ثمّ سلّمُ الجوائز ورصيدُه وأصنافُ الجوائز: **«على قدّ جائزتك» ثمّ «بتدفع الفرق»**، **وبلا رصيدٍ
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
        // **وفراغٌ تحت آخر الصفحة** لزرّ الدردشة العائم — كان يغطّي آخرَ شرط (المالك ٢٠٢٦-١١-٢٥).
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = Rahal.space.lg, top = Rahal.space.lg, end = Rahal.space.lg, bottom = Rahal.space.lg * 6,
        ),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        // **رأسٌ واحد**: متى تبلش وكم يوم باقي، و«لست ضمن المسابقة بعد» (ملاحظةُ المالك ٢٠٢٦-١١-٢٤).
        item {
            val launch = me.launchOn.ifBlank { "2026-12-01" }
            PrizeHero(
                label = stringResource(R.string.prizes_hero_starts),
                date = PrizesLogic.launchLabel(launch),
                days = PrizesLogic.daysUntilDate(launch),
                inside = false,
                statusTitle = stringResource(R.string.prizes_out_title),
                statusBody = stringResource(R.string.prizes_out_before, PrizesLogic.launchLabel(launch)),
            )
        }
        item { Podium(me.amounts) }
        // **والهدايا تحتها** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — عرضٌ فقط قبل الافتتاح.
        if (vm.items.isNotEmpty()) {
            item { SectionHeader(stringResource(R.string.prizes_gifts_title)) }
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
        item { PointsTable(me) }
        item { PrizeTerms(me) }
    }
}

@Composable
private fun Active(vm: PrizesViewModel, me: PrizesMe) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    // **رمزُ الدعوة نفسُه المستعمَل في «ادعُ صديقاً»** — يُجلب إن لم يكن قد جاء.
    LaunchedEffect(Unit) { if (InviteNudge.referral == null) InviteNudge.refresh(scope) }

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
        // **وفراغٌ تحت آخر الصفحة** لزرّ الدردشة العائم — كان يغطّي آخرَ شرط (المالك ٢٠٢٦-١١-٢٥).
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = Rahal.space.lg, top = Rahal.space.lg, end = Rahal.space.lg, bottom = Rahal.space.lg * 6,
        ),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        item {
            if (vm.error.isNotEmpty()) {
                com.rahalgo.ui.Note(vm.error, Rahal.colors.danger)
            }
        }
        // **رأسٌ واحد**: آخرُ يومٍ لجمع النقاط وكم يوم، متأهّلٌ أو كم باقي، ونقاطُه وترتيبُه، وزرُّ الدعوة.
        item {
            val r = InviteNudge.referral
            val rank = if (me.rank > 0) me.rank.toString() else "—"
            PrizeHero(
                label = stringResource(R.string.prizes_hero_cutoff),
                date = PrizesLogic.dayLabel(me.cutoffAt),
                days = PrizesLogic.daysUntilInstant(me.cutoffAt),
                inside = me.qualified,
                statusTitle = stringResource(
                    when {
                        me.excluded -> R.string.prizes_excluded_title
                        me.qualified -> R.string.prizes_qualified_title
                        else -> R.string.prizes_out_title
                    },
                ),
                statusBody = when {
                    me.excluded -> stringResource(R.string.prizes_excluded_body)
                    me.qualified -> stringResource(R.string.prizes_qualified_body)
                    else -> stringResource(R.string.prizes_to_qualify, me.toQualify.coerceAtLeast(1))
                },
                progress = if (me.excluded) null else PrizesLogic.qualifyProgress(me),
                progressLine = stringResource(R.string.prizes_points_line, me.points.toString(), rank),
                action = {
                    RahalButton(
                        onClick = { if (r != null) InviteNudge.share(context, r) },
                        enabled = r != null && r.code.isNotBlank(),
                        tone = Tone.Accent,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(R.string.prizes_share)) }
                },
            )
        }
        if (me.lastRank > 0) {
            item { com.rahalgo.ui.Note(stringResource(R.string.prizes_last_won, me.lastRank, money(me.lastAmount)), Rahal.colors.success) }
        }
        if (!me.excluded) {
            item {
                PrizeConditions(
                    me = me,
                    busy = vm.busy,
                    onCheckIn = vm::checkIn,
                    onFollow = { network ->
                        val url = PrizesLogic.pageUrl(if (network == "facebook") me.facebookUrl else me.instagramUrl)
                        if (url.isNotEmpty()) {
                            runCatching {
                                context.startActivity(
                                    android.content.Intent(android.content.Intent.ACTION_VIEW, android.net.Uri.parse(url)),
                                )
                            }
                        }
                        vm.follow(network)
                    },
                )
            }
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
                item { SectionHeader(stringResource(R.string.prizes_gifts_title)) }
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

        item { PointsTable(me) }
        item { PrizeTerms(me) }
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
                    !enabled -> stringResource(R.string.prizes_win_first)
                    else -> ""
                },
                color = when {
                    enabled && covered -> Rahal.colors.success
                    !enabled -> Rahal.colors.inkMuted
                    else -> Rahal.colors.accentInk
                },
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

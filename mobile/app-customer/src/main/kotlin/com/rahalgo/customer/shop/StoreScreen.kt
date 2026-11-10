package com.rahalgo.customer.shop

import androidx.compose.ui.unit.sp
import androidx.compose.foundation.text.TextAutoSize
import androidx.compose.foundation.text.BasicText
import com.rahalgo.shared.model.Section
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.FeaturedStore
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.StoreHeader
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalLoader
import com.rahalgo.ui.RemoteImage
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ «متاجر مميّزة» — مربّعاتٌ لا دوائر** (قرارُ المالك ٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مربّعٌ بزوايا مدوّرة** فيه الشعار، **والاسمُ تحته**. ومتجرٌ بلا شعارٍ
 * يُرسم **حرفَه الأوّلَ على لونٍ هادئ** (`RemoteImage`). **والدوائرُ لشريط
 * الأقسام** — شكلان لبابين.
 */
@Composable
internal fun FeaturedStrip(
    stores: List<FeaturedStore>,
    media: (String?) -> String?,
    onOpen: (String) -> Unit,
) {
    Column(Modifier.fillMaxWidth().padding(vertical = Rahal.space.xs)) {
        Text(
            text = stringResource(R.string.shop_featured_stores),
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
            modifier = Modifier.padding(horizontal = 14.dp),
        )
        Spacer(Modifier.height(Rahal.space.sm))
        val nameSize = rememberFeaturedNameSize(stores.map { it.name }, 76.dp)
        LazyRow(
            contentPadding = PaddingValues(horizontal = 14.dp),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            items(stores, key = { it.id }) { s ->
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    modifier = Modifier
                        .width(76.dp)
                        .clip(Rahal.shape.md)
                        .clickable { onOpen(s.id) },
                ) {
                    RemoteImage(
                        url = media(s.logoThumbUrl ?: s.logoUrl),
                        name = s.name,
                        modifier = Modifier
                            .size(72.dp)
                            .clip(Rahal.shape.md)
                            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                            // **والمغلقُ أخفت** — يُفتح ويُتصفّح، **ولا يُوهِم أنّه يستقبل.**
                            .alpha(if (s.openNow) 1f else 0.7f),
                    )
                    Spacer(Modifier.height(Rahal.space.xs))
                    FeaturedName(s.name, nameSize)
                }
            }
        }
    }
}

/**
 * **حجمُ خطٍّ واحدٌ لكلّ أسماء الشريط** (قرارُ المالك ٢٠٢٦-١١-٠٤: «خط واحد للكل») — **أكبرُ حجمٍ
 * يسع عنده كلُّ اسمٍ في سطرين على عرض المربّع**، يُحسب مرّةً للقائمة بقياس النصّ فعلاً، وأرضيّتُه ٩.
 * **ولا نقاطَ في آخر الاسم أبداً** — ينزل لسطرٍ ثانٍ.
 */
@Composable
private fun rememberFeaturedNameSize(names: List<String>, tile: androidx.compose.ui.unit.Dp): androidx.compose.ui.unit.TextUnit {
    val base = MaterialTheme.typography.labelMedium
    val measurer = androidx.compose.ui.text.rememberTextMeasurer()
    val density = androidx.compose.ui.platform.LocalDensity.current
    return remember(names, tile, density, base) {
        val widthPx = with(density) { tile.roundToPx() }
        val top = base.fontSize.value
        var size = top
        while (size > FEATURED_MIN_SP) {
            val fits = names.all { n ->
                val r = measurer.measure(
                    text = n,
                    style = base.copy(fontSize = size.sp, textAlign = TextAlign.Center),
                    maxLines = 2,
                    softWrap = true,
                    constraints = androidx.compose.ui.unit.Constraints(maxWidth = widthPx),
                )
                !r.didOverflowHeight && !r.didOverflowWidth
            }
            if (fits) break
            size -= 0.5f
        }
        size.coerceAtLeast(FEATURED_MIN_SP).sp
    }
}

private const val FEATURED_MIN_SP = 9f

@Composable
private fun FeaturedName(name: String, size: androidx.compose.ui.unit.TextUnit) {
    BasicText(
        text = name,
        style = MaterialTheme.typography.labelMedium.copy(
            color = Rahal.colors.ink,
            textAlign = TextAlign.Center,
            fontSize = size,
        ),
        softWrap = true,
        overflow = TextOverflow.Clip,
        modifier = Modifier.fillMaxWidth(),
    )
}

/**
 * **دوامُ اليوم بلفظ الزبون** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨: «مافي شي اسمو مفتوح هلق… تحط ساعات
 * الدوام… كل يوم بيومو») — «مفتوح ٢٤ ساعة» أو «اليوم من ٩:٠٠ ص لـ ١١:٠٠ م» أو «مغلق اليوم».
 */
@Composable
private fun statusText(h: StoreHeader): String = when {
    h.today.open24h -> stringResource(R.string.store_open_24h)
    h.today.closed || h.today.open.isBlank() -> stringResource(R.string.store_closed_today)
    else -> {
        // **وحرفا الصبح والمسا من ملف النصوص** — لا مكتوبَين جوّا الشيفرة.
        val am = stringResource(R.string.clock_am)
        val pm = stringResource(R.string.clock_pm)
        stringResource(R.string.store_today_hours, clock12(h.today.open, am, pm), clock12(h.today.close, am, pm))
    }
}

/** «18:30» ← «6:30 م». */
internal fun clock12(hhmm: String, am: String, pm: String): String {
    val p = hhmm.split(":")
    val h = p.getOrNull(0)?.toIntOrNull() ?: return hhmm
    val m = p.getOrNull(1) ?: "00"
    val h12 = when (val x = h % 12) { 0 -> 12; else -> x }
    return "$h12:$m " + if (h < 12) am else pm
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحةُ المتجر — رأسٌ وشريطُ أقسامٍ وأصنافُه مجموعة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والرأسُ والشريطُ ثابتان والشبكةُ تمرّ تحتهما** — فالشريطُ لاصقٌ بلا
 * حيلة، **ويُضيء القسمَ الذي تقف فيه الشبكة** (`StoreMenu.sectionAt`).
 *
 * **والبطاقةُ بطاقةُ السوق** و«+» قرارُ السوق نفسُه (`onAdd`) — **والمغلقُ
 * للتصفّح**: أصنافُه تصل `sourceClosed` فلا «+» عليها.
 */
@Composable
internal fun StoreScreen(
    vm: ShopViewModel,
    media: (String?) -> String?,
    liked: Set<String>,
    onLike: (Item) -> Unit,
    onAdd: (Item) -> Unit,
    onBack: () -> Unit,
    /** **ضغطةُ البطاقة تفتح نافذةَ الصنف** (٢٠٢٦-١٠-٣٠) — كما في السوق. */
    onOpen: (Item) -> Unit = {},
    /**
     * **لافتةُ السوق نفسُها حين يُطفأ «+»** — عنوانٌ خارجَ التغطية أو استعراض
     * (`ServiceBlockNotice`)، **ولا «+» يُطفأ بصمت.**
     */
    notice: @Composable () -> Unit = {},
) {
    Column(Modifier.fillMaxSize()) {
        val page = vm.store
        // ══════════════════════════════════════════════════════════════
        // **شريطٌ علويٌّ: سهمُ رجوعٍ واسمُ المتجر** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **كان زرّاً بإطارٍ فوق الشعار** فيُقرأ زرّاً في نصّ الصفحة. **والسهمُ
        // مع كلمة «رجوع للتسوق» باقٍ** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨: «زر واضح مع
        // ايقونة مناسبة») — **ومساحةُ لمسه ٤٨ لا أقلّ.**
        // **رأسٌ واحدٌ لا اثنان (تدقيق D16، ٢٠٢٦-١١-١١)**: سهمُ الرجوع صار داخلَ بطاقة المتجر؛ وهذا
        // الشريطُ المنفصلُ يبقى فقط أثناءَ التحميل أو الخطأ حيث لا بطاقةَ بعد.
        if (page == null || vm.storeError.isNotEmpty()) Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Rahal.space.xs, vertical = Rahal.space.xs),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .clip(Rahal.shape.md)
                    .clickable(onClick = onBack)
                    .heightIn(min = 48.dp)
                    .padding(horizontal = Rahal.space.sm),
            ) {
                Icon(
                    painter = painterResource(com.rahalgo.ui.R.drawable.ic_arrow_back),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(22.dp),
                )
                Spacer(Modifier.width(Rahal.space.xs))
                Text(
                    text = stringResource(R.string.store_back),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelLarge,
                )
            }
        }

        when {
            vm.storeError.isNotEmpty() ->
                LoadState(loading = false, error = vm.storeError, onRetry = vm::loadStore)

            page == null ->
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { RahalLoader() }

            else -> {
                val h = page.store
                // **بطاقةُ الرأس: الشعارُ والاسمُ وشارةُ «مفتوح/مسكّر» والدوامُ تحتها.**
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = Rahal.space.md)
                        .clip(Rahal.shape.md)
                        .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                        .padding(Rahal.space.md),
                ) {
                    Box(
                        Modifier
                            .size(48.dp)
                            .clip(CircleShape)
                            .clickable(onClickLabel = stringResource(R.string.store_back), onClick = onBack),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            painter = painterResource(com.rahalgo.ui.R.drawable.ic_arrow_back),
                            contentDescription = stringResource(R.string.store_back),
                            tint = Rahal.colors.brand,
                            modifier = Modifier.size(22.dp),
                        )
                    }
                    Spacer(Modifier.width(Rahal.space.xs))
                    RemoteImage(
                        url = media(h.logoThumbUrl ?: h.logoUrl),
                        name = h.name,
                        modifier = Modifier
                            .size(64.dp)
                            .clip(Rahal.shape.md)
                            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
                    )
                    Spacer(Modifier.width(Rahal.space.md))
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = h.name,
                            color = Rahal.colors.ink,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleMedium,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Spacer(Modifier.height(Rahal.space.xs))
                        // **دوامُ اليوم وحدَه بلا شارة «مفتوح/مسكّر»** (قرارُ المالك ٢٠٢٦-١٠-٢٨) —
                        // **والإغلاقُ الطارئُ يُقال بلفظه** فلا يُقرأ الدوامُ وعداً بأنّه مفتوح.
                        Text(
                            text = if (h.emergencyClosed) {
                                stringResource(R.string.store_emergency_closed)
                            } else {
                                statusText(h)
                            },
                            color = if (h.emergencyClosed) Rahal.colors.danger else Rahal.colors.inkMuted,
                            fontWeight = if (h.emergencyClosed) FontWeight.Bold else FontWeight.Normal,
                            style = MaterialTheme.typography.labelMedium,
                        )
                        // **زمنُ التحضير إن أرسله المحرّك (تدقيق D16)** — لا تقييمَ ولا غلافَ في ردّ المتجر
                        // اليوم، ولا يُختلَق حقل.
                        if (h.prepMinutes > 0 && !h.emergencyClosed) {
                            Text(
                                text = stringResource(R.string.store_prep, com.rahalgo.ui.minutes(h.prepMinutes.toLong())),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.labelMedium,
                            )
                        }
                    }
                }
                if (!h.openNow) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Column(Modifier.padding(horizontal = Rahal.space.md)) {
                        com.rahalgo.ui.Note(stringResource(R.string.store_browse_only), Rahal.colors.danger)
                    }
                }
                Spacer(Modifier.height(Rahal.space.sm))
                notice()

                // **Sections with counts from the summary, items page by page** (owner 2026-11-07:
                // «ما يصير يكون بطيء») — see `StoreList`.
                val sections = page.sections.filter { it.count > 0 }
                if (sections.isEmpty()) {
                    Empty(
                        text = stringResource(R.string.store_empty),
                        hint = stringResource(R.string.store_empty_hint),
                        actionLabel = "",
                        onAction = null,
                    )
                    return@Column
                }
                val list = vm.storeList

                // **كالسوق تماماً** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨: «لازم نفس الاقسام دوائر من فوق وكل
                // قسم فيه اصنافه») — دوائرُ الأقسام بصورها، **وتحتها أصنافُ المختار وحدَه.**
                Spacer(Modifier.height(Rahal.space.sm))
                // **Search inside the store** — on the server, across its sections (2026-11-07).
                androidx.compose.material3.OutlinedTextField(
                    value = list.query,
                    onValueChange = { list.search(it) },
                    placeholder = { Text(stringResource(com.rahalgo.ui.R.string.ow_search)) },
                    leadingIcon = {
                        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_search), contentDescription = null, tint = Rahal.colors.inkMuted)
                    },
                    trailingIcon = if (list.query.isNotEmpty()) {
                        {
                            androidx.compose.material3.IconButton(onClick = { list.search("") }) {
                                Icon(
                                    painterResource(com.rahalgo.ui.R.drawable.ic_close),
                                    contentDescription = stringResource(com.rahalgo.ui.R.string.act_cancel),
                                    tint = Rahal.colors.inkMuted,
                                )
                            }
                        }
                    } else {
                        null
                    },
                    singleLine = true,
                    // **بنمط حقل الرئيسيّة نفسِه (تدقيق D16)**: دائريٌّ بحدٍّ عاديّ — لا مستطيلٌ رماديٌّ آخر.
                    shape = Rahal.shape.pill,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = Rahal.space.sm),
                )
                if (list.query.isBlank()) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    SectionRail(
                        sections.map { Section(id = it.id, name = it.name, icon = it.icon, imageUrl = it.imageUrl, imageThumbUrl = it.imageThumbUrl) },
                        list.section.ifEmpty { sections.first().id },
                        media,
                    ) { list.pick(it) }
                }

                LazyVerticalGrid(
                    columns = GridCells.Fixed(ShopCols),
                    contentPadding = PaddingValues(start = 12.dp, end = 12.dp, top = 8.dp, bottom = 16.dp + com.rahalgo.ui.LocalChatFabInset.current),
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                    verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
                ) {
                    items(list.items, key = { it.id }) { item ->
                        ItemCard(
                            item = item,
                            media = media,
                            liked = item.id in liked,
                            onAdd = { onAdd(item) },
                            onOpen = { onOpen(item) },
                            onLike = { onLike(item) },
                            // **ولا رابطَ إلى الصفحة التي هو فيها** (`StoreLink`).
                            onStore = StoreLink.target(item, vm.storeId)?.let { id -> { vm.openStore(id) } },
                            showStoreName = false,
                        )
                    }
                    val full: androidx.compose.foundation.lazy.grid.LazyGridItemSpanScope.() -> GridItemSpan = { GridItemSpan(maxLineSpan) }
                    when {
                        list.items.isEmpty() && list.loading -> item(span = full) {
                            Box(Modifier.fillMaxWidth().padding(vertical = Rahal.space.xl), contentAlignment = Alignment.Center) { RahalLoader(size = 48.dp) }
                        }
                        list.items.isEmpty() && list.error.isNotEmpty() -> item(span = full) {
                            LoadState(loading = false, error = list.error, onRetry = list::reload)
                        }
                        list.items.isEmpty() -> item(span = full) {
                            Text(
                                stringResource(com.rahalgo.ui.R.string.grid_search_none),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodyMedium,
                                textAlign = TextAlign.Center,
                                modifier = Modifier.fillMaxWidth().padding(vertical = Rahal.space.xl),
                            )
                        }
                        list.hasMore -> item(span = full, key = "more") {
                            androidx.compose.runtime.LaunchedEffect(list.items.size) { list.loadMore() }
                            Box(Modifier.fillMaxWidth().padding(vertical = Rahal.space.md), contentAlignment = Alignment.Center) { RahalLoader(size = 48.dp) }
                        }
                    }
                }
            }
        }
    }
}

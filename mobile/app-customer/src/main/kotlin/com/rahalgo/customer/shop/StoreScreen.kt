package com.rahalgo.customer.shop

import androidx.compose.ui.unit.sp
import androidx.compose.foundation.text.TextAutoSize
import androidx.compose.foundation.text.BasicText
import com.rahalgo.shared.model.Section
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.foundation.border
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
                    FeaturedName(s.name)
                }
            }
        }
    }
}

/**
 * **اسمُ المتجر المميّز بسطرٍ واحد** (طلبُ المالك ٢٠٢٦-١٠-١٩: «بدي بسطر واحد بس يصغر خطه
 * اذا كان طويل شوي») — **يصغر الخطُّ حتّى يسع**، وما لا يسعه ولو صغر يُختم بنقاط.
 */
@Composable
private fun FeaturedName(name: String) {
    val base = MaterialTheme.typography.labelMedium.copy(
        color = Rahal.colors.ink,
        textAlign = TextAlign.Center,
    )
    BasicText(
        text = name,
        style = base,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        // **وأصغرُ خطٍّ ١٠** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — ٨ ما بينقرا على شاشة صغيرة.
        autoSize = TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = base.fontSize),
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
        Row(
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

                val sections = StoreMenu.visible(page.sections)
                if (sections.isEmpty()) {
                    Empty(
                        text = stringResource(R.string.store_empty),
                        hint = stringResource(R.string.store_empty_hint),
                        actionLabel = "",
                        onAction = null,
                    )
                    return@Column
                }

                // **كالسوق تماماً** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨: «لازم نفس الاقسام دوائر من فوق وكل
                // قسم فيه اصنافه») — دوائرُ الأقسام بصورها، **وتحتها أصنافُ المختار وحدَه.**
                var picked by rememberSaveable(sections) { mutableStateOf(sections.first().id) }
                val shown = sections.firstOrNull { it.id == picked } ?: sections.first()
                Spacer(Modifier.height(Rahal.space.sm))
                SectionRail(
                    sections.map { Section(id = it.id, name = it.name, icon = it.icon, imageUrl = it.imageUrl, imageThumbUrl = it.imageThumbUrl) },
                    shown.id,
                    media,
                ) { picked = it }

                LazyVerticalGrid(
                    columns = GridCells.Fixed(ShopCols),
                    contentPadding = PaddingValues(start = 12.dp, end = 12.dp, top = 8.dp, bottom = 96.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    items(shown.items, key = { shown.id + "/" + it.id }) { item ->
                        ItemCard(
                            item = item,
                            media = media,
                            liked = item.id in liked,
                            onAdd = { onAdd(item) },
                            onLike = { onLike(item) },
                            // **ولا رابطَ إلى الصفحة التي هو فيها** (`StoreLink`).
                            onStore = StoreLink.target(item, vm.storeId)?.let { id -> { vm.openStore(id) } },
                        )
                    }
                }
            }
        }
    }
}

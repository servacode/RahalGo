package com.rahalgo.customer.shop

import com.rahalgo.shared.model.Section
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.foundation.border
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
                    Text(
                        text = s.name,
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.labelMedium,
                        minLines = 2,
                        maxLines = 2,
                        overflow = TextOverflow.Visible,
                        textAlign = TextAlign.Center,
                    )
                }
            }
        }
    }
}

/** **حالُ المتجر بلفظ الزبون** — «مفتوح هلّق» أو «مغلق — يفتح الساعة …». */
@Composable
private fun statusText(h: StoreHeader): String {
    if (h.openNow) return stringResource(R.string.store_open)
    val t = com.rahalgo.ui.backAtText(h.opensAt)
    return if (t != null) stringResource(R.string.store_closed_opens_at, t) else stringResource(R.string.store_closed)
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
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm)
                .clip(Rahal.shape.sm)
                .clickable(onClick = onBack)
                .padding(Rahal.space.xs),
        ) {
            Icon(
                painter = painterResource(com.rahalgo.ui.R.drawable.ic_close),
                contentDescription = null,
                tint = Rahal.colors.brand,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(Rahal.space.xs))
            Text(
                text = stringResource(R.string.store_back),
                color = Rahal.colors.brand,
                style = MaterialTheme.typography.labelLarge,
            )
        }

        val page = vm.store
        when {
            vm.storeError.isNotEmpty() ->
                LoadState(loading = false, error = vm.storeError, onRetry = vm::loadStore)

            page == null ->
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { RahalLoader() }

            else -> {
                val h = page.store
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp),
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
                        )
                        Spacer(Modifier.height(Rahal.space.xs))
                        Text(
                            text = statusText(h),
                            color = if (h.openNow) Rahal.colors.success else Rahal.colors.danger,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelMedium,
                        )
                        if (h.prepMinutes > 0) {
                            Text(
                                text = stringResource(R.string.store_prep, h.prepMinutes.toString()),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.labelSmall,
                            )
                        }
                    }
                }
                if (!h.openNow) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Column(Modifier.padding(horizontal = 12.dp)) {
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

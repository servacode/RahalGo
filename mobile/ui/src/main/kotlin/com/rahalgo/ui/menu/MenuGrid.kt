package com.rahalgo.ui.menu

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridItemSpanScope
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.Empty
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RemoteImage

/**
 * **صنفٌ كما تعرضه شبكةُ الأصناف** — يحوّله كلُّ تطبيقٍ من نوعه (المتجر والمندوب).
 */
data class GridItem(
    val id: String,
    val name: String,
    val description: String,
    val imageUrl: String?,
    val price: String,
    /** **سعرُ الزبون إن زاد على سعر المتجر** — يُعرض أخضرَ بجانبه (المندوب). */
    val customerPrice: String? = null,
    val available: Boolean,
    val approved: Boolean = true,
    val rejected: Boolean = false,
    val reviewNote: String = "",
    val closedNow: Boolean = false,
)

/** **قسمٌ في الشبكة** — اسمُه وأصنافُه. */
data class GridGroup(val name: String, val items: List<GridItem>)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شبكةُ الأصناف — التصميمُ المعتمد ٢٠٢٦-١٠-٢٩، للمتجر والمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * من فوق لتحت: **البحث · مربّعا الأرقام · الأدوات الثلاث · «أضف صنفاً» و«أنشئ عرضاً»
 * · رقائقُ الأقسام · ثمّ الأصناف بطاقتين في كلّ سطر.**
 *
 * **نسخةٌ واحدةٌ للتطبيقين** (طلبُ المالك ٢٠٢٦-١٠-٢٩: «تطبيق المندوب يطبق عليه كلشي
 * تطبق على المتجر») — ومن غيّر أحدَهما وحدَه أعاد الفرق.
 */
@Composable
fun MenuItemsGrid(
    groups: List<GridGroup>,
    query: String,
    onQuery: (String) -> Unit,
    openName: String?,
    onlyOff: Boolean,
    onAll: () -> Unit,
    onOnlyOff: () -> Unit,
    onOpen: (String) -> Unit,
    showTools: Boolean,
    onPresets: () -> Unit,
    onBulk: () -> Unit,
    onDrinks: () -> Unit,
    addLabel: String,
    onAdd: () -> Unit,
    offerLabel: String,
    onOffer: (() -> Unit)?,
    emptyText: String,
    toggleEnabled: Boolean,
    onToggle: (GridItem) -> Unit,
    onEdit: (GridItem) -> Unit,
    header: (LazyGridScope.() -> Unit)? = null,
) {
    val all = groups.flatMap { it.items }
    val off = all.count { !it.available }
    val q = query.trim()
    val shown = when {
        q.isNotEmpty() -> all.filter { it.name.contains(q, ignoreCase = true) }
        onlyOff -> all.filter { !it.available }
        openName != null -> groups.firstOrNull { it.name == openName }?.items.orEmpty()
        else -> all
    }
    val full: (LazyGridItemSpanScope.() -> GridItemSpan) = { GridItemSpan(maxLineSpan) }
    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Rahal.space.lg, end = Rahal.space.lg, top = Rahal.space.lg, bottom = Rahal.space.xl),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        header?.invoke(this)
        // ── ١ · البحث ──
        item(span = full) {
            OutlinedTextField(
                value = query,
                onValueChange = { onQuery(it.take(60)) },
                placeholder = { Text(stringResource(R.string.ow_search)) },
                leadingIcon = {
                    Icon(painterResource(R.drawable.ic_search), contentDescription = null, tint = Rahal.colors.inkMuted)
                },
                trailingIcon = if (query.isNotEmpty()) {
                    {
                        IconButton(onClick = { onQuery("") }) {
                            Icon(
                                painterResource(R.drawable.ic_close),
                                contentDescription = stringResource(R.string.act_cancel),
                                tint = Rahal.colors.inkMuted,
                            )
                        }
                    }
                } else {
                    null
                },
                singleLine = true,
                shape = Rahal.shape.md,
                colors = OutlinedTextFieldDefaults.colors(
                    unfocusedContainerColor = Rahal.colors.field,
                    focusedContainerColor = Rahal.colors.field,
                    unfocusedBorderColor = Color.Transparent,
                ),
                modifier = Modifier.fillMaxWidth(),
            )
        }
        // ── ٢ · مربّعا الأرقام — و«غير المتوفرة» يصفّي بضغطة ──
        item(span = full) {
            Row(
                Modifier.fillMaxWidth().height(IntrinsicSize.Min),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                StatBox(R.drawable.ic_store, stringResource(R.string.grid_stat_total), all.size, Rahal.colors.brand, false, Modifier.weight(1f), onAll)
                StatBox(
                    R.drawable.ic_block,
                    stringResource(R.string.grid_stat_off),
                    off,
                    if (off > 0) Rahal.colors.danger else Rahal.colors.inkMuted,
                    onlyOff,
                    Modifier.weight(1f),
                ) { if (onlyOff) onAll() else onOnlyOff() }
            }
        }
        if (showTools) {
            // ── ٣ · الأدواتُ الثلاث ──
            item(span = full) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    ToolTile(R.drawable.ic_custom, stringResource(R.string.presets_title), Modifier.weight(1f), onPresets)
                    ToolTile(R.drawable.ic_banknotes, stringResource(R.string.bulk_title), Modifier.weight(1f), onBulk)
                    ToolTile(R.drawable.ic_cart, stringResource(R.string.grid_drinks), Modifier.weight(1f), onDrinks)
                }
            }
            // ── ٤ · «أضف صنفاً» و«أنشئ عرضاً» ──
            item(span = full) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
                    RahalButton(onClick = onAdd, modifier = Modifier.weight(1f).height(52.dp)) {
                        Icon(painterResource(R.drawable.ic_plus), contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(Rahal.space.sm))
                        Text(addLabel, fontWeight = FontWeight.Bold, maxLines = 1)
                    }
                    if (onOffer != null) {
                        RahalOutlineButton(onClick = onOffer, modifier = Modifier.weight(1f).height(52.dp)) {
                            Icon(painterResource(R.drawable.ic_offer), contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(Modifier.width(Rahal.space.sm))
                            Text(offerLabel, fontWeight = FontWeight.Bold, maxLines = 1)
                        }
                    }
                }
            }
        }
        // ── ٥ · رقائقُ الأقسام ──
        if (groups.isNotEmpty() && q.isEmpty()) {
            item(span = full) {
                Row(
                    Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                ) {
                    SectionChip(stringResource(R.string.grid_all), all.size, openName == null && !onlyOff, onAll)
                    groups.forEach { g ->
                        SectionChip(g.name, g.items.size, openName == g.name && !onlyOff) { onOpen(g.name) }
                    }
                }
            }
        }
        // ── ٦ · الأصناف — بطاقتان في كلّ سطر ──
        when {
            groups.isEmpty() -> item(span = full) { Empty(emptyText) }
            shown.isEmpty() -> item(span = full) {
                Text(
                    stringResource(if (onlyOff && q.isEmpty()) R.string.grid_off_none else R.string.grid_search_none),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(vertical = Rahal.space.md),
                )
            }
            else -> items(shown, key = { it.id }) { item ->
                ItemCard(item, toggleEnabled, { onToggle(item) }, { onEdit(item) })
            }
        }
    }
}

/** **مربّعُ رقم** — أيقونةٌ في دائرة، والرقمُ كبيراً، واسمُه تحته. والمختارُ بحدّ لونه. */
@Composable
private fun StatBox(
    icon: Int,
    label: String,
    value: Int,
    tint: Color,
    selected: Boolean,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    Row(
        modifier
            .fillMaxHeight()
            .clip(Rahal.shape.lg)
            .background(if (selected) tint.copy(alpha = 0.08f) else Rahal.colors.surface)
            .border(
                if (selected) Rahal.stroke.focus else Rahal.stroke.hair,
                if (selected) tint else Rahal.colors.line,
                Rahal.shape.lg,
            )
            .clickable(onClick = onClick)
            .padding(Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier.size(40.dp).clip(Rahal.shape.pill).background(tint.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(20.dp))
        }
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                value.toString(),
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.headlineSmall,
                maxLines = 1,
            )
            Text(
                label,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** **زرُّ أداة** — أيقونةٌ فوق اسمها في مربّعٍ فاتح. */
@Composable
private fun ToolTile(icon: Int, label: String, modifier: Modifier, onClick: () -> Unit) {
    Column(
        modifier
            .height(68.dp)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.surface)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .clickable(onClick = onClick)
            .padding(horizontal = Rahal.space.xs),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = Rahal.colors.brand, modifier = Modifier.size(22.dp))
        Spacer(Modifier.height(Rahal.space.xs))
        Text(label, style = MaterialTheme.typography.labelMedium, color = Rahal.colors.ink, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

/** **رقاقةُ قسم** — اسمُه وعددُه؛ المختارةُ ممتلئةٌ بلون العلامة، والباقيةُ مؤطَّرة. */
@Composable
private fun SectionChip(text: String, count: Int, selected: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .clip(Rahal.shape.pill)
            .background(if (selected) Rahal.colors.brand else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (selected) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.pill)
            .clickable(onClick = onClick)
            .padding(start = Rahal.space.lg, end = Rahal.space.sm, top = Rahal.space.sm, bottom = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text,
            color = if (selected) Rahal.colors.onBrand else Rahal.colors.ink,
            fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
        )
        Spacer(Modifier.width(Rahal.space.sm))
        Text(
            count.toString(),
            color = if (selected) Rahal.colors.brand else Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier
                .clip(Rahal.shape.pill)
                .background(if (selected) Rahal.colors.onBrand else Rahal.colors.field)
                .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
        )
    }
}

/**
 * **بطاقةُ صنف** — صورتُه فوق، ثمّ اسمُه ووصفُه وسعرُه، ثمّ مفتاحُ توفّره.
 * **والبطاقةُ كلُّها تفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر. وغيرُ المتوفّر باهت.**
 */
@Composable
private fun ItemCard(item: GridItem, toggleEnabled: Boolean, onToggle: () -> Unit, onEdit: () -> Unit) {
    val dim = if (item.available) 1f else 0.5f
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg)
            .clickable(onClick = onEdit),
    ) {
        Box(Modifier.fillMaxWidth().aspectRatio(1.15f).alpha(dim)) {
            RemoteImage(url = item.imageUrl, name = item.name.ifBlank { "?" }, modifier = Modifier.fillMaxSize())
            val badge = when {
                item.rejected -> stringResource(R.string.grid_rejected)
                !item.approved -> stringResource(R.string.mn_pending_review)
                item.closedNow -> stringResource(R.string.grid_closed_now)
                else -> null
            }
            if (badge != null) {
                Text(
                    badge,
                    color = Rahal.colors.onBrand,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    modifier = Modifier
                        .align(Alignment.TopStart)
                        .padding(Rahal.space.sm)
                        .clip(Rahal.shape.pill)
                        .background(if (item.rejected) Rahal.colors.danger else Rahal.colors.accent)
                        .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
                )
            }
        }
        Column(Modifier.fillMaxWidth().padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm)) {
            Column(Modifier.alpha(dim)) {
                Text(
                    item.name,
                    minLines = 2,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.ink,
                    style = MaterialTheme.typography.bodyMedium,
                )
                Text(
                    item.description,
                    minLines = 2,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    item.price,
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (item.customerPrice != null) {
                    Text(
                        "← " + item.customerPrice,
                        color = Rahal.colors.success,
                        style = MaterialTheme.typography.labelSmall,
                        maxLines = 1,
                    )
                }
                if (item.reviewNote.isNotBlank()) {
                    Text(
                        item.reviewNote,
                        color = Rahal.colors.danger,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Spacer(Modifier.height(Rahal.space.xs))
            HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(if (item.available) R.string.mn_available else R.string.mn_unavailable),
                    color = if (item.available) Rahal.colors.success else Rahal.colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelMedium,
                    maxLines = 1,
                    modifier = Modifier.weight(1f),
                )
                Switch(
                    checked = item.available,
                    enabled = toggleEnabled,
                    onCheckedChange = { onToggle() },
                    colors = SwitchDefaults.colors(
                        checkedTrackColor = Rahal.colors.success,
                        checkedThumbColor = Rahal.colors.onPanel,
                        checkedBorderColor = Rahal.colors.success,
                    ),
                )
            }
        }
    }
}

package com.rahalgo.merchant.menu

import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridItemSpanScope
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.ui.draw.alpha
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.SwitchDefaults
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource

import com.rahalgo.ui.menu.ItemDraft
import com.rahalgo.ui.menu.ItemEditor
import com.rahalgo.ui.RahalOutlineButton
import androidx.compose.material3.Switch
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Checkbox
import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.ModifierOption
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.Backend
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.MenuItem
import com.rahalgo.ui.Card
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.rememberImagePicker
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأصناف — شبكةُ أقسام السوق ثمّ ما فيها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «التسميةُ غلط، يجب أن تكون الأصناف — وتكون
 *  عبارةً عن الأقسام الموجودة بالسوق والتي يوجد لديّ أصنافٌ بها».)
 *
 * # والنمطُ نفسُه الذي في الويب
 *
 * **شبكةٌ بالصور ثمّ قسمٌ يُفتح** (`MenuManager`) — **ومن رآها في اللوحة
 * يعرفها هنا.** وشكلان لشيءٍ واحدٍ يجعلان المنصّةَ تبدو منصّتين.
 *
 * # والصورةُ أوّلاً لأنّها هويّةُ القسم لا زينتُه
 *
 * **قائمةُ أسماءٍ تُقرأ سطراً سطراً، وشبكةُ صورٍ تُعرف بلمحة** — وصاحبُ
 * المتجر ينظر إلى الشاشة ثانيتين.
 *
 * # والتوفّرُ بضغطةٍ على السطر كلِّه
 *
 * **أوسعُ هدفٍ ممكنٍ لإصبعٍ مبلَّلةٍ في مطبخ** — وزرٌّ صغيرٌ في طرف السطر
 * يُخطئه الإصبعُ فيفتح ما لا يريد.
 */
@Composable
fun MenuScreen(
    vm: MenuViewModel,
    // **«أضف صنفاً» كتبويب الإضافة** — وبلا نداءٍ يفتح المحرّرَ هنا.
    onAddItem: (() -> Unit)? = null,
    // **«أنشئ عرضاً»** — يفتح صفحةَ العروض؛ وبلا نداءٍ لا يظهر الزرّ.
    onOffer: (() -> Unit)? = null,
) {
    val context = LocalContext.current
    val media = { path: String? -> Backend.of(context).media(path) }

    if (vm.loading || (vm.error.isNotEmpty() && vm.groups.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(R.string.menu_title), stringResource(R.string.menu_hint))
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    // **والمحرّرُ يغطّي كلَّ شيء** — كنافذة الويب.
    vm.editing?.let { draft ->
        BackHandler { vm.cancelEdit() }
        // **والمحرّرُ من الوحدة المشتركة** — انظر `ui.menu.ItemEditor`:
        // **نسخةٌ واحدةٌ للمتجر والمندوب.**
        ItemEditor(
            d = draft,
            sections = vm.sections,
            busy = vm.busy,
            media = media,
            onEdit = { f -> vm.editDraft(f) },
            onPickImage = { bytes -> vm.pickImage(bytes) },
            onClearImage = { vm.clearImage() },
            onSave = { vm.saveItem() },
            error = vm.editError,
            onCancel = { vm.cancelEdit() },
            onDelete = { vm.deleteItem(draft.itemId) },
            exchangeRate = vm.exchangeRate,
            presets = vm.presets.chosen,
            onDuplicate = if (vm.suspended) null else { { vm.duplicate(draft.itemId) } },
            dirty = vm.dirty,
            // **والموقوفُ يرى ولا يحفظ ولا يحذف** (تدقيقُ ٢٠٢٦-١٠-٢٨).
            readOnly = vm.suspended,
        )
        return
    }

    // **خياراتُ الأصناف وتغييرُ الأسعار تغطّيان الشبكةَ كالمحرّر** (٢٠٢٦-١٠-٢٧).
    if (vm.presets.open) {
        BackHandler { vm.presets.close() }
        com.rahalgo.ui.menu.OptionPresetsEditor(vm.presets)
        return
    }
    if (vm.bulk.open) {
        BackHandler { vm.bulk.close() }
        com.rahalgo.ui.menu.BulkPriceEditor(vm.bulk, vm.sections)
        return
    }

    // **قائمةُ المشروبات تغطّي الشبكةَ كالمحرّر** (٢٠٢٦-١٠-١٩).
    if (vm.drinks.open) {
        BackHandler { vm.drinks.close() }
        com.rahalgo.ui.menu.DrinksEditor(vm.drinks)
        return
    }

    val open = vm.open
    // **والرجوعُ يعيد إلى «الكل» لا يُخرج** — من اختار قسماً ثمّ ضغط رجوعاً
    // **يتوقّع قائمتَه كلَّها لا أن يخرج.**
    BackHandler(enabled = open != null || vm.onlyOff) { vm.back() }
    ItemsList(vm, media, open, onAddItem, onOffer)
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأصناف — التصميمُ المعتمد ٢٠٢٦-١٠-٢٩**
 * ══════════════════════════════════════════════════════════════════════
 *
 * من فوق لتحت: **البحث · مربّعا الأرقام (عدد الأصناف · غير المتوفرة) · الأدوات
 * الثلاث · «أضف صنفاً» و«أنشئ عرضاً» · رقائقُ الأقسام · ثمّ الأصناف بطاقتين في كلّ سطر.**
 *
 * **والرقاقةُ هي قسمُ السوق نفسُه** (`vm.open`) — «الكل» يعرض كلَّ الأصناف.
 * **وشبكةٌ واحدةٌ تمرّر** — لا تمريران متداخلان.
 */
@Composable
private fun ItemsList(
    vm: MenuViewModel,
    media: (String?) -> String?,
    open: MenuGroup?,
    onAddItem: (() -> Unit)?,
    onOffer: (() -> Unit)?,
) {
    val all = vm.groups.flatMap { it.items }
    val off = all.count { !it.available }
    val q = vm.query.trim()
    val shown = when {
        q.isNotEmpty() -> all.filter { it.name.contains(q, ignoreCase = true) }
        vm.onlyOff -> all.filter { !it.available }
        open != null -> vm.groups.firstOrNull { it.name == open.name }?.items ?: open.items
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
        // ── ١ · البحث — حقلٌ ممتلئٌ بلا إطار ──
        item(span = full) {
            OutlinedTextField(
                value = vm.query,
                onValueChange = { vm.query = it.take(60) },
                placeholder = { Text(stringResource(com.rahalgo.ui.R.string.ow_search)) },
                leadingIcon = {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_search),
                        contentDescription = null,
                        tint = Rahal.colors.inkMuted,
                    )
                },
                trailingIcon = if (vm.query.isNotEmpty()) {
                    {
                        IconButton(onClick = { vm.query = "" }) {
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
                Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                StatBox(
                    icon = com.rahalgo.ui.R.drawable.ic_store,
                    label = stringResource(R.string.menu_stat_total),
                    value = all.size,
                    tint = Rahal.colors.brand,
                    selected = false,
                    modifier = Modifier.weight(1f),
                ) { vm.back() }
                StatBox(
                    icon = R.drawable.ic_block,
                    label = stringResource(R.string.menu_stat_off),
                    value = off,
                    tint = if (off > 0) Rahal.colors.danger else Rahal.colors.inkMuted,
                    selected = vm.onlyOff,
                    modifier = Modifier.weight(1f),
                ) { if (vm.onlyOff) vm.back() else vm.showUnavailable() }
            }
        }
        // ── ٣ · الأدواتُ الثلاث — والموقوفُ لا يرى أدواتٍ لا يملكها (B6) ──
        if (!vm.suspended) {
            item(span = full) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    ToolTile(com.rahalgo.ui.R.drawable.ic_custom, stringResource(com.rahalgo.ui.R.string.presets_title), Modifier.weight(1f)) { vm.presets.show() }
                    ToolTile(R.drawable.ic_banknotes, stringResource(com.rahalgo.ui.R.string.bulk_title), Modifier.weight(1f)) { vm.bulk.show() }
                    ToolTile(com.rahalgo.ui.R.drawable.ic_cart, stringResource(R.string.menu_tool_drinks), Modifier.weight(1f)) { vm.drinks.show() }
                }
            }
            // ── ٤ · «أضف صنفاً» و«أنشئ عرضاً» ──
            item(span = full) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
                    RahalButton(
                        onClick = {
                            // **ومن داخل قسمٍ يُضاف إليه** — فلا يُعيد اختيارَ ما هو فيه.
                            when {
                                open != null -> vm.newItem(open.name)
                                onAddItem != null -> onAddItem()
                                else -> vm.newItem()
                            }
                        },
                        modifier = Modifier
                            .weight(1f)
                            .height(52.dp),
                    ) {
                        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_plus), contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(Rahal.space.sm))
                        Text(stringResource(R.string.item_add), fontWeight = FontWeight.Bold, maxLines = 1)
                    }
                    if (onOffer != null) {
                        RahalOutlineButton(
                            onClick = onOffer,
                            modifier = Modifier
                                .weight(1f)
                                .height(52.dp),
                        ) {
                            Icon(painterResource(com.rahalgo.ui.R.drawable.ic_offer), contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(Modifier.width(Rahal.space.sm))
                            Text(stringResource(R.string.menu_create_offer), fontWeight = FontWeight.Bold, maxLines = 1)
                        }
                    }
                }
            }
        }
        // ── ٥ · رقائقُ الأقسام — «الكل» ثمّ كلُّ قسمٍ بعدده ──
        if (vm.groups.isNotEmpty() && q.isEmpty()) {
            item(span = full) {
                Row(
                    Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                ) {
                    SectionChip(stringResource(R.string.menu_all), all.size, open == null && !vm.onlyOff) { vm.back() }
                    vm.groups.forEach { g ->
                        SectionChip(g.name, g.items.size, open?.name == g.name && !vm.onlyOff) { vm.openGroup(g) }
                    }
                }
            }
        }
        // ── ٦ · الأصناف — بطاقتان في كلّ سطر ──
        when {
            vm.groups.isEmpty() -> item(span = full) { Empty(stringResource(R.string.menu_empty)) }
            shown.isEmpty() -> item(span = full) {
                // **والمرشّحُ «غير متوفر» بلا بحثٍ له رسالتُه** (ملاحظةُ المالك ٢٠٢٦-١٠-٢٩).
                Text(
                    stringResource(if (vm.onlyOff && q.isEmpty()) R.string.menu_off_none else R.string.menu_search_none),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(vertical = Rahal.space.md),
                )
            }
            else -> items(shown, key = { it.id }) { item ->
                ItemCard(
                    item = item,
                    media = media,
                    suspended = vm.suspended,
                    onToggle = { vm.toggle(item.id) },
                    onEdit = { vm.editItem(item) },
                )
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
            Modifier
                .size(40.dp)
                .clip(Rahal.shape.pill)
                .background(tint.copy(alpha = 0.12f)),
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
        Text(
            label,
            style = MaterialTheme.typography.labelMedium,
            color = Rahal.colors.ink,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
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
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٩: «زرُّ متوفر/غير متوفر لكلّ صنف».) **والبطاقةُ كلُّها
 * تفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر. وغيرُ المتوفّر باهت.**
 */
@Composable
private fun ItemCard(
    item: MenuItem,
    media: (String?) -> String?,
    suspended: Boolean,
    onToggle: () -> Unit,
    onEdit: () -> Unit,
) {
    val dim = if (item.available) 1f else 0.5f
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg)
            .clickable(onClick = onEdit),
    ) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(1.15f)
                .alpha(dim),
        ) {
            RemoteImage(
                url = media(item.imageThumbUrl ?: item.imageUrl),
                name = item.name.ifBlank { "?" },
                modifier = Modifier.fillMaxSize(),
            )
            if (!item.approved) {
                Text(
                    stringResource(R.string.mn_pending_review),
                    color = Rahal.colors.onBrand,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    modifier = Modifier
                        .align(Alignment.TopStart)
                        .padding(Rahal.space.sm)
                        .clip(Rahal.shape.pill)
                        .background(Rahal.colors.accent)
                        .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
                )
            }
        }
        Column(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        ) {
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
                    item.description.trim().ifBlank {
                        item.modifiers.map { it.name }.filter { it.isNotBlank() }.joinToString(" · ")
                    },
                    minLines = 2,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    // **والدولاريُّ بدولاره وليرته** (٢٠٢٦-١٠-١٢).
                    com.rahalgo.ui.menu.priceLine(item.price, item.merchantPriceUsd),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (!item.approved && item.reviewNote.isNotBlank()) {
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
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    stringResource(
                        if (item.available) R.string.mn_available else R.string.mn_unavailable,
                    ),
                    color = if (item.available) Rahal.colors.success else Rahal.colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelMedium,
                    maxLines = 1,
                    modifier = Modifier.weight(1f),
                )
                // **والموقوفُ لا يبدّل الإتاحة** (B6) — المفتاحُ يُعطَّل.
                Switch(
                    checked = item.available,
                    enabled = !suspended,
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

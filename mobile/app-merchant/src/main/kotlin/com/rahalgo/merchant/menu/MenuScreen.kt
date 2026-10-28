package com.rahalgo.merchant.menu

import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
fun MenuScreen(vm: MenuViewModel) {
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
    BackHandler(enabled = open != null) { vm.back() }
    ItemsList(vm, media, open)
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأصناف — عنوانٌ وبحثٌ وأدواتٌ ورقائقُ أقسامٍ ثمّ القائمة** (التصميمُ المعتمد ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والرقاقةُ هي قسمُ السوق نفسُه** (`vm.open`) — «الكل» يعرض كلَّ الأصناف،
 * والقسمُ يعرض ما فيه مع زرّ «أضف صنفاً» فيه. **وقائمةٌ واحدةٌ تمرّر** — لا
 * تمريران متداخلان.
 */
@Composable
private fun ItemsList(vm: MenuViewModel, media: (String?) -> String?, open: MenuGroup?) {
    val all = vm.groups.flatMap { it.items }
    val off = all.count { !it.available }
    val q = vm.query.trim()
    val shown = when {
        q.isNotEmpty() -> all.filter { it.name.contains(q, ignoreCase = true) }
        open != null -> vm.groups.firstOrNull { it.name == open.name }?.items ?: open.items
        else -> all
    }
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = Rahal.space.lg, end = Rahal.space.lg, top = Rahal.space.lg, bottom = Rahal.space.xl),
    ) {
        item {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(com.rahalgo.ui.R.string.mn_items),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineSmall,
                    modifier = Modifier.weight(1f),
                )
                if (all.isNotEmpty()) {
                    Text(
                        stringResource(R.string.menu_count, all.size, off),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
            Spacer(Modifier.height(Rahal.space.md))
        }
        // ── البحث — حقلٌ ممتلئٌ بلا إطار ──
        item {
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
            Spacer(Modifier.height(Rahal.space.md))
        }
        // ── الأدواتُ الثلاث — خياراتُ الأصناف · تغييرُ الأسعار · المشروبات ──
        // **والموقوفُ لا يرى أدواتٍ لا يملكها** (B6).
        if (!vm.suspended) {
            item {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    ToolTile(com.rahalgo.ui.R.drawable.ic_custom, stringResource(com.rahalgo.ui.R.string.presets_title), Modifier.weight(1f)) { vm.presets.show() }
                    ToolTile(com.rahalgo.ui.R.drawable.ic_offer, stringResource(com.rahalgo.ui.R.string.bulk_title), Modifier.weight(1f)) { vm.bulk.show() }
                    ToolTile(com.rahalgo.ui.R.drawable.ic_cart, stringResource(R.string.menu_tool_drinks), Modifier.weight(1f)) { vm.drinks.show() }
                }
                Spacer(Modifier.height(Rahal.space.md))
            }
        }
        // ── رقائقُ الأقسام ──
        if (vm.groups.isNotEmpty() && q.isEmpty()) {
            item {
                Row(
                    Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                ) {
                    SectionChip(stringResource(R.string.menu_all), open == null) { vm.back() }
                    vm.groups.forEach { g ->
                        SectionChip("${g.name} ${g.items.size}", open?.name == g.name) { vm.openGroup(g) }
                    }
                }
                Spacer(Modifier.height(Rahal.space.sm))
            }
        }
        // **وإضافةُ صنفٍ من داخل قسمه** — فلا يُعيد اختيارَ ما هو فيه.
        if (open != null && q.isEmpty() && !vm.suspended) {
            item {
                RahalOutlineButton(onClick = { vm.newItem(open.name) }, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.item_add))
                }
                Spacer(Modifier.height(Rahal.space.xs))
            }
        }
        when {
            vm.groups.isEmpty() -> item { Empty(stringResource(R.string.menu_empty)) }
            shown.isEmpty() -> item {
                Text(
                    stringResource(R.string.menu_search_none),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(vertical = Rahal.space.md),
                )
            }
            else -> items(shown, key = { it.id }) { item ->
                ItemRow(
                    item = item,
                    media = media,
                    suspended = vm.suspended,
                    onToggle = { vm.toggle(item.id) },
                    onEdit = { vm.editItem(item) },
                )
                HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
            }
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

/** **رقاقةُ قسم** — المختارةُ داكنةٌ بلون اللوح، والباقيةُ مؤطَّرة. */
@Composable
private fun SectionChip(text: String, selected: Boolean, onClick: () -> Unit) {
    Text(
        text,
        color = if (selected) Rahal.colors.onPanel else Rahal.colors.ink,
        fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
        style = MaterialTheme.typography.bodyMedium,
        maxLines = 1,
        modifier = Modifier
            .clip(Rahal.shape.pill)
            .background(if (selected) Rahal.colors.panel else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (selected) Rahal.colors.panel else Rahal.colors.line, Rahal.shape.pill)
            .clickable(onClick = onClick)
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.sm),
    )
}

@Composable
private fun ItemRow(
    item: MenuItem,
    media: (String?) -> String?,
    suspended: Boolean,
    onToggle: () -> Unit,
    onEdit: () -> Unit,
) {
    // ══════════════════════════════════════════════════════════════════
    // **صنفٌ بصورته وخياراته وسعره ومفتاحِ توفّره**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٩: «زرُّ متوفر/غير متوفر لكلّ صنف».) **والسطرُ كلُّه
    // يفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر.**
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onEdit)
            .padding(vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RemoteImage(
            url = media(item.imageThumbUrl ?: item.imageUrl),
            name = item.name.ifBlank { "?" },
            modifier = Modifier
                .size(60.dp)
                .clip(Rahal.shape.md),
        )
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                item.name,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                fontWeight = FontWeight.Bold,
                color = if (item.available) Rahal.colors.ink else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyLarge,
            )
            Text(
                item.modifiers.map { it.name }.filter { it.isNotBlank() }.joinToString(" · ")
                    .ifBlank { stringResource(R.string.menu_no_options) },
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            Text(
                // **والدولاريُّ بدولاره وليرته** (٢٠٢٦-١٠-١٢).
                com.rahalgo.ui.menu.priceLine(item.price, item.merchantPriceUsd),
                color = if (item.available) Rahal.colors.brand else Rahal.colors.inkMuted,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            if (!item.approved) {
                Text(
                    stringResource(R.string.mn_pending_review),
                    color = Rahal.colors.accent,
                    style = MaterialTheme.typography.labelSmall,
                )
                if (item.reviewNote.isNotBlank()) {
                    Text(
                        item.reviewNote,
                        color = Rahal.colors.danger,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
        Spacer(Modifier.width(Rahal.space.sm))
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
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
            Text(
                stringResource(
                    if (item.available) R.string.mn_available else R.string.mn_unavailable,
                ),
                color = if (item.available) Rahal.colors.success else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
        }
    }
}

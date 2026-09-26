package com.rahalgo.merchant.menu

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
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.items
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
            onCancel = { vm.cancelEdit() },
            onDelete = { vm.deleteItem(draft.itemId) },
        )
        return
    }

    val open = vm.open
    // **والرجوعُ يغلق القسمَ لا التطبيق** — من فتح قسماً ثمّ ضغط رجوعاً
    // **يتوقّع شبكتَه لا أن يخرج.**
    BackHandler(enabled = open != null) { vm.back() }

    if (open == null) {
        Grid(vm, media)
        return
    }

    Screen {
        RahalTextButton(onClick = { vm.back() }) {
            Text(stringResource(R.string.menu_back), color = Rahal.colors.brand)
        }
        ScreenTitle(open.name, stringResource(R.string.menu_hint))
        Spacer(Modifier.height(6.dp))
        // **وإضافةُ صنفٍ من داخل قسمه** — **فلا يُعيد اختيارَ ما هو فيه.**
        // **والموقوفُ لا يضيف** (B6) — يُخفى الزرُّ لا يُعرَض معطّلاً بلا سبب.
        if (!vm.suspended) {
            RahalButton(onClick = { vm.newItem(open.name) }) {
                Text(stringResource(R.string.item_add))
            }
            Spacer(Modifier.height(10.dp))
        }
        Card {
            open.items.forEach { item ->
                ItemRow(
                    item = item,
                    media = media,
                    suspended = vm.suspended,
                    onToggle = { vm.toggle(item.id) },
                    onEdit = { vm.editItem(item) },
                )
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **شبكةُ الأقسام** — وما ليس فيه صنفٌ لا يُعرض.
 *
 * # وهي التي تمرّر — لا عمودٌ يحملها
 *
 * **كانت داخلَ `Screen` الذي يمرّر بارتفاعٍ مقطوع** (٦٢٠) — **فتمريران
 * متداخلان**: يسحب فتتحرّك الشبكةُ أو الصفحةُ ولا يعرف أيَّهما، **وتُقطع
 * الصفوفُ عند الحدّ فتُرى تسميةٌ بلا صورتها.** (قِيس على الجهاز
 * ٢٠٢٦-٠٨-٢٣.)
 *
 * **والعنوانُ صار أوّلَ عنصرٍ فيها** — يمرّ معها ولا يثبت فوقها.
 */
@Composable
private fun Grid(vm: MenuViewModel, media: (String?) -> String?) {
    if (vm.groups.isEmpty()) {
        Screen {
            ScreenTitle(
                stringResource(R.string.menu_title),
                stringResource(R.string.menu_sections_hint),
            )
            Empty(stringResource(R.string.menu_empty))
        }
        return
    }

    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 14.dp, end = 14.dp, top = 4.dp, bottom = 24.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item(span = { GridItemSpan(maxLineSpan) }) {
            Column {
                ScreenTitle(
                    stringResource(R.string.menu_title),
                    stringResource(R.string.menu_sections_hint),
                )
                Spacer(Modifier.height(6.dp))
            }
        }
        items(vm.groups, key = { it.name }) { g ->
            Column(
                Modifier
                    .clip(Rahal.shape.md)
                    .clickable { vm.openGroup(g) },
            ) {
                RemoteImage(
                    url = media(g.imageUrl),
                    name = g.name,
                    modifier = Modifier.fillMaxWidth().aspectRatio(4f / 3f).clip(Rahal.shape.sm),
                )
                Spacer(Modifier.height(6.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        g.name,
                        modifier = Modifier.weight(1f),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    // **وعددُ ما فيه يُقال** — **وقسمٌ بلا عددٍ يُفتح ليُعرف
                    // ما فيه**، وذلك نقرةٌ زائدةٌ في كلّ مرّة.
                    Text(
                        g.items.size.toString(),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.labelMedium,
                    )
                }
            }
        }
    }
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
    // **صنفٌ بصورته وسعره ومفتاحِ توفّره**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٩: «الأصناف يجب أن تُعرض بالقسم بشكل صحيح
    //  مثل السوق، وليس قائمةً بدون أيّ صورة… ويجب أن يكون فيه زرُّ
    //  متوفر/غير متوفر لكلّ صنف».)
    //
    // **وكان صفّاً بنقطةٍ واسم** — لا صورةَ ولا سعر.
    //
    // **وصاحبُ المتجر يرى قائمتَه كما يراها زبونُه أو لا يراها**: من لم
    // يرَ صورةَ صنفه لا يعرف أنّها ناقصةٌ أو مقلوبةٌ أو لصنفٍ آخر.
    //
    // **والنقطةُ قطرُها اثنتا عشرةَ نقطة** تتبدّل بين لونين — **من رآها
    // لم يعرف أنّها تُضغط**، ومن ضغطها لم يتيقّن أنّ شيئاً وقع.
    //
    // **ونفادُ الصنف يقع في ذروة الطلب** — والمفتاحُ يُقلب بإبهامٍ واحدٍ
    // وصاحبُه واقفٌ على النار.
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onEdit)
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RemoteImage(
            url = media(item.imageThumbUrl ?: item.imageUrl),
            name = item.name.ifBlank { "?" },
            modifier = Modifier
                .size(52.dp)
                .clip(Rahal.shape.sm),
        )
        Spacer(Modifier.size(10.dp))
        Column(Modifier.weight(1f)) {
            Text(
                item.name,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                color = if (item.available) Rahal.colors.ink else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyLarge,
            )
            Text(
                money(item.price),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
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
        Spacer(Modifier.size(8.dp))
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            // **والموقوفُ لا يبدّل الإتاحة** (B6) — المفتاحُ يُعطَّل.
            Switch(
                checked = item.available,
                enabled = !suspended,
                onCheckedChange = { onToggle() },
            )
            Text(
                stringResource(
                    if (item.available) R.string.mn_available else R.string.mn_unavailable,
                ),
                color = if (item.available) Rahal.colors.brand else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
        }
    }
}

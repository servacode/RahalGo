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
    // **والشبكةُ نفسُها عند المندوب** — `ui.menu.MenuItemsGrid` (طلبُ المالك ٢٠٢٦-١٠-٢٩).
    val byId = vm.groups.flatMap { it.items }.associateBy { it.id }
    com.rahalgo.ui.menu.MenuItemsGrid(
        groups = vm.groups.map { g ->
            com.rahalgo.ui.menu.GridGroup(
                g.name,
                g.items.map { item ->
                    com.rahalgo.ui.menu.GridItem(
                        id = item.id,
                        name = item.name,
                        description = item.description.trim().ifBlank {
                            item.modifiers.map { it.name }.filter { it.isNotBlank() }.joinToString(" · ")
                        },
                        imageUrl = media(item.imageThumbUrl ?: item.imageUrl),
                        // **والدولاريُّ بدولاره وليرته** (٢٠٢٦-١٠-١٢).
                        price = com.rahalgo.ui.menu.priceLine(item.price, item.merchantPriceUsd),
                        available = item.available,
                        approved = item.approved,
                        reviewNote = if (item.approved) "" else item.reviewNote,
                    )
                },
            )
        },
        query = vm.query,
        onQuery = { vm.query = it },
        openName = open?.name,
        onlyOff = vm.onlyOff,
        onAll = { vm.back() },
        onOnlyOff = { vm.showUnavailable() },
        onOpen = { name -> vm.groups.firstOrNull { it.name == name }?.let(vm::openGroup) },
        // **والموقوفُ لا يرى أدواتٍ لا يملكها** (B6).
        showTools = !vm.suspended,
        onPresets = { vm.presets.show() },
        onBulk = { vm.bulk.show() },
        onDrinks = { vm.drinks.show() },
        addLabel = stringResource(R.string.item_add),
        onAdd = {
            // **ومن داخل قسمٍ يُضاف إليه** — فلا يُعيد اختيارَ ما هو فيه.
            when {
                open != null -> vm.newItem(open.name)
                onAddItem != null -> onAddItem()
                else -> vm.newItem()
            }
        },
        offerLabel = stringResource(R.string.menu_create_offer),
        onOffer = onOffer,
        emptyText = stringResource(R.string.menu_empty),
        toggleEnabled = !vm.suspended,
        onToggle = { vm.toggle(it.id) },
        onEdit = { g -> byId[g.id]?.let(vm::editItem) },
    )
}

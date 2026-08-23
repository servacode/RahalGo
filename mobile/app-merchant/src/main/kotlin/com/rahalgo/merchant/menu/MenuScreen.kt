package com.rahalgo.merchant.menu

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
        ItemEditor(vm, draft, media)
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
        RahalButton(onClick = { vm.newItem(open.name) }) {
            Text(stringResource(R.string.item_add))
        }
        Spacer(Modifier.height(10.dp))
        Card {
            open.items.forEach { item ->
                ItemRow(
                    item = item,
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
private fun ItemRow(item: MenuItem, onToggle: () -> Unit, onEdit: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            // ══════════════════════════════════════════════════════════
            // **والسطرُ يفتح المحرّر، والنقطةُ تبدّل التوفّر**
            // ══════════════════════════════════════════════════════════
            //
            // **فعلان في سطرٍ واحدٍ يحتاجان هدفين منفصلين** — ولو كان
            // السطرُ كلُّه يبدّل التوفّرَ **لَما وُجد سبيلٌ إلى التعديل
            // إلّا زرٌّ ثالث.**
            //
            // **والنقطةُ هدفٌ صغير** — فوُسّعت بحشوةٍ حولَها (٤٤ نقطة،
            // أدنى ما يوصي به أندرويد للمس).
            .clickable(onClick = onEdit)
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // **ونقطةُ لونٍ تُقرأ قبل النصّ** — من مرّ بعينه على عشرين صنفاً
        // يرى النواقصَ بلا قراءة.
        Box(
            Modifier
                .size(44.dp)
                .clip(CircleShape)
                .clickable(onClick = onToggle),
            contentAlignment = Alignment.Center,
        ) {
            Box(
                Modifier
                    .size(12.dp)
                    .clip(CircleShape)
                    .background(if (item.available) Rahal.colors.brand else Rahal.colors.line),
            )
        }
        Column(Modifier.weight(1f)) {
            Text(
                item.name,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                color = if (item.available) Rahal.colors.ink else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyLarge,
            )
            Row {
                // **واللفظُ يقول الحالَ لا الفعل** — توحيدٌ في التطبيقات
                // الأربعة والويب: «متوفر / غير متوفر».
                Text(
                    stringResource(
                        if (item.available) R.string.menu_available else R.string.menu_unavailable,
                    ),
                    color = if (item.available) Rahal.colors.brand else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
                )
                // **وما لم يُراجَع بعدُ يُقال ويُقال سببُه** — **وصنفٌ لا
                // يظهر للزبون ولا يُعرف لماذا يُقرأ عطباً في المنصّة.**
                if (!item.approved) {
                    Spacer(Modifier.size(8.dp))
                    Text(
                        stringResource(R.string.menu_pending),
                        color = Rahal.colors.accent,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
            }
            if (!item.approved && item.reviewNote.isNotBlank()) {
                Text(
                    item.reviewNote,
                    color = Rahal.colors.danger,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
        // **والسعرُ سعرُ شرائه** — **والمحرّكُ يحجب سعرَ البيع عنه** عمداً:
        // «هو يضع سعرَه ويقبض عليه».
        Text(
            money(item.price),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الصنف — كنموذج الويب حرفاً بحرف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «بنفس طريقة الويب: إضافةُ صورةٍ وتعديلُ
 *  معلوماتٍ ومتوفّر/غير متوفّر وحذف».)
 *
 * # والسعرُ سعرُ الشراء لا البيع
 *
 * **ولو حُرِّر سعرُ البيع لَضاع الهامشُ في أوّل تعديل**: يفتح الصنفَ فيرى
 * رقماً أكبرَ من سعره، **فيحفظ فيصير سعرُ شرائه ذاك**، ويُضاف عليه
 * الهامشُ من جديد — **ورقمٌ يرتفع بكلّ فتحةٍ للنافذة.**
 *
 * # ولا مفتاحَ توفّرٍ هنا
 *
 * **يُقلب من نقطة السطر وحدَها** — **وحقلٌ يُكتب من موضعين يمحو أحدُهما
 * ما فعله الآخر**: من أوقف صنفاً ثمّ عدّل اسمَه أعاده متوفّراً.
 */
@Composable
private fun ItemEditor(vm: MenuViewModel, d: Draft, media: (String?) -> String?) {
    val pick = rememberImagePicker { bytes -> vm.pickImage(bytes) }
    var confirmDelete by remember { mutableStateOf(false) }

    Screen {
        ScreenTitle(
            stringResource(if (d.itemId.isEmpty()) R.string.item_new else R.string.item_edit),
            "",
        )

        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(d.imageThumb),
                name = d.name.ifBlank { "?" },
                modifier = Modifier
                    .size(84.dp)
                    .clip(Rahal.shape.sm)
                    .clickable(enabled = !vm.busy) { pick() },
            )
            Spacer(Modifier.size(12.dp))
            Column {
                RahalTextButton(onClick = { pick() }, enabled = !vm.busy) {
                    Text(stringResource(R.string.item_pick_image), color = Rahal.colors.brand)
                }
                if (d.imageThumb != null || !d.imageMediaId.isNullOrEmpty()) {
                    RahalTextButton(onClick = { vm.clearImage() }, enabled = !vm.busy) {
                        Text(
                            stringResource(R.string.item_clear_image),
                            color = Rahal.colors.inkMuted,
                        )
                    }
                }
            }
        }

        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = d.name,
            onValueChange = { v -> vm.editDraft { it.copy(name = v) } },
            label = { Text(stringResource(R.string.item_name)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = d.description,
            onValueChange = { v -> vm.editDraft { it.copy(description = v) } },
            label = { Text(stringResource(R.string.item_desc)) },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = d.price,
            onValueChange = { v -> vm.editDraft { it.copy(price = v.filter(Char::isDigit)) } },
            label = { Text(stringResource(R.string.item_price)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Text(
            stringResource(R.string.item_price_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )

        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.item_section),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelLarge,
        )
        Spacer(Modifier.height(4.dp))
        Card {
            vm.sections.forEach { sec ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clickable { vm.editDraft { it.copy(platformSectionId = sec.id) } }
                        .padding(vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        Modifier
                            .size(16.dp)
                            .clip(CircleShape)
                            .background(
                                if (sec.id == d.platformSectionId) Rahal.colors.brand
                                else Rahal.colors.line,
                            ),
                    )
                    Spacer(Modifier.size(10.dp))
                    Text(sec.name, style = MaterialTheme.typography.bodyMedium)
                }
            }
        }

        Spacer(Modifier.height(16.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(onClick = { vm.saveItem() }, enabled = !vm.busy) {
                Text(stringResource(R.string.save))
            }
            RahalTextButton(onClick = { vm.cancelEdit() }, enabled = !vm.busy) {
                Text(stringResource(R.string.cancel))
            }
        }

        if (d.itemId.isNotEmpty()) {
            Spacer(Modifier.height(20.dp))
            if (!confirmDelete) {
                RahalTextButton(onClick = { confirmDelete = true }, enabled = !vm.busy) {
                    Text(stringResource(R.string.item_delete), color = Rahal.colors.danger)
                }
            } else {
                Card {
                    Text(
                        stringResource(R.string.item_delete_ask),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Spacer(Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        RahalButton(
                            onClick = { vm.deleteItem(d.itemId) },
                            enabled = !vm.busy,
                            tone = com.rahalgo.ui.Tone.Danger,
                        ) { Text(stringResource(R.string.item_delete)) }
                        RahalTextButton(onClick = { confirmDelete = false }) {
                            Text(stringResource(R.string.cancel))
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

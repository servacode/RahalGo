package com.rahalgo.rep.menu

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.MenuItem
import com.rahalgo.shared.rep.MenuSection
import com.rahalgo.shared.rep.ModifierGroup
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.money
import com.rahalgo.ui.rememberImagePicker
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأصناف — يبنيها المندوبُ نيابةً عن عميله**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نفس الفورم الموجود عند مدير المنصّة
 *  والموجود عند المتجر موجودٌ عند المندوب… باللوحة والتطبيق أيضا».)
 *
 * # وهي نسخةُ `MenuManager` في الويب — لا شاشةٌ تشبهها
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لا يجوز أن يشعر الشخصُ بالفرق بين الويب
 *  والتطبيق أصلاً — أيَّهما يفتح يكون العملُ موحّدا».)
 *
 * **فالكلماتُ من `shared.menuEditor` حرفا**، **وترتيبُ الحقول ترتيبَه**
 * (الاسم · السعر · قسم المنصة · الوصف · الصورة)، **والإتاحةُ زرٌّ في
 * السطر لا مفتاحاً في النموذج** — كما هي هناك.
 *
 * **ومن غيّر أحدَهما وحدَه أعاد الفرق.**
 *
 * # وقسمُ المنصة ليس شرطاً — إنّما يُحذَّر منه
 *
 * **الويبُ يحفظ بلا قسمٍ ويكتب تحذيراً**، **ومنعُ الحفظ يوقف المندوبَ
 * في السوق أمام صاحب متجرٍ ينتظر** — والصنفُ يبقى قابلاً للطلب من صفحة
 * متجره، ويراه الأدمنُ بلا قسمٍ فيصنّفه.
 *
 * # ولا زرَّ لإنشاء قسم
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٨: «الأقسامُ الإدارةُ هي التي تضعها، والمتجرُ
 *  أو المندوبُ يختار منتجَه بأيّ قسمٍ سينزل».)
 */
@Composable
fun MenuScreen(vm: MenuViewModel) {
    BackHandler { if (vm.editing != null) vm.cancelEdit() else vm.close() }

    if (vm.editing != null) {
        ItemForm(vm)
        return
    }

    val list = vm.sections
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load() }
        return
    }

    Screen {
        ScreenTitle(stringResource(R.string.mn_title), stringResource(R.string.mn_hint))

        RahalButton(
            onClick = { vm.newItem() },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.mn_add_item)) }

        Spacer(Modifier.height(12.dp))

        if (list.isEmpty()) {
            Empty(stringResource(R.string.mn_empty))
            return@Screen
        }

        list.forEach { sec -> SectionBlock(sec, vm) }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **قسمٌ بأصنافه.**
 *
 * **واسمُه يُعرض ولا يُعدَّل** — تزرعه الإدارةُ ويراه المتاجرُ كلُّهم
 * سواء، **ومن سمّاه بيده رأى سوقاً لا يشبه سوقَ جاره.**
 */
@Composable
private fun SectionBlock(sec: MenuSection, vm: MenuViewModel) {
    SectionTitle(sec.name + "  (" + sec.items.size + ")")
    // **وزرُّ القسم اسمُه اسمُ الزرّ الأعلى** — كما في الويب: `addItem`
    // في الترويسة وفي كلّ قسم.
    RahalTextButton(onClick = { vm.newItem(sec.id) }) {
        Text(stringResource(R.string.mn_add_item))
    }
    if (sec.items.isEmpty()) {
        Text(
            text = stringResource(R.string.mn_no_items),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
    }
    sec.items.forEach { ItemRow(it, vm) }
    Spacer(Modifier.height(10.dp))
    HorizontalDivider()
    Spacer(Modifier.height(6.dp))
}

/**
 * **سطرُ صنف — بصورته وسعره وحاله.**
 *
 * **و«نافد» تُقال بلونٍ لا بغياب**: صنفٌ يختفي حين يُطفأ يُضاف ثانيةً
 * وثالثة، **فتمتلئ القائمةُ نسخاً من الشيء الواحد.**
 *
 * **والسعرُ الثاني لا يُعرض إلّا إن كان أعلى** — كما في الويب حرفا:
 * **ومساواةُ الرقمين تعني «لا هامش»، وعرضُهما متساويين يسأل صاحبَه عن
 * فرقٍ لا وجودَ له.**
 */
@Composable
private fun ItemRow(item: MenuItem, vm: MenuViewModel) {
    val context = LocalContext.current
    Spacer(Modifier.height(8.dp))
    Card {
        Row(
            Modifier.fillMaxWidth().clickable { vm.editItem(item) },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            RemoteImage(
                url = Backend.of(context).media(item.imageThumbURL),
                name = item.name,
                modifier = Modifier.size(46.dp).clip(Rahal.shape.sm),
            )
            Spacer(Modifier.size(10.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = item.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = money(if (item.merchantPrice > 0) item.merchantPrice else item.price),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    if (item.merchantPrice > 0 && item.price > item.merchantPrice) {
                        Spacer(Modifier.size(6.dp))
                        Text(
                            text = "← " + money(item.price),
                            color = Rahal.colors.success,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
            }
        }
        // **ومجموعاتُ المُعدِّلات تُلمَح في السطر** — كما في الويب:
        // اسمُها وعددُ خياراتها. **ومن لا يراها لا يعرف أنّ الصنفَ يُطلب
        // بأحجامٍ أو بإضافات، فيبني ثانياً مثلَه بحجمٍ آخر.**
        if (item.modifiers.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                item.modifiers.forEach { g ->
                    Chip(g.name + " (" + g.options.size + ")", Rahal.colors.inkMuted)
                }
            }
        }
        Spacer(Modifier.height(6.dp))
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            // **والمعلَّقُ يحلّ محلّ حال التوفّر** — كما في الويب:
            // **صنفٌ لم يُنشر بعدُ لا يعني الزبونَ أمتوفّرٌ هو أم نافد.**
            if (!item.approved) {
                Chip(
                    stringResource(
                        if (item.reviewNote.isNotEmpty()) R.string.mn_rejected
                        else R.string.mn_pending_review,
                    ),
                    Rahal.colors.accent,
                )
            } else {
                Chip(
                    stringResource(
                        if (item.available) R.string.mn_available else R.string.mn_unavailable,
                    ),
                    if (item.available) Rahal.colors.success else Rahal.colors.inkMuted,
                )
            }
            // **و«خارجَ الدوام» يقوله الوقتُ لا صاحبُ المتجر** — ومن
            // خلط بينهما جعله يطفئ أصنافَه كلَّ ليلةٍ ويشعلها كلَّ صباح.
            if (item.sourceClosed) {
                Chip(stringResource(R.string.mn_closed_now), Rahal.colors.accent)
            }
            Spacer(Modifier.weight(1f))
            // **والإتاحةُ زرٌّ لا مفتاح** — كما في الويب: **زرٌّ يقول ما
            // سيقع، ومفتاحٌ يقول ما هو قائم**، والخلطُ بينهما يجعل من
            // يقرأ «متوفر» على مفتاحٍ مرفوعٍ يظنّ أنّه يُطفئه بالرفع.
            RahalOutlineButton(onClick = { vm.toggleAvailable(item) }, enabled = !vm.busy) {
                Text(
                    stringResource(
                        if (item.available) R.string.mn_mark_unavailable
                        else R.string.mn_mark_available,
                    ),
                )
            }
        }
        if (!item.approved && item.reviewNote.isNotEmpty()) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = item.reviewNote,
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/**
 * **نموذجُ الصنف — بترتيب حقول الويب نفسِه.**
 *
 * الاسم · السعر · قسم المنصة · الوصف · الصورة.
 *
 * **ولا حقلَ إتاحةٍ فيه** — الويبُ يقلبها من السطر لا من النموذج،
 * **وحقلٌ في موضعين يفترق أحدُهما عن الآخر.**
 */
@Composable
private fun ItemForm(vm: MenuViewModel) {
    val d = vm.editing ?: return
    val context = LocalContext.current
    val pick = rememberImagePicker { bytes -> vm.pickImage(bytes) }
    var confirmDelete by remember { mutableStateOf(false) }

    Screen {
        ScreenTitle(
            stringResource(if (d.isNew) R.string.mn_add_item else R.string.mn_edit_item),
            vm.merchantName,
        )

        Card {
            OutlinedTextField(
                value = d.name,
                onValueChange = { v -> vm.editDraft { it.copy(name = v) } },
                label = { Text(stringResource(R.string.mn_item_name)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(
                value = d.price,
                // **والأرقامُ وحدَها** — لوحةُ المفاتيح تسمح بغيرها.
                onValueChange = { v ->
                    vm.editDraft { it.copy(price = v.filter { c -> c.isDigit() }) }
                },
                label = { Text(stringResource(R.string.mn_price)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **وقسمُ المنصة** — يُختار ولا يُنشأ
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        SectionTitle(stringResource(R.string.mn_platform_section))
        Card {
            // **و«بلا قسم» خيارٌ صريحٌ لا فراغ** — كما في الويب:
            // **ومن لم يجد ما يختاره لا يعرف أنّ له أن يترك.**
            SectionChoice(
                label = stringResource(R.string.mn_no_platform_section),
                chosen = d.platformSectionID.isEmpty(),
                onPick = { vm.editDraft { it.copy(platformSectionID = "") } },
            )
            vm.platformSections.forEach { ps ->
                SectionChoice(
                    label = ps.name,
                    chosen = d.platformSectionID == ps.id,
                    onPick = { vm.editDraft { it.copy(platformSectionID = ps.id) } },
                )
            }
            if (d.platformSectionID.isEmpty()) {
                Spacer(Modifier.height(4.dp))
                Text(
                    text = stringResource(R.string.mn_no_platform_section_hint),
                    color = Rahal.colors.accent,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }

        Spacer(Modifier.height(12.dp))
        Card {
            OutlinedTextField(
                value = d.description,
                onValueChange = { v -> vm.editDraft { it.copy(description = v) } },
                label = { Text(stringResource(R.string.mn_desc)) },
                modifier = Modifier.fillMaxWidth(),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والصورة**
        // ══════════════════════════════════════════════════════════════
        //
        // **والمندوبُ واقفٌ عند الصنف** — **فصورةٌ تُلتقط الآن خيرٌ من
        // صورةٍ يَعِد بها صاحبُ المتجر.**
        Spacer(Modifier.height(12.dp))
        Card {
            Row(verticalAlignment = Alignment.CenterVertically) {
                RemoteImage(
                    url = Backend.of(context).media(d.imageThumb),
                    name = d.name.ifEmpty { "؟" },
                    modifier = Modifier.size(64.dp).clip(Rahal.shape.sm),
                )
                Spacer(Modifier.size(10.dp))
                Column {
                    RahalOutlineButton(onClick = pick, enabled = !vm.busy) {
                        Text(stringResource(R.string.mn_item_image))
                    }
                    // **والإزالةُ صريحةٌ** — الفراغُ يعني «أزِلها»،
                    // **والغيابُ يعني «لا تمسّها»**، وخلطُهما يمحو صورةً
                    // كلَّما بُدّل اسم.
                    if (d.imageThumb != null || !d.imageMediaID.isNullOrEmpty()) {
                        RahalTextButton(onClick = { vm.clearImage() }) {
                            Text(
                                stringResource(R.string.mn_remove_image),
                                color = Rahal.colors.danger,
                            )
                        }
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والمُعدِّلات** — الحجمُ والإضافاتُ وما شابه
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «مجموعاتٌ ومعدّلاتٌ غيرُ موجودةٍ
        //  بالتطبيق».)
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.mn_modifiers),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            RahalOutlineButton(onClick = { vm.addGroup() }) {
                Text(stringResource(R.string.mn_add_group))
            }
        }
        d.groups.forEachIndexed { gi, g -> GroupCard(gi, g, vm) }

        Spacer(Modifier.height(16.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(
                onClick = { vm.saveItem() },
                enabled = !vm.busy && d.name.isNotBlank() && d.price.isNotBlank(),
            ) { Text(stringResource(R.string.mn_save)) }
            RahalTextButton(onClick = { vm.cancelEdit() }) {
                Text(stringResource(R.string.mn_cancel))
            }
            if (!d.isNew) {
                RahalTextButton(onClick = { confirmDelete = true }) {
                    Text(stringResource(R.string.mn_delete), color = Rahal.colors.danger)
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }

    // **والحذفُ يُستأذَن فيه** — كما في الويب: **ضغطةٌ واحدةٌ تمحو صنفاً
    // بُني بصورته وسعره، ولا رجعةَ فيها.**
    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.mn_confirm_delete)) },
            confirmButton = {
                RahalTextButton(onClick = {
                    confirmDelete = false
                    vm.deleteItem(d.itemID)
                    vm.cancelEdit()
                }) {
                    Text(stringResource(R.string.mn_delete), color = Rahal.colors.danger)
                }
            },
            dismissButton = {
                RahalTextButton(onClick = { confirmDelete = false }) {
                    Text(stringResource(R.string.mn_cancel))
                }
            },
        )
    }
}

/**
 * **مجموعةُ مُعدِّلاتٍ واحدة — باسمها وحدَّيها وخياراتها.**
 *
 * **و«إلزاميّ» تُشتقّ من أدنى اختيارٍ ولا تُكتب** — كما في الويب حرفا:
 * **رقمٌ واحدٌ لا حقلان يتناقضان.** ومن رفع رايةَ الإلزام وترك الأدنى
 * صفراً ترك حالاً لا يعرف المحرّكُ أيَّ طرفيها يصدّق.
 *
 * **والأرقامُ تُقرأ نصّاً ثمّ تُحوَّل** — **وحقلٌ رقميٌّ يُفرَّغ يصير
 * صفراً في الحال، فلا يستطيع صاحبُه أن يمحو «١٠» ليكتب «٢».**
 */
@Composable
private fun GroupCard(index: Int, g: ModifierGroup, vm: MenuViewModel) {
    Spacer(Modifier.height(8.dp))
    Card {
        OutlinedTextField(
            value = g.name,
            onValueChange = { v -> vm.updateGroup(index) { it.copy(name = v) } },
            label = { Text(stringResource(R.string.mn_group_name)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(
                value = if (g.minSelect == 0) "" else g.minSelect.toString(),
                onValueChange = { v ->
                    val n = v.filter { it.isDigit() }.toIntOrNull() ?: 0
                    vm.updateGroup(index) { it.copy(minSelect = n) }
                },
                label = { Text(stringResource(R.string.mn_min_select)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            OutlinedTextField(
                value = if (g.maxSelect == 0) "" else g.maxSelect.toString(),
                onValueChange = { v ->
                    val n = v.filter { it.isDigit() }.toIntOrNull() ?: 0
                    vm.updateGroup(index) { it.copy(maxSelect = n) }
                },
                label = { Text(stringResource(R.string.mn_max_select)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
        }
        Spacer(Modifier.height(8.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Chip(
                stringResource(
                    if (g.minSelect > 0) R.string.mn_required else R.string.mn_optional,
                ),
                if (g.minSelect > 0) Rahal.colors.accent else Rahal.colors.inkMuted,
            )
            RahalTextButton(onClick = { vm.removeGroup(index) }) {
                Text(stringResource(R.string.mn_delete), color = Rahal.colors.danger)
            }
        }

        g.options.forEachIndexed { oi, o ->
            Spacer(Modifier.height(6.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OutlinedTextField(
                    value = o.name,
                    onValueChange = { v ->
                        vm.updateOption(index, oi) { it.copy(name = v) }
                    },
                    label = { Text(stringResource(R.string.mn_option_name)) },
                    singleLine = true,
                    modifier = Modifier.weight(1.4f),
                )
                OutlinedTextField(
                    value = if (o.priceDelta == 0L) "" else o.priceDelta.toString(),
                    onValueChange = { v ->
                        val n = v.filter { it.isDigit() }.toLongOrNull() ?: 0L
                        vm.updateOption(index, oi) { it.copy(priceDelta = n) }
                    },
                    label = { Text(stringResource(R.string.mn_price_delta)) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                )
                RahalTextButton(onClick = { vm.removeOption(index, oi) }) {
                    Text(stringResource(R.string.mn_delete), color = Rahal.colors.danger)
                }
            }
        }
        Spacer(Modifier.height(4.dp))
        RahalTextButton(onClick = { vm.addOption(index) }) {
            Text(stringResource(R.string.mn_add_option))
        }
    }
}

/**
 * **خيارُ قسمٍ واحد.**
 *
 * **والمختارُ يُعرف بلونه وثقله لا بكلمةٍ بجانبه** — **ولفظٌ يُخترع هنا
 * («مختار») ليس في معجم الويب فيعود الفرقُ من حيث أُغلق.**
 */
@Composable
private fun SectionChoice(label: String, chosen: Boolean, onPick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onPick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (chosen) FontWeight.Bold else FontWeight.Normal,
            color = if (chosen) Rahal.colors.brand else Rahal.colors.ink,
        )
    }
    Spacer(Modifier.height(6.dp))
}

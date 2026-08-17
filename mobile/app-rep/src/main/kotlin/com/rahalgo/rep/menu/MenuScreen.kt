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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
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
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.money
import com.rahalgo.ui.rememberImagePicker

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أصنافُ العميل — يبنيها المندوبُ نيابةً عنه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نفس الفورم الموجود عند مدير المنصّة
 *  والموجود عند المتجر موجودٌ عند المندوب… باللوحة والتطبيق أيضا».)
 *
 * # والرقمان يُعرضان
 *
 * **بخلاف صاحب المتجر**: يضع سعرَه ويقبض عليه، **وما تبيع به المنصّةُ
 * ليس شأنَه.** **والمندوبُ يبني نيابةً فيحتاج الاثنين** — يقول لصاحب
 * المتجر «تقبض هذا» وللزبون «يُباع بهذا».
 *
 * # ولا زرَّ لإنشاء قسم
 *
 * **القسمُ هو قسمُ السوق تزرعه الإدارة** — يظهر لأنّ فيه صنفاً. **فلا
 * يُنشأ من هنا**، ويُختار داخل نموذج الصنف.
 *
 * # و«غيرُ متاح» تُقلب من القائمة
 *
 * **«نفد الصنف» تقع عشرَ مرّاتٍ في اليوم** — **ومن فتح لها نموذجاً
 * بثمانية حقولٍ لم يقلبها**، فيبقى الصنفُ يُطلب وهو ناقص.
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
        ScreenTitle(stringResource(R.string.mn_title), vm.merchantName)

        if (vm.error.isNotEmpty()) {
            Note(vm.error, Rahal.colors.danger)
            Spacer(Modifier.height(8.dp))
        }

        Button(
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
    // **والإضافةُ من داخل القسم تملأ اختيارَه سلفا** — **ومن أضاف من
    // هنا لا يُسأل عن قسمٍ يقف فيه.**
    TextButton(onClick = { vm.newItem(sec.id) }) {
        Text(stringResource(R.string.mn_add_here))
    }
    sec.items.forEach { ItemRow(it, vm) }
    Spacer(Modifier.height(10.dp))
    HorizontalDivider()
    Spacer(Modifier.height(6.dp))
}

/**
 * **سطرُ صنف — بصورته والرقمين وحاله.**
 *
 * **و«غيرُ متاح» تُقال بلونٍ لا بغياب**: صنفٌ يختفي حين يُطفأ يُضاف
 * ثانيةً وثالثة، **فتمتلئ القائمةُ نسخاً من الشيء الواحد.**
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
                modifier = Modifier.size(46.dp).clip(RoundedCornerShape(10.dp)),
            )
            Spacer(Modifier.size(10.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = item.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
                // **والرقمان في سطر** — ما يقبضه المتجرُ وما يُباع به.
                Text(
                    text = stringResource(
                        R.string.mn_two_prices,
                        money(item.merchantPrice),
                        money(item.price),
                    ),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            // **وقلبُ «متاح» بضغطةٍ واحدة** — انظر أعلى الملفّ.
            Switch(
                checked = item.available,
                onCheckedChange = { vm.toggleAvailable(item) },
                enabled = !vm.busy,
            )
        }
        Spacer(Modifier.height(6.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Chip(
                stringResource(
                    if (item.available) R.string.mn_available else R.string.mn_unavailable,
                ),
                if (item.available) Rahal.colors.success else Rahal.colors.inkMuted,
            )
            // **و«خارجَ الدوام» يقوله الوقتُ لا صاحبُ المتجر** — ومن
            // خلط بينهما جعله يطفئ أصنافَه كلَّ ليلةٍ ويشعلها كلَّ صباح.
            if (item.sourceClosed) {
                Chip(stringResource(R.string.mn_closed_now), Rahal.colors.accent)
            }
            if (!item.approved) {
                Chip(stringResource(R.string.mn_pending), Rahal.colors.accent)
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
 * **نموذجُ الصنف — نفسُ حقول الإدارة والمتجر.**
 *
 * **والسعرُ المُدخَل سعرُ المتجر** — ما يقبضه، **وسعرُ البيع تحسبه
 * المنصّةُ بهامشها فلا يُكتب هنا.**
 *
 * **وقسمُ السوق شرطٌ لا اختيار**: **صنفٌ بلا قسمٍ لا يراه زبونٌ
 * يتصفّح** — والمحرّكُ يردّه، **فيُعطَّل الحفظُ حتّى يُختار** لا أن
 * يُضغط فيُردّ.
 */
@Composable
private fun ItemForm(vm: MenuViewModel) {
    val d = vm.editing ?: return
    val context = LocalContext.current
    val pick = rememberImagePicker { bytes -> vm.pickImage(bytes) }

    Screen {
        ScreenTitle(
            stringResource(if (d.isNew) R.string.mn_new_item else R.string.mn_edit_item),
            vm.merchantName,
        )

        if (vm.error.isNotEmpty()) {
            Note(vm.error, Rahal.colors.danger)
            Spacer(Modifier.height(8.dp))
        }

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
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(
                value = d.description,
                onValueChange = { v -> vm.editDraft { it.copy(description = v) } },
                label = { Text(stringResource(R.string.mn_desc)) },
                modifier = Modifier.fillMaxWidth(),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **وصورةٌ من الكاميرا أو المعرض**
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
                    modifier = Modifier.size(64.dp).clip(RoundedCornerShape(12.dp)),
                )
                Spacer(Modifier.size(10.dp))
                Column {
                    OutlinedButton(onClick = pick, enabled = !vm.busy) {
                        Text(stringResource(R.string.mn_pick_image))
                    }
                    // **والإزالةُ صريحةٌ** — الفراغُ يعني «أزِلها»،
                    // **والغيابُ يعني «لا تمسّها»**، وخلطُهما يمحو صورةً
                    // كلَّما بُدّل اسم.
                    if (d.imageThumb != null || !d.imageMediaID.isNullOrEmpty()) {
                        TextButton(onClick = { vm.clearImage() }) {
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
        // **وقسمُ السوق** — **وبلاه لا يراه زبونٌ يتصفّح**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        SectionTitle(stringResource(R.string.mn_market_section))
        Card {
            if (vm.platformSections.isEmpty()) {
                Text(
                    text = stringResource(R.string.mn_no_market_sections),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            vm.platformSections.forEach { ps ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clickable { vm.editDraft { it.copy(platformSectionID = ps.id) } },
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(ps.name, style = MaterialTheme.typography.bodyMedium)
                    if (d.platformSectionID == ps.id) {
                        Chip(stringResource(R.string.mn_chosen), Rahal.colors.brand)
                    }
                }
                Spacer(Modifier.height(4.dp))
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **ومتاحٌ أو غيرُ متاح**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        Card {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.mn_available_label),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Text(
                        stringResource(R.string.mn_available_hint),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                Switch(
                    checked = d.available,
                    onCheckedChange = { v -> vm.editDraft { it.copy(available = v) } },
                )
            }
        }

        Spacer(Modifier.height(16.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(
                onClick = { vm.saveItem() },
                // **والقسمُ شرطٌ** — انظر أعلى الدالّة.
                enabled = !vm.busy && d.name.isNotBlank() &&
                    d.price.isNotBlank() && d.platformSectionID.isNotEmpty(),
            ) { Text(stringResource(R.string.mn_save)) }
            TextButton(onClick = { vm.cancelEdit() }) {
                Text(stringResource(R.string.mn_cancel))
            }
            if (!d.isNew) {
                TextButton(onClick = { vm.deleteItem(d.itemID); vm.cancelEdit() }) {
                    Text(stringResource(R.string.mn_delete), color = Rahal.colors.danger)
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

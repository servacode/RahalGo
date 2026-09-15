package com.rahalgo.rep.menu

import com.rahalgo.ui.menu.ItemEditor
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Switch
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
fun MenuScreen(vm: MenuViewModel, onOffers: () -> Unit = {}) {
    val ctx = LocalContext.current
    BackHandler { if (vm.editing != null) vm.cancelEdit() else vm.close() }

    if (vm.editing != null) {
        // **والمحرّرُ من الوحدة المشتركة** — نسخةٌ واحدةٌ للمتجر
        // والمندوب (`ui.menu.ItemEditor`). **وكان نموذجان بمئتين وستّين
        // سطراً بينهما يفعلان الشيءَ نفسَه.**
        ItemEditor(
            d = vm.editing!!,
            sections = vm.platformSections.map {
                com.rahalgo.shared.merchant.PlatformSectionRef(id = it.id, name = it.name, imageUrl = null, imageThumbUrl = null)
            },
            busy = vm.busy,
            media = { path -> Backend.of(ctx).media(path) },
            onEdit = { f -> vm.editDraft(f) },
            onPickImage = { bytes -> vm.pickImage(bytes) },
            onClearImage = { vm.clearImage() },
            onSave = { vm.saveItem() },
            onCancel = { vm.cancelEdit() },
            onDelete = { vm.editing?.itemId?.let(vm::deleteItem) },
            // **وألفاظُ المندوب هي ألفاظُ الويب حرفاً** — يحرسها
            // `pnpm check:menu-parity`. **والمتجرُ يقرأ غيرَها** لأنّه
            // يكتب سعرَ نفسِه لا سعرَ متجرِ غيره.
            priceLabel = R.string.mn_price,
            descLabel = R.string.mn_desc,
            sectionLabel = R.string.mn_platform_section,
            sectionHint = R.string.mn_no_platform_section_hint,
        )
        return
    }

    val list = vm.sections
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load() }
        return
    }

    Screen {
        ScreenTitle(stringResource(R.string.mn_items), stringResource(R.string.mn_hint))

        // **وبابُ عروضه من حيث قائمتُه** — **والخصمُ على صنفٍ يُفتح
        // من مكان الصنف لا من قائمةٍ بعيدة.**
        RahalTextButton(onClick = onOffers, modifier = Modifier.fillMaxWidth()) {
            Text(stringResource(R.string.menu_offers))
        }

        RahalButton(
            onClick = { vm.newItem() },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.mn_item_new)) }

        Spacer(Modifier.height(12.dp))

        if (list.isEmpty()) {
            Empty(stringResource(R.string.mn_empty))
            return@Screen
        }

        // ══════════════════════════════════════════════════════════════
        // **شبكةُ أقسام السوق — كما في تطبيق المتجر**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «الأصناف لازم أوّل شي تُعرض بشكل
        //  أقسام وندخل بالقسم مثل المتجر وليس بشكل مختلف».)
        //
        // **وكانت كتلاً متتابعةً بأقسام المتجر الداخليّة** — والمندوبُ
        // يملأ القائمةَ نفسَها التي يملؤها صاحبُ المتجر، **فشاشتان
        // مختلفتان لعملٍ واحدٍ تجعلان من عرف إحداهما يتعثّر بالأخرى.**
        val open = vm.open
        if (open == null) {
            vm.groups.forEach { g ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.md)
                        .clickable { vm.openGroup(g) }
                        .padding(vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    RemoteImage(
                        url = Backend.of(ctx).media(g.imageUrl),
                        name = g.name,
                        modifier = Modifier.size(56.dp).clip(Rahal.shape.sm),
                    )
                    Spacer(Modifier.width(10.dp))
                    Text(
                        g.name,
                        modifier = Modifier.weight(1f),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyLarge,
                    )
                    Text(
                        "${g.items.size}",
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
                HorizontalDivider()
            }
        } else {
            RahalTextButton(onClick = { vm.back() }) {
                Text(stringResource(R.string.mn_back), color = Rahal.colors.brand)
            }
            Text(
                open.name,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.height(6.dp))
            open.items.forEach { item -> ItemRow(item, vm) }
        }
        Spacer(Modifier.height(24.dp))
    }
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
                url = Backend.of(context).media(item.imageThumbUrl),
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
            // ══════════════════════════════════════════════════════════
            // **ومفتاحٌ ذكيٌّ بدل زرٍّ يقول الفعل**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «ما في داعي لكلمة متوفر، زرّ
            //  ذكيّ متوفر/غير متوفر».)
            //
            // **وكان زرّاً نصُّه ينقلب**: يقول «متوفر» حين يكون متوفراً
            // فيُقرأ حالاً، **ويُقرأ فعلاً أيضاً** — أيُّهما؟ ومن ضغطه
            // لم يتيقّن أنّه قلبه أم أكّده.
            //
            // **والمفتاحُ لا يلتبس**: موضعُه هو الحال، وقلبُه هو الفعل.
            // **وهو نفسُه في تطبيق المتجر** — والشاشتان لغرضٍ واحد.
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Switch(
                    checked = item.available,
                    onCheckedChange = { vm.toggleAvailable(item) },
                    enabled = !vm.busy,
                )
                Text(
                    stringResource(
                        if (item.available) R.string.mn_available
                        else R.string.mn_unavailable,
                    ),
                    color = if (item.available) Rahal.colors.brand else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
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

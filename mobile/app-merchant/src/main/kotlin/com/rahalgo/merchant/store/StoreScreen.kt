package com.rahalgo.merchant.store

import com.rahalgo.ui.RahalOutlineButton
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.DropdownMenu
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.Card
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.StatBox
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متجري — حالُه وأرقامُ يومه وساعاتُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣، بنداً بند.)
 *
 * # ولا شرحَ تحت المفاتيح
 *
 * **«مفتوحٌ الآن الزبائنُ يرونك… وساعاتُ دوامك تغلقه تلقائيّاً خارج
 * أوقاتها» — حُذف كلُّه.** قال المالك: **«ما له داعٍ، خلص، معروفٌ
 * أصلاً»**. **وشرحُ الواضح يُطيل الشاشةَ فيُمرَّر ما تحته** — والساعاتُ
 * تحته.
 *
 * # والطارئُ أعلى الشاشة لا أسفلَها
 *
 * **كان في القاع** بحجّة أنّه فعلٌ ثقيلٌ لا يُغرى به. **وذاك خطأ**:
 * الطارئُ يقع والمطبخُ يحترق أو تنقطع الكهرباء — **ولحظتَها لا يُمرَّر
 * نصفُ شاشةٍ ليصل زرّاً.** فهو الآن أحمرُ تحت الاسم مباشرةً.
 *
 * # وما خرج منها
 *
 * **الإنذاراتُ صارت قسماً في الدرج** — انظر `drawer/DrawerScreens.kt`.
 */
@Composable
fun StoreScreen(vm: StoreViewModel, onPickPoint: () -> Unit = {}) {
    val store = vm.store
    if (vm.loading || store == null) {
        Screen {
            ScreenTitle(stringResource(R.string.store_title), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }

    var renaming by remember { mutableStateOf<String?>(null) }
    var confirming by remember { mutableStateOf(false) }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            // ══════════════════════════════════════════════════════════
            // **١ · اسمُه بيده — وقلمٌ بجانبه**
            // ══════════════════════════════════════════════════════════
            //
            // **وكان للأدمن وحدَه**، فمن أخطأ حرفاً يومَ سُجّل **بقي
            // الخطأُ في كلّ إشعارٍ يصل سائقَه.**
            val editingName = renaming
            if (editingName == null) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        store.name,
                        modifier = Modifier.weight(1f),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.headlineSmall,
                    )
                    RahalTextButton(onClick = { renaming = store.name }) {
                        Icon(
                            painter = painterResource(com.rahalgo.ui.R.drawable.ic_edit),
                            contentDescription = stringResource(R.string.store_rename),
                            tint = Rahal.colors.brand,
                            modifier = Modifier.size(20.dp),
                        )
                    }
                }
            } else {
                OutlinedTextField(
                    value = editingName,
                    onValueChange = { renaming = it },
                    label = { Text(stringResource(R.string.mn_store_name)) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(6.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    RahalButton(
                        onClick = { vm.rename(editingName); renaming = null },
                        enabled = !vm.saving && editingName.isNotBlank(),
                    ) { Text(stringResource(R.string.act_save)) }
                    RahalTextButton(onClick = { renaming = null }) {
                        Text(stringResource(R.string.act_cancel))
                    }
                }
            }

            // ══════════════════════════════════════════════════════════
            // **٢ · الإغلاقُ الطارئ — زرٌّ أحمرُ في المقدّمة**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «زرُّ إغلاقٍ طارئٍ بالأعلى بجانب
            //  اسم المتجر ليكون واضحاً سريعَ الوصول — زرٌّ حقيقيٌّ بلونٍ
            //  أحمرَ وأيقونة».)
            //
            // **ولا يُعرض على متجرٍ مغلقٍ أصلاً** — **وزرُّ إغلاقٍ في
            // متجرٍ مغلقٍ يُضغط فلا يتبدّل شيءٌ فيُقرأ عطباً.**
            //
            // **ويُسأل قبل أن يُنفَّذ**: **ضغطةٌ واحدةٌ تغلق متجراً وتُبلّغ
            // الإدارة** — والإبهامُ يزلّ.
            if (!store.emergencyClosed) {
                Spacer(Modifier.height(8.dp))
                Button(
                    onClick = { confirming = true },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Rahal.colors.danger,
                        contentColor = Color.White,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Icon(
                        painter = painterResource(com.rahalgo.ui.R.drawable.ic_power),
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(stringResource(R.string.store_emergency))
                }
            }
            if (confirming) {
                Spacer(Modifier.height(8.dp))
                Card {
                    Text(
                        stringResource(R.string.store_emergency_hint),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Spacer(Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(
                            onClick = { vm.emergency(); confirming = false },
                            colors = ButtonDefaults.buttonColors(
                                containerColor = Rahal.colors.danger,
                                contentColor = Color.White,
                            ),
                        ) { Text(stringResource(R.string.store_emergency_do)) }
                        RahalTextButton(onClick = { confirming = false }) {
                            Text(stringResource(R.string.act_cancel))
                        }
                    }
                }
            }

            // ══════════════════════════════════════════════════════════
            // **٣ · مفتوحٌ أو مغلق — بلا شرحٍ تحته**
            // ══════════════════════════════════════════════════════════
            //
            // **و`emergency_closed` جزءٌ من معادلة «مفتوح» نفسِها**
            // (`orders/hours.go`) — فالمفتاحُ يفتح ويغلق حرفيّاً.
            Spacer(Modifier.height(10.dp))
            Card {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(
                            if (!store.emergencyClosed) R.string.store_open
                            else R.string.store_closed,
                        ),
                        modifier = Modifier.weight(1f),
                        fontWeight = FontWeight.Bold,
                        color = if (!store.emergencyClosed) Rahal.colors.brand
                        else Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Switch(
                        checked = !store.emergencyClosed,
                        enabled = !vm.saving,
                        onCheckedChange = { vm.setOpen(it) },
                    )
                }
            }

            // ══════════════════════════════════════════════════════════
            // **٤ · أرقامُ اليوم — أربعةٌ لا خامسَ لها**
            // ══════════════════════════════════════════════════════════
            //
            // **والمستحقُّ مبيعاتُه ناقصَ العمولة** (قرارُ المالك
            // ٢٠٢٦-٠٨-١٠) — **ورقمٌ يشمل هامشَ المنصّة يجعله يحسب
            // أرباحاً ليست له.**
            vm.report?.let { r ->
                Spacer(Modifier.height(12.dp))
                // **ومربّعاتٌ لا أسطر** — (طلبُ المالك ٢٠٢٦-٠٨-٣١:
                // «تصبح أيضاً مربّعاتٍ تحتها الأرقام»).
                //
                // **والمستحقُّ في صفٍّ وحدَه**: الثلاثةُ فوقه عددٌ،
                // **وهو مال** — ومربّعٌ رابعٌ بينها يُقرأ عدّاً رابعا.
                Card {
                    StatRow {
                        StatBox(
                            label = stringResource(R.string.reports_today_orders),
                            value = r.summary.orders.toString(),
                            modifier = Modifier.weight(1f),
                        )
                        StatBox(
                            label = stringResource(R.string.reports_delivered),
                            value = r.summary.delivered.toString(),
                            modifier = Modifier.weight(1f),
                        )
                        StatBox(
                            label = stringResource(R.string.reports_cancelled),
                            value = r.summary.cancelled.toString(),
                            modifier = Modifier.weight(1f),
                        )
                    }
                    Spacer(Modifier.height(8.dp))
                    StatRow {
                        StatBox(
                            label = stringResource(R.string.reports_due_today),
                            value = money(r.summary.due),
                            modifier = Modifier.weight(1f),
                            color = Rahal.colors.brand,
                        )
                    }
                }
            }

            // ══════════════════════════════════════════════════════════
            // **٥ · مدّةُ التحضير — رقمٌ يَعِد به الزبون**
            // ══════════════════════════════════════════════════════════
            //
            // **ورقمٌ متفائلٌ يكسر الوعدَ لا يُسرّع المطبخ** — من كتب
            // عشراً وهو يحتاج ثلاثين **جعل السائقَ ينتظر عشرين.**
            Spacer(Modifier.height(12.dp))
            Card {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(R.string.store_prep),
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyLarge,
                    )
                    RahalTextButton(
                        onClick = { vm.setPrepMinutes(store.prepMinutes - 5) },
                        enabled = !vm.saving && store.prepMinutes > 5,
                    ) { Text("−", style = MaterialTheme.typography.titleLarge) }
                    Text(
                        "${store.prepMinutes} ${stringResource(R.string.store_prep_unit)}",
                        fontWeight = FontWeight.Bold,
                    )
                    RahalTextButton(
                        onClick = { vm.setPrepMinutes(store.prepMinutes + 5) },
                        enabled = !vm.saving,
                    ) { Text("+", style = MaterialTheme.typography.titleLarge) }
                }
            }

            // ══════════════════════════════════════════════════════════
            // **٦ · الساعات — تُقرأ وتُكتب**
            // ══════════════════════════════════════════════════════════
            HoursEditor(vm)
            SectionsEditor(vm)

            // ══════════════════════════════════════════════════════════
            // **٧ · العنوانُ والدبّوس — آخرُ الشاشة**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «يجب أن يكون آخرَ شيءٍ العنوانُ
            //  أيضاً والخريطة، من أجل تحديد الموقع وكتابة العنوان»،
            //  ثمّ: «سطرٌ أعلى لكتابة العنوان أيضاً».)
            //
            // **وموضعُه الآخِرُ صحيح**: يُبدَّل مرّةً في العمر — يومَ
            // ينتقل. **وما يُفعل مرّةً لا يُوضع فوقَ ما يُفعل كلَّ يوم.**
            //
            // # والسطرُ فوق الخريطة لا تحتها
            //
            // **الخريطةُ تقول «أين» والسطرُ يقول «كيف تصل»** — طابقٌ
            // ثانٍ، بابٌ خلفَ المحلّ، علامةٌ يعرفها أهلُ الحيّ.
            // **ودبّوسٌ دقيقٌ لا يقول أيَّ بابٍ يُطرق.**
            //
            // # ولا يُحفظان إلّا معاً
            //
            // **زرٌّ واحدٌ للاثنين** — **وزرّان يجعلان من كتب عنوانَه ثمّ
            // حرّك دبّوسَه يحفظ أحدَهما ويظنّ الآخرَ محفوظاً.**
            AddressEditor(vm, onPickPoint)

            Spacer(Modifier.height(24.dp))
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الساعات — سبعةُ أيّامٍ تُحفظ دفعةً واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «ساعاتُ العمل ما فيه مربّعُ المغلق، وثاني
 *  شيء الساعاتُ ما نقدر نعدّلها — ثابتةٌ لا تتغيّر بالتطبيق، أصلحها».)
 *
 * **والبابُ كان مفتوحاً في المحرّك** (`PUT /merchant/stores/{id}/hours`)
 * **والتطبيقُ يقرأ ولا يكتب** — فبدت الساعاتُ كأنّها قرارُ المنصّة وهي
 * قرارُه هو.
 *
 * **ودوامُه يغلق متجرَه** (`OpenNowSQL`) — **فمن لم يملك تعديلَها لم
 * يملك أن يفتح صباحاً قبل موعده.**
 *
 * # ولماذا السبعةُ معاً
 *
 * **المحرّكُ يرفض ما دونها**: `SetHours` يردّ `ErrBadHours` إن لم تكن
 * الأيّامُ سبعةً (`catalog/hours.go`) — **فحفظُ يومٍ واحدٍ يُردّ.**
 *
 * # ومربّعُ «مغلق» لكلّ يوم
 *
 * **ويومُ عطلته لا يُكتب بساعاتٍ متساوية** — من كتب `00:00 — 00:00`
 * ظنّاً أنّه إغلاقٌ **فتح متجرَه لحظةً في منتصف الليل.** فالإغلاقُ
 * مفتاحٌ صريحٌ يحمله المحرّكُ في عموده (`closed`).
 *
 * # والوقتُ يُكتب ولا يُنتقى بعقربٍ دوّار
 *
 * **`HH:MM` نصٌّ في المحرّك** — **ومنتقي وقتٍ يفتح نافذةً لكلّ حقلٍ
 * أربعةَ عشرَ مرّةً** لضبط أسبوع.
 */
/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ السوق — يعلن ما يبيع فتُقصر عليه القائمة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «قسم السوق يجب أن يكون مخصّصاً — مو معقول
 *  كل الأقسام تطلع عند كل المتاجر. مطعم تطلع المأكولات فقط، شو علاقته
 *  بالأحذية؟»)
 *
 * # ولماذا يختار هو
 *
 * **وكان يمكن أن نشتقّها من تصنيف المتجر** — لكنّ الكافتيريا تبيع
 * شاورما وحلوياتٍ ومشروبات، **وتصنيفٌ واحدٌ يحصرها في قسم.**
 *
 * **وهو أعرفُ بما يبيع** — ويبدّله في يومٍ يوسّع فيه بضاعته.
 *
 * # وفارغٌ يعني الكلّ
 *
 * **ومتجرٌ لم يختر بعدُ يرى القائمةَ كاملةً** — فلا يُحبس متجرٌ قائمٌ
 * بلا أقسامٍ يومَ يصله التحديث.
 */
@Composable
private fun SectionsEditor(vm: StoreViewModel) {
    if (vm.allSections.isEmpty()) return

    var picked by remember(vm.mySections) {
        mutableStateOf(vm.mySections.map { it.id }.toSet())
    }
    var open by remember { mutableStateOf(false) }
    val dirty = picked != vm.mySections.map { it.id }.toSet()
    val chosen = vm.allSections.filter { it.id in picked }
    val rest = vm.allSections.filter { it.id !in picked }

    Spacer(Modifier.height(12.dp))
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.store_sections),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        if (dirty) {
            RahalButton(
                onClick = { vm.saveSections(picked.toList()) },
                enabled = !vm.saving,
            ) { Text(stringResource(R.string.act_save)) }
        }
    }
    Spacer(Modifier.height(4.dp))
    Text(
        stringResource(R.string.store_sections_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(8.dp))

    // ══════════════════════════════════════════════════════════════════
    // **وأقسامُه مربّعاتٌ لا أسطر — وحذفُها أيقونة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **(طلبُ المالك ٢٠٢٦-٠٨-٣١:** «الأقسامُ التي تخصّ المتجر تصبح
    // مربّعات، مع إضافة أيقونة حذفٍ لحذف القسم».)
    //
    // **وسطرٌ لكلّ قسمٍ يجعل خمسةَ أقسامٍ خمسةَ أسطر** — والاسمُ فيها
    // كلمةٌ والباقي فراغ. **والمربّعاتُ تصفّها في سطرين.**
    //
    // **وكلمةُ «حذف» بجانب كلّ قسمٍ تزاحم اسمَه** — والأيقونةُ تقولها
    // بحجمٍ أصغرَ ومعنًى واحد.
    Card {
        if (chosen.isEmpty()) {
            Text(
                stringResource(R.string.store_sections_empty),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.padding(vertical = 6.dp),
            )
        } else {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                chosen.forEach { sec ->
                    Row(
                        Modifier
                            .padding(vertical = 4.dp)
                            .clip(Rahal.shape.md)
                            .background(Rahal.colors.inkMuted.copy(alpha = 0.07f))
                            .padding(start = 12.dp, end = 4.dp, top = 4.dp, bottom = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(sec.name, style = MaterialTheme.typography.bodyMedium)
                        IconButton(
                            onClick = { picked = picked - sec.id },
                            modifier = Modifier.size(32.dp),
                        ) {
                            Icon(
                                painter = painterResource(com.rahalgo.ui.R.drawable.ic_close),
                                // **والوصفُ يسمّي قسمَه** — قارئُ الشاشة
                                // يقرأ «حذف» ستَّ مرّاتٍ بلا تمييز.
                                contentDescription =
                                    stringResource(R.string.act_delete) + " " + sec.name,
                                tint = Rahal.colors.inkMuted,
                                modifier = Modifier.size(16.dp),
                            )
                        }
                    }
                }
            }
        }
    }

    Spacer(Modifier.height(8.dp))
    Box {
        RahalOutlineButton(
            onClick = { if (rest.isNotEmpty()) open = true },
            enabled = rest.isNotEmpty(),
        ) {
            Text(
                stringResource(
                    if (rest.isEmpty()) R.string.store_sections_all_in
                    else R.string.store_sections_add,
                ),
            )
        }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            rest.forEach { sec ->
                DropdownMenuItem(
                    text = { Text(sec.name) },
                    onClick = {
                        picked = picked + sec.id
                        open = false
                    },
                )
            }
        }
    }
}

@Composable
private fun HoursEditor(vm: StoreViewModel) {
    if (vm.hours.isEmpty()) return

    val sorted = vm.hours.sortedBy { it.dayOfWeek }
    // **والمسوّدةُ محلّيّةٌ حتّى يُضغط الحفظ** — **وحفظٌ عند كلّ حرفٍ
    // يُرسل نداءً ونصفُ الوقتِ غيرُ مكتوبٍ بعد.**
    var draft by remember(sorted) { mutableStateOf(sorted) }
    val dirty = draft != sorted

    Spacer(Modifier.height(12.dp))
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.store_hours),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        if (dirty) {
            RahalButton(
                onClick = { vm.saveHours(draft) },
                enabled = !vm.saving,
            ) { Text(stringResource(R.string.act_save)) }
        }
    }
    Spacer(Modifier.height(6.dp))
    Card {
        draft.forEachIndexed { i, d ->
            if (i > 0) HorizontalDivider()
            HoursRow(
                d = d,
                onChange = { changed ->
                    draft = draft.map { if (it.dayOfWeek == changed.dayOfWeek) changed else it }
                },
            )
        }
    }
}

/**
 * **سطرُ يومٍ — مربّعٌ ثمّ اسمٌ ثمّ وقتان.**
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «ما عجبني الشكل، كبّرتَها أكثر من اللازم.
 *  ما في داعٍ من وإلى وهيك كلّ هالشي — خلّي الشكل مرتّباً أكثر: قائمةُ
 *  مربّعاتٍ على اليمين مفتوحٌ أو مغلق».)
 *
 * # وما كان قبلَه
 *
 * **كلُّ يومٍ كان ثلاثةَ أسطرٍ ومفتاحاً وحقلين بعنوانين** — سبعةُ أيّامٍ
 * تملأ شاشتين. **وجدولٌ يُقرأ في لمحةٍ صار سَرْداً يُمرَّر.**
 *
 * # والمربّعُ لا المفتاح
 *
 * **سبعةُ مفاتيحَ متتاليةٍ تُقرأ سبعةَ إعداداتٍ منفصلة** — **والمربّعاتُ
 * تُقرأ جدولاً**: عمودٌ واحدٌ تمسحه العينُ من فوقُ إلى تحت فترى أيّامَ
 * عملك كلَّها دفعةً.
 *
 * **وهو مُعلَّمٌ حين يكون مفتوحاً** — **ومربّعٌ معناه «مغلق» يُقرأ
 * مقلوباً**: العلامةُ في ذهن الناس إثباتٌ لا نفي.
 *
 * # ولا عنوانَ فوق الوقتين
 *
 * **`08:00 — 23:00` يقول نفسَه** — **و«من» و«إلى» فوق حقلين متجاورين
 * شرحٌ لما لا يحتاج شرحاً**، ويرفع السطرَ ضعفَ ارتفاعه.
 */
@Composable
private fun HoursRow(d: DayHours, onChange: (DayHours) -> Unit) {
    val name = when (d.dayOfWeek) {
        0 -> R.string.days_sun
        1 -> R.string.days_mon
        2 -> R.string.days_tue
        3 -> R.string.days_wed
        4 -> R.string.days_thu
        5 -> R.string.days_fri
        else -> R.string.days_sat
    }
    Row(
        Modifier.fillMaxWidth().padding(vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Checkbox(
            checked = !d.closed,
            onCheckedChange = { onChange(d.copy(closed = !it)) },
            modifier = Modifier.size(36.dp),
        )
        Spacer(Modifier.width(4.dp))
        Text(
            stringResource(name),
            modifier = Modifier.width(56.dp),
            style = MaterialTheme.typography.bodyMedium,
        )
        Spacer(Modifier.weight(1f))
        if (d.closed) {
            Text(
                stringResource(R.string.store_day_closed),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        } else {
            TimeField(d.openTime) { onChange(d.copy(openTime = it)) }
            Text(
                " — ",
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            TimeField(d.closeTime) { onChange(d.copy(closeTime = it)) }
        }
    }
}

/**
 * **حقلُ وقتٍ `HH:MM`** — صندوقٌ صغيرٌ بلا عنوانٍ ولا إطارِ مادّة.
 *
 * **و`OutlinedTextField` أدناه ٥٦ نقطة** — **وأربعةَ عشرَ منها تصنع
 * الشاشتين اللتين شكا منهما المالك.**
 *
 * **والمحرّكُ يحوّله إلى `::time`** — **وحرفٌ زائدٌ يردّ `ErrBadHours`
 * على الأيّام السبعة معاً**، فلا يُحفظ شيء.
 */
@Composable
private fun TimeField(value: String, onChange: (String) -> Unit) {
    val context = LocalContext.current
    // **والثواني تُقطع للعرض** — المحرّكُ يردّ `08:00:00`.
    val shown = value.take(5).ifBlank { "00:00" }

    // ══════════════════════════════════════════════════════════════════
    // **والساعةُ تُختار لا تُكتب**
    // ══════════════════════════════════════════════════════════════════
    //
    // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «تعديلُ ساعات العمل صعبٌ جدّاً، يجب أن
    //  يكون اختيارُ الساعة أسهلَ من كتابتها بشكلٍ يدويّ».)
    //
    // # ولماذا كان صعباً
    //
    // **وكان حقلاً نصّيّاً حرّاً بلوحة أرقام**: يكتب `0` ثمّ `8` ثمّ
    // **ينقّط بيده** `:` ثمّ `0` و`0`. **خمسُ ضغطاتٍ لرقمٍ واحد** —
    // وفي سبعة أيّامٍ مرّتين: **سبعون ضغطة.**
    //
    // **ولا يمنع خطأً**: من كتب `25:70` حُفظ، **ومتجرٌ ساعتُه غلطٌ لا
    // يُفتح أبداً ولا يعرف صاحبُه لماذا.**
    //
    // # ومنتقي النظام لا منتقٍ نصنعه
    //
    // **وصاحبُ المتجر يعرف ساعةَ هاتفه** — رآها في المنبّه والتقويم.
    // **ومنتقٍ نصنعه بأيدينا غريبٌ عليه** مهما أتقنّاه.
    //
    // **و`is24Hour = true`** — والمنصّةُ تحفظ `HH:mm`، **فلا ترجمةَ
    // بين صباحٍ ومساءٍ تضيع في الطريق.**
    val picker = {
        val h = shown.substringBefore(':').toIntOrNull()?.coerceIn(0, 23) ?: 0
        val m = shown.substringAfter(':').toIntOrNull()?.coerceIn(0, 59) ?: 0
        // ══════════════════════════════════════════════════════════════
        // **وزخرفةٌ في الأرقام لا تُسقط تطبيقاً**
        // ══════════════════════════════════════════════════════════════
        //
        // **انهار التطبيقُ ٢٠٢٦-٠٨-٢٩** عند فتح ساعات العمل:
        // `BadTokenException` — سياقٌ بلا نافذةِ نشاط.
        //
        // **والسببُ أنّي بنيتُ سياقاً جديداً بدل أن ألفَّ القائم.**
        // أُصلح، **لكنّ الدرسَ أكبرُ من السطر**: تبديلُ نظام الأرقام
        // شكلٌ لا وظيفة، **ومن جعل الشكلَ قادراً على إسقاط شاشةٍ أخطأ
        // مرّتين.**
        //
        // **فإن تعذّر اللفُّ على جهازٍ ما، يُفتح الحوارُ بسياقه الأصليّ**
        // — أرقامٌ عربيّةٌ في المنتقي أهونُ من تطبيقٍ يُغلق.
        val themed = runCatching { latinDigits(context) }.getOrDefault(context)
        android.app.TimePickerDialog(
            themed,
            // ══════════════════════════════════════════════════════════
            // **والساعةُ تُكتب بأرقامٍ لاتينيّةٍ مهما كانت لغةُ الجهاز**
            // ══════════════════════════════════════════════════════════
            //
            // (بلاغُ المالك ٢٠٢٦-٠٨-٢٩: «الأرقام عربيّة وما يقبل ساعاتِ
            //  العمل، يعتبرها غير صالحة».)
            //
            // **و`"%02d".format(9)` تتبع لغةَ الجهاز** — فتُنتج «٠٩» على
            // جهازٍ عربيّ. **والبلاغان بلاغٌ واحد:**
            //
            //   الحقلُ يُرسل   open_time = "٠٩:٣٠"
            //   وبوستغرس يقرأ  $4::time  فيفشل
            //   فيردّ المحرّك   ErrBadHours → «غير صالح»
            //
            // **فالأرقامُ العربيّةُ هي نفسُها سببُ الرفض** — لا عطبان.
            //
            // **و`Locale.ROOT` لا لغةُ الجهاز**: هذه قيمةُ بروتوكولٍ
            // تذهب إلى خادمٍ لا نصٌّ يُقرأ، **ومن ترجم أرقامَ بروتوكولٍ
            // كسره.**
            { _, hh, mm ->
                onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
            },
            h, m, true,
        ).let { dialog ->
            runCatching { dialog.show() }.onFailure {
                // **والسقوطُ الأخير**: حوارٌ بالسياق الأصليّ بلا لفّ.
                android.app.TimePickerDialog(
                    context,
                    { _, hh, mm ->
                        onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
                    },
                    h, m, true,
                ).show()
            }
        }
    }

    Box(
        Modifier
            .width(72.dp)
            .border(1.dp, Rahal.colors.line, Rahal.shape.sm)
            .clickable { picker() }
            .padding(vertical = 8.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = shown,
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عنوانُه ودبّوسُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا لا تُرسم الخريطةُ في الشاشة
 *
 * **بطاقةُ خريطةٍ حيّةٍ في شاشةٍ تُمرَّر تلتقط الإصبعَ فتتحرّك** —
 * فيظنّ صاحبُها أنّه نقل دبّوسَه وهو يمرّر. **والخريطةُ تُفتح ملءَ
 * الشاشة عند الطلب** كما في تطبيق المندوب.
 *
 * **وخريطةٌ حيّةٌ تُهيَّأ مع كلّ فتحةٍ لـ«متجري»** تحمّل نمطاً
 * وأقراصاً على شبكةٍ ضعيفة — **لشيءٍ يُبدَّل مرّةً في العمر.**
 *
 * # وما يُعرض بدلَها
 *
 * **الإحداثيّان نصّاً** — **ومن لا يرى شيئاً لا يعرف أدبّوسُه موضوعٌ
 * أصلاً أم لا**، فيحرّكه بلا داعٍ أو يتركه وهو فارغ.
 */
@Composable
private fun AddressEditor(vm: StoreViewModel, onPickPoint: () -> Unit) {
    val store = vm.store ?: return

    // **والمسوّدةُ تبدأ ممّا في المتجر** — وتُعاد حين يتبدّل من الخريطة.
    var text by remember(store.addressText) { mutableStateOf(store.addressText) }
    val dirty = text.trim() != store.addressText || vm.pickedLat != null

    Spacer(Modifier.height(12.dp))
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.act_address),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        if (dirty) {
            RahalButton(
                onClick = { vm.saveAddress(text, vm.pickedLat, vm.pickedLng) },
                enabled = !vm.saving,
            ) { Text(stringResource(R.string.act_save)) }
        }
    }
    Spacer(Modifier.height(6.dp))
    OutlinedTextField(
        value = text,
        onValueChange = { text = it },
        placeholder = { Text(stringResource(R.string.store_address_hint)) },
        modifier = Modifier.fillMaxWidth(),
        minLines = 2,
    )
    Spacer(Modifier.height(8.dp))
    Card {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    stringResource(R.string.store_point),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
                val lat = vm.pickedLat ?: store.lat
                val lng = vm.pickedLng ?: store.lng
                Text(
                    if (lat == null || lng == null) {
                        stringResource(R.string.store_point_none)
                    } else {
                        // **ولا تُكتب بحروفٍ عربيّة** — الأرقامُ
                        // اللاتينيّة هي ما يُلصق في خريطةٍ أخرى.
                        String.format(java.util.Locale.US, "%.5f, %.5f", lat, lng)
                    },
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            RahalTextButton(onClick = onPickPoint) {
                Icon(
                    painter = painterResource(com.rahalgo.ui.R.drawable.ic_pin),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(4.dp))
                Text(stringResource(R.string.store_point_pick))
            }
        }
    }
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **سياقٌ بأرقامٍ لاتينيّة — للنافذة نفسِها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ونافذةُ الوقت من النظام لا منّا** — ترسم أرقامَها بلغة الجهاز،
 * **فيرى صاحبُ المتجر «٠٩:٣٠» في المنتقي** ثمّ «09:30» في الحقل.
 * **ورقمان لشيءٍ واحدٍ في شاشةٍ واحدةٍ يربكان.**
 *
 * **و`ar-u-nu-latn` عربيّةٌ بأرقامٍ لاتينيّة** — تبقى الكلماتُ عربيّةً
 * («إلغاء» و«موافق») **ويتبدّل نظامُ الأرقام وحدَه.**
 */
private fun latinDigits(context: android.content.Context): android.content.Context {
    val config = android.content.res.Configuration(context.resources.configuration)
    config.setLocale(java.util.Locale.forLanguageTag("ar-u-nu-latn"))
    // ══════════════════════════════════════════════════════════════════
    // **ويُلَفُّ السياقُ لفّاً — ولا يُبنى من جديد**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان `createConfigurationContext`** فانهار التطبيقُ عند فتح
    // ساعاتِ العمل (قِيس على الجهاز ٢٠٢٦-٠٨-٢٩):
    //
    //   BadTokenException: Unable to add window — token null is not
    //   valid; is your activity running?
    //
    // **لأنّها تصنع سياقاً جديداً بلا نافذةِ نشاط** — والحوارُ يحتاج
    // رمزَ النافذة ليُعلَّق عليها. **فيُبنى ولا يجد أين يظهر.**
    //
    // **و`ContextThemeWrapper` يلفّ النشاطَ نفسَه** — فيبقى الرمزُ
    // والسمةُ، **ويتبدّل نظامُ الأرقام وحدَه.**
    return android.view.ContextThemeWrapper(context, 0).apply {
        applyOverrideConfiguration(config)
    }
}

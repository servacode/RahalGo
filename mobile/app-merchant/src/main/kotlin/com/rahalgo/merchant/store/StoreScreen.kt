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
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
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
import androidx.compose.runtime.saveable.rememberSaveable
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
internal fun SectionsEditor(vm: StoreViewModel) {
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
    }
    // **تنحفظ لحالها** — بلا زرّ «حفظ» (طلبُ المالك ٢٠٢٦-١٠-١٢).
    androidx.compose.runtime.LaunchedEffect(picked) {
        if (dirty && vm.store?.status != "suspended") {
            kotlinx.coroutines.delay(600)
            vm.saveSections(picked.toList())
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
                            .padding(start = 12.dp, end = 0.dp, top = 0.dp, bottom = 0.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(sec.name, style = MaterialTheme.typography.bodyMedium)
                        // **وزرُّ الشيل ٤٨ لا ٣٢** — حدُّ الإصبع المريح (تدقيقُ ٢٠٢٦-١٠-٢٧).
                        IconButton(
                            onClick = { picked = picked - sec.id },
                            modifier = Modifier.size(48.dp),
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
internal fun HoursEditor(vm: StoreViewModel) {
    if (vm.hours.isEmpty()) return

    val sorted = vm.hours.sortedBy { it.dayOfWeek }
    // **والمسوّدةُ محلّيّةٌ حتّى يُضغط الحفظ** — **وحفظٌ عند كلّ حرفٍ
    // يُرسل نداءً ونصفُ الوقتِ غيرُ مكتوبٍ بعد.**
    var draft by remember(sorted) { mutableStateOf(sorted) }
    val dirty = draft != sorted
    // ══════════════════════════════════════════════════════════════════
    // **وفتحٌ بعد منتصف الليل يُسأل عنه قبل الحفظ** (٢٠٢٦-١٠-١٣)
    // ══════════════════════════════════════════════════════════════════
    //
    // (قِيس على الإنتاج: «شاورما بصمون» محفوظٌ ١:٠٠–١١:٠٠ و«فلافل الخير»
    //  ٦:٠٠–٩:٠٠ — **أرادا الظهرَ والمساء فحُفظ الليلُ والصبح**، والزبونُ يرى
    //  «مغلق» العصرَ كلَّه. **والغلطُ في الشاشة لا فيهما** — قالها المالك.)
    var nightCheck by remember { mutableStateOf<List<com.rahalgo.shared.merchant.DayHours>?>(null) }
    // **ومفتوحٌ اليومَ كلَّه لا يُسأل** — منتصفُ الليل إلى ما قبله دوامٌ كاملٌ لا غلط.
    val nightOpen: (com.rahalgo.shared.merchant.DayHours) -> Boolean = { d -> opensAtNight(d) }
    val save: (List<com.rahalgo.shared.merchant.DayHours>) -> Unit = { days ->
        if (days.any(nightOpen)) nightCheck = days else vm.saveHours(days)
    }
    nightCheck?.let { days ->
        val d = days.first(nightOpen)
        androidx.compose.material3.AlertDialog(
            onDismissRequest = { nightCheck = null },
            title = { Text(stringResource(R.string.hours_night_title)) },
            text = { Text(stringResource(R.string.hours_night_text, clock12(d.openTime), clock12(d.closeTime))) },
            confirmButton = {
                RahalButton(onClick = { nightCheck = null; vm.saveHours(days) }) { Text(stringResource(R.string.hours_night_yes)) }
            },
            dismissButton = {
                RahalOutlineButton(onClick = { nightCheck = null }) { Text(stringResource(R.string.hours_night_fix)) }
            },
        )
    }

    Spacer(Modifier.height(12.dp))
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.store_hours),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
    }
    Spacer(Modifier.height(6.dp))
    // ══════════════════════════════════════════════════════════════════
    // **وقتٌ واحدٌ لكلّ الأيّام بكبسة** (طلبُ المالك ٢٠٢٦-١٠-١٢: «مشان ما يضبط الدوام
    // كل يوم بيوم، زر فوق ساعات العمل من الساعة الى الساعة ويطبّق على كل الايام»).
    // ويُحفظ فوراً — **ثمّ يعدّل يوماً بعينه إن شاء** (الجمعة مثلاً).
    // ══════════════════════════════════════════════════════════════════
    var allOpen by rememberSaveable { mutableStateOf(sorted.firstOrNull { !it.closed }?.openTime ?: "09:00") }
    var allClose by rememberSaveable { mutableStateOf(sorted.firstOrNull { !it.closed }?.closeTime ?: "23:00") }
    // **«مفتوح ٢٤ ساعة» لكلّ الأيّام** (قرارُ المالك ٢٠٢٦-١٠-١٨) — يُخفي الوقتين.
    var all24 by rememberSaveable { mutableStateOf(sorted.firstOrNull { !it.closed }?.isOpen24h ?: false) }
    Card {
        Text(stringResource(R.string.hours_all_days), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.height(6.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(
                checked = all24,
                onCheckedChange = { all24 = it },
                modifier = Modifier.size(48.dp),
            )
            Text(stringResource(R.string.hours_open_24h), style = MaterialTheme.typography.bodyMedium)
        }
        if (!all24) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.hours_from))
                TimeField(allOpen) { allOpen = it }
                Text(stringResource(R.string.hours_to))
                TimeField(allClose) { allClose = it }
            }
        }
        // **والدوامُ بالكلام** — «من 1:00 م لـ 11:00 م» يُقرأ قبل الحفظ.
        Spacer(Modifier.height(4.dp))
        Text(
            if (all24) stringResource(R.string.hours_open_24h)
            else stringResource(R.string.hours_summary, clock12(allOpen), clock12(allClose)),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(8.dp))
        RahalButton(
            onClick = {
                draft = draft.map {
                    if (all24) it.with24h(true)
                    else it.copy(closed = false, open24h = false, openTime = allOpen, closeTime = allClose)
                }
                save(draft)
            },
            enabled = !vm.saving && vm.store?.status != "suspended" &&
                (all24 || (allOpen.isNotBlank() && allClose.isNotBlank())),
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.hours_apply_all)) }
    }
    Spacer(Modifier.height(8.dp))
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
        // **والحفظُ تحت ما تغيّر لا فوقه** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان بعنوان القسم بعيداً
        // عن اليوم الذي عدّله.
        if (dirty && vm.store?.status != "suspended") {
            Spacer(Modifier.height(8.dp))
            RahalButton(
                onClick = { save(draft) },
                enabled = !vm.saving,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.act_save)) }
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
internal fun HoursRow(d: DayHours, onChange: (DayHours) -> Unit) {
    val name = when (d.dayOfWeek) {
        0 -> R.string.days_sun
        1 -> R.string.days_mon
        2 -> R.string.days_tue
        3 -> R.string.days_wed
        4 -> R.string.days_thu
        5 -> R.string.days_fri
        else -> R.string.days_sat
    }
    // ══════════════════════════════════════════════════════════════════
    // **سطرٌ لليوم وسطرٌ لوقته** (تدقيقُ ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان كلُّه في سطرٍ أعرضَ من الشاشة** — مربّعٌ واسمٌ ومربّعُ «٢٤ ساعة» ووقتان
    // كلٌّ منهما ٩٦، **فينضغط على شاشةٍ عرضُها ٣٦٠، والمربّعاتُ ٣٦ أصغرُ من الإصبع.**
    // **فاليومُ ومربّعاه فوق، ووقتاه تحته** — ولليوم المغلق سطرٌ واحد.
    Column(Modifier.fillMaxWidth().padding(vertical = 2.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Checkbox(
                checked = !d.closed,
                onCheckedChange = { onChange(d.copy(closed = !it)) },
                modifier = Modifier.size(48.dp),
            )
            Text(
                stringResource(name),
                modifier = Modifier.weight(1f),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            if (d.closed) {
                Text(
                    stringResource(R.string.store_day_closed),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            } else {
                // **«٢٤ ساعة» لليوم وحدَه** — يُخفي الوقتين، وإطفاؤه يعيد ٩–٢٣.
                Checkbox(
                    checked = d.isOpen24h,
                    onCheckedChange = { onChange(d.with24h(it)) },
                    modifier = Modifier.size(48.dp),
                )
                Text(
                    stringResource(R.string.hours_open_24h_short),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
        if (!d.closed && !d.isOpen24h) {
            Row(
                Modifier.fillMaxWidth().padding(start = 48.dp, bottom = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
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
internal fun TimeField(value: String, onChange: (String) -> Unit) {
    val context = LocalContext.current
    // **والثواني تُقطع** — المحرّكُ يردّ `08:00:00`.
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
            // **صباحاً ومساءً لا ٢٤ ساعة** (٢٠٢٦-١٠-١٣) — كان `true` فاختار
            // أصحابُ المتاجر «١» و«١١» يريدون الظهرَ والليل، فحُفظ الليلُ والصبح.
            // **والمحفوظُ يبقى `HH:mm`** — الترجمةُ في الشاشة وحدَها.
            h, m, false,
        ).let { dialog ->
            runCatching { dialog.show() }.onFailure {
                // **والسقوطُ الأخير**: حوارٌ بالسياق الأصليّ بلا لفّ.
                android.app.TimePickerDialog(
                    context,
                    { _, hh, mm ->
                        onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
                    },
                    h, m, false,
                ).show()
            }
        }
    }

    Box(
        Modifier
            .width(96.dp)
            .border(1.dp, Rahal.colors.line, Rahal.shape.sm)
            .clickable { picker() }
            .padding(vertical = 8.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = clock12(shown),
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
 * **حالُ الدبّوس كلمةً** — «محدّد على الخريطة» أو «لسّا ما حدّدته» (تدقيقُ
 * ٢٠٢٦-١٠-٢٧). **ومن لا يرى شيئاً لا يعرف أدبّوسُه موضوعٌ أصلاً أم لا**،
 * والإحداثيّان كانا أرقاماً لا تعني له شيئاً.
 */
@Composable
internal fun AddressEditor(vm: StoreViewModel, onPickPoint: () -> Unit) {
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
                val set = lat != null && lng != null
                // **حالٌ لا أرقام** (تدقيقُ ٢٠٢٦-١٠-٢٧) — «35.95012, 39.01234» لا تعني شيئاً
                // لصاحب المتجر. **فمحدَّدٌ بالأخضر، أو لسّا بالأحمر.**
                Text(
                    stringResource(if (set) R.string.store_point_set else R.string.store_point_none),
                    color = if (set) Rahal.colors.success else Rahal.colors.danger,
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
                Text(
                    stringResource(
                        if ((vm.pickedLat ?: store.lat) != null) R.string.store_point_change else R.string.store_point_pick,
                    ),
                )
            }
        }
    }
    // **والحفظُ تحت العنوان والدبّوس معاً** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان بعنوان القسم.
    if (dirty && vm.store?.status != "suspended") {
        Spacer(Modifier.height(8.dp))
        RahalButton(
            onClick = { vm.saveAddress(text, vm.pickedLat, vm.pickedLng) },
            enabled = !vm.saving,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.act_save)) }
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

/**
 * **«13:00» ⇒ «1:00 م»** — الساعةُ كما يقرؤها صاحبُ المتجر في منبّه هاتفه
 * (٢٠٢٦-١٠-١٣). والأرقامُ لاتينيّةٌ كبقيّة الشاشة.
 */
internal fun clock12(hhmm: String): String {
    val h = hhmm.take(2).toIntOrNull() ?: return hhmm
    val m = hhmm.drop(3).take(2).ifBlank { "00" }
    val h12 = when {
        h == 0 -> 12
        h > 12 -> h - 12
        else -> h
    }
    return "$h12:$m " + if (h < 12) "ص" else "م"
}

/** **أدنى سعرِ صرفٍ يُحفظ وحدَه** — دونه كتابةٌ لم تكتمل لا سعر (تدقيقُ ٢٠٢٦-١٠-٢٧). */
internal const val MIN_RATE = 10.0

/**
 * **أيفتح بعد منتصف الليل؟** (٢٠٢٦-١٠-١٣) — يُسأل عنه قبل الحفظ في المحرّر والمعالج معاً.
 * **و«مفتوح ٢٤ ساعة» ليس غلطاً** — ٠٠:٠٠ ← ٠٠:٠٠ مقصودة (٢٠٢٦-١٠-١٨).
 */
internal fun opensAtNight(d: DayHours): Boolean {
    val open = if (d.isOpen24h) -1 else d.openTime.take(2).toIntOrNull() ?: 9
    val close = d.closeTime.take(2).toIntOrNull() ?: 23
    return !d.closed && open in 0..5 && !(open == 0 && close >= 23)
}

/**
 * **سؤالٌ وزرّان كبيران** (طلبُ المالك ٢٠٢٦-١٠-١٢) — المختارُ ممتلئٌ والآخرُ مؤطَّر،
 * **وبلا اختيارٍ يُقال تحتهما «اختار وحدة»** فيُعرف أنّه إلزاميّ.
 */
@Composable
internal fun BigChoice(question: String, options: List<Pair<String, Boolean>>, enabled: Boolean, onPick: (Int) -> Unit) {
    Text(question, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
    Spacer(Modifier.height(8.dp))
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        options.forEachIndexed { i, (label, on) ->
            if (on) {
                RahalButton(onClick = { onPick(i) }, enabled = enabled, modifier = Modifier.weight(1f)) {
                    Text(label, fontWeight = FontWeight.Bold)
                }
            } else {
                RahalOutlineButton(onClick = { onPick(i) }, enabled = enabled, modifier = Modifier.weight(1f)) {
                    Text(label)
                }
            }
        }
    }
    if (options.none { it.second }) {
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.choose_one), color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
    }
}

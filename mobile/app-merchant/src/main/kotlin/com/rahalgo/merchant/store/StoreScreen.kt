package com.rahalgo.merchant.store

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.border
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.Card
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
                    label = { Text(stringResource(R.string.store_name)) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(6.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    RahalButton(
                        onClick = { vm.rename(editingName); renaming = null },
                        enabled = !vm.saving && editingName.isNotBlank(),
                    ) { Text(stringResource(R.string.save)) }
                    RahalTextButton(onClick = { renaming = null }) {
                        Text(stringResource(R.string.cancel))
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
                            Text(stringResource(R.string.cancel))
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
                Card {
                    KeyValue(
                        stringResource(R.string.reports_today_orders),
                        r.summary.orders.toString(),
                    )
                    KeyValue(
                        stringResource(R.string.reports_delivered),
                        r.summary.delivered.toString(),
                    )
                    KeyValue(
                        stringResource(R.string.reports_cancelled),
                        r.summary.cancelled.toString(),
                    )
                    HorizontalDivider()
                    Spacer(Modifier.height(6.dp))
                    KeyValue(
                        stringResource(R.string.reports_due_today),
                        money(r.summary.due),
                        valueColor = Rahal.colors.brand,
                    )
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
            ) { Text(stringResource(R.string.save)) }
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
    BasicTextField(
        // **والثواني تُقطع للعرض** — المحرّكُ يردّ `08:00:00`.
        value = value.take(5),
        onValueChange = { onChange(it.take(5)) },
        singleLine = true,
        textStyle = MaterialTheme.typography.bodyMedium.copy(
            color = Rahal.colors.ink,
            textAlign = TextAlign.Center,
        ),
        cursorBrush = SolidColor(Rahal.colors.brand),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier
            .width(62.dp)
            .border(1.dp, Rahal.colors.line, Rahal.shape.sm)
            .padding(vertical = 6.dp),
    )
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
            stringResource(R.string.store_address),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        if (dirty) {
            RahalButton(
                onClick = { vm.saveAddress(text, vm.pickedLat, vm.pickedLng) },
                enabled = !vm.saving,
            ) { Text(stringResource(R.string.save)) }
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

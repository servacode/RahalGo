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

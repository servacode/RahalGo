package com.rahalgo.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.material3.AlertDialog
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.heightIn
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأزرار — ثلاثةُ أشكالٍ وأربعُ نبرات، ولا رابع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «بدنا نوحّد ستايلَ التطبيقات… الألوان
 *
 *	والأزرار وشكلها وكلّ شيءٍ بيها يكون موحّد — يعني مو كلّ تطبيقٍ وكلّ
 *	صفحةٍ وكلّ قسمٍ تصميمه على كيفه».)
 *
 * # ما كان يقع
 *
 * **مئةٌ وخمسةَ عشرَ زرَّ Material خاما** — كلٌّ يختار شكلَه وحشوتَه
 * ولونَه في موضعه. **فزرٌّ في تطبيق السائق نصفُ قطره ١٢ وفي الزبون ٨
 * وفي المندوب ١٦** — ولا أحدَ قرّر ذلك.
 *
 * **وأخطرُ من اختلاف الشكل اختلافُ المعنى**: زرُّ الإلغاء أحمرُ في
 * شاشةٍ ورماديٌّ في أخرى، **فيُقرأ الخطرُ في موضعٍ ولا يُقرأ في مثله.**
 *
 * # ولماذا نبرةٌ لا لون
 *
 * **`tone = Tone.Danger` تقول ما يفعله الزرّ**، و`containerColor =
 * Rahal.colors.danger` تقول ما لونُه. **والأولى تبقى صحيحةً يومَ
 * يتبدّل اللون** — والثانية تصير موضعاً يُبحث عنه.
 *
 * **ولا نبرةَ خامسة**: لونٌ خامسٌ يعني معنًى خامساً لا يعرفه المستخدم.
 *
 * # وما ليس هنا
 *
 * **لا زرَّ بأيقونةٍ ولا زرَّ عائم** — لكلٍّ منهما قطعتُه حين تلزم.
 * **وقطعةٌ تحمل كلَّ الحالات تصير مفتاحَ تشغيلٍ بعشرة أذرع.**
 */
enum class Tone {
    /** **الفعلُ الأساسيّ** — حفظٌ وإرسالٌ ومتابعة. */
    Brand,

    /** **فعلٌ ثانويٌّ ملفت** — كالتقاط الموقع وإضافة عنوان. */
    Accent,

    /** **ما يُتمّ ويُؤكّد** — تسليمٌ وقبولٌ واتّفاق. */
    Success,

    /** **ما لا يُستردّ** — حذفٌ وإلغاءٌ ورفض. */
    Danger,

    /** On a dark/brand background: white outline and text (driver offer banner). */
    OnDark,

    /**
     * **البرتقاليُّ الغامقُ بخطٍّ أبيض** — زرُّ «إنشاء حساب جديد» (طلبُ المالك ٢٠٢٦-١١-٠٦: «لون
     * الزر برتقالي ليتميّز بكتابة بخط أبيض»). **أغمقُ من `Accent` درجة** ليُقرأ الأبيضُ عليه في الشمس.
     */
    AccentSolid,
}

/**
 * **نافذةُ تأكيد السعر — القبولُ أخضرُ والرفضُ أحمر** (طلبُ المالك ٢٠٢٦-١٠-١٨)
 *
 * **في كلّ نافذةِ سعرٍ أو مبلغ**: عرضُ الطلب الخاصّ، وثمنُ البضاعة، ورفعُ
 * المبلغ، ومبلغُ الإحضار عند المتجر. **وزرٌّ أزرقُ للقبول يُقرأ «متابعة»** لا
 * «موافق على المبلغ». **وموضعٌ واحدٌ للنبرتين** — فلا تفترق نافذتان.
 */
object PriceConfirm {
    val Accept: Tone = Tone.Success
    val Reject: Tone = Tone.Danger
}

/**
 * **أرضُ النبرة وحبرُها لغير الأزرار** — فقاعةُ الشرح تأخذ لونَ زرّ «إنشاء حساب»
 * نفسَه (`SignupGuide.kt`)، **ولا يُعاد حسابُ البرتقاليّ الغامق في موضعٍ ثانٍ.**
 */
@Composable
fun toneFill(tone: Tone): Color = tone.color()

@Composable
fun toneOn(tone: Tone): Color = tone.onColor()

/** لونُ النبرة — **من اللوح لا من رقمٍ يُكتب هنا.** */
@Composable
private fun Tone.color(): Color = when (this) {
    Tone.Brand -> Rahal.colors.brand
    Tone.Accent -> Rahal.colors.accent
    Tone.Success -> Rahal.colors.success
    Tone.Danger -> Rahal.colors.danger
    Tone.OnDark -> Color.White
    // **الأبيضُ على البرتقاليّ ≥ ٤٫٥:١** (D15: كان ٣٫٢) — الفاتحةُ بحبر البرتقاليّ الغامق نفسِه
    // (`accentInk` ≈ ٥:١ مع الأبيض)، والغامقةُ تمزج نصفَها إلى اللوح (≈ ٥٫٦:١).
    Tone.AccentSolid -> if (Rahal.colors.dark) {
        androidx.compose.ui.graphics.lerp(Rahal.colors.accent, Rahal.colors.panel, 0.55f)
    } else {
        Rahal.colors.accentInk
    }
}

/**
 * **ما يُكتب على النبرة المملوءة** — من اللوح حسب السمة لا أبيضُ ثابت.
 *
 * **والأبيضُ على سماويّ الغامقة (`#4FB8E8`) تباينُه قرابةَ ٢ من ٢١**، وعلى
 * البرتقاليّ مثلُه في السمتين — فالبرتقاليُّ يُكتب عليه حبرٌ غامق، والباقي
 * `onBrand` (أبيضُ في الفاتحة وكحليٌّ غامقٌ في الغامقة).
 */
@Composable
private fun Tone.onColor(): Color = when (this) {
    Tone.Accent -> Rahal.colors.onLight
    Tone.OnDark -> Color.White
    Tone.AccentSolid -> Rahal.colors.onPanel
    else -> Rahal.colors.onBrand
}

/** **حبرُ النبرة للزرّ المحدَّد والنصّيّ** — البرتقاليُّ نصّاً بحبره الغامق. */
@Composable
private fun Tone.inkColor(): Color = when (this) {
    Tone.Accent, Tone.AccentSolid -> Rahal.colors.accentInk
    else -> color()
}

/**
 * **حشوةٌ واحدةٌ لكلّ الأزرار** — والارتفاعُ يتبعها.
 *
 * **وحشوةٌ تُكتب في الموضع تجعل زرّين متجاورين مختلفَي الارتفاع** —
 * وهو ما يُقرأ «تصميمٌ مهمَل» ولا يُعرف سببُه.
 */
private val Pad = PaddingValues(horizontal = 20.dp, vertical = 12.dp)

/**
 * **حشوةٌ ضيّقةٌ لصفٍّ من ثلاثة** — شاشةُ الهاتف لا تتّسع لثلاثةٍ
 * عريضة، **والزرُّ الأوّلُ ينزل إلى سطرين.**
 *
 * **وهي الاستثناءُ الوحيدُ المسموح** — ولا تُفتح للنداء بحشوةٍ من
 * عنده: **وسيطٌ يقبل أيَّ حشوةٍ يعيد التشتّتَ بابا خلفيّا.**
 */
private val PadTight = PaddingValues(horizontal = 6.dp, vertical = 10.dp)

/**
 * **الزرُّ المملوء** — الفعلُ الأساسيّ في الشاشة.
 *
 * **وواحدٌ في الشاشة لا اثنان**: زرّان مملوءان يتنازعان العين،
 * **فلا يُعرف أيُّهما المقصود.**
 */
@Composable
fun RahalButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    compact: Boolean = false,
    content: @Composable RowScope.() -> Unit,
) {
    Button(
        onClick = onClick,
        modifier = modifier.dialogFill(),
        enabled = enabled,
        shape = Rahal.shape.md,
        contentPadding = if (compact) PadTight else Pad,
        colors = ButtonDefaults.buttonColors(
            containerColor = tone.color(),
            // **والنصُّ من اللوح حسب النبرة والسمة** — لا أبيضُ ثابت.
            contentColor = tone.onColor(),
            // ══════════════════════════════════════════════════════════
            // **والمعطَّلُ يُرى معطَّلاً لا يختفي**
            // ══════════════════════════════════════════════════════════
            //
            // (بلاغُ المالك ٢٠٢٦-٠٨-٢٥: «زرّ الدخول أيضاً» — في السمة
            //  الغامقة.)
            //
            // **وافتراضُ Material يخفض الشفافيّةَ إلى ١٢٪ للأرض و٣٨٪
            // للحبر** — وهي نسبٌ حُسبت لأرضٍ بيضاء. **وعلى أرضٍ غامقةٍ
            // تذوب**، فيبدو الزرُّ غائباً لا معطّلاً.
            //
            // **والمعطَّلُ يجب أن يُرى**: هو ما سيعمل حين يملأ حقلَه،
            // **ومن لم يرَه لم يعرف أنّ ثمّة زرّا.**
            disabledContainerColor = tone.color().copy(alpha = 0.30f),
            disabledContentColor = tone.onColor().copy(alpha = 0.55f),
        ),
        content = content,
    )
}

/**
 * **الزرُّ المحدَّد** — البديلُ المتاح لا الفعلُ المقصود.
 *
 * **وحدُّه بلون نبرته**: حدٌّ رماديٌّ يجعل «إلغاءَ الحساب» و«رجوعاً»
 * سواءً في العين.
 */
@Composable
fun RahalOutlineButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    /** **حشوةٌ ضيّقة** — لزرٍّ في صفٍّ من ثلاثة، كـ`RahalButton`. */
    compact: Boolean = false,
    content: @Composable RowScope.() -> Unit,
) {
    val c = tone.inkColor()
    OutlinedButton(
        onClick = onClick,
        modifier = modifier.dialogFill(),
        enabled = enabled,
        shape = Rahal.shape.md,
        contentPadding = if (compact) PadTight else Pad,
        border = BorderStroke(Rahal.stroke.hair, c),
        colors = ButtonDefaults.outlinedButtonColors(contentColor = c),
        content = content,
    )
}

/**
 * **الزرُّ الثانويّ الخفيف** — يُقرأ زرّا لا رابطا (قرارُ المالك ٢٠٢٦-١١-١٠: «كلّ زرٍّ لازم
 * يبين إنو زر»).
 *
 * **أرضٌ بلون النبرة بشفافيّةٍ خفيفة + حدٌّ شعريٌّ بلونها + `Rahal.shape.md` + ارتفاعٌ
 * لا يقلّ عن ٤٨dp + خطٌّ متوسّط.** نصٌّ بلا أرضٍ ولا حدٍّ لا يعرف الهاتفُ أنّه يُضغط
 * (لا تحويمَ على اللمس).
 */
@Composable
fun RahalTextButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    content: @Composable RowScope.() -> Unit,
) {
    TintedTextButton(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled,
        ink = tone.inkColor(),
        minHeight = 48.dp,
        padding = Pad,
        borderAlpha = 0.55f,
        fillAlpha = 0.10f,
        textStyle = null,
        content = content,
    )
}

/**
 * **الزرُّ الصغيرُ الثانويّ** — لنسيان كلمة المرور وتعديل الرقم وإعادة الإرسال: **نبرةٌ واحدةٌ**
 * (النصُّ والأرضُ الخفيفةُ والحدُّ من اللون نفسِه)، مدمجٌ، ويُوضع بمحاذاة البداية. **ولا يلتصق
 * بما حوله**: المنادي يضع `Rahal.space.md` فوقه وتحته.
 */
@Composable
fun RahalSmallButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    /** **حبرٌ مخصّص** (رماديّ للثانويّ جدّاً) — وبلا قيمةٍ يؤخذ حبرُ النبرة. */
    ink: Color? = null,
    content: @Composable RowScope.() -> Unit,
) {
    TintedTextButton(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled,
        ink = ink ?: tone.inkColor(),
        minHeight = 40.dp,
        padding = PaddingValues(horizontal = 14.dp, vertical = 6.dp),
        borderAlpha = 0.45f,
        fillAlpha = 0.08f,
        textStyle = 13.sp,
        content = content,
    )
}

/**
 * **الزرُّ النصّيُّ المحدَّد الوحيد** — أرضٌ وحدٌّ ونصٌّ من اللون نفسِه. **موضعٌ واحدٌ** يبني `TextButton`
 * فلا يتكاثر الشكلُ الخامّ (حارسُ `check-design-tokens`).
 */
@Composable
private fun TintedTextButton(
    onClick: () -> Unit,
    modifier: Modifier,
    enabled: Boolean,
    ink: Color,
    minHeight: androidx.compose.ui.unit.Dp,
    padding: PaddingValues,
    borderAlpha: Float,
    fillAlpha: Float,
    textStyle: androidx.compose.ui.unit.TextUnit?,
    content: @Composable RowScope.() -> Unit,
) {
    TextButton(
        onClick = onClick,
        modifier = modifier.dialogFill().heightIn(min = minHeight),
        enabled = enabled,
        shape = Rahal.shape.md,
        contentPadding = padding,
        border = BorderStroke(Rahal.stroke.hair, ink.copy(alpha = if (enabled) borderAlpha else borderAlpha / 2f)),
        colors = ButtonDefaults.textButtonColors(
            containerColor = ink.copy(alpha = fillAlpha),
            contentColor = ink,
            disabledContainerColor = ink.copy(alpha = fillAlpha / 2f),
            disabledContentColor = ink.copy(alpha = 0.45f),
        ),
    ) {
        val base = androidx.compose.material3.LocalTextStyle.current
        androidx.compose.runtime.CompositionLocalProvider(
            androidx.compose.material3.LocalTextStyle provides
                (if (textStyle != null) base.copy(fontWeight = FontWeight.Medium, fontSize = textStyle, color = ink)
                else base.copy(fontWeight = FontWeight.Medium)),
        ) { content() }
    }
}

/**
 * **زرُّ «أكّد» في نافذة الحوار** — مملوءٌ دائماً (`confirmButton`)، والخطِرُ بـ`Tone.Danger`.
 * **موضعٌ واحدٌ لكلّ النوافذ**، فلا تفترق نافذتان.
 */
@Composable
fun DialogConfirmButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    content: @Composable RowScope.() -> Unit,
) = RahalButton(onClick = onClick, modifier = modifier, enabled = enabled, tone = tone, content = content)

/** **زرُّ «تراجع» في نافذة الحوار** — محدَّدٌ دائماً (`dismissButton`). */
@Composable
fun DialogDismissButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    content: @Composable RowScope.() -> Unit,
) = RahalOutlineButton(onClick = onClick, modifier = modifier, enabled = enabled, tone = tone, content = content)

/**
 * **نصُّ زرٍّ يصغر حتّى يسع ولا يُقصّ** (بلاغُ المالك ٢٠٢٦-١٠-٣٠: «افة صنف» و«شاء عرض») —
 * لأزرارٍ ثلاثةٍ في صفٍّ على شاشةٍ ضيّقة.
 */
@Composable
fun FitButtonText(text: String) {
    val style = androidx.compose.material3.MaterialTheme.typography.labelLarge.copy(color = androidx.compose.material3.LocalContentColor.current)
    androidx.compose.foundation.text.BasicText(
        text = text,
        style = style,
        maxLines = 1,
        autoSize = androidx.compose.foundation.text.TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = style.fontSize),
    )
}


// ══════════════════════════════════════════════════════════════════════
// **أزرارُ النافذة — عرضان متساويان وفاصلٌ بينهما** (ملاحظتا المالك ٢٠٢٦-١١-١١: «رجوع» و«إرسال»
// ملتصقان ومكوّمان في جهة، و«تراجع» و«نعم، اعتمده» كذلك)
// ══════════════════════════════════════════════════════════════════════

/** **هل الزرُّ داخل صفِّ نافذةٍ بزرّين؟** — فيملأ عرضَ خانته. */
private val LocalDialogFill = compositionLocalOf { false }

@Composable
private fun Modifier.dialogFill(): Modifier =
    if (LocalDialogFill.current) this.fillMaxWidth() else this

/** **الفاصلُ بين زرّي النافذة** — بين ٨ و١٢. */
private val DialogButtonGap = 10.dp

/**
 * **صفُّ زرّي النافذة** — `dismiss` و`confirm` بعرضين متساويين على كامل العرض وفاصلٍ بينهما.
 * **يُستعمل داخل `RahalDialog`**، ومن بنى نافذةً بيده ناداه في خانة `confirmButton` وحدَها.
 */
@Composable
fun DialogButtons(
    confirm: @Composable () -> Unit,
    dismiss: @Composable (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    if (dismiss == null) {
        Row(modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) { confirm() }
        return
    }
    CompositionLocalProvider(LocalDialogFill provides true) {
        Row(modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(DialogButtonGap)) {
            Box(Modifier.weight(1f)) { confirm() }
            Box(Modifier.weight(1f)) { dismiss() }
        }
    }
}

/**
 * **نافذةُ الحوار الموحَّدة** — بديلُ `AlertDialog` في التطبيقات كلِّها: الزرّان بعرضين متساويين
 * وفاصل (`DialogButtons`). **ومن نادى `AlertDialog` خاماً رجع الالتصاقُ.**
 */
@Composable
fun RahalDialog(
    onDismissRequest: () -> Unit,
    confirmButton: @Composable () -> Unit,
    modifier: Modifier = Modifier,
    dismissButton: @Composable (() -> Unit)? = null,
    icon: @Composable (() -> Unit)? = null,
    title: @Composable (() -> Unit)? = null,
    text: @Composable (() -> Unit)? = null,
    shape: Shape = androidx.compose.material3.AlertDialogDefaults.shape,
    properties: DialogProperties = DialogProperties(),
) {
    AlertDialog(
        onDismissRequest = onDismissRequest,
        confirmButton = { DialogButtons(confirm = confirmButton, dismiss = dismissButton) },
        modifier = modifier,
        icon = icon,
        title = title,
        text = text,
        shape = shape,
        properties = properties,
    )
}

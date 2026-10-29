package com.rahalgo.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.RowScope
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
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

/** لونُ النبرة — **من اللوح لا من رقمٍ يُكتب هنا.** */
@Composable
private fun Tone.color(): Color = when (this) {
    Tone.Brand -> Rahal.colors.brand
    Tone.Accent -> Rahal.colors.accent
    Tone.Success -> Rahal.colors.success
    Tone.Danger -> Rahal.colors.danger
    Tone.OnDark -> Color.White
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
    else -> Rahal.colors.onBrand
}

/** **حبرُ النبرة للزرّ المحدَّد والنصّيّ** — البرتقاليُّ نصّاً بحبره الغامق. */
@Composable
private fun Tone.inkColor(): Color = when (this) {
    Tone.Accent -> Rahal.colors.accentInk
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
        modifier = modifier,
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
    content: @Composable RowScope.() -> Unit,
) {
    val c = tone.inkColor()
    OutlinedButton(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled,
        shape = Rahal.shape.md,
        contentPadding = Pad,
        border = BorderStroke(Rahal.stroke.hair, c),
        colors = ButtonDefaults.outlinedButtonColors(contentColor = c),
        content = content,
    )
}

/**
 * **الزرُّ النصّيّ** — ما يُقرأ رابطاً لا زرّا.
 *
 * **ولا يحمل الفعلَ الخطر**: نصٌّ بلا إطارٍ يُضغط بالخطأ، **وحذفٌ
 * خلفَه لا يُستردّ.**
 */
@Composable
fun RahalTextButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tone: Tone = Tone.Brand,
    content: @Composable RowScope.() -> Unit,
) {
    TextButton(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled,
        shape = Rahal.shape.md,
        colors = ButtonDefaults.textButtonColors(contentColor = tone.inkColor()),
        content = content,
    )
}

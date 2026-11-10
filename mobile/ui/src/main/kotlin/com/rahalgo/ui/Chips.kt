package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نبرةُ الحال — خمسُ معانٍ لكلّ شارةٍ وبطاقةٍ ملوّنة** (نظامُ التصميم ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **النبرةُ تقول المعنى لا اللون** — `StatusTone.Danger` تبقى صحيحةً يومَ يتبدّل الأحمر.
 * (وأزرارُ الفعل لها `Tone` في `Buttons.kt`: الزرُّ فعلٌ والشارةُ حال.)
 */
enum class StatusTone {
    /** **تمّ** — مُسلَّم، مدفوع، يعمل. أخضر. */
    Success,

    /** **انتبه** — بانتظار، ناقص، قريبٌ من الحدّ. برتقاليّ. */
    Warning,

    /** **خطر** — ملغى، مرفوض، اقبض. أحمر. */
    Danger,

    /** **معلومة** — جديد، قيد التنفيذ. لونُ العلامة. */
    Info,

    /** **عاديّ** — متوقّف، قديم، بلا حال. رماديّ. */
    Neutral,
}

/** **لونُ النبرة للأرض والحدّ والنقطة.** */
@Composable
fun StatusTone.color(): Color = when (this) {
    StatusTone.Success -> Rahal.colors.success
    StatusTone.Warning -> Rahal.colors.accent
    StatusTone.Danger -> Rahal.colors.danger
    StatusTone.Info -> Rahal.colors.brand
    StatusTone.Neutral -> Rahal.colors.inkMuted
}

/** **حبرُ النبرة للنصّ** — البرتقاليُّ نصّاً بحبره الغامق (البرتقاليُّ نفسُه على الأبيض لا يُقرأ). */
@Composable
fun StatusTone.ink(): Color = when (this) {
    StatusTone.Warning -> Rahal.colors.accentInk
    else -> color()
}

/**
 * **رقاقةُ الحال** — كلمةٌ قصيرةٌ في أرضٍ من لونها: «مُسلَّم»، «بانتظار المتجر»، «ملغى».
 *
 * ```
 * StatusChip("مُسلَّم", StatusTone.Success)
 * StatusChip("بانتظار", StatusTone.Warning, icon = R.drawable.ic_time)
 * ```
 */
@Composable
fun StatusChip(
    text: String,
    tone: StatusTone,
    modifier: Modifier = Modifier,
    @androidx.annotation.DrawableRes icon: Int? = null,
) {
    ChipBody(text = text, ink = tone.ink(), fill = tone.color(), modifier = modifier, icon = icon)
}

/** **شارةٌ** — اسمٌ آخرُ لـ[StatusChip] لِما ليس حالاً: «جديد»، «-١٠٪»، «مميّز». */
@Composable
fun Badge(
    text: String,
    tone: StatusTone = StatusTone.Info,
    modifier: Modifier = Modifier,
    @androidx.annotation.DrawableRes icon: Int? = null,
) = StatusChip(text = text, tone = tone, modifier = modifier, icon = icon)

/**
 * **نقطةُ حالٍ ونصُّها** — «يعمل» أخضر و«متوقّف» رماديّ. أخفُّ من الرقاقة، لسطرٍ تحت اسم.
 *
 * ```
 * StatusDot("يعمل", Rahal.colors.success)        // بلون
 * StatusDot("يعمل", StatusTone.Success)          // بنبرة
 * ```
 */
@Composable
fun StatusDot(text: String, color: Color, modifier: Modifier = Modifier) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(Rahal.size.dot).clip(Rahal.shape.pill).background(color))
        Spacer(Modifier.size(Rahal.space.xs))
        Text(text, color = color, style = Rahal.type.label, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
fun StatusDot(text: String, tone: StatusTone, modifier: Modifier = Modifier) =
    StatusDot(text = text, color = tone.ink(), modifier = modifier)

/** **جسدُ الرقاقة الواحد** — لـ`StatusChip` و`Chip` القديمة. */
@Composable
internal fun ChipBody(
    text: String,
    ink: Color,
    fill: Color,
    modifier: Modifier = Modifier,
    @androidx.annotation.DrawableRes icon: Int? = null,
) {
    Row(
        modifier
            .clip(Rahal.shape.sm)
            .background(fill.copy(alpha = 0.12f))
            .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xxs),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
    ) {
        if (icon != null) {
            Icon(painterResource(icon), contentDescription = null, tint = ink, modifier = Modifier.size(Rahal.icon.xs))
        }
        Text(text, color = ink, style = Rahal.type.badge, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

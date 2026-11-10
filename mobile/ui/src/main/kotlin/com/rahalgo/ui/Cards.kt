package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.Dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البطاقة — أربعةُ أشكالٍ ولا خامس** (نظامُ التصميم الموحَّد ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * - [CardVariant.Elevated] **مرفوعةٌ بظلٍّ خفيف** — البطاقةُ العاديّة: طلبٌ، متجرٌ، قسمُ إعدادات.
 * - [CardVariant.Outlined] **بحدٍّ بلا ظلّ** — بطاقةٌ داخل بطاقة، أو قائمةٌ طويلةٌ متلاصقة.
 * - [CardVariant.Plain] **أرضٌ رماديّةٌ خفيفةٌ بلا حدٍّ ولا ظلّ** — مجموعةٌ هادئة، ملخّص.
 * - [CardVariant.Tinted] **أرضٌ بلون الحال وحدٌّ منه** — تنبيهٌ، تمامٌ، رسومٌ ومعلومة. **ولا ظلَّ لها**:
 *   بطاقةُ معلومةٍ بظلٍّ تُقرأ زرّاً (D9).
 *
 * **والضغطُ اختياريّ** (`onClick`): بطاقةٌ تُفتح تأخذ دورَ الزرّ لقارئ الشاشة.
 */
enum class CardVariant { Elevated, Outlined, Plain, Tinted }

/**
 * **البطاقةُ الموحَّدة** — كلُّ صندوقٍ في الشاشة يُبنى بها.
 *
 * ```
 * RahalCard { Text("…") }                                     // مرفوعة
 * RahalCard(variant = CardVariant.Outlined) { … }
 * RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Warning) { … }
 * RahalCard(onClick = { open(id) }) { ListRow(…) }
 * ```
 *
 * @param tone لونُ الحال لـ`Tinted` (ويُقبل مع غيرها فيلوّن الحدَّ وحدَه). وبلا قيمةٍ: لونُ العلامة.
 * @param padding حشوةُ الداخل — `Rahal.space.md` افتراضاً، و`0.dp` لبطاقةِ صورةٍ تملأ حوافَّها.
 */
@Composable
fun RahalCard(
    modifier: Modifier = Modifier,
    variant: CardVariant = CardVariant.Elevated,
    tone: StatusTone? = null,
    onClick: (() -> Unit)? = null,
    padding: Dp = Rahal.space.md,
    content: @Composable ColumnScope.() -> Unit,
) {
    RahalCardSurface(
        modifier = modifier,
        variant = variant,
        toneColor = tone?.color(),
        onClick = onClick,
        padding = padding,
        content = content,
    )
}

/**
 * **جسدُ البطاقة بلونٍ لا بنبرة** — لـ`Card(tone = Rahal.colors.success)` القديمة. **والجديدُ يمرّ بـ`RahalCard`.**
 */
@Composable
internal fun RahalCardSurface(
    modifier: Modifier,
    variant: CardVariant,
    toneColor: Color?,
    onClick: (() -> Unit)?,
    padding: Dp,
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = Rahal.shape.lg
    Column(
        modifier
            .fillMaxWidth()
            .cardVariant(variant, toneColor, shape)
            .then(if (onClick != null) Modifier.clickable(role = Role.Button, onClick = onClick) else Modifier)
            .padding(padding),
        content = content,
    )
}

/** **سطحُ الشكل الواحد** — يُستعمل في `RahalCard` ولمن احتاج السطحَ بلا عمود (صورةٌ، صفّ). */
@Composable
fun Modifier.cardVariant(
    variant: CardVariant,
    tone: Color? = null,
    shape: Shape = Rahal.shape.lg,
): Modifier = when (variant) {
    CardVariant.Elevated -> cardSurface(shape = shape, tone = tone)
    CardVariant.Outlined -> this
        .clip(shape)
        .background(Rahal.colors.canvas)
        .border(Rahal.stroke.hair, tone?.copy(alpha = 0.45f) ?: Rahal.colors.line, shape)
    CardVariant.Plain -> this
        .clip(shape)
        .background(Rahal.colors.surface)
    CardVariant.Tinted -> {
        val c = tone ?: Rahal.colors.brand
        this
            .clip(shape)
            .background(Rahal.colors.canvas)
            .background(c.copy(alpha = 0.08f))
            .border(Rahal.stroke.hair, c.copy(alpha = 0.30f), shape)
    }
}

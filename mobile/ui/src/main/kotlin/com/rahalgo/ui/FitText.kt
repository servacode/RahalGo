package com.rahalgo.ui

import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.LocalTextStyle
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.takeOrElse
import androidx.compose.ui.layout.IntrinsicMeasurable
import androidx.compose.ui.layout.IntrinsicMeasureScope
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.layout.Measurable
import androidx.compose.ui.layout.MeasurePolicy
import androidx.compose.ui.layout.MeasureResult
import androidx.compose.ui.layout.MeasureScope
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.text
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.sp

/**
 * **نصٌّ يصغر حتّى يسع — ولا تنكسر كلمتُه نصفين** (الفحصُ البصريّ ٢٠٢٦-١١-١١).
 *
 * على ٣٦٠ نقطةً بخطٍّ ١٫٣ كانت الأسماءُ القصيرةُ في المربّعات والدرج والشريط السفليّ
 * **تنكسر وسطَ الكلمة** («العرو/ض»، «المشروبا/ت») **أو تُقصّ إلى كلمتها الأولى**
 * («سياسة» بدل «سياسة الخصوصية»). **فيُقاس النصُّ أوّلاً**: يُصغَّر الخطُّ درجةً درجةً
 * حتّى تسع كلُّ كلمةٍ سطرَها ويسع النصُّ كلُّه في [maxLines] — ولا ينزل تحت [minSize].
 * **وبالحجم الأصليّ ما دام يسع** — فلا يصغر نصٌّ بلا داعٍ.
 *
 * **ويُرسم بتخطيطٍ عاديٍّ لا بـ`BoxWithConstraints`** — هذا لا يُجيب عن القياس الذاتيّ
 * فيُسقط `ActionRow` (يسأل عن أطول قطعة) — **وقع على المحاكي في أوّل نسخة.**
 */
@Composable
fun FitText(
    text: String,
    modifier: Modifier = Modifier,
    style: TextStyle = LocalTextStyle.current,
    color: Color = Color.Unspecified,
    maxLines: Int = 2,
    minSize: TextUnit = 10.sp,
    textAlign: TextAlign? = null,
) {
    val measurer = rememberTextMeasurer()
    val ink = color.takeOrElse { style.color.takeOrElse { LocalContentColor.current } }
    val base = style.copy(color = ink, textAlign = textAlign ?: style.textAlign)
    val last = remember { arrayOfNulls<TextLayoutResult>(1) }
    val policy = remember(text, base, maxLines, minSize, measurer) {
        FitPolicy(text, base, maxLines, minSize, measurer, last)
    }
    Layout(
        modifier = modifier
            .semantics { this.text = AnnotatedString(text) }
            // **واللونُ يُمرَّر عند الرسم** (ملاحظةُ المالك ٢٠٢٦-١١-١١: «تسوق» تظلّ برتقاليّةً) —
            // تبديلُ التبويب يغيّر اللونَ لا الحجم، فلا يُعاد القياس ويبقى المرسومُ بلونه القديم.
            // **ولامدا جديدةٌ بلونٍ جديدٍ تُعيد الرسم.**
            .drawBehind { last[0]?.let { drawText(it, color = ink) } },
        measurePolicy = policy,
    )
}

private class FitPolicy(
    val text: String,
    val style: TextStyle,
    val maxLines: Int,
    val minSize: TextUnit,
    val measurer: TextMeasurer,
    val last: Array<TextLayoutResult?>,
) : MeasurePolicy {
    private val words = text.split(' ', '\n').filter { it.isNotEmpty() }

    private fun at(size: Float): TextStyle = style.copy(
        fontSize = size.sp,
        lineHeight = if (style.lineHeight.isSp && style.fontSize.isSp) {
            (style.lineHeight.value * size / style.fontSize.value).sp
        } else {
            style.lineHeight
        },
    )

    /** **أكبرُ حجمٍ يسع فيه** — كلُّ كلمةٍ في سطرها والنصُّ في [maxLines]. */
    private fun fit(maxW: Int): TextLayoutResult {
        val bounded = maxW != Constraints.Infinity
        var size = if (style.fontSize.isSp) style.fontSize.value else 0f
        if (bounded && style.fontSize.isSp) {
            while (size > minSize.value) {
                val s = at(size)
                val wordsFit = words.all {
                    measurer.measure(it, s, softWrap = false, maxLines = 1).size.width <= maxW
                }
                val whole = measurer.measure(text, s, maxLines = maxLines, constraints = Constraints(maxWidth = maxW))
                if (wordsFit && !whole.hasVisualOverflow) break
                size -= 1f
            }
            size = size.coerceAtLeast(minSize.value)
        }
        val s = if (style.fontSize.isSp) at(size) else style
        return measurer.measure(
            text, s,
            overflow = TextOverflow.Ellipsis,
            maxLines = maxLines,
            constraints = if (bounded) Constraints(maxWidth = maxW) else Constraints(),
        )
    }

    override fun MeasureScope.measure(measurables: List<Measurable>, constraints: Constraints): MeasureResult {
        val r = fit(constraints.maxWidth)
        last[0] = r
        return layout(
            r.size.width.coerceIn(constraints.minWidth, constraints.maxWidth),
            r.size.height.coerceIn(constraints.minHeight, constraints.maxHeight),
        ) {}
    }

    override fun IntrinsicMeasureScope.minIntrinsicHeight(measurables: List<IntrinsicMeasurable>, width: Int) =
        fit(width).size.height

    override fun IntrinsicMeasureScope.maxIntrinsicHeight(measurables: List<IntrinsicMeasurable>, width: Int) =
        fit(width).size.height

    override fun IntrinsicMeasureScope.maxIntrinsicWidth(measurables: List<IntrinsicMeasurable>, height: Int) =
        measurer.measure(text, style, softWrap = false, maxLines = 1).size.width

    override fun IntrinsicMeasureScope.minIntrinsicWidth(measurables: List<IntrinsicMeasurable>, height: Int) =
        words.maxOfOrNull { measurer.measure(it, at(minSize.value), softWrap = false, maxLines = 1).size.width } ?: 0
}

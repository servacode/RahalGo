package com.rahalgo.ui

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مسارُ الطلب — قضيبٌ واحدٌ تمشي عليه الدرّاجة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهو مسارُ الويب نفسُه** (`packages/ui/src/ordertrack.tsx`) — الذي
 * أقرّه المالكُ في شاشة الزبون.
 *
 * # ولماذا أُعيد بناؤه
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٥: «شريط الرحلة أحسّه مختلفاً عن المتّفق عليه،
 *  مو واضحة الرحلة».)
 *
 * **وكان في الجوّال شكلاً ثانيا**: خمسُ خاناتٍ متساويةٍ في كلٍّ منها
 * نقطةٌ وخيطان — **وهو الشكلُ الذي رُفض في الويب (٢٠٢٦-٠٨-٠٧).**
 *
 * **وعلّتُه أنّ الخانةَ ليست الطريق**: النقطةُ تتوسّط خُمسَ العرض،
 * **والطريقُ يبدأ من طرفٍ وينتهي في طرف** — فلا تقع نقطةٌ حيث ينبغي
 * إلّا في الوسط. **ومسارٌ مقطَّعٌ إلى خاناتٍ يُقرأ خياراتٍ لا رحلة.**
 *
 * # وما يقوله هذا الشكل
 *
 * **القضيبُ يقول «كم بقي»** — طولٌ ممتلئٌ وطولٌ فارغ، يُقرأ بلمحةٍ بلا
 * عدٍّ للنقاط.
 *
 * **والدرّاجةُ تقول «أنا هنا»** — (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «شو رأيك
 * نخلّيه أيقونة موتسكل هي تتحرّك وتمشي المراحل»). **وهي أصدقُ ما يقوله
 * تطبيقُ توصيل.**
 *
 * **وتتحرّك في الجاري وحدَه** — وطلبٌ سُلّم أو أُلغي درّاجتُه ساكنة:
 * **حركةٌ في عشر بطاقاتٍ مغلقةٍ ضجيجٌ يُتعب العينَ ولا يدلّ.**
 *
 * # وكلُّ شيءٍ يقف على نسبته
 *
 * **الاسمُ يقف حيث تقف عُقدتُه** — لا في خانةٍ بجوارها. (قرارُ المالك
 * ٢٠٢٦-٠٨-٠٧: «النصوصُ تحت الطريق يجب أن تكون مضبوطةً بشكلٍ صحيح».)
 *
 * **والطريقُ من حافّةٍ إلى حافّة** (قرارُ المالك ٢٠٢٦-١٠-١٥): **كانت حشوةٌ
 * جانبيّةٌ بنصف خانة الاسم تُقصّر المسار** — والآن العقدتان الطرفيّتان على
 * الحافّتين، **واسماهما يحاذيانهما** فلا يخرج نصفُ أحد. **وكلُّ اسمٍ سطرٌ
 * واحد** — والمُنادي يمرّر أسماءً قصيرةً للشريط وحدَه.
 *
 * # ولماذا في الوحدة المشتركة
 *
 * **يقرؤه الزبونُ اليومَ وحدَه** — **والمتجرُ والمندوبُ لم يُبنَ
 * بعدُ**، وكلٌّ منهما يسأل السؤالَ نفسَه: أين وصل هذا الطلب؟
 *
 * **ونسخةٌ في كلّ تطبيقٍ هي ما وقع بين الويب والجوّال**: أُصلح شكلُ
 * الويب (٢٠٢٦-٠٨-٠٧) وبقي الجوّالُ على الشكل المرفوض **ثمانيةَ
 * أيّام** — لا يعلم أحدٌ أنّهما افترقا حتّى رآه المالك.
 */
@Composable
fun Stages(
    labels: List<String>,
    current: Int,
    modifier: Modifier = Modifier,
    /** **الحركةُ للجاري وحدَه** — والمنتهي يسكن. */
    live: Boolean = false,
) {
    if (labels.isEmpty()) return

    val last = (labels.size - 1).coerceAtLeast(1)
    val at = current.coerceIn(0, labels.size - 1)
    // **والنسبةُ تُتدرَّج لا تقفز** — الدرّاجةُ تنزلق إلى موضعها
    // الجديد، **ومن رآها تقفز لم يعرف أنّها تحرّكت.**
    val pct by animateFloatAsState(
        targetValue = at.toFloat() / last,
        animationSpec = tween(600, easing = FastOutSlowInEasing),
        label = "stage",
    )

    // ══════════════════════════════════════════════════════════════════
    // **وقفزةٌ خفيفةٌ ما دام في الطريق — تُقرأ عند الرسم لا التركيب**
    // ══════════════════════════════════════════════════════════════════
    //
    // **إشارةٌ أنّ الأمرَ ما زال يجري** — لا حركةَ تُلهي.
    //
    // # ولماذا `State` لا قيمةٌ مقروءة
    //
    // (قِيس ٢٠٢٦-٠٨-٢٠ على جهاز المالك: شاشةُ الطلبات **٩٩٪ من إطاراتها
    //  ضائعة** — والتسوّقُ بصورِه ١٧٪. **والصورُ لم تكن السبب.**)
    //
    // **وقراءةُ `.value` في جسد المُركِّب تُعيد تركيبَ الصفّ كلَّ إطار**
    // — والصفُّ فيه `BoxWithConstraints`، **وهي تركيبٌ فرعيٌّ يُعاد
    // قياسُه ستّين مرّةً في الثانية.**
    //
    // **وفي قائمةٍ غيرِ كسولةٍ تعمل لكلّ طلبٍ جارٍ** — عشرةٌ في قائمة
    // المالك، **حتّى ما هو خارجَ الشاشة.**
    //
    // **فتُمرَّر حالةً وتُقرأ داخل `graphicsLayer`** — وهي تعمل في طور
    // الرسم: **تتحرّك الطبقةُ ولا يُعاد بناءُ شيء.**
    //
    // **وقِيس بعد النقل**: ٩٩٪ ← ١٪. (والحذفُ الكاملُ أعطى الرقمَ نفسَه،
    // **فلا ثمنَ للحركة إذا قُرئت في موضعها.**)
    val bobState = if (live) {
        val t = rememberInfiniteTransition(label = "ride")
        t.animateFloat(
            initialValue = 0f,
            targetValue = -2f,
            animationSpec = infiniteRepeatable(
                tween(800, easing = FastOutSlowInEasing),
                RepeatMode.Reverse,
            ),
            label = "bob",
        )
    } else {
        null
    }

    // ══════════════════════════════════════════════════════════════════
    // **والأسماءُ سطرٌ واحدٌ لا تتراكب** (قرارُ المالك ٢٠٢٦-١٠-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان الاسمُ يلتفّ حتّى ثلاثة أسطر وينزل الصفّان متناوبين** — فصار
    // الشريطُ سلّماً من الكلمات. **والآن أسماءٌ قصيرةٌ في سطرٍ واحد**
    // («بالانتظار · مقبول · تحضير…») — يمرّرها المُنادي.
    //
    // **والطريقُ من حافّةٍ إلى حافّة**: كانت حشوةٌ جانبيّةٌ بنصف الخانة تُقصّر
    // القضيب. **والآن العقدةُ الأولى على الحافّة الأولى والأخيرةُ على الأخيرة،
    // واسماهما يحاذيان الحافّتين** لا يتوسّطان عقدتيهما — فلا يخرج نصفُهما.
    //
    // **والاسمُ تحت عقدته ما أمكن** — **ويُزاح عنها بقدر ما يمنع التراكب**
    // (`stageLabelStarts`): الطرفُ العريضُ يدفع جارَه قليلاً بدل أن يُقصّ.
    //
    // **وإن لم يسع مجموعُ الأسماء العرضَ صغُر الخطُّ قليلاً** (`StageScales`) —
    // من توكن `labelSmall` نفسِه لا رقمٍ جديد. **والقياسُ بالخطّ الذي يُرسم.**
    val baseStyle = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold)
    val measurer = rememberTextMeasurer()
    val dens = LocalDensity.current

    BoxWithConstraints(modifier.fillMaxWidth().heightIn(min = RowH)) {
        val inset = NodeH.value / 2
        val fit = remember(labels, maxWidth, baseStyle) {
            fitStageLabels(StageScales, maxWidth.value) { scale ->
                val style = baseStyle.copy(fontSize = baseStyle.fontSize * scale)
                labels.map { label ->
                    with(dens) {
                        measurer.measure(label, style, maxLines = 1, softWrap = false).size.width.toDp().value
                    } + LabelSlack
                }
            }.let { (scale, measured) ->
                val widths = stageLabelWidths(maxWidth.value, measured)
                Triple(scale, widths, stageLabelStarts(maxWidth.value, widths, inset))
            }
        }
        val labelStyle = baseStyle.copy(fontSize = baseStyle.fontSize * fit.first)
        val widths = fit.second
        val starts = fit.third
        // **والطريقُ بين مركزي العقدتين الطرفيّتين** — والعقدتان داخل الحافّتين.
        val track = maxWidth - NodeH
        val center = { f: Float -> NodeH / 2 + track * f }

        // ══════════════════════════════════════════════════════════════
        // **القضيبُ وما امتلأ منه**
        // ══════════════════════════════════════════════════════════════
        //
        // **والملءُ من جهة البداية** — وهي اليمين في العربيّة:
        // الإزاحةُ والعرضُ في صندوقٍ محاذاته `Start` يتبعان اتّجاه الشاشة،
        // **فلا يُكتب اتّجاهٌ بيد.**
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(y = VehicleH)
                .fillMaxWidth()
                .height(RailH)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.inkMuted.copy(alpha = 0.18f)),
        )
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(y = VehicleH)
                .width(center(pct))
                .height(RailH)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.accent),
        )

        // ══════════════════════════════════════════════════════════════
        // **والعُقَدُ وأسماؤها — كلٌّ على نسبته**
        // ══════════════════════════════════════════════════════════════
        labels.forEachIndexed { i, label ->
            val f = i.toFloat() / last
            val done = i <= at
            val edge = when {
                labels.size == 1 -> TextAlign.Center
                i == 0 -> TextAlign.Start
                i == labels.size - 1 -> TextAlign.End
                else -> TextAlign.Center
            }
            val w = widths[i].dp

            Box(
                Modifier
                    .align(Alignment.TopStart)
                    .offset(x = center(f) - NodeH / 2, y = VehicleH + (RailH - NodeH) / 2)
                    .size(NodeH)
                    .clip(CircleShape)
                    .background(
                        if (done) Rahal.colors.accent
                        else Rahal.colors.inkMuted.copy(alpha = 0.30f),
                    ),
            )

            Text(
                text = label,
                // **والقادمُ باهتٌ لا مخفيّ** — يعرف ما ينتظره.
                color = if (done) Rahal.colors.accent else Rahal.colors.inkMuted,
                fontWeight = if (i == at) FontWeight.Bold else FontWeight.Normal,
                style = labelStyle,
                textAlign = edge,
                maxLines = 1,
                softWrap = false,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    // **وبحشوةٍ لا بإزاحة** — الحشوةُ تدخل في ارتفاع الشريط
                    // فيتّسع للسطر، **والإزاحةُ لا تُحسب فيُقصّ ما تحتها.**
                    .padding(top = VehicleH + RailH + 6.dp)
                    .offset(x = starts[i].dp)
                    .width(w),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والدرّاجةُ فوق القضيب**
        // ══════════════════════════════════════════════════════════════
        //
        // **ووجهُها نحو مسيرها**: الطريقُ يمشي من اليمين إلى اليسار
        // كاتّجاه القراءة، **ورسمُ المركبة موجَّهٌ إلى اليمين** فتبدو
        // ماشيةً إلى الخلف. **وقلبُها أفقيّاً يجعلها تسير حيث تنظر.**
        // **ولا تخرج عن الحافّتين** — مركزُها مركزُ العقدة ما اتّسع لها.
        Icon(
            painter = painterResource(R.drawable.ic_moto),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier
                .align(Alignment.TopStart)
                .offset(x = (center(pct) - VehicleH / 2).coerceIn(0.dp, (maxWidth - VehicleH).coerceAtLeast(0.dp)))
                // **والقفزةُ في طبقة الرسم** — تُقرأ الحالُ هنا فلا
                // يُعاد تركيبُ شيء. انظر الشرحَ عند `bobState`.
                .graphicsLayer {
                    translationY = (bobState?.value ?: 0f) * density
                }
                .size(VehicleH)
                .scale(scaleX = -1f, scaleY = 1f),
        )
    }
}

/**
 * **مقاييسُ خطّ الأسماء بالترتيب** — الكاملُ أوّلاً، ثمّ أصغرُ قليلاً.
 *
 * **وتصغيرٌ لا التفاف** (قرارُ المالك ٢٠٢٦-١٠-١٥): اسمٌ في سطرين يرفع الشريط،
 * **وخطٌّ أصغرُ بعُشرٍ أو خُمسٍ لا يكاد يُرى.**
 */
internal val StageScales = listOf(1f, 0.9f, 0.8f)

/** **فراغٌ بين اسمين متجاورين** — فلا يلتصق اسمٌ باسم. */
internal const val LabelGap = 4f

/** **هامشُ القياس** — تقريبُ البكسل لا يقصّ آخرَ حرف. */
private const val LabelSlack = 1f
private val VehicleH = 24.dp
private val RailH = 6.dp
private val NodeH = 12.dp

/** **ارتفاعُ الشريط** — درّاجةٌ فقضيبٌ ففراغٌ فسطرٌ واحد. */
private val RowH = VehicleH + RailH + 6.dp + 16.dp

/**
 * **الموضعُ المرغوبُ لبداية الاسم** (dp، من جهة البداية) — **والطرفان يحاذيان الحافّتين.**
 *
 * الأوّلُ يبدأ من الصفر، والأخيرُ ينتهي عند `width`، **والأوسطُ يتوسّط عقدتَه**:
 * `inset + (width − 2·inset) · i/(n−1)`.
 */
internal fun stageLabelPreferred(i: Int, count: Int, width: Float, w: Float, inset: Float = 0f): Float = when {
    count <= 1 -> (width - w) / 2
    i == 0 -> 0f
    i == count - 1 -> width - w
    else -> inset + (width - 2 * inset) * i / (count - 1) - w / 2
}

/** **هل تسع الأسماءُ سطراً واحداً؟** — مجموعُ عروضها وفراغاتها لا يتجاوز العرض. */
internal fun stageLabelsFit(width: Float, widths: List<Float>): Boolean =
    widths.sum() + LabelGap * (widths.size - 1).coerceAtLeast(0) <= width + 0.001f

/**
 * **أوّلُ مقياسٍ تسع به الأسماء** — وإلّا أصغرُها.
 *
 * @param measure عروضُ الأسماء (dp) بمقياسٍ ما — تُقاس بالخطّ في الشاشة.
 * @return المقياسُ المختارُ وعروضُه المقيسة.
 */
internal inline fun fitStageLabels(
    scales: List<Float>,
    width: Float,
    measure: (Float) -> List<Float>,
): Pair<Float, List<Float>> {
    var last = 1f to emptyList<Float>()
    for (k in scales) {
        last = k to measure(k)
        if (stageLabelsFit(width, last.second)) return last
    }
    return last
}

/**
 * **العروضُ النهائيّة** — المقيسةُ كما هي إن وسعت، **وإلّا تُقصّ بنسبةٍ واحدة.**
 *
 * **والقصُّ آخرُ حيلة** بعد أصغر مقياس — **فلا يتراكب اثنان مهما ضاق العرض.**
 */
internal fun stageLabelWidths(width: Float, widths: List<Float>): List<Float> {
    if (stageLabelsFit(width, widths)) return widths
    val room = (width - LabelGap * (widths.size - 1).coerceAtLeast(0)).coerceAtLeast(0f)
    val k = room / widths.sum()
    return widths.map { it * k }
}

/**
 * **بداياتُ الأسماء** — كلٌّ تحت عقدته ما أمكن، **ويُزاح بقدر ما يمنع التراكب.**
 *
 * **والطرفان لا يُزاحان عن الحافّتين.** مسحةٌ من البداية تدفع كلَّ اسمٍ بعد
 * جاره، **ومسحةٌ من النهاية تردّه قبل جاره** — فإن وسع المجموعُ العرضَ لم
 * يتراكب اثنان ولم يخرج أحدٌ عن الحافّتين.
 */
internal fun stageLabelStarts(width: Float, widths: List<Float>, inset: Float = 0f): List<Float> {
    val n = widths.size
    if (n == 0) return emptyList()
    if (n == 1) return listOf((width - widths[0]) / 2)
    val x = FloatArray(n) { stageLabelPreferred(it, n, width, widths[it], inset) }
    for (i in 1 until n) x[i] = maxOf(x[i], x[i - 1] + widths[i - 1] + LabelGap)
    x[n - 1] = width - widths[n - 1]
    for (i in n - 2 downTo 0) x[i] = minOf(x[i], x[i + 1] - LabelGap - widths[i])
    return x.toList()
}

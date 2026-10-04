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
import androidx.compose.ui.unit.Constraints
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
 * **وحشوةٌ جانبيّةٌ تسع نصفَ أعرضِ ما يقف**: الواقفُ على الطرف يتوسّطه
 * **فيخرج نصفُه** لولاها. **والمسارُ يقصر بمقدارها ولا يفيض أحد.**
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
    // **والأسماءُ لا تتراكب — كلٌّ في خانته** (فحصُ القبول ٢٠٢٦-١٠-٠٣)
    // ══════════════════════════════════════════════════════════════════
    //
    // **رآها المالكُ تدخل بعضُها في بعض**: «بانتظار القبول · مقبول · قيد
    // التحضير…» و«السائق في طريقه لشراء طلبك». **وعلّتُها أنّ عرضَ الاسم
    // كان ثابتاً (٥٦dp) والمسافةَ بين عقدتين أقلُّ منه** — ستُّ عقدٍ على
    // هاتفٍ بعرض ٣٦٠ تترك ٤٨dp لكلٍّ منها.
    //
    // **فعرضُ الخانة يُحسب من العرض المتاح وعدد المراحل** (`stageLabelSlots`)،
    // **وإن لم يسع اسمٌ خانتَه في ثلاثة أسطر نزلت الأسماءُ صفّين متناوبين**
    // فتتّسع خانةُ كلٍّ منها قرابةَ الضعف. **والقياسُ بالخطّ نفسِه الذي يُرسم.**
    val labelStyle = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold)
    val measurer = rememberTextMeasurer()
    val dens = LocalDensity.current

    BoxWithConstraints(modifier.fillMaxWidth().heightIn(min = RowH)) {
        val slots = remember(labels, maxWidth, labelStyle) {
            stageLabelSlots(maxWidth.value, labels.size) { w ->
                val px = with(dens) { w.dp.roundToPx() }.coerceAtLeast(1)
                labels.all { label ->
                    // **وكلمةٌ أعرضُ من خانتها تُكسر في منتصفها** — فلا تُعدّ «تسع».
                    label.split(' ').all { word ->
                        measurer.measure(word, labelStyle, maxLines = 1).size.width <= px
                    } && !measurer.measure(
                        label, labelStyle, maxLines = LabelLines,
                        constraints = Constraints(maxWidth = px),
                    ).hasVisualOverflow
                }
            }
        }
        val labelW = slots.width.dp
        // **والحشوةُ نصفُ الخانة** — فلا يخرج نصفُ الواقف على الطرف.
        val pad = labelW / 2
        // **وطولُ الطريق ما بقي بعد الحشوتين.**
        val track = maxWidth - pad * 2
        // **والصفُّ الثاني تحت الأوّل بارتفاع أطول أسمائه** — يُقاس لا يُخمَّن.
        val firstRowH = if (!slots.staggered) 0.dp else with(dens) {
            labels.filterIndexed { i, _ -> i % 2 == 0 }.maxOf { label ->
                measurer.measure(
                    label, labelStyle, maxLines = LabelLines,
                    constraints = Constraints(maxWidth = labelW.roundToPx().coerceAtLeast(1)),
                ).size.height
            }.toDp()
        }

        // ══════════════════════════════════════════════════════════════
        // **القضيبُ وما امتلأ منه**
        // ══════════════════════════════════════════════════════════════
        //
        // **والملءُ من جهة البداية** — وهي اليمين في العربيّة:
        // `fillMaxWidth` في صندوقٍ محاذاته `Start` يتبع اتّجاه الشاشة،
        // **فلا يُكتب اتّجاهٌ بيد.**
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(y = VehicleH)
                .padding(horizontal = pad)
                .fillMaxWidth()
                .height(RailH)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.inkMuted.copy(alpha = 0.18f)),
        )
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(y = VehicleH)
                .padding(horizontal = pad)
                .fillMaxWidth(pct)
                .height(RailH)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.accent),
        )

        // ══════════════════════════════════════════════════════════════
        // **والعُقَدُ وأسماؤها — كلٌّ على نسبته**
        // ══════════════════════════════════════════════════════════════
        labels.forEachIndexed { i, label ->
            val f = i.toFloat() / last
            val x = pad + track * f
            val done = i <= at

            Box(
                Modifier
                    .align(Alignment.TopStart)
                    .offset(x = x - NodeH / 2, y = VehicleH + (RailH - NodeH) / 2)
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
                style = MaterialTheme.typography.labelSmall,
                textAlign = TextAlign.Center,
                maxLines = LabelLines,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    // **وبحشوةٍ لا بإزاحة** — الحشوةُ تدخل في ارتفاع الشريط
                    // فيتّسع لأطول اسم، **والإزاحةُ لا تُحسب فيُقصّ ما تحتها.**
                    .padding(
                        top = VehicleH + RailH + 6.dp +
                            if (slots.staggered && i % 2 == 1) firstRowH + 2.dp else 0.dp,
                    )
                    .offset(x = x - labelW / 2)
                    .width(labelW),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والدرّاجةُ فوق القضيب**
        // ══════════════════════════════════════════════════════════════
        //
        // **ووجهُها نحو مسيرها**: الطريقُ يمشي من اليمين إلى اليسار
        // كاتّجاه القراءة، **ورسمُ المركبة موجَّهٌ إلى اليمين** فتبدو
        // ماشيةً إلى الخلف. **وقلبُها أفقيّاً يجعلها تسير حيث تنظر.**
        Icon(
            painter = painterResource(R.drawable.ic_moto),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier
                .align(Alignment.TopStart)
                .offset(x = pad + track * pct - VehicleH / 2)
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

/** **أعرضُ خانةٍ لاسمٍ في صفٍّ واحد** — وما كان قبلُ عرضَها الثابت. */
private const val LabelMaxOneRow = 56f
/** **أعرضُ خانةٍ في الصفّين المتناوبين.** */
private const val LabelMaxStaggered = 84f
/** **فراغٌ بين خانتين متجاورتين** — فلا يلتصق اسمٌ باسم. */
internal const val LabelGap = 4f
/** **ثلاثةُ أسطرٍ أقصى ما يأخذه اسم** — وما زاد يُختصر بنقاط. */
private const val LabelLines = 3
private val VehicleH = 24.dp
private val RailH = 6.dp
private val NodeH = 12.dp

/** **أقلُّ ارتفاعٍ للشريط** — درّاجةٌ فقضيبٌ ففراغٌ فسطران، **ويطول بأطول اسم.** */
private val RowH = VehicleH + RailH + 6.dp + 30.dp

/**
 * **خانةُ اسم المرحلة — عرضُها وهل تنزل الأسماءُ صفّين.**
 *
 * @param width عرضُ الشريط كلِّه (dp).
 * @param count عددُ المراحل.
 * @param fits **أيسعُ كلُّ اسمٍ خانةً بهذا العرض؟** — يُقاس بالخطّ في الشاشة.
 *
 * **والحسابُ**: العقدُ على مسافاتٍ متساوية `s = (W - w) / (n - 1)` بعد حشوةٍ
 * نصفُها خانة. **وصفٌّ واحدٌ لا يتراكب إن `w + فراغ ≤ s`**، **وصفّان
 * متناوبان إن `w + فراغ ≤ 2s`** — جارُ الاسم في صفّه على مسافة عقدتين.
 */
internal data class StageLabelSlots(val width: Float, val staggered: Boolean)

internal fun stageLabelSlots(width: Float, count: Int, fits: (Float) -> Boolean): StageLabelSlots {
    if (count <= 1) return StageLabelSlots(minOf(LabelMaxOneRow, width).coerceAtLeast(0f), false)
    val one = minOf(LabelMaxOneRow, (width - LabelGap * (count - 1)) / count).coerceAtLeast(0f)
    if (fits(one)) return StageLabelSlots(one, false)
    val two = minOf(LabelMaxStaggered, (2 * width - LabelGap * (count - 1)) / (count + 1)).coerceAtLeast(0f)
    return StageLabelSlots(two, true)
}

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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
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

    // **وقفزةٌ خفيفةٌ ما دام في الطريق** — لا حركةَ تُلهي، **إنّما
    // إشارةٌ أنّ الأمرَ ما زال يجري.**
    val bob = if (live) {
        val t = rememberInfiniteTransition(label = "ride")
        t.animateFloat(
            initialValue = 0f,
            targetValue = -2f,
            animationSpec = infiniteRepeatable(
                tween(800, easing = FastOutSlowInEasing),
                RepeatMode.Reverse,
            ),
            label = "bob",
        ).value
    } else {
        0f
    }

    BoxWithConstraints(modifier.fillMaxWidth().height(RowH)) {
        // **وطولُ الطريق ما بقي بعد الحشوتين.**
        val track = maxWidth - Pad * 2

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
                .padding(horizontal = Pad)
                .fillMaxWidth()
                .height(RailH)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.inkMuted.copy(alpha = 0.18f)),
        )
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(y = VehicleH)
                .padding(horizontal = Pad)
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
            val x = Pad + track * f
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
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .offset(x = x - LabelW / 2, y = VehicleH + RailH + 6.dp)
                    .width(LabelW),
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
                .offset(
                    x = Pad + track * pct - VehicleH / 2,
                    y = bob.dp,
                )
                .size(VehicleH)
                .scale(scaleX = -1f, scaleY = 1f),
        )
    }
}

/** **نصفُ أعرضِ ما يقف على الطريق** — فلا يخرج نصفُ الواقف على طرفه. */
private val Pad = 28.dp
private val LabelW = 56.dp
private val VehicleH = 24.dp
private val RailH = 6.dp
private val NodeH = 12.dp

/** **ارتفاعُ الشريط كلِّه** — درّاجةٌ فقضيبٌ ففراغٌ فسطران. */
private val RowH = VehicleH + RailH + 6.dp + 30.dp

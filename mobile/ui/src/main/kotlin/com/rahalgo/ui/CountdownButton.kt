package com.rahalgo.ui

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.layout
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **زرُّ القبول والعدّادُ فيه — لا بجانبه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «العدّادُ يكون داخل زرّ موافق بطريقةٍ
 *  احترافيّة، ليكون واضحاً أنّه بمجرّد انتهاء الزمن سيختفي الطلب».)
 *
 * # ولماذا فيه لا فوقه
 *
 * **العدّادُ في زاوية البطاقة رقمٌ يُقرأ ثمّ يُنسى** — والزرُّ في مكانٍ
 * آخر، **فلا يربط الذهنُ بينهما.** ومن قرأ «٠:٠٨» لا يعلم أنّها مهلةُ
 * هذا الزرّ بعينه.
 *
 * **وفيه يصير الزمنُ صفةَ الفعل**: «موافق — وبقي ثمانٍ». **والأرضُ
 * تنحسر تحت إصبعه** فيرى المهلةَ تنفد لا يقرؤها.
 *
 * # وما يقوله الانحسار
 *
 * **الأرضُ الممتلئةُ تنقص من جهة النهاية** — وهي جهةُ اليسار في
 * العربيّة. **فيُقرأ أنّ شيئاً يفرغ** لا أنّ شيئاً يمتلئ: **شريطُ تقدّمٍ
 * يمتلئ يقول «انتظر»، وهذا يقول «أسرع».**
 *
 * # ونسبةُ الامتلاء من أطول ما رُئي
 *
 * **المحرّكُ يرسل لحظةَ الانتهاء لا طولَ المهلة** — فلا يُعرف كم كانت.
 * **فيُحفظ أطولُ ما رُئي** ويُقاس عليه.
 *
 * **ومن فتح التطبيقَ ومهلتُه في نصفها** يرى الشريطَ ممتلئاً ثمّ ينحسر
 * أسرع. **ولحظةُ النفاد صادقةٌ على كلّ حال** — وهي التي تعني شيئا.
 */
@Composable
fun CountdownButton(
    expiresAt: String?,
    onExpired: () -> Unit,
    onClick: () -> Unit,
    label: String,
    icon: Int,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    busy: Boolean = false,
) {
    val end = remember(expiresAt) {
        expiresAt?.let { runCatching { java.time.Instant.parse(it).toEpochMilli() }.getOrNull() }
    }

    var left by remember(expiresAt) {
        mutableStateOf(end?.minus(System.currentTimeMillis()) ?: 0L)
    }
    // **وأطولُ ما رُئي مقياسُ الامتلاء** — ولا يقلّ عن ثانيةٍ لئلّا
    // تُقسَم على صفر.
    var span by remember(expiresAt) { mutableStateOf(maxOf(left, 1L)) }

    LaunchedEffect(expiresAt) {
        if (end == null) return@LaunchedEffect
        while (left > 0) {
            kotlinx.coroutines.delay(250)
            left = end - System.currentTimeMillis()
            if (left > span) span = left
        }
        onExpired()
    }

    val seconds = (left / 1000).toInt()
    val running = end != null && left > 0
    // **وعشرُ ثوانٍ فأقلّ تصير برتقاليّة** — ومن يقود يقرأ اللونَ قبل
    // الرقم، **والمهلةُ التي تنقضي بلا إنذارٍ تُقرأ عطبا.**
    val urgent = running && seconds <= 10
    val fill = if (urgent) Rahal.colors.accent else Rahal.colors.success
    val fraction by animateFloatAsState(
        targetValue = if (running) (left.toFloat() / span).coerceIn(0f, 1f) else 1f,
        animationSpec = tween(250),
        label = "offer-countdown",
    )

    val shape = Rahal.shape.lg
    Box(
        modifier
            .height(48.dp)
            .clip(shape)
            // **والأرضُ الباهتةُ هي المسار** — وما انحسر عنه يبقى مرئيّاً
            // فيُعرف كم ذهب.
            .background(fill.copy(alpha = 0.28f))
            .clickable(enabled = enabled && !busy, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (running) {
            Box(
                Modifier
                    .fillMaxHeight()
                    // **وينحسر من جهة النهاية** — يسارُ الشاشة في
                    // العربيّة، **وهو ما يفعله `layout` بترك الفائض
                    // خارج الحدّ البادئ.**
                    .layout { measurable, constraints ->
                        val w = (constraints.maxWidth * fraction).toInt()
                        val placeable = measurable.measure(
                            constraints.copy(minWidth = w, maxWidth = w),
                        )
                        layout(w, placeable.height) { placeable.place(0, 0) }
                    }
                    .background(fill),
            )
        } else {
            Box(Modifier.fillMaxWidth().fillMaxHeight().background(fill))
        }

        Row(
            Modifier.padding(horizontal = 14.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (busy) {
                CircularProgressIndicator(
                    Modifier.size(18.dp),
                    strokeWidth = 2.dp,
                    color = Color.White,
                )
                return@Row
            }
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = Color.White,
                modifier = Modifier.size(18.dp),
            )
            Spacer(Modifier.size(6.dp))
            Text(
                text = label,
                color = Color.White,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            if (running) {
                Spacer(Modifier.size(8.dp))
                // **والرقمُ في حبّةٍ داكنةٍ داخل الزرّ** — ليُقرأ فوق
                // الأرض الممتلئة والفارغة معا: **نصٌّ أبيضُ يمرّ على
                // الحدّ بينهما فيختفي نصفُه.**
                Box(
                    Modifier
                        .clip(Rahal.shape.sm)
                        .background(Color.Black.copy(alpha = 0.22f))
                        .padding(horizontal = 8.dp, vertical = 2.dp),
                ) {
                    Text(
                        text = "%d:%02d".format(seconds / 60, seconds % 60),
                        color = Color.White,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
        }
    }
}

/**
 * **نصُّ «انقضت المهلة»** — يُعرض مكانَ الزرّ لا فيه.
 *
 * **وزرٌّ يُضغط بعد انقضائها يردّ «ليس عرضك»** — فيُقرأ عطباً في
 * التطبيق لا انقضاءً في الوقت.
 */
@Composable
fun OfferExpired(modifier: Modifier = Modifier) {
    Box(modifier.height(48.dp), contentAlignment = Alignment.Center) {
        Text(
            text = stringResource(R.string.offer_expired),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

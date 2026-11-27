package com.rahalgo.ui

import android.content.Context
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.CompositingStrategy
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawOutline
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.platform.LocalLayoutDirection
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.DailyCard
import com.rahalgo.shared.model.StampCard
import kotlinx.coroutines.delay

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بطاقةُ عضويةٍ فاخرة** (اختيارُ المالك ٢٠٢٦-١١-٢٧، النموذج ١): الأولى كحليّةٌ عميقةٌ بنقشةٍ خفيفةٍ وشعارٍ كبيرٍ باهتٍ
 * في الخلفيّة، دوائرُها في صفّين (١٠ ⇒ ٥×٢) **والمختومةُ بشعار رحّال غو** والأخيرةُ هديّةٌ ذهبيّة. وتحتها **كرتُ اليوم**
 * كتذكرةٍ برتقاليّةٍ بخطّ قصٍّ منقَّط ونتوءين جانبيّين. ألوانُهما ثابتةٌ بالوضعين (الليليّ والنهاريّ) لأنّهما بلونهما.
 * **والختمُ الجديدُ منذ آخر نظرةٍ يُختم أمامه** (العددُ المرئيُّ محفوظٌ بالجهاز بمفتاح `seenKey`).
 * **والمطفأةُ لا تُرسم.** وتطبيقُ الزبون وتطبيقُ المتجر يرسمانها بهذه وحدَها، والويبُ يقول الكلامَ نفسَه (`loyalty.*`).
 */

// ألوانُ البطاقتين من النموذج المعتمد — ثابتةٌ بالوضعين.
private val StampBlueLight = Color(0xFF0A8FC4)
private val StampBlueMid = Color(0xFF02678F)
private val StampBlueDeep = Color(0xFF033A52)
private val LoyaltyGold = Color(0xFFF4C766)
private val DailyLight = Color(0xFFFFD38A)
private val DailyMid = Color(0xFFFFB547)
private val DailyDeep = Color(0xFFFE9501)
private val DailyInk = Color(0xFF3B2200)


@Composable
fun LoyaltyCards(card: StampCard, seenKey: String, modifier: Modifier = Modifier) {
    if (!card.enabled && !card.daily.enabled) return
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
        if (card.enabled && card.size > 0) StampCardBox(card, "$seenKey.stamp")
        if (card.daily.enabled && card.daily.size > 0) DailyCardBox(card.daily, "$seenKey.daily")
    }
}

/** **كم بقي** — بالعدد العربيّ: واحدة · اثنتان · ٣–١٠ جمع · ١١+ مفرد. */
@Composable
private fun leftText(n: Int, one: Int, two: Int, few: Int, many: Int): String = when {
    n <= 1 -> stringResource(one)
    n == 2 -> stringResource(two)
    n <= 10 -> stringResource(few, n)
    else -> stringResource(many, n)
}

/** **كم ختماً رآه آخرَ مرّة** — ويُحفظ ما يراه الآن. `-1` أوّلَ مرّة (فلا يُختم شيءٌ أمامه). */
@Composable
private fun rememberSeen(key: String, stamps: Int): Int {
    val ctx = LocalContext.current
    val prefs = remember { ctx.getSharedPreferences("loyalty_cards", Context.MODE_PRIVATE) }
    val seen = remember(key, stamps) { prefs.getInt(key, -1) }
    LaunchedEffect(key, stamps) { prefs.edit().putInt(key, stamps).apply() }
    return seen
}

/** **«٧ / ١٠»** بأرقامٍ غربيّة وباتجاهٍ ثابتٍ من اليسار كي لا ينقلب ترتيبُها في الصفحة العربيّة. */
private fun chipText(n: Int, size: Int): String = "⁦%d / %d⁩".format(java.util.Locale.US, n, size)

/** **ساعةُ الإغلاق** («١١:٠٠ م») بتوقيت دمشق، أو `null` إن لم يصل وقتٌ صالح. */
@Composable
private fun closesText(iso: String): String? {
    val t = remember(iso) {
        runCatching {
            val z = java.time.OffsetDateTime.parse(iso).atZoneSameInstant(java.time.ZoneId.of("Asia/Damascus"))
            if (z.year < 2000) null else z.toLocalTime()
        }.getOrNull()
    } ?: return null
    val h12 = (t.hour % 12).let { if (it == 0) 12 else it }
    val hm = "%d:%02d".format(java.util.Locale.US, h12, t.minute)
    return stringResource(if (t.hour < 12) R.string.loyalty_time_am else R.string.loyalty_time_pm, hm)
}

// ───────────────────────── البطاقة الكحليّة ─────────────────────────

@Composable
private fun StampCardBox(card: StampCard, seenKey: String) {
    val seen = rememberSeen(seenKey, card.stamps)
    val full = card.freeReady
    Box(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .drawBehind {
                drawRect(
                    Brush.radialGradient(
                        colorStops = arrayOf(0f to StampBlueLight, 0.38f to StampBlueMid, 1f to StampBlueDeep),
                        center = Offset(size.width, 0f),
                        radius = maxOf(size.width, size.height) * 1.15f,
                    ),
                )
                // نقشةُ خطوطٍ مائلةٍ خفيفة
                val step = 20.dp.toPx()
                val stripe = Color.White.copy(alpha = 0.045f)
                var x = 0f
                while (x < size.width + size.height) {
                    drawLine(stripe, Offset(x, 0f), Offset(x - size.height, size.height), strokeWidth = 2.dp.toPx())
                    x += step
                }
            }
            .then(if (full) Modifier.border(BorderStroke(2.dp, Rahal.colors.accent), Rahal.shape.lg) else Modifier),
    ) {
        // **الشعارُ الكبير الباهت** أسفلَ اليسار الماديّ — بغضّ النظر عن اتجاه الصفحة.
        CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
            Box(Modifier.matchParentSize()) {
                Image(
                    painterResource(R.drawable.rahal_mark_color),
                    contentDescription = null,
                    colorFilter = ColorFilter.tint(Color.White),
                    modifier = Modifier
                        .align(Alignment.BottomStart)
                        .offset(x = (-30).dp, y = 34.dp)
                        .size(170.dp)
                        .alpha(0.10f),
                )
            }
        }
        Column(Modifier.padding(start = Rahal.space.lg, end = Rahal.space.lg, top = 18.dp, bottom = Rahal.space.lg)) {
            Row(verticalAlignment = Alignment.Top) {
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.loyalty_stamp_title),
                        color = Color.White,
                        style = Rahal.type.sectionTitle.copy(fontWeight = FontWeight.ExtraBold),
                    )
                    Text(
                        stringResource(R.string.loyalty_stamp_sub),
                        color = Color.White.copy(alpha = 0.8f),
                        style = Rahal.type.meta,
                    )
                }
                Spacer(Modifier.padding(horizontal = Rahal.space.xs))
                Text(
                    chipText(card.stamps, card.size),
                    color = Color.White,
                    style = Rahal.type.badge,
                    modifier = Modifier
                        .clip(Rahal.shape.pill)
                        .background(Color.White.copy(alpha = 0.16f))
                        .border(BorderStroke(1.dp, Color.White.copy(alpha = 0.25f)), Rahal.shape.pill)
                        .padding(horizontal = 10.dp, vertical = 3.dp),
                )
            }
            Spacer(Modifier.height(Rahal.space.lg))
            SlotGrid(
                n = card.size, stamps = card.stamps, seen = seen, perRow = if (card.size <= 10) (card.size + 1) / 2 else 5,
                style = StampSlotStyle, giftIndex = card.size - 1, spacing = 10.dp,
            )
            Spacer(Modifier.height(Rahal.space.md))
            val frac = if (card.size > 0) (card.stamps.toFloat() / card.size).coerceIn(0f, 1f) else 0f
            Box(
                Modifier.fillMaxWidth().height(8.dp).clip(CircleShape).background(Color.White.copy(alpha = 0.18f)),
            ) {
                Box(
                    Modifier
                        .fillMaxWidth(frac)
                        .height(8.dp)
                        .clip(CircleShape)
                        .background(Brush.horizontalGradient(listOf(LoyaltyGold, Color.White))),
                )
            }
            Spacer(Modifier.height(Rahal.space.sm))
            when {
                full -> Highlight(stringResource(R.string.loyalty_stamp_full), Rahal.colors.accent, Rahal.colors.onLight)
                card.rewardPending -> Text(
                    stringResource(R.string.loyalty_stamp_pending),
                    color = Color.White, style = Rahal.type.bodyStrong,
                )
                else -> Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        leftText(
                            card.size - card.stamps, R.string.loyalty_foot_left_1, R.string.loyalty_foot_left_2,
                            R.string.loyalty_foot_left_few, R.string.loyalty_foot_left_many,
                        ),
                        color = Color.White, style = Rahal.type.caption,
                    )
                    Text(
                        stringResource(R.string.loyalty_foot_next_free),
                        color = LoyaltyGold, style = Rahal.type.caption.copy(fontWeight = FontWeight.Bold),
                    )
                }
            }
            if (card.freeCap > 0) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    stringResource(R.string.loyalty_cap, money(card.freeCap)),
                    color = Color.White.copy(alpha = 0.85f), style = Rahal.type.meta,
                )
            }
        }
    }
}

// ───────────────────────── كرت اليوم (تذكرة) ─────────────────────────

@Composable
private fun DailyCardBox(card: DailyCard, seenKey: String) {
    val seen = rememberSeen(seenKey, card.stamps)
    val full = card.freeReady && card.open
    val until = if (card.open) closesText(card.closesAt) else null
    Column(
        Modifier
            .fillMaxWidth()
            // طبقةٌ منفصلة كي يقصَّ النتوءان (BlendMode.Clear) من البطاقة فيظهر ما خلفها
            .graphicsLayer { compositingStrategy = CompositingStrategy.Offscreen }
            .clip(Rahal.shape.lg)
            .drawBehind {
                drawRect(
                    Brush.radialGradient(
                        colorStops = arrayOf(0f to DailyLight, 0.35f to DailyMid, 1f to DailyDeep),
                        center = Offset(0f, 0f),
                        radius = maxOf(size.width, size.height) * 1.15f,
                    ),
                )
            }
            .then(if (full) Modifier.border(BorderStroke(2.dp, Rahal.colors.success), Rahal.shape.lg) else Modifier),
    ) {
        Column(Modifier.padding(start = Rahal.space.lg, end = Rahal.space.lg, top = Rahal.space.lg, bottom = Rahal.space.xs)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.loyalty_daily_title),
                    color = DailyInk,
                    style = Rahal.type.sectionTitle.copy(fontWeight = FontWeight.ExtraBold),
                    modifier = Modifier.weight(1f),
                )
                if (card.open) {
                    Text(
                        if (until != null) stringResource(R.string.loyalty_daily_until, until) else stringResource(R.string.loyalty_daily_renews),
                        color = DailyInk,
                        style = Rahal.type.badge,
                        modifier = Modifier
                            .clip(Rahal.shape.pill)
                            .background(DailyInk.copy(alpha = 0.12f))
                            .padding(horizontal = 10.dp, vertical = 3.dp),
                    )
                }
            }
            Text(
                stringResource(R.string.loyalty_daily_sub, card.size),
                color = DailyInk.copy(alpha = 0.8f), style = Rahal.type.meta,
            )
        }
        // خطُّ القصّ المنقَّط والنتوءان
        Box(
            Modifier.fillMaxWidth().height(24.dp).drawBehind {
                val y = size.height / 2
                val r = 11.dp.toPx()
                drawLine(
                    DailyInk.copy(alpha = 0.18f), Offset(r, y), Offset(size.width - r, y),
                    strokeWidth = 2.dp.toPx(),
                    pathEffect = PathEffect.dashPathEffect(floatArrayOf(8.dp.toPx(), 6.dp.toPx())),
                )
                drawCircle(Color.Black, r, Offset(0f, y), blendMode = BlendMode.Clear)
                drawCircle(Color.Black, r, Offset(size.width, y), blendMode = BlendMode.Clear)
            },
        )
        Column(Modifier.padding(start = Rahal.space.lg, end = Rahal.space.lg, bottom = Rahal.space.lg)) {
            SlotGrid(
                n = card.size + 1, stamps = card.stamps, seen = seen, perRow = card.size + 1,
                style = dailySlotStyle(), giftIndex = card.size, spacing = 8.dp,
            )
            Spacer(Modifier.height(Rahal.space.md))
            val msg = Rahal.type.caption.copy(fontWeight = FontWeight.Bold)
            when {
                !card.open -> Text(stringResource(R.string.loyalty_daily_closed), color = DailyInk, style = msg)
                full -> Highlight(stringResource(R.string.loyalty_daily_full), Rahal.colors.success, Rahal.colors.onPanel)
                card.rewardPending -> Text(stringResource(R.string.loyalty_daily_pending), color = DailyInk, style = msg)
                else -> Text(
                    leftText(
                        card.size - card.stamps, R.string.loyalty_daily_left_1, R.string.loyalty_daily_left_2,
                        R.string.loyalty_daily_left_few, R.string.loyalty_daily_left_many,
                    ),
                    color = DailyInk, style = msg,
                )
            }
            if (card.freeCap > 0) {
                Spacer(Modifier.height(Rahal.space.xxs))
                Text(stringResource(R.string.loyalty_cap, money(card.freeCap)), color = DailyInk.copy(alpha = 0.75f), style = Rahal.type.meta)
            }
        }
    }
}

/** **سطرٌ بارزٌ بأرضٍ ملوّنة** — حالُ الامتلاء. */
@Composable
private fun Highlight(text: String, bg: Color, ink: Color) {
    Text(
        text,
        color = ink,
        style = Rahal.type.bodyStrong,
        fontWeight = FontWeight.Bold,
        modifier = Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(bg)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
    )
}

// ───────────────────────── الخانات ─────────────────────────

/** **هيئةُ الخانة** — دائرةٌ للبطاقة الكحليّة ومربّعٌ مدوَّر لكرت اليوم. */
private class SlotStyle(
    val shape: Shape,
    val dash: Color,
    val numberInk: Color,
    val tilt: Float,
    val giftRing: Color,
    val giftTint: Color,
    val maxSize: Dp,
)

private val StampSlotStyle = SlotStyle(
    shape = CircleShape, dash = Color.White.copy(alpha = 0.38f), numberInk = Color.White.copy(alpha = 0.55f),
    tilt = -14f, giftRing = LoyaltyGold, giftTint = LoyaltyGold.copy(alpha = 0.35f), maxSize = 52.dp,
)
@Composable
private fun dailySlotStyle() = SlotStyle(
    shape = Rahal.shape.md, dash = DailyInk.copy(alpha = 0.3f), numberInk = DailyInk.copy(alpha = 0.45f),
    tilt = -12f, giftRing = Color.White, giftTint = Color.White.copy(alpha = 0.35f), maxSize = 48.dp,
)

/** **حدٌّ منقَّط** على شكل الخانة. */
private fun Modifier.dashedOutline(shape: Shape, color: Color): Modifier = drawBehind {
    val sw = 2.dp.toPx()
    val outline = shape.createOutline(Size(size.width - sw, size.height - sw), layoutDirection, this)
    translate(sw / 2, sw / 2) {
        drawOutline(outline, color, style = Stroke(sw, pathEffect = PathEffect.dashPathEffect(floatArrayOf(sw * 2.5f, sw * 2f))))
    }
}

/**
 * **الخاناتُ في صفوف** — الأولى المختومةُ بشعار رحّال غو، **والجديدةُ منذ آخر نظرةٍ تُختم أمامه** بنبضةٍ خفيفة،
 * والفارغةُ منقَّطةٌ برقمها، **وخانةُ `giftIndex` هديّةٌ** (🎁).
 */
@Composable
private fun SlotGrid(n: Int, stamps: Int, seen: Int, perRow: Int, style: SlotStyle, giftIndex: Int, spacing: Dp) {
    val stampedCd = stringResource(R.string.loyalty_stamped)
    val emptyCd = stringResource(R.string.loyalty_empty)
    Column(verticalArrangement = Arrangement.spacedBy(spacing)) {
        for (start in 0 until n step perRow) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(spacing)) {
                for (i in start until start + perRow) {
                    Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                        if (i < n) {
                            val gift = i == giftIndex && i >= stamps // الخانةُ العاشرةُ تُختم أيضاً؛ هديّتُها ما دامت فارغة
                            val stamped = !gift && i < stamps
                            val fresh = stamped && seen in 0..i
                            LoyaltySlot(
                                index = i, stamped = stamped, gift = gift, fresh = fresh,
                                delayIndex = i - maxOf(seen, 0), style = style,
                                cd = if (stamped) stampedCd else emptyCd,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun LoyaltySlot(
    index: Int,
    stamped: Boolean,
    gift: Boolean,
    fresh: Boolean,
    delayIndex: Int,
    style: SlotStyle,
    cd: String,
) {
    val anim = remember(stamped, fresh) { Animatable(if (fresh) 0f else 1f) }
    LaunchedEffect(stamped, fresh) {
        if (fresh) {
            delay(250L + 140L * delayIndex)
            anim.animateTo(1f, spring(dampingRatio = Spring.DampingRatioMediumBouncy, stiffness = Spring.StiffnessLow))
        }
    }
    val base = Modifier
        .widthIn(max = style.maxSize)
        .fillMaxWidth()
        .aspectRatio(1f)
        .semantics { contentDescription = cd }
    when {
        gift -> Box(
            base
                .clip(style.shape)
                .background(Brush.radialGradient(listOf(style.giftTint, Color.Transparent)))
                .border(BorderStroke(2.dp, style.giftRing), style.shape),
            contentAlignment = Alignment.Center,
        ) {
            Text("🎁", style = Rahal.type.sectionTitle)
        }
        !stamped -> Box(base.dashedOutline(style.shape, style.dash), contentAlignment = Alignment.Center) {
            Text(
                "${index + 1}", color = style.numberInk,
                style = Rahal.type.caption.copy(fontWeight = FontWeight.ExtraBold),
            )
        }
        else -> Box(
            base
                .scale(1.6f - 0.6f * anim.value)
                .alpha(anim.value.coerceIn(0f, 1f))
                .shadow(3.dp, style.shape)
                .clip(style.shape)
                .background(Color.White),
            contentAlignment = Alignment.Center,
        ) {
            Image(
                painterResource(R.drawable.rahal_mark_color),
                contentDescription = null,
                modifier = Modifier.fillMaxSize(0.7f).rotate(style.tilt),
            )
        }
    }
}

/** **سطرُ «مجاناً» ومصدرُه** — للأجرة حين تُعفى بجائزة بطاقة أو بالعرض. */
@Composable
fun freeSourceText(source: String): String = when (source) {
    "stamp" -> stringResource(R.string.loyalty_free_stamp)
    "daily" -> stringResource(R.string.loyalty_free_daily)
    else -> stringResource(R.string.loyalty_free_campaign)
}

/**
 * **صفحةُ «بطاقاتي»** (قرارُ المالك ٢٠٢٦-١١-٢٧) — البطاقتان كبيرتين بدل أن تُحشرا في الرئيسيّة؛ تُفتح من أيقونة
 * الشريط العلويّ بجانب المحفظة. **والمطفأةُ لا تُرسم** (في `LoyaltyCards`)، وسهمُ الرجوع كالمحفظة.
 */
@Composable
fun LoyaltyCardsScreen(card: StampCard, seenKey: String, onBack: () -> Unit) {
    androidx.activity.compose.BackHandler(onBack = onBack)
    Screen {
        PageHeader(stringResource(R.string.loyalty_cards_title), onBack)
        Spacer(Modifier.height(Rahal.space.md))
        LoyaltyCards(card, seenKey)
    }
}

/** **تقدّمُ الختم لشارة الأيقونة** («٧/١٠» بأرقام غربيّة كالرصيد) — وفارغٌ إن لم تُفعَّل بطاقةٌ بعدد. */
fun StampCard.cardsBadge(): String = when {
    enabled && size > 0 -> "%d/%d".format(java.util.Locale.US, stamps, size)
    daily.enabled && daily.size > 0 -> "%d/%d".format(java.util.Locale.US, daily.stamps, daily.size)
    else -> ""
}

/** **أيُّ بطاقةٍ مفعّلة** — شرطُ ظهور الأيقونة. */
val StampCard.anyCardEnabled: Boolean get() = enabled || daily.enabled

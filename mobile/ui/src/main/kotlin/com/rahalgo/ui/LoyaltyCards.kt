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
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.DailyCard
import com.rahalgo.shared.model.StampCard
import kotlinx.coroutines.delay

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * بطاقةٌ بلون العلامة بدوائرَ في صفّين (١٠ ⇒ ٥×٢)، **والمختومةُ بشعار رحّال غو**. وتحتها **كرتُ
 * اليوم** بدوائره. **والختمُ الجديدُ منذ آخر نظرةٍ يُختم أمامه** (العددُ المرئيُّ محفوظٌ بالجهاز
 * بمفتاح `seenKey`). **والمطفأةُ لا تُرسم.** وتطبيقُ الزبون وتطبيقُ المتجر يرسمانها بهذه وحدَها،
 * والويبُ يقول الكلامَ نفسَه (`loyalty.*`).
 */
@Composable
fun LoyaltyCards(card: StampCard, seenKey: String, modifier: Modifier = Modifier) {
    if (!card.enabled && !card.daily.enabled) return
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
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

@Composable
private fun StampCardBox(card: StampCard, seenKey: String) {
    val seen = rememberSeen(seenKey, card.stamps)
    val brand = Rahal.colors.brand
    val full = card.freeReady
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Brush.linearGradient(listOf(brand, brand.copy(alpha = 0.82f))))
            .then(if (full) Modifier.border(BorderStroke(2.dp, Rahal.colors.accent), Rahal.shape.lg) else Modifier)
            .padding(Rahal.space.lg),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Image(
                painterResource(R.drawable.rahal_mark_color),
                contentDescription = null,
                modifier = Modifier.height(22.dp).clip(CircleShape).background(Rahal.colors.onBrand).padding(2.dp),
            )
            Spacer(Modifier.padding(horizontal = Rahal.space.xs))
            Column(Modifier.weight(1f)) {
                Text(stringResource(R.string.loyalty_stamp_title), color = Rahal.colors.onBrand, style = Rahal.type.sectionTitle)
                Text(
                    stringResource(R.string.loyalty_stamp_sub),
                    color = Rahal.colors.onBrand.copy(alpha = 0.85f),
                    style = Rahal.type.meta,
                )
            }
            Text(
                stringResource(R.string.loyalty_count, card.stamps, card.size),
                color = Rahal.colors.onBrand,
                style = Rahal.type.badge,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.onBrand.copy(alpha = 0.18f))
                    .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xxs),
            )
        }
        Spacer(Modifier.height(Rahal.space.md))
        StampGrid(
            size = card.size, stamps = card.stamps, seen = seen,
            emptyRing = Rahal.colors.onBrand.copy(alpha = 0.55f),
            emptyFill = Rahal.colors.onBrand.copy(alpha = 0.08f),
            stampFill = Rahal.colors.onBrand,
            numberInk = Rahal.colors.onBrand.copy(alpha = 0.6f),
        )
        Spacer(Modifier.height(Rahal.space.md))
        when {
            full -> Highlight(stringResource(R.string.loyalty_stamp_full), Rahal.colors.accent, Rahal.colors.onLight)
            card.rewardPending -> Text(
                stringResource(R.string.loyalty_stamp_pending),
                color = Rahal.colors.onBrand, style = Rahal.type.bodyStrong,
            )
            else -> Text(
                leftText(
                    card.size - card.stamps, R.string.loyalty_stamp_left_1, R.string.loyalty_stamp_left_2,
                    R.string.loyalty_stamp_left_few, R.string.loyalty_stamp_left_many,
                ),
                color = Rahal.colors.onBrand, style = Rahal.type.bodyStrong,
            )
        }
        if (card.freeCap > 0) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(R.string.loyalty_cap, money(card.freeCap)),
                color = Rahal.colors.onBrand.copy(alpha = 0.85f), style = Rahal.type.meta,
            )
        }
    }
}

@Composable
private fun DailyCardBox(card: DailyCard, seenKey: String) {
    val seen = rememberSeen(seenKey, card.stamps)
    val full = card.freeReady && card.open
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.surface)
            .border(BorderStroke(if (full) 2.dp else 1.dp, if (full) Rahal.colors.success else Rahal.colors.line), Rahal.shape.lg)
            .padding(Rahal.space.lg),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.height(10.dp).aspectRatio(1f).clip(CircleShape).background(Rahal.colors.accent),
            )
            Spacer(Modifier.padding(horizontal = Rahal.space.xs))
            Text(
                stringResource(R.string.loyalty_daily_title),
                color = Rahal.colors.ink, style = Rahal.type.cardTitle, modifier = Modifier.weight(1f),
            )
            Text(
                stringResource(R.string.loyalty_count, card.stamps, card.size),
                color = Rahal.colors.accentInk,
                style = Rahal.type.badge,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.warnTint)
                    .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xxs),
            )
        }
        Spacer(Modifier.height(Rahal.space.md))
        StampGrid(
            size = card.size, stamps = card.stamps, seen = seen,
            emptyRing = Rahal.colors.line,
            emptyFill = Rahal.colors.field,
            stampFill = Rahal.colors.surface,
            numberInk = Rahal.colors.inkMuted,
            stampRing = Rahal.colors.accent,
        )
        Spacer(Modifier.height(Rahal.space.md))
        when {
            !card.open -> Text(stringResource(R.string.loyalty_daily_closed), color = Rahal.colors.inkMuted, style = Rahal.type.body)
            full -> Highlight(stringResource(R.string.loyalty_daily_full), Rahal.colors.success, Rahal.colors.onPanel)
            card.rewardPending -> Text(stringResource(R.string.loyalty_daily_pending), color = Rahal.colors.ink, style = Rahal.type.bodyStrong)
            else -> Text(
                leftText(
                    card.size - card.stamps, R.string.loyalty_daily_left_1, R.string.loyalty_daily_left_2,
                    R.string.loyalty_daily_left_few, R.string.loyalty_daily_left_many,
                ),
                color = Rahal.colors.ink, style = Rahal.type.bodyStrong,
            )
        }
        if (card.open) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(stringResource(R.string.loyalty_daily_renews), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
        }
        if (card.freeCap > 0) {
            Text(stringResource(R.string.loyalty_cap, money(card.freeCap)), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
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

/**
 * **الدوائرُ في صفوف** — حتّى ١٠ صفّان متساويان (١٠ ⇒ ٥×٢)، وما فوقها صفوفٌ من خمس.
 * والمختومةُ بشعار رحّال غو، **والجديدةُ منذ آخر نظرةٍ تُختم أمامه** بنبضةٍ خفيفة.
 */
@Composable
private fun StampGrid(
    size: Int,
    stamps: Int,
    seen: Int,
    emptyRing: Color,
    emptyFill: Color,
    stampFill: Color,
    numberInk: Color,
    stampRing: Color? = null,
) {
    val perRow = if (size <= 10) (size + 1) / 2 else 5
    val stampedCd = stringResource(R.string.loyalty_stamped)
    val emptyCd = stringResource(R.string.loyalty_empty)
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        for (start in 0 until size step perRow) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                for (i in start until start + perRow) {
                    Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                        if (i < size) {
                            val stamped = i < stamps
                            val fresh = stamped && seen in 0..i
                            StampCircle(
                                index = i, stamped = stamped, fresh = fresh, delayIndex = i - maxOf(seen, 0),
                                emptyRing = emptyRing, emptyFill = emptyFill, stampFill = stampFill,
                                numberInk = numberInk, stampRing = stampRing,
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
private fun StampCircle(
    index: Int,
    stamped: Boolean,
    fresh: Boolean,
    delayIndex: Int,
    emptyRing: Color,
    emptyFill: Color,
    stampFill: Color,
    numberInk: Color,
    stampRing: Color?,
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
        .widthIn(max = 52.dp)
        .fillMaxWidth()
        .aspectRatio(1f)
        .semantics { contentDescription = cd }
    if (!stamped) {
        Box(
            base.clip(CircleShape).background(emptyFill).border(BorderStroke(1.5.dp, emptyRing), CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Text("${index + 1}", color = numberInk, style = Rahal.type.meta)
        }
        return
    }
    Box(
        base
            .scale(1.6f - 0.6f * anim.value)
            .alpha(anim.value.coerceIn(0f, 1f))
            .clip(CircleShape)
            .background(stampFill)
            .then(if (stampRing != null) Modifier.border(BorderStroke(2.dp, stampRing), CircleShape) else Modifier),
        contentAlignment = Alignment.Center,
    ) {
        Image(
            painterResource(R.drawable.rahal_mark_color),
            contentDescription = null,
            modifier = Modifier.fillMaxSize().padding(Rahal.space.xs),
        )
    }
}

/** **سطرُ «مجاناً» ومصدرُه** — للأجرة حين تُعفى بجائزة بطاقة أو بالعرض. */
@Composable
fun freeSourceText(source: String): String = when (source) {
    "stamp" -> stringResource(R.string.loyalty_free_stamp)
    "daily" -> stringResource(R.string.loyalty_free_daily)
    else -> stringResource(R.string.loyalty_free_campaign)
}

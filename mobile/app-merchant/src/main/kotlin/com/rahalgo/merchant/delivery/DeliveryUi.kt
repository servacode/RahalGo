package com.rahalgo.merchant.delivery

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.ui.ConfirmDialog

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قطعُ تصميم «توصيلاتي»** — طلبُ المالك ٢٠٢٦-١٠-٠١
 * ══════════════════════════════════════════════════════════════════════
 *
 * (نصُّه: «صمّم توصيلاتي بتصميمٍ احترافيٍّ واضح، وخصوصاً زرّ إلغاء التوصيلة،
 *  و«بانتظار سائق» خلّيها تومض ليشعر المتجر أنّه يتمّ البحث عن سائق».)
 *
 * - **شارةُ الحال بلونها**: برتقاليٌّ للانتظار · أزرقُ للطريق · أخضرُ للتسليم ·
 *   أحمرُ للإلغاء — **فيُقرأ الحالُ من بعيد قبل الكلمة.**
 * - **«جاري البحث عن سائق…» بدوائرَ تنبض حول الموتور** ما دامت `dispatching`.
 * - **زرُّ الإلغاء أحمرُ بعرض البطاقة، ويسأل قبل أن يُلغي** — كان رابطاً باهتاً
 *   يُلغي من أوّل لمسة.
 */

/** **ما يُلغى من المتجر** — ما دام الغرضُ عنده (الخادمُ يحكم أيضاً). */
internal val cancellable = setOf("dispatching", "assigned", "at_pickup")

internal enum class Tone { Waiting, Moving, Done, Ended }

internal fun toneOf(status: String): Tone = when (status) {
    "dispatching" -> Tone.Waiting
    "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff" -> Tone.Moving
    "delivered" -> Tone.Done
    else -> Tone.Ended
}

@Composable
internal fun toneColor(t: Tone): Color = when (t) {
    Tone.Waiting -> Rahal.colors.accent
    Tone.Moving -> Rahal.colors.brand
    Tone.Done -> Rahal.colors.success
    Tone.Ended -> Rahal.colors.danger
}

@Composable
internal fun deliveryStatusText(status: String): String = stringResource(
    when (status) {
        "dispatching" -> R.string.md_searching
        "assigned" -> R.string.os_assigned
        "at_pickup" -> R.string.os_at_pickup
        "picked_up" -> R.string.os_picked_up
        "on_the_way" -> R.string.os_on_way
        "at_dropoff" -> R.string.os_at_dropoff
        "delivered" -> com.rahalgo.ui.R.string.ord_st_delivered
        "cancelled" -> R.string.os_cancelled
        "failed" -> R.string.os_failed
        else -> R.string.md_searching
    },
)

/** **نقطةٌ تنبض** — تتّسع وتخفت بلا توقّف. */
@Composable
internal fun PulseDot(color: Color, size: Dp = 10.dp) {
    val pulse = rememberInfiniteTransition(label = "dot")
    val a by pulse.animateFloat(
        initialValue = 1f,
        targetValue = 0.25f,
        animationSpec = infiniteRepeatable(tween(700, easing = FastOutSlowInEasing), RepeatMode.Reverse),
        label = "dotAlpha",
    )
    Box(Modifier.size(size).alpha(a).clip(CircleShape).background(color))
}

/** **شارةُ الحال** — والانتظارُ نقطتُه تنبض. */
@Composable
internal fun StatusPill(status: String) {
    val tone = toneOf(status)
    val c = toneColor(tone)
    Row(
        Modifier
            .clip(CircleShape)
            .background(c.copy(alpha = 0.12f))
            .padding(horizontal = 10.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (tone == Tone.Waiting) PulseDot(c, 8.dp)
        else Box(Modifier.size(8.dp).clip(CircleShape).background(c))
        Spacer(Modifier.width(6.dp))
        Text(
            deliveryStatusText(status),
            color = c,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelMedium,
        )
    }
}

/**
 * **«جاري البحث عن سائق…»** — دائرتان تتّسعان وتخفتان حول أيقونة الموتور،
 * **كرادارٍ يمسح** — فيرى المتجرُ أنّ شيئاً يجري لا أنّ التطبيقَ واقف.
 */
@Composable
internal fun SearchingBanner(compact: Boolean = false) {
    val c = Rahal.colors.accent
    val wave = rememberInfiniteTransition(label = "radar")
    val p1 by wave.animateFloat(
        0f, 1f, infiniteRepeatable(tween(1600, easing = LinearEasing)), label = "w1",
    )
    val p2 by wave.animateFloat(
        0f, 1f,
        infiniteRepeatable(tween(1600, delayMillis = 800, easing = LinearEasing)),
        label = "w2",
    )
    val box = if (compact) 44.dp else 64.dp
    val core = if (compact) 28.dp else 40.dp
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(c.copy(alpha = 0.08f))
            .border(Rahal.stroke.hair, c.copy(alpha = 0.35f), Rahal.shape.md)
            .padding(if (compact) 10.dp else 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(box), contentAlignment = Alignment.Center) {
            for (p in listOf(p1, p2)) {
                Box(
                    Modifier
                        .size(box)
                        .scale(0.45f + 0.55f * p)
                        .alpha(1f - p)
                        .clip(CircleShape)
                        .background(c.copy(alpha = 0.35f)),
                )
            }
            Box(
                Modifier.size(core).clip(CircleShape).background(c),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_moto),
                    contentDescription = null,
                    tint = Rahal.colors.onBrand,
                    modifier = Modifier.size(if (compact) 16.dp else 22.dp),
                )
            }
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.md_searching),
                color = c,
                fontWeight = FontWeight.Bold,
                style = if (compact) MaterialTheme.typography.bodyMedium else MaterialTheme.typography.titleMedium,
            )
            Text(
                stringResource(R.string.md_searching_hint),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/** **سطرُ معلومةٍ بأيقونته** — المستلِمُ والرقمُ والعنوانُ والأجرة. */
@Composable
internal fun InfoLine(icon: Int, text: String, strong: Boolean = false, muted: Boolean = false) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 3.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painterResource(icon),
            contentDescription = null,
            tint = Rahal.colors.inkMuted,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(
            text,
            fontWeight = if (strong) FontWeight.Bold else FontWeight.Normal,
            color = if (muted) Rahal.colors.inkMuted else Rahal.colors.ink,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/**
 * **زرُّ «إلغاء التوصيلة»** — أحمرُ بعرض البطاقة، **ويسأل قبل أن يُلغي.**
 *
 * **و«تراجع» هو الافتراض** — لمسةٌ خاطئةٌ لا تُلغي توصيلةً بُحث لها عن سائق.
 */
@Composable
internal fun CancelDeliveryButton(busy: Boolean, onConfirm: () -> Unit) {
    var asking by rememberSaveable { mutableStateOf(false) }
    // **ونبرةُ الخطر المشتركة** (`Tone.Danger`) — زرُّ «اعتذر عن الطلب» بها أيضاً.
    com.rahalgo.ui.RahalOutlineButton(
        onClick = { asking = true },
        enabled = !busy,
        tone = com.rahalgo.ui.Tone.Danger,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_close),
            contentDescription = null,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(
            stringResource(if (busy) R.string.md_cancelling else R.string.md_cancel),
            fontWeight = FontWeight.Bold,
        )
    }
    if (asking) {
        ConfirmDialog(
            title = stringResource(R.string.md_cancel_q),
            body = stringResource(R.string.md_cancel_body),
            confirm = stringResource(R.string.md_cancel_yes),
            onConfirm = onConfirm,
            onDismiss = { asking = false },
        )
    }
}

/** **فاصلٌ رفيع** بين رأس البطاقة وتفاصيلها. */
@Composable
internal fun Hairline() {
    Box(
        Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp)
            .height(1.dp)
            .background(Rahal.colors.line),
    )
}

package com.rahalgo.customer.prizes

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal

/**
 * **زرُّ الهديّة بجانب البحث** — يفتح «جوائز رحّال غو» وعليه شارةُ نقاطه هذا الشهر.
 *
 * **ويَنبض ثلاثَ مرّاتٍ لا أكثر**: حين تُفتح الشاشةُ، وحين يتبدّل العدّاد (`PrizesLogic.shouldPulse`).
 * **ونبضٌ لا ينتهي يُتعب العينَ ويستنزف البطّاريّة** — لفتُ النظر مرّةً يكفي.
 * **والشارةُ مخفيّةٌ قبل الافتتاح** (`badge == null`).
 */
@Composable
fun PrizesButton(badge: String?, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val pulse = remember { Animatable(0f) }
    var opened by remember { mutableStateOf(false) }
    var seen by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(badge) {
        val play = !opened || PrizesLogic.shouldPulse(seen, badge)
        opened = true
        seen = badge
        if (play) {
            repeat(PULSES) {
                pulse.animateTo(1f, tween(PULSE_MS))
                pulse.animateTo(0f, tween(PULSE_MS))
            }
        }
    }
    Box(
        modifier
            .padding(horizontal = Rahal.space.xs)
            .size(Rahal.size.touch)
            .clip(CircleShape)
            .clickable(role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_gift),
            contentDescription = stringResource(R.string.prizes_open),
            tint = Rahal.colors.accent,
            modifier = Modifier
                .size(Rahal.icon.md)
                .scale(1f + 0.25f * pulse.value)
                .alpha(1f - 0.3f * pulse.value),
        )
        if (badge != null) {
            Box(
                Modifier
                    .align(Alignment.TopEnd)
                    .clip(CircleShape)
                    .background(Rahal.colors.brand)
                    .defaultMinSize(minWidth = Rahal.size.count)
                    .padding(horizontal = Rahal.space.xs),
                contentAlignment = Alignment.Center,
            ) {
                Text(badge, color = Rahal.colors.onBrand, style = Rahal.type.badge, maxLines = 1)
            }
        }
    }
}

private const val PULSES = 3
private const val PULSE_MS = 300

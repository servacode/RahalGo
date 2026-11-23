package com.rahalgo.customer.prizes

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
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
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal

/**
 * **هديّةٌ عائمةٌ ثلاثيّةُ الأبعاد فوق السوق** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤: «ايقونة الهدية لازم
 * عائمة ثري دي واضحة مو بهذا الشكل») — دائرةٌ بظلٍّ فيها علبةُ هديّةٍ مجسّمة (`gift_3d`)،
 * **وعليها شارةُ التقدّم «٣/٥»** بعد الافتتاح.
 *
 * **تقفز ثلاثَ قفزاتٍ لا أكثر** — حين يُفتح السوق وحين يتبدّل العدّاد (`PrizesLogic.shouldPulse`)؛
 * **وحركةٌ لا تنتهي تُتعب العينَ وتُقرأ إعلاناً مزعجاً.**
 */
@Composable
fun FloatingGift(badge: String?, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val hop = remember { Animatable(0f) }
    var opened by remember { mutableStateOf(false) }
    var seen by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(badge) {
        val play = !opened || PrizesLogic.shouldPulse(seen, badge)
        opened = true
        seen = badge
        if (play) {
            repeat(HOPS) {
                hop.animateTo(1f, tween(HOP_MS, easing = FastOutSlowInEasing))
                hop.animateTo(0f, tween(HOP_MS, easing = FastOutSlowInEasing))
            }
        }
    }
    Box(modifier.graphicsLayer { translationY = -HOP_PX * hop.value }) {
        Box(
            Modifier
                .size(SIZE)
                .shadow(Rahal.space.sm, CircleShape)
                .clip(CircleShape)
                .background(Rahal.colors.surface)
                .clickable(role = Role.Button, onClick = onClick),
            contentAlignment = Alignment.Center,
        ) {
            Image(
                painter = painterResource(R.drawable.gift_3d),
                contentDescription = stringResource(R.string.prizes_open),
                modifier = Modifier
                    .size(GIFT)
                    .graphicsLayer {
                        val k = 1f + 0.08f * hop.value
                        scaleX = k
                        scaleY = k
                    },
            )
        }
        if (badge != null) {
            Box(
                Modifier
                    .align(Alignment.TopStart)
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

private const val HOPS = 3
private const val HOP_MS = 260
private const val HOP_PX = 28f
private val SIZE = 68.dp
private val GIFT = 54.dp

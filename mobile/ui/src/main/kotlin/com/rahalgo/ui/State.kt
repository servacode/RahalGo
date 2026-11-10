package com.rahalgo.ui

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالةٌ واحدةٌ للفراغ والخطأ وانقطاع الشبكة** (D11، ٢٠٢٦-١١-١١: «نصٌّ عاريٌّ بلا شكل»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أيقونةٌ في دائرةٍ + عنوانٌ + سطرٌ + زرٌّ اختياريّ** — بالشكل نفسِه في كلّ التطبيقات.
 * `Empty` و`LoadState` يناديانها، فلا تُبنى حالةٌ بيد.
 */
@Composable
fun StateMessage(
    title: String,
    modifier: Modifier = Modifier,
    @androidx.annotation.DrawableRes icon: Int = R.drawable.ic_info,
    line: String = "",
    actionLabel: String = "",
    onAction: (() -> Unit)? = null,
    /** **لونُ الدائرة** — العلامةُ للفراغ، والخطرُ للخطأ. */
    danger: Boolean = false,
) {
    val tone = if (danger) Rahal.colors.danger else Rahal.colors.brand
    Box(modifier.fillMaxWidth().padding(vertical = Rahal.space.xl), contentAlignment = Alignment.Center) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.padding(horizontal = Rahal.space.xl),
        ) {
            Box(
                Modifier.size(EmptyIconCircle).clip(Rahal.shape.pill).background(tone.copy(alpha = 0.10f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(painterResource(icon), contentDescription = null, tint = tone, modifier = Modifier.size(Rahal.space.xxl))
            }
            Spacer(Modifier.height(Rahal.space.md))
            Text(
                title,
                color = if (danger) Rahal.colors.danger else Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                textAlign = TextAlign.Center,
            )
            if (line.isNotBlank()) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(line, color = Rahal.colors.inkMuted, textAlign = TextAlign.Center, style = MaterialTheme.typography.bodySmall)
            }
            if (actionLabel.isNotBlank() && onAction != null) {
                Spacer(Modifier.height(Rahal.space.md))
                RahalOutlineButton(onClick = onAction) { Text(actionLabel) }
            }
        }
    }
}

/** **حالةُ انقطاع الاتّصال** — بأيقونة «لا إشارة» وزرِّ إعادة. */
@Composable
fun OfflineState(onRetry: (() -> Unit)? = null, modifier: Modifier = Modifier) {
    StateMessage(
        title = stringResource(R.string.state_offline_title),
        line = stringResource(R.string.state_offline_line),
        icon = R.drawable.ic_no_signal,
        actionLabel = if (onRetry != null) stringResource(R.string.act_retry) else "",
        onAction = onRetry,
        modifier = modifier,
    )
}


/**
 * **هيكلُ تحميلٍ لقائمةِ بطاقات** — بطاقاتٌ رماديّةٌ تنبض بدل دوّارةٍ فوق فراغ. **`count` بطاقة** بصورةٍ
 * وسطرين، بالألوان من اللوح.
 */
@Composable
fun SkeletonList(count: Int = 4, modifier: Modifier = Modifier) {
    val t = rememberInfiniteTransition(label = "skeleton")
    val a by t.animateFloat(
        initialValue = 0.35f,
        targetValue = 0.8f,
        animationSpec = infiniteRepeatable(tween(900), RepeatMode.Reverse),
        label = "pulse",
    )
    val block = Rahal.colors.inkMuted.copy(alpha = 0.18f * a)
    Column(modifier.fillMaxWidth().padding(Rahal.space.lg), verticalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
        repeat(count) {
            Row(
                Modifier.fillMaxWidth().clip(Rahal.shape.md).background(block.copy(alpha = block.alpha * 0.5f)).padding(Rahal.space.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(Modifier.size(56.dp).clip(Rahal.shape.sm).background(block))
                Spacer(Modifier.size(Rahal.space.md))
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    Box(Modifier.fillMaxWidth(0.7f).height(14.dp).clip(Rahal.shape.sm).background(block))
                    Box(Modifier.fillMaxWidth(0.45f).height(12.dp).clip(Rahal.shape.sm).background(block))
                }
            }
        }
    }
}

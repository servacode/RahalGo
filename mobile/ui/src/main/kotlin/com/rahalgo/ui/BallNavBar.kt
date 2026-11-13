package com.rahalgo.ui

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.rahalgo.design.Rahal
import kotlin.math.abs

/** **بندٌ في شريط الكرة** — أيقونةٌ واسمٌ وشارةُ عدد، ووسمُ الأتمتة. */
data class BallTab(
    val icon: Int,
    val label: Int,
    val badge: Int = 0,
    val tag: String = "",
)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ «الكرة»** (قرارُ المالك ٢٠٢٦-١١-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بدي الدائرة تتحول للبرتقالي وتوصل للطلباتي تصير زرقة… وكانها كرة تتحرك للقسم الثاني، والايقونة
 * والاسم بالدائرة كمان». **دائرةٌ مرفوعةٌ فوق المختار فيها أيقونتُه واسمُه**؛ وحين يُختار غيرُه تنزلق
 * إليه **برتقاليّةً في الطريق وزرقاءَ حين تصل**. والبنودُ الأخرى أيقونةٌ واسمٌ عاديّان.
 *
 * **والموضعُ بجهة القراءة** (`offset` يتبع RTL): البندُ الأوّلُ أيمنُها كالشريط القديم.
 */
@Composable
fun RahalBallNavBar(tabs: List<BallTab>, selected: Int, onSelect: (Int) -> Unit) {
    if (tabs.isEmpty()) return
    val sel = selected.coerceIn(0, tabs.lastIndex)
    val barH = 62.dp
    val ball = 58.dp
    val lift = 20.dp
    val pos = remember { Animatable(sel.toFloat()) }
    LaunchedEffect(sel) {
        pos.animateTo(sel.toFloat(), tween(durationMillis = 340, easing = FastOutSlowInEasing))
    }
    val moving = abs(pos.value - sel) > 0.02f
    val ballColor by animateColorAsState(
        targetValue = if (moving) Rahal.colors.accent else Rahal.colors.brand,
        animationSpec = tween(160),
        label = "ballColor",
    )
    val line = Rahal.colors.line
    val hair = Rahal.stroke.hair
    BoxWithConstraints(
        Modifier
            .fillMaxWidth()
            .windowInsetsPadding(WindowInsets.navigationBars)
            .height(barH + lift),
    ) {
        val slot = maxWidth / tabs.size
        // **الأرضُ** — بيضاءُ بخطٍّ رفيعٍ في أعلاها كالشريط المشترك.
        Box(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .height(barH)
                .background(Rahal.colors.canvas)
                .drawBehind {
                    drawLine(line, Offset(0f, 0f), Offset(size.width, 0f), hair.toPx())
                },
        )
        Row(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .height(barH),
        ) {
            tabs.forEachIndexed { i, t ->
                Column(
                    Modifier
                        .weight(1f)
                        .fillMaxHeight()
                        .then(if (t.tag.isNotEmpty()) Modifier.testTag(t.tag) else Modifier)
                        .clickable(
                            interactionSource = remember { MutableInteractionSource() },
                            indication = null,
                            role = Role.Tab,
                        ) { onSelect(i) },
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                ) {
                    if (i == sel) {
                        // **مكانُه تحت الكرة** — اسمُه في الكرة نفسِها.
                        Spacer(Modifier.height(1.dp))
                    } else {
                        Box {
                            Icon(
                                painterResource(t.icon),
                                contentDescription = stringResource(t.label),
                                tint = Rahal.colors.inkMuted,
                                modifier = Modifier.size(Rahal.icon.md),
                            )
                            if (t.badge > 0) {
                                CountBadge(
                                    count = t.badge,
                                    color = Rahal.colors.accent,
                                    modifier = Modifier
                                        .align(Alignment.TopEnd)
                                        .offset(x = 8.dp, y = (-6).dp),
                                )
                            }
                        }
                        Text(
                            stringResource(t.label),
                            color = Rahal.colors.inkMuted,
                            fontSize = 11.sp,
                            maxLines = 1,
                        )
                    }
                }
            }
        }
        // **الكرة** — تنزلق بين البنود، وفيها أيقونةُ المختار واسمُه.
        val cur = tabs[sel]
        Box(
            Modifier
                .align(Alignment.TopStart)
                .offset(x = slot * pos.value + (slot - ball) / 2)
                .size(ball)
                .shadow(Rahal.elevation.card, CircleShape)
                .clip(CircleShape)
                .background(ballColor),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Box {
                    Icon(
                        painterResource(cur.icon),
                        contentDescription = null,
                        tint = Rahal.colors.onBrand,
                        modifier = Modifier.size(22.dp),
                    )
                    if (cur.badge > 0) {
                        CountBadge(
                            count = cur.badge,
                            color = Rahal.colors.accent,
                            modifier = Modifier
                                .align(Alignment.TopEnd)
                                .offset(x = 10.dp, y = (-6).dp),
                        )
                    }
                }
                Text(
                    stringResource(cur.label),
                    color = Rahal.colors.onBrand,
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

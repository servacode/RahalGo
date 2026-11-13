package com.rahalgo.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.offset
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.size
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشريطُ السفليّ الموحَّد لتطبيقَي المتجر والمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مراجعةُ التصميم الشاملة ٢٠٢٦-١١-٠٩.) **كان للمتجر شريطٌ مصقولٌ مكتوبٌ داخل
 * `MainActivity` وللمندوب شريطٌ افتراضيٌّ بألوان مادّة** — فاختلف التطبيقان
 * في أوّل ما يراه المستخدمُ وآخره. **وموضعٌ واحدٌ هنا** للأرض البيضاء والخطّ
 * الرفيع في أعلاه ولون المختار وشارة العدد.
 */
/**
 * **شريطٌ بلون العلامة** (قرارُ المالك ٢٠٢٦-١١-١٣: «خلي لون الشريط كله ازرق») — للزبون وحدَه الآن:
 * الأرضُ زرقاء، والأيقوناتُ والأسماءُ بيضاء، ودائرةُ الوسط بيضاءُ بأيقونةٍ زرقاء.
 */
val LocalNavOnBrand = androidx.compose.runtime.compositionLocalOf { false }

@Composable
fun RahalNavigationBar(onBrand: Boolean = false, content: @Composable RowScope.() -> Unit) {
    val line = Rahal.colors.line
    val hair = Rahal.stroke.hair
    NavigationBar(
        containerColor = if (onBrand) Rahal.colors.brand else Rahal.colors.canvas,
        tonalElevation = 0.dp,
        modifier = Modifier.drawBehind {
            drawLine(
                color = line,
                start = Offset(0f, 0f),
                end = Offset(size.width, 0f),
                strokeWidth = hair.toPx(),
            )
        },
        content = { androidx.compose.runtime.CompositionLocalProvider(LocalNavOnBrand provides onBrand) { content() } },
    )
}

/** **تبويبٌ واحدٌ في الشريط** — أيقونةٌ واسمٌ وشارةُ عددٍ اختيارية. */
@Composable
fun RowScope.RahalNavTab(
    selected: Boolean,
    onClick: () -> Unit,
    icon: Int,
    label: Int,
    badge: Int = 0,
    /** **وسمُ الأتمتة وغيرُه** — يُمرَّر لبند الشريط. */
    modifier: Modifier = Modifier,
    /** **لونُ شارة العدد** — أحمرُ افتراضاً، والسلّةُ بلون العلامة. */
    badgeColor: androidx.compose.ui.graphics.Color = Rahal.colors.danger,
) {
    NavigationBarItem(
        modifier = modifier,
        selected = selected,
        onClick = onClick,
        icon = {
            // **نبضةٌ خفيفةٌ للمختار** (قرارُ المالك ٢٠٢٦-١١-١٣: «حركة تاثير وقت الانتقال») — تكبر وترجع.
            val pop by androidx.compose.animation.core.animateFloatAsState(
                targetValue = if (selected) 1.12f else 1f,
                animationSpec = androidx.compose.animation.core.spring(dampingRatio = 0.45f, stiffness = 500f),
                label = "navPop",
            )
            Box(Modifier.graphicsLayer { scaleX = pop; scaleY = pop }) {
                Icon(painterResource(icon), contentDescription = null)
                if (badge > 0) {
                    CountBadge(
                        count = badge,
                        color = badgeColor,
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .offset(x = 8.dp, y = (-6).dp),
                    )
                }
            }
        },
        label = {
            // **يصغر ولا يُقصّ** (الفحصُ البصريّ ٢٠٢٦-١١-١١: «الطلبا…»، «أضف …» بستّة تبويبات على ٣٦٠).
            FitText(
                stringResource(label),
                maxLines = 1,
                style = androidx.compose.material3.LocalTextStyle.current.copy(
                    fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
                ),
            )
        },
        colors = NavigationBarItemDefaults.colors(
            selectedIconColor = if (LocalNavOnBrand.current) Rahal.colors.onBrand else Rahal.colors.brand,
            selectedTextColor = if (LocalNavOnBrand.current) Rahal.colors.onBrand else Rahal.colors.brand,
            indicatorColor = if (LocalNavOnBrand.current) Rahal.colors.onBrand.copy(alpha = 0.18f) else Rahal.colors.brand.copy(alpha = 0.12f),
            unselectedIconColor = if (LocalNavOnBrand.current) Rahal.colors.onBrand.copy(alpha = 0.72f) else Rahal.colors.inkMuted,
            unselectedTextColor = if (LocalNavOnBrand.current) Rahal.colors.onBrand.copy(alpha = 0.72f) else Rahal.colors.inkMuted,
        ),
    )
}

/**
 * **زرُّ الوسط الكبير** (قرارُ المالك ٢٠٢٦-١١-١٣: «زر تسوق بالنص… اكبر من بقية العناصر») — دائرةٌ بلون
 * العلامة وظلٍّ خفيف، **تصغر لحظةَ الضغط وترجع**، واسمُه تحتها كباقي البنود.
 */
@Composable
fun RowScope.RahalNavCenter(
    selected: Boolean,
    onClick: () -> Unit,
    icon: Int,
    label: Int,
    modifier: Modifier = Modifier,
) {
    val press = remember { androidx.compose.foundation.interaction.MutableInteractionSource() }
    val down by press.collectIsPressedAsState()
    val scale by androidx.compose.animation.core.animateFloatAsState(
        targetValue = if (down) 0.9f else if (selected) 1.05f else 1f,
        animationSpec = androidx.compose.animation.core.spring(dampingRatio = 0.5f, stiffness = 600f),
        label = "navCenter",
    )
    androidx.compose.foundation.layout.Column(
        modifier
            .weight(1f)
            .align(Alignment.CenterVertically)
            .clickable(interactionSource = press, indication = null, role = androidx.compose.ui.semantics.Role.Tab, onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center,
    ) {
        Box(
            Modifier
                .graphicsLayer { scaleX = scale; scaleY = scale }
                .size(54.dp)
                .shadow(Rahal.elevation.card, CircleShape)
                .clip(CircleShape)
                .background(if (LocalNavOnBrand.current) Rahal.colors.onBrand else Rahal.colors.brand),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(icon),
                contentDescription = null,
                tint = if (LocalNavOnBrand.current) Rahal.colors.brand else Rahal.colors.onBrand,
                modifier = Modifier.size(28.dp),
            )
        }
        Text(
            stringResource(label),
            color = when {
                LocalNavOnBrand.current -> if (selected) Rahal.colors.onBrand else Rahal.colors.onBrand.copy(alpha = 0.72f)
                selected -> Rahal.colors.brand
                else -> Rahal.colors.inkMuted
            },
            fontWeight = FontWeight.Bold,
            style = androidx.compose.material3.MaterialTheme.typography.labelSmall,
            maxLines = 1,
        )
    }
}

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
@Composable
fun RahalNavigationBar(content: @Composable RowScope.() -> Unit) {
    val line = Rahal.colors.line
    val hair = Rahal.stroke.hair
    NavigationBar(
        containerColor = Rahal.colors.canvas,
        tonalElevation = 0.dp,
        modifier = Modifier.drawBehind {
            drawLine(
                color = line,
                start = Offset(0f, 0f),
                end = Offset(size.width, 0f),
                strokeWidth = hair.toPx(),
            )
        },
        content = content,
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
) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = {
            Box {
                Icon(painterResource(icon), contentDescription = null)
                if (badge > 0) {
                    CountBadge(
                        count = badge,
                        color = Rahal.colors.danger,
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .offset(x = 8.dp, y = (-6).dp),
                    )
                }
            }
        },
        label = {
            Text(
                stringResource(label),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
            )
        },
        colors = NavigationBarItemDefaults.colors(
            selectedIconColor = Rahal.colors.brand,
            selectedTextColor = Rahal.colors.brand,
            indicatorColor = Rahal.colors.brand.copy(alpha = 0.12f),
            unselectedIconColor = Rahal.colors.inkMuted,
            unselectedTextColor = Rahal.colors.inkMuted,
        ),
    )
}

package com.rahalgo.ui

import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * **نجمةُ التقييم — موضعٌ واحدٌ لشكلها** (D10، ٢٠٢٦-١١-١١: الفارغةُ شبهُ مخفيّة، تباينُها ١٫٢:١)
 *
 * **المختارةُ ممتلئةٌ برتقاليّة، والفارغةُ مفرّغةٌ بحدٍّ رماديٍّ غامق** (`inkMuted`، تباينُه أكثرُ من ٣:١ مع
 * الأرضيّة في السمتين) — فلا تُقرأ النجمةُ الفارغةُ غائبة. **والحجمُ الافتراضيّ ٤٠dp** لخانةٍ تُضغط.
 */
@Composable
fun RatingStar(filled: Boolean, modifier: Modifier = Modifier, size: Dp = 40.dp) {
    Icon(
        painter = painterResource(if (filled) R.drawable.ic_star else R.drawable.ic_star_outline),
        contentDescription = null,
        tint = if (filled) Rahal.colors.accent else Rahal.colors.inkMuted,
        modifier = modifier.size(size),
    )
}

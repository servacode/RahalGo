package com.rahalgo.rep.clients

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RemoteImage

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قطعُ بطاقة العميل** (طلبُ المالك ٢٠٢٦-١١-١١: «بطاقةُ المتجر هواة وغيرُ واضحة»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مربّعُ وظيفةٍ** — أيقونةٌ فوق اسمِ الوظيفة وقيمتِها الحاليّة، ويُضغط كلُّه (٤٤dp فأكثر). **وصفٌّ منها
 * بدل أسطرٍ تُقرأ طولاً وأزرارٍ مؤطَّرةٍ كبيرة.** (قطعةٌ محلّيّةٌ في التطبيق؛ تُنقل إلى `ui` إن احتاجها غيرُه.)
 */
@Composable
internal fun InfoTile(
    icon: Int,
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    valueColor: Color = Rahal.colors.inkMuted,
    onClick: () -> Unit,
) {
    Column(
        modifier
            .fillMaxHeight()
            .heightIn(min = 88.dp)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.brand.copy(alpha = 0.07f))
            .clickable(role = Role.Button, onClick = onClick)
            .padding(horizontal = 6.dp, vertical = 10.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(3.dp, Alignment.CenterVertically),
    ) {
        Icon(
            painterResource(icon),
            contentDescription = null,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(24.dp),
        )
        Text(
            label,
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelMedium,
            textAlign = TextAlign.Center,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
        if (value.isNotEmpty()) {
            Text(
                value,
                color = valueColor,
                style = MaterialTheme.typography.labelSmall,
                textAlign = TextAlign.Center,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** **أيقونةُ تواصلٍ مدوَّرة** — ٤٤dp، بلا نصّ (الوصفُ لقارئ الشاشة). */
@Composable
internal fun RoundIconAction(
    icon: Int,
    description: String,
    tint: Color,
    onClick: () -> Unit,
) {
    Box(
        Modifier
            .size(44.dp)
            .clip(CircleShape)
            .background(tint.copy(alpha = 0.12f))
            .border(BorderStroke(1.dp, tint.copy(alpha = 0.35f)), CircleShape)
            .clickable(role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(painterResource(icon), contentDescription = description, tint = tint, modifier = Modifier.size(22.dp))
    }
}

/** **الشعارُ وعليه كاميرا صغيرة** — يُضغط فيفتح «كاميرا أم معرض». */
@Composable
internal fun LogoWithCamera(url: String?, name: String, enabled: Boolean, description: String, onPick: () -> Unit) {
    Box(Modifier.size(72.dp)) {
        RemoteImage(
            url = url,
            name = name,
            modifier = Modifier
                .size(68.dp)
                .clip(Rahal.shape.md)
                .clickable(enabled = enabled, onClick = onPick),
        )
        Box(
            Modifier
                .align(Alignment.BottomEnd)
                .size(28.dp)
                .clip(CircleShape)
                .background(Rahal.colors.brand)
                .border(BorderStroke(2.dp, Rahal.colors.canvas), CircleShape)
                .clickable(enabled = enabled, role = Role.Button, onClick = onPick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_photo),
                contentDescription = description,
                tint = Color.White,
                modifier = Modifier.size(15.dp),
            )
        }
    }
}

/** **نقطةُ حالةٍ ونصُّها** — «يعمل» أخضر و«متوقف» رماديّ. */
@Composable
internal fun StatusDot(text: String, color: Color) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(8.dp).clip(CircleShape).background(color))
        Spacer(Modifier.size(5.dp))
        Text(text, color = color, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelMedium)
    }
}

@Composable
internal fun FullWidthRow(content: @Composable androidx.compose.foundation.layout.RowScope.() -> Unit) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp), content = content)
}

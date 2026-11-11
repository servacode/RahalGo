package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحٌ مقسوم — خياران أو ثلاثة بعرضٍ متساوٍ على أرضٍ واحدة** (إعادةُ التصميم ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **«جارية · منتهية» في طلباتي و«مشتريات · أمانة» في الطلب الخاصّ كانا شكلين**: حبّتان
 * بإطارٍ هنا، وزرٌّ مملوءٌ وآخرُ محدَّدٌ هناك — **فيُقرأ الثاني فعلين لا اختياراً.** والآن
 * شكلٌ واحد: **أرضُ الحقل، والمختارُ مرفوعٌ بلون العلامة**، وكلُّ قسمٍ بارتفاع لمسٍ مريح.
 *
 * **وهي القطعةُ المشتركة** — نُقلت من تطبيق الزبون (`Segmented`) بالواجهة نفسِها، فكلُّ تطبيقٍ
 * يبدّل إليها بلا تغييرِ نداء. ومرآتُها في الويب `SegmentedToggle` في `@rahalgo/ui`.
 */
@Composable
fun SegmentedToggle(
    options: List<String>,
    selected: Int,
    onSelect: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier
            .fillMaxWidth()
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.field)
            .padding(Rahal.space.xs)
            .selectableGroup(),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
    ) {
        options.forEachIndexed { i, label ->
            val on = i == selected
            Box(
                Modifier
                    .weight(1f)
                    .heightIn(min = Rahal.size.iconBox)
                    .clip(Rahal.shape.pill)
                    .then(if (on) Modifier.background(Rahal.colors.brand) else Modifier)
                    .selectable(selected = on, role = Role.Tab, onClick = { onSelect(i) })
                    .padding(horizontal = Rahal.space.sm),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = label,
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                    fontWeight = if (on) FontWeight.Bold else FontWeight.Medium,
                    style = Rahal.type.label,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

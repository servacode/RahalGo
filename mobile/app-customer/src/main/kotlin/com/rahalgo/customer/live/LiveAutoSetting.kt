package com.rahalgo.customer.live

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.clickable
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal

/**
 * **«شارك موقعي المباشر مع السائق أثناء الطلب» — في «حسابي ← الإعدادات»** (قرارُ المالك ٢٠٢٦-١١-٢٤:
 * «شارك موقعي ننقلو بالاعدادات ونشيلو من العنوان مشان ماحدا يطفي بالغلط»).
 *
 * **كان في صفحة العنوان** فيطفئه من يعدّل عنوانَه ولا يدري. **والتفضيلُ نفسُه** (`LiveAuto`)
 * — فمن أطفأه من قبل بقي مطفأً.
 */
@Composable
fun LiveAutoSetting() {
    val context = LocalContext.current
    LaunchedEffect(Unit) { LiveAuto.load(context) }
    Row(
        Modifier
            .fillMaxWidth()
            .clickable { LiveAuto.set(context, !LiveAuto.on) }
            .padding(vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            stringResource(R.string.addrp_live),
            fontWeight = FontWeight.Medium,
            style = Rahal.type.body,
            modifier = Modifier.weight(1f),
        )
        // **حالُ المفتاح بالكلمة (تدقيق D18)** — المفتاحُ وحدَه لا يقول «شغّال» أم «واقف» لكلّ العيون.
        Text(
            stringResource(if (LiveAuto.on) R.string.addrp_live_on else R.string.addrp_live_off),
            color = if (LiveAuto.on) Rahal.colors.success else Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = Rahal.type.label,
        )
        Spacer(Modifier.size(Rahal.space.sm))
        Switch(checked = LiveAuto.on, onCheckedChange = { LiveAuto.set(context, it) })
    }
}

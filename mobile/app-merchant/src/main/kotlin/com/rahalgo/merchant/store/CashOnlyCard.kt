package com.rahalgo.merchant.store

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R

/**
 * **طريقةُ استلام الأموال — نقدٌ لا غير** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٩: شيلت «المحفظة»).
 *
 * بطاقةٌ ثابتةٌ لا اختيار: السائقُ يدفع للمتجر حقَّ البضاعة نقداً عند الاستلام ناقصَ العمولة.
 * **ونفسُ الكلام على الموقع** (`store.wizard.cashOnly`). ورصيدُ المحفظة القديمُ يبقى ويُسحب كما كان.
 */
@Composable
internal fun CashOnlyCard() {
    // **البطاقةُ الملوّنةُ الموحّدة** (تصميمُ ٢٠٢٦-١١-١١) — `RahalCard.Tinted` بنبرة المعلومة.
    com.rahalgo.ui.RahalCard(variant = com.rahalgo.ui.CardVariant.Tinted, tone = com.rahalgo.ui.StatusTone.Info) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(Rahal.size.touch)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.brand),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(R.drawable.ic_banknotes),
                contentDescription = null,
                tint = Rahal.colors.onBrand,
                modifier = Modifier.size(Rahal.icon.md),
            )
        }
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.wz_pay_cash),
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = Rahal.type.sectionTitle,
            )
            Text(
                stringResource(R.string.settlement_cash_only),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
    }
    }
}

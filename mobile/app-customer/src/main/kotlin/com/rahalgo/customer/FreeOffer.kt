package com.rahalgo.customer

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.FreeDelivery
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.money
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * **عرضُ التوصيل المجاني — أهليّةُ الزبون كما يقولها الخادم** (٢٠٢٦-١٠-٢٦)
 *
 * **وحالٌ واحدةٌ للتطبيق** — تقرؤها لافتةُ السوق وشاشةُ الطلب الخاصّ معاً.
 * **والضيفُ لا عرضَ له** — ولا نداءَ بلا جلسة.
 */
object FreeOffer {

    var state: FreeDelivery by mutableStateOf(FreeDelivery())
        private set

    fun refresh(scope: CoroutineScope) {
        val core = AppCore.get()
        if (core.session.refreshToken().isEmpty()) {
            state = FreeDelivery()
            return
        }
        scope.launch { state = CustomerApi(core.api).freeDelivery() }
    }
}

/**
 * **سطرُ أجرة التوصيل** — وإن أسقط العرضُ شيئاً: الأصلُ مشطوباً وبجانبه «مجاني 🎁»
 * أو ما بقي (كما في الويب).
 */
@Composable
fun DeliveryFeeRow(label: String, fee: Long, waived: Long) {
    if (waived <= 0) {
        com.rahalgo.ui.KeyValue(label, money(fee))
        return
    }
    Row(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                money(fee + waived),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
                textDecoration = TextDecoration.LineThrough,
            )
            Spacer(Modifier.width(8.dp))
            if (fee > 0) {
                Text(money(fee), style = MaterialTheme.typography.bodyMedium)
            } else {
                Text(
                    stringResource(R.string.free_badge),
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
    }
}

/** **لافتةُ «توصيلتك الجاية مجانية»** — لمن يحقّ له أيُّ نوعٍ والطلبُ مفتوح (كما في الويب). */
@Composable
fun FreeDeliveryBanner() {
    if (!FreeOffer.state.any || !Serving.available) return
    androidx.compose.foundation.layout.Column(Modifier.padding(horizontal = 12.dp)) {
        com.rahalgo.ui.Note(stringResource(R.string.free_banner), Rahal.colors.success)
    }
}

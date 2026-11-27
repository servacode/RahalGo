package com.rahalgo.customer

import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.StampCard
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.LoyaltyCards
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * **بطاقة رحّال غو وكرت اليوم للزبون** (قرارُ المالك ٢٠٢٦-١١-٢٧) — حالٌ واحدةٌ للتطبيق، تقرؤها
 * الرئيسيّةُ وصفحةُ العروض. **والضيفُ لا بطاقةَ له** — ولا نداءَ بلا جلسة.
 */
object Loyalty {

    var state: StampCard by mutableStateOf(StampCard())
        private set

    fun refresh(scope: CoroutineScope) {
        val core = AppCore.get()
        if (core.session.refreshToken().isEmpty()) {
            state = StampCard()
            return
        }
        scope.launch { state = CustomerApi(core.api).stampCard() }
    }
}

/** **البطاقتان بهامش الشاشة** — ولا شيءَ إن كانتا مطفأتين. */
@Composable
fun LoyaltyBlock(modifier: Modifier = Modifier) {
    LoyaltyCards(Loyalty.state, seenKey = "customer", modifier = modifier.padding(horizontal = Rahal.space.md))
}

package com.rahalgo.customer

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.StampCard
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * **بطاقة رحّال غو وكرت اليوم للزبون** (قرارُ المالك ٢٠٢٦-١١-٢٧) — حالٌ واحدةٌ للتطبيق، تقرؤها
 * أيقونةُ الشريط العلويّ وصفحةُ «بطاقاتي». **والضيفُ لا بطاقةَ له** — ولا نداءَ بلا جلسة.
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

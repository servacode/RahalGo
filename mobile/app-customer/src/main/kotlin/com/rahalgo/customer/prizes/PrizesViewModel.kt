package com.rahalgo.customer.prizes

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.PrizeItem
import com.rahalgo.shared.customer.PrizeWinner
import com.rahalgo.shared.customer.PrizesMe
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.apiError
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ «جوائز رحّال غو»** — حالُه وأصنافُ الجوائز ورابحو الشهر الماضي
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهو واحدٌ للتطبيق** (يقرؤه زرُّ الهديّة في السوق وشاشةُ الجوائز) — فالشارةُ والشاشةُ
 * تقرآن حالاً واحدة. **وقبل الافتتاح لا يُنادى غيرُ `prizes/me`** فلا أصنافَ ولا رابحين.
 *
 * **ورابحو الشهر الماضي لا يُسقطون الشاشة** — فشلُهم يعني قائمةً فارغةً مخفيّة.
 */
class PrizesViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var me by mutableStateOf<PrizesMe?>(null)
        private set
    var items by mutableStateOf<List<PrizeItem>>(emptyList())
        private set
    var winners by mutableStateOf<List<PrizeWinner>>(emptyList())
        private set
    var winnersMonth by mutableStateOf<String?>(null)
        private set

    /** **المتجرُ الذي تُطلب منه الجوائز** — من الأصناف أوّلاً، وإلّا من حال الزبون. */
    var storeId by mutableStateOf<String?>(null)
        private set

    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set

    private var job: Job? = null

    /** **أقبلَ الافتتاح؟** — يُحسب من اليوم بتوقيت دمشق. */
    val teaser: Boolean
        get() {
            val m = me ?: return true
            return PrizesLogic.isTeaser(m.enabled, m.launchOn, PrizesLogic.today())
        }

    /** **شارةُ الزرّ في السوق** — `null` قبل الافتتاح أو قبل أن يصل الحال. */
    val badge: String? get() = PrizesLogic.badge(me, teaser)

    /**
     * **يُجلب الحالُ** — ومنه تُجلب الأصنافُ والرابحون حين يكون الافتتاحُ قد جاء.
     * **ومن كان عنده بياناتٌ يُبقيها أثناء التحديث** فلا تومض الشاشة.
     */
    fun load() {
        job?.cancel()
        job = viewModelScope.launch {
            loading = true
            error = ""
            try {
                val m = api.prizesMe()
                me = m
                storeId = m.storeId
                // **والهدايا تُعرض قبل الافتتاح أيضاً** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — يراها ولا يطلبها.
                val page = runCatching { api.prizeItems() }.getOrNull()
                if (page != null) {
                    items = page.items
                    if (!page.storeId.isNullOrBlank()) storeId = page.storeId
                }
                if (PrizesLogic.isTeaser(m.enabled, m.launchOn, PrizesLogic.today())) {
                    winners = emptyList()
                } else {
                    val w = runCatching { api.prizeWinners() }.getOrNull()
                    winners = w?.winners.orEmpty()
                    winnersMonth = w?.month
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                // **فشلُ القراءة ليس فراغاً** — ومن عنده حالٌ قديمٌ يُبقيها ويُرى الخطأ فوقها.
                error = apiError(getApplication(), e)
            } finally {
                loading = false
            }
        }
    }

    /** **عند الخروج من الحساب يُمحى الحال** فلا يرثه حسابٌ تالٍ. */
    fun reset() {
        job?.cancel()
        me = null
        items = emptyList()
        winners = emptyList()
        winnersMonth = null
        storeId = null
        error = ""
        loading = false
    }
}

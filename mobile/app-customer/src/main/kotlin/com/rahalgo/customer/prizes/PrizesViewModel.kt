package com.rahalgo.customer.prizes

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.PrizeItem
import com.rahalgo.shared.customer.PrizesMe
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.apiError
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ «جوائز رحّال غو»** — حالُه في مسابقة النقاط وأصنافُ الجوائز
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهو واحدٌ للتطبيق** (يقرؤه زرُّ الهديّة في السوق وشاشةُ الجوائز) — فالشارةُ والشاشةُ
 * تقرآن حالاً واحدة. **ولا يُجلب شيءٌ عن زبونٍ آخر** (قرارُ المالك ٢٠٢٦-١١-٢٥): نقاطُه وترتيبُه
 * وشروطُه ونتيجتُه هو وحدَه.
 */
class PrizesViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var me by mutableStateOf<PrizesMe?>(null)
        private set
    var items by mutableStateOf<List<PrizeItem>>(emptyList())
        private set

    /** **المتجرُ الذي تُطلب منه الجوائز** — من الأصناف أوّلاً، وإلّا من حال الزبون. */
    var storeId by mutableStateOf<String?>(null)
        private set

    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set

    /** **حضورٌ أو متابعةٌ قيد الإرسال** — فلا يُضغط الزرُّ مرّتين. */
    var busy by mutableStateOf(false)
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
     * **يُجلب الحالُ** ثمّ الأصناف. **ومن كان عنده بياناتٌ يُبقيها أثناء التحديث** فلا تومض الشاشة.
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

    /** **«سجّل حضوري»** — والحالُ الجديدُ من الردّ نفسِه. */
    fun checkIn() = act { api.prizeCheckIn() }

    /** **«تابع»** بالأمانة — الشاشةُ تفتح الصفحة، وهنا يُعلَّم. */
    fun follow(network: String) = act { api.prizeFollow(network) }

    private fun act(call: suspend () -> PrizesMe) {
        if (busy) return
        viewModelScope.launch {
            busy = true
            error = ""
            try {
                me = call()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            } finally {
                busy = false
            }
        }
    }

    /** **عند الخروج من الحساب يُمحى الحال** فلا يرثه حسابٌ تالٍ. */
    fun reset() {
        job?.cancel()
        me = null
        items = emptyList()
        storeId = null
        error = ""
        loading = false
        busy = false
    }
}

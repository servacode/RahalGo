package com.rahalgo.merchant.offers

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.merchant.noStoreMsg
import com.rahalgo.shared.merchant.MenuItem
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.offers.NewOffer
import com.rahalgo.shared.offers.StoreOffer
import com.rahalgo.shared.offers.StoreOffersApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.err
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.format.DateTimeFormatter

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عروضُ متجره — يبنيها بنفسه** (`MO`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا حسبةَ سعرٍ هنا
 *
 * **والسعرُ بعد الخصم يجيء من المحرّك** — **ولا يُضرَب في الجهاز**:
 * **حسبةٌ في موضعين تفترق يوماً، فيرى سعراً ويُحاسَب بآخر.**
 *
 * # والحالُ يقولها المحرّك
 *
 * **ولا تُشتقّ من `ends_at` بساعة الجهاز** — **ومن قدّم ساعتَه رأى
 * منتهياً ساريا.** **والشاشةُ تعرض ما يُقال لها.**
 *
 * # والمدّةُ تُترجَم ولا تُخزَّن
 *
 * **يختار «يومين» فتُحسب لحظةُ نهاية وتُرسَل** — **و`ends_at` هي
 * الحقيقة.** **ومدّةٌ محفوظةٌ ثانيةً تفترق عنها.**
 */
class OffersViewModel(app: Application) : AndroidViewModel(app) {

    private val core = AppCore.get()
    private val merchant = MerchantApi(core.api)
    private val offers = StoreOffersApi.merchant(core.api)

    var storeId by mutableStateOf("")
        private set
    var rows by mutableStateOf<List<StoreOffer>?>(null)
        private set

    /** **أصنافُه** — **ليختار عليها، ولا يُكتب معرّفٌ بالأصابع.** */
    var items by mutableStateOf<List<MenuItem>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set

    /**
     * **العرضُ الذي يُنزَل الآن** — **ليدور زرُّه وحدَه.**
     *
     * **وضغطتان على زرٍّ واحدٍ نداءان** — **ورايةٌ واحدةٌ لكلّ الشاشة
     * تُعطّل الأزرارَ كلَّها**، **فيظنّ أنّ التطبيق علق.**
     */
    var stopping by mutableStateOf<String?>(null)
        private set

    init {
        refresh()
    }

    fun refresh() {
        if (busy) return
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                val store = storeId.ifEmpty {
                    merchant.stores().stores.firstOrNull()?.id
                        ?: throw IllegalStateException(noStoreMsg())
                }
                storeId = store
                rows = offers.list(store).offers
                items = merchant.menu(store).flatMap { it.items }.filter { it.approved }
            } catch (e: Exception) {
                error = describe(e)
            } finally {
                busy = false
            }
        }
    }

    /**
     * **ينشئ عرضاً** — **والمدّةُ ساعاتٌ تُترجَم لحظةَ نهاية.**
     *
     * **والفحصُ الظاهرُ هنا لا يُغني عن المحرّك** — **يقول في اللحظة
     * ما يُعرَف في اللحظة**، **والحكمُ هناك** (`bad_offer_window`).
     */
    fun create(itemId: String, percent: Int, hours: Int, title: String) {
        if (busy) return
        if (itemId.isBlank() || percent !in 1..90 || !OfferDuration.valid(hours)) {
            error = getApplication<Application>().getString(
                com.rahalgo.merchant.R.string.offer_bad_input,
            )
            return
        }
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                val ends = Instant.ofEpochMilli(
                    OfferDuration.endsAtMillis(System.currentTimeMillis(), hours),
                )
                offers.create(
                    storeId,
                    NewOffer(
                        title = title.ifBlank {
                            getApplication<Application>().getString(
                                com.rahalgo.merchant.R.string.offer_default_title,
                            )
                        },
                        menuItemId = itemId,
                        discountPercent = percent,
                        endsAt = DateTimeFormatter.ISO_INSTANT.format(ends),
                    ),
                )
                busy = false
                refresh()
            } catch (e: Exception) {
                error = describe(e)
                busy = false
            }
        }
    }

    /**
     * **يُنزل عرضاً** — **والحالُ من الردّ لا من تفاؤل.**
     *
     * **ومن بدّل الصفَّ متفائلاً أرى صاحبَه «موقوف» وهو سارٍ** — فيظنّ
     * أنّه أوقفه ويمضي.
     */
    fun stop(offerId: String) {
        // **وضغطةٌ ثانيةٌ على العرض نفسِه لا تفتح نداءً ثانياً.**
        if (stopping != null) return
        stopping = offerId
        error = ""
        viewModelScope.launch {
            try {
                val updated = offers.stop(storeId, offerId)
                rows = rows?.map { if (it.id == updated.id) updated else it }
            } catch (e: Exception) {
                error = describe(e)
            } finally {
                stopping = null
            }
        }
    }

    private fun describe(e: Exception): String = err(e)
}

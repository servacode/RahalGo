package com.rahalgo.rep.offers

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.offers.NewOffer
import com.rahalgo.shared.offers.StoreOffer
import com.rahalgo.shared.offers.StoreOffersApi
import com.rahalgo.shared.rep.MenuItem
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.err
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.format.DateTimeFormatter

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عروضُ عميله — في نطاقٍ قائمٍ لا مُخترَع** (`RO`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا متجرَ يُختار بمعرّفه
 *
 * **والمتجرُ يُفتَح من شاشة عملائه** (`/rep/merchants`) — **وهي التي
 * يردّها المحرّكُ بعلاقته**: `merchants.sales_rep_user_id`.
 *
 * **ولو كُتب معرّفٌ بيدٍ لَرُدّ من الخادم** (`repClient`) — **والحارسُ
 * هناك لا هنا**: **وشاشةٌ تحرس وحدَها ليست حارساً.**
 *
 * # واسمُ المتجر ظاهرٌ دائماً
 *
 * **ومندوبٌ يفتح عشرةَ متاجرَ في ساعة** — **ومن أنزل خصماً على متجرٍ
 * ظنّه غيرَه أضرّ برزق رجل.**
 */
class RepOffersViewModel(app: Application) : AndroidViewModel(app) {

    private val core = AppCore.get()
    private val rep = RepApi(core.api)
    private val offers = StoreOffersApi.rep(core.api)

    var merchantID by mutableStateOf("")
        private set

    /** **اسمُ المتجر الذي يُعمل فيه الآن** — **يبقى في أعلى الشاشة.** */
    var merchantName by mutableStateOf("")
        private set

    var rows by mutableStateOf<List<StoreOffer>?>(null)
        private set
    var items by mutableStateOf<List<MenuItem>>(emptyList())
        private set
    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var stopping by mutableStateOf<String?>(null)
        private set

    /** **يُفتح في سياق متجرٍ بعينه** — **ولا يُفتح بلا متجر.** */
    fun open(id: String, name: String) {
        merchantID = id
        merchantName = name
        rows = null
        error = ""
        load()
    }

    fun close() {
        merchantID = ""
        rows = null
    }

    fun load() {
        if (merchantID.isEmpty() || busy) return
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                rows = offers.list(merchantID).offers
                items = rep.menu(merchantID).flatMap { it.items }
            } catch (e: Exception) {
                error = err(e)
            } finally {
                busy = false
            }
        }
    }

    fun create(itemId: String, percent: Int, hours: Int) {
        if (busy || merchantID.isEmpty()) return
        if (itemId.isBlank() || percent !in 1..90 || !OfferDuration.valid(hours)) {
            error = getApplication<Application>().getString(
                com.rahalgo.rep.R.string.offer_bad_input,
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
                    merchantID,
                    NewOffer(
                        title = getApplication<Application>().getString(
                            com.rahalgo.rep.R.string.offer_default_title,
                        ),
                        menuItemId = itemId,
                        discountPercent = percent,
                        endsAt = DateTimeFormatter.ISO_INSTANT.format(ends),
                    ),
                )
                busy = false
                load()
            } catch (e: Exception) {
                error = err(e)
                busy = false
            }
        }
    }

    fun stop(offerId: String) {
        if (stopping != null) return
        stopping = offerId
        error = ""
        viewModelScope.launch {
            try {
                val updated = offers.stop(merchantID, offerId)
                rows = rows?.map { if (it.id == updated.id) updated else it }
            } catch (e: Exception) {
                error = err(e)
            } finally {
                stopping = null
            }
        }
    }
}

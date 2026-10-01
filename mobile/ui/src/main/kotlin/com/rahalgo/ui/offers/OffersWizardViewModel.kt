package com.rahalgo.ui.offers

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.offers.NewOffer
import com.rahalgo.shared.offers.StoreOffer
import com.rahalgo.shared.offers.StoreOffersApi
import com.rahalgo.ui.Attempt
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.R
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.err
import com.rahalgo.ui.isDecided
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.format.DateTimeFormatter

/** متجرٌ يُختار بالاسم — **ولا معرّفٌ يُكتب.** */
data class OfferStore(val id: String, val name: String)

/** قسمٌ بأصنافه كما ردّه المحرّك. */
data class OfferSection(val id: String, val name: String, val items: List<OfferItem>)

data class OfferItem(val id: String, val name: String, val price: Long)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ العروض المركزيّة — للمندوب وللمتجر معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «شاشة العروض يجب أن تكون مركزيّة بين المندوب
 *  والمتجر، وإذا لزمتنا بغير مكان أيضاً».)
 *
 * **كانت نسختين**: المندوبُ بخمس خطواتٍ ومبلغٍ ثابتٍ واستبدالٍ يُقال،
 * **والمتجرُ بنموذجٍ قديمٍ بالمئة وحدَها** — فكلُّ إصلاحٍ في واحدةٍ تنساه
 * الأخرى. **فصار المنطقُ هنا والتطبيقُ يقول شيئين فقط**: من أين تأتي
 * المتاجر، ومن أين تأتي القائمة.
 *
 * - **المندوب** (`picksStore = true`): يختار عميلاً من قائمته، ثمّ الخطواتُ الأربع.
 * - **المتجر** (`picksStore = false`): الفرعُ المختارُ يُفتح وحدَه — **أربعُ خطوات.**
 */
abstract class OffersWizardViewModel(app: Application) : AndroidViewModel(app) {

    protected abstract val offers: StoreOffersApi

    /** **هل يختار متجراً؟** المندوبُ نعم، والمتجرُ فرعُه مختارٌ سلفاً. */
    abstract val picksStore: Boolean

    /**
     * **هل يمرّ بالقسم قبل الصنف؟** المندوبُ نعم (طلبُ المالك ٢٠٢٦-٠٩-٣٠)،
     * **والمتجرُ يختار الصنفَ مباشرةً** بحثاً (طلبُه ٢٠٢٦-١٠-٠١).
     */
    open val picksSection: Boolean = true

    /** المتاجرُ التي يعمل عليها — **للمتجر: الفرعُ المختارُ وحدَه.** */
    protected abstract suspend fun stores(): List<OfferStore>

    /** أقسامُ قائمة المتجر بأصنافها — **المعتمَدةُ وحدَها.** */
    protected abstract suspend fun menu(storeId: String): List<OfferSection>

    var merchantID by mutableStateOf("")
        private set

    /** **اسمُ المتجر الذي يُعمل فيه الآن** — **يبقى في أعلى الشاشة.** */
    var merchantName by mutableStateOf("")
        private set

    var rows by mutableStateOf<List<StoreOffer>?>(null)
        private set
    var items by mutableStateOf<List<OfferItem>>(emptyList())
        private set
    var sections by mutableStateOf<List<OfferSection>>(emptyList())
        private set
    var clients by mutableStateOf<List<OfferStore>>(emptyList())
        private set
    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var stopping by mutableStateOf<String?>(null)
        private set

    /** عددُ ما أُنشئ بنجاح — تقرؤه الشاشةُ فتمسح القيمة. */
    var created by mutableStateOf(0)
        private set
    var createdShown by mutableStateOf(false)
        private set

    var pickedSection by mutableStateOf("")
        private set
    var pickedItem by mutableStateOf("")
        private set

    /** **طريقةُ الخصم** — نسبةٌ أو مبلغٌ ثابت (قرارُ المالك ٢٠٢٦-٠٩-٣٠). */
    var byPercent by mutableStateOf(true)
        private set

    fun clearCreated() {
        createdShown = false
    }

    fun pickMode(percent: Boolean) {
        byPercent = percent
    }

    /**
     * **يُحمَّل عند الفتح بلا متجر.** للمندوب: لوحُ عملائه. **وللمتجر: فرعُه
     * المختارُ يُفتح مباشرةً** — ويُعاد قراءتُه في كلّ فتح، فمن بدّل فرعَه
     * لا يُنشئ عرضاً على الآخر.
     */
    fun loadClients() {
        if (busy) return
        if (picksStore && clients.isNotEmpty()) return
        busy = true
        error = ""
        viewModelScope.launch {
            val list = try {
                stores()
            } catch (e: Exception) {
                error = err(e)
                null
            }
            busy = false
            if (list == null) return@launch
            clients = list
            // **والمتجرُ يُفتح بعد إسقاط الراية** — وإلّا ردّه حارسُ `load`.
            if (!picksStore) list.firstOrNull()?.let { open(it.id, it.name) }
        }
    }

    /** **يُفتح في سياق متجرٍ بعينه.** */
    fun open(id: String, name: String) {
        merchantID = id
        merchantName = name
        rows = null
        error = ""
        pickedSection = ""
        pickedItem = ""
        load()
    }

    fun clearMerchant() {
        merchantID = ""
        merchantName = ""
        rows = null
        items = emptyList()
        sections = emptyList()
        pickedSection = ""
        pickedItem = ""
        error = ""
        loadClients()
    }

    /** **والقسمُ يُختار فيُنسى الصنفُ** — صنفٌ من قسمٍ آخرَ اختيارٌ قديم. */
    fun pickSection(id: String) {
        pickedSection = id
        pickedItem = ""
    }

    fun pickItem(id: String) {
        pickedItem = id
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
                val secs = menu(merchantID)
                sections = secs
                items = secs.flatMap { it.items }
                // **وقسمٌ واحدٌ لا يُسأل عنه.**
                if (secs.size == 1) pickedSection = secs.first().id
            } catch (e: Exception) {
                error = err(e)
            } finally {
                busy = false
            }
        }
    }

    /**
     * **ينشئ الخصمَ بطريقتِه — وبمفتاح منع التكرار** (`Attempt.OFFER`): عرضٌ
     * أُرسل وضاع ردُّه فأُعيد **يُعاد ردُّه لا يُنشأ ثانيةً.**
     */
    fun create(itemId: String, value: Long, hours: Int) {
        if (busy || merchantID.isEmpty()) return
        val okValue = if (byPercent) value in 1..90 else value > 0
        if (itemId.isBlank() || !okValue || !OfferDuration.valid(hours)) {
            error = getApplication<Application>().getString(R.string.ow_bad_input)
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
                        title = getApplication<Application>().getString(R.string.ow_default_title),
                        menuItemId = itemId,
                        // **ويُرسَل أحدُهما لا كلاهما.**
                        discountPercent = if (byPercent) value.toInt() else null,
                        discountAmount = if (byPercent) null else value,
                        endsAt = DateTimeFormatter.ISO_INSTANT.format(ends),
                    ),
                    idempotencyKey = Attempt.key(Attempt.OFFER),
                )
                Attempt.clear(Attempt.OFFER)
                busy = false
                created += 1
                createdShown = true
                load()
                Refresh.bump()
            } catch (e: Exception) {
                if (isDecided(e)) Attempt.clear(Attempt.OFFER)
                error = err(e)
                createdShown = false
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
                Refresh.bump()
            } catch (e: Exception) {
                error = err(e)
            } finally {
                stopping = null
            }
        }
    }
}

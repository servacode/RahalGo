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

    // ══════════════════════════════════════════════════════════════════
    //  **ومسارٌ متدرّج: متجرٌ ثمّ قسمٌ ثمّ صنفٌ ثمّ نسبةٌ ثمّ مدّة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٩-٣٠ نصّاً: «يختار المتجر ثمّ يختار القسم ثمّ
    //  يختار الصنف ثمّ يحدّد النسبة ثمّ يحدّد المدّة».)
    //
    // # ولماذا لم تكن الشاشةُ تخيّر متجراً
    //
    // **`RO-01` (٢٠٢٦-٠٩-١٥) قصدَ ذلك**: تُفتح من داخل المتجر فيبقى اسمُه
    // فوق الشاشة، **ومن أنزل خصماً على متجرٍ ظنّه غيرَه أضرّ برزق رجل.**
    //
    // **والطلبُ لا ينقض ذلك بل يوسّعه**: **يُختار من قائمة عملائه
    // بالاسم** — لا بمعرّفٍ يُكتب — **واسمُه يبقى ظاهراً في كلّ خطوةٍ
    // بعده.** فيُنال التدرّجُ ويبقى الحارس.
    //
    // # والأقسامُ كانت تُسحق
    //
    // **كان `menu(...).flatMap { it.items }`** — **فصفٌّ واحدٌ يضمّ كلَّ
    // أصناف المتجر** بلا قسم. ومتجرٌ فيه ثمانون صنفاً يصير قائمةَ بحثٍ
    // بالعين. **فتُحفَظ الأقسامُ كما ردّها المحرّك**، والصنفُ يُبلَغ في
    // قسمه.

    /** **عملاؤه** — يُقرَأون من `/rep/merchants` بعلاقته، لا بمعرّفٍ يُكتب. */
    var clients by mutableStateOf<List<com.rahalgo.shared.rep.RepMerchant>>(emptyList())
        private set

    /** **أقسامُ قائمة المتجر المفتوح** — بأصنافها كما ردّها المحرّك. */
    var sections by mutableStateOf<List<com.rahalgo.shared.rep.MenuSection>>(emptyList())
        private set

    var pickedSection by mutableStateOf("")
        private set

    var pickedItem by mutableStateOf("")
        private set

    /**
     * **ويُحمَّل لوحُ العملاء قبل أن يُختار أحدٌ** — **وشاشةٌ تفتح على
     * فراغٍ تُقرأ عطباً.**
     */
    fun loadClients() {
        if (clients.isNotEmpty() || busy) return
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                clients = rep.merchants()
            } catch (e: Exception) {
                error = err(e)
            } finally {
                busy = false
            }
        }
    }

    /** **يُفتح في سياق متجرٍ بعينه** — **ولا يُفتح بلا متجر.** */
    fun open(id: String, name: String) {
        merchantID = id
        merchantName = name
        rows = null
        error = ""
        pickedSection = ""
        pickedItem = ""
        load()
    }

    /**
     * **ويُعاد اختيارُ المتجر بلا خروجٍ من الشاشة** — **ومن أراد متجراً
     * آخرَ لا يُطالَب بالرجوع خطوتين.**
     */
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

    /** **والقسمُ يُختار فيُنسى الصنفُ** — **وصنفٌ من قسمٍ آخرَ اختيارٌ قديم.** */
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
                val secs = rep.menu(merchantID)
                sections = secs
                items = secs.flatMap { it.items }
                // **وقسمٌ واحدٌ لا يُسأل عنه** — **وخطوةٌ جوابُها واحدٌ
                // زحمةٌ لا اختيار.**
                if (secs.size == 1) pickedSection = secs.first().id
            } catch (e: Exception) {
                error = err(e)
            } finally {
                busy = false
            }
        }
    }

    /**
     * **طريقةُ الخصم** — نسبةٌ أو مبلغٌ ثابت (قرارُ المالك ٢٠٢٦-٠٩-٣٠).
     *
     * **وواحدةٌ لا اثنتان** — يحرسه قيدُ القاعدة `offers_one_discount_kind`،
     * **ويقولها الخادمُ `bad_offer_discount` قبل أن تبلغه.**
     */
    var byPercent by mutableStateOf(true)
        private set

    fun pickMode(percent: Boolean) {
        byPercent = percent
    }

    /**
     * create **ينشئ الخصمَ بطريقتِه.**
     *
     * **والقيمةُ واحدةٌ تُقرأ بحسب الطريقة** — **ولا حقلان في الشاشة
     * يملأ المندوبُ أحدَهما وينسى الآخر**، فيُرسَل عرضٌ بنسبةٍ ومبلغٍ
     * معاً ويُردّ.
     */
    fun create(itemId: String, value: Long, hours: Int) {
        if (busy || merchantID.isEmpty()) return
        val okValue = if (byPercent) value in 1..90 else value > 0
        if (itemId.isBlank() || !okValue || !OfferDuration.valid(hours)) {
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
                        // **ويُرسَل أحدُهما لا كلاهما** — والفارغُ يُحذف من
                        // الجسم (`encodeDefaults` مطفأ)، **فالخادمُ يرى حقلاً
                        // واحداً كما يشترط.**
                        discountPercent = if (byPercent) value.toInt() else null,
                        discountAmount = if (byPercent) null else value,
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

package com.rahalgo.customer.orders

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.ComplaintReason
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.MyOrder
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.apiError
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ الزبون — الجاري وحدَه أو المنتهي وحدَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا نموذجٌ واحدٌ لشاشتين
 *
 * **«طلباتي» و«سجلّ الطلبات» بابٌ واحدٌ بشرطٍ مختلف** (`open_only`) —
 * **ونموذجان يعنيان نسختين من ترجمة الحال ومن حسبة المهلة**، تُصلَح
 * إحداهما وتبقى الأخرى.
 *
 * # ولا يُخلط الجاري بالمنتهي
 *
 * **من بحث عن طلبٍ يجري وسط مئةٍ منتهيةٍ لا يجده** — **ومن فتح
 * «طلباتي» فوجده فارغاً ظنّ أنّ طلبَه ضاع.**
 */
class OrdersViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var open by mutableStateOf<List<MyOrder>>(emptyList())
        private set

    var history by mutableStateOf<List<MyOrder>>(emptyList())
        private set

    // ══════════════════════════════════════════════════════════════════
    // **ترقيمُ سجلّ الطلبات** (`CAF-14`، `CUST-14-026`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان يُطلب أوّلُ ثلاثين فقط** — **فمن تجاوز طلباتُه الثلاثين لم
    // يرَ أقدمَها أبداً.** فتُجلب صفحةٌ صفحةً عند بلوغ آخر القائمة،
    // **بلا تكرارِ بطاقةٍ وبلا خلطِ ترتيب** (`mergeById`).
    var historyHasMore by mutableStateOf(false)
        private set

    var loadingMore by mutableStateOf(false)
        private set

    private var loadedCount = 0
    private var lastPage = 1

    /**
     * **من قيّمه من طلباته** — ومنه يُعرف أين يُعرض زرُّ النجوم.
     *
     * **ولا يُقرأ من الطلب نفسه**: المحرّكُ لا يرسل `rated` في القائمة.
     */
    var rated by mutableStateOf<Set<String>>(emptySet())
        private set

    // ══════════════════════════════════════════════════════════════════
    // **وأيُّ طلبٍ يُسأل صاحبُه عن رأيه الآن**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «لا يظهر التقييمُ بشكلٍ تلقائيّ بعد
    //  استلام الطلب، وهيك المفروض».)
    //
    // **والزرُّ وحدَه لا يكفي**: الطلبُ المسلَّمُ ينزل إلى «سجلّ
    // الطلبات» — **فمن أراد أن يقيّم يفتح القائمةَ ثمّ السجلَّ ثمّ
    // يبحث عن طلبه.** ولا يفعلها أحد.
    //
    // **واللحظةُ التي يُقيَّم فيها هي لحظةُ الاستلام** — وبعدها بساعةٍ
    // يكون قد نسي.
    var askRate by mutableStateOf<MyOrder?>(null)
        private set

    /**
     * **ومن أُغلقت نافذتُه لا يُسأل ثانيةً في هذه الجلسة.**
     *
     * **وسؤالٌ يتكرّر كلّما فُتح التطبيقُ عقوبةٌ لا طلبُ رأي** —
     * **والزرُّ يبقى في بطاقته** لمن أراد بعدها.
     */
    private var skipped = setOf<String>()

    fun skipRate() {
        askRate?.let { skipped = skipped + it.id }
        askRate = null
    }

    /**
     * **رايةُ السحب وحدَه** — لا `busy` العامّة.
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «عند سحب الشاشة إلى الأسفل يتحدّث
     *  البرنامج، تحسّباً لأمرِ تحديثٍ لحظيٍّ لم يصل».)
     *
     * **و`busy` تُرفع في كلّ جلبٍ داخليّ** — فلو قادت الدوّارةَ
     * **لَظهرت من تلقائها كلَّما وصلت نبضةُ تحديث.**
     */
    var refreshing by mutableStateOf(false)
        private set

    /** **يُنعش بسحبةٍ** — ويُطفئ رايتَه مهما وقع. */
    fun refresh() {
        if (refreshing) return
        refreshing = true
        viewModelScope.launch {
            runCatching {
                val p = api.orders(openOnly = false, page = 1)
                open = p.orders.filterNot { it.status in ENDED }
                history = p.orders.filter { it.status in ENDED }
                lastPage = 1
                loadedCount = p.orders.size
                historyHasMore = loadedCount < p.total
                error = ""
            }.onFailure { error = apiError(getApplication(), it as Exception) }
            refreshing = false
        }
    }

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    /**
     * **خطأُ فعلٍ لا خطأُ جلب** — ولا تمحوه إعادةُ القراءة.
     *
     * **وقع في تطبيق السائق (٢٠٢٦-٠٨-١٣)**: الإلغاءُ يسقط فيُكتب خطؤه،
     * **ثمّ يُعاد الجلبُ فيمسحه** — فيرى صاحبُه أنّ شيئاً لم يقع.
     */
    var actionError by mutableStateOf("")
        private set

    init {
        load()
        // **وما يتبدّل يُقرأ** — الوصلةُ الحيّةُ تقول «تغيّر شيء».
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        busy = true
        viewModelScope.launch {
            try {
                // ══════════════════════════════════════════════════════
                // **والحالُ تحكم لا العمودُ وحدَه**
                // ══════════════════════════════════════════════════════
                //
                // **`open_only` في المحرّك يقرأ `closed_at IS NULL`** —
                // **وصفوفٌ أُغلقت قبل أن يُكتب هذا العمودُ تُقرأ
                // جارية.** (شكوى المالك ٢٠٢٦-٠٨-١٥: رأى «تعذّر» في
                // «طلباتي».)
                //
                // **والنهاياتُ خمسٌ تعرفها الحالُ نفسُها** — فلا يرى
                // الزبونُ طلباً منتهياً بين الجارية، **ولا يُنتظر
                // ترحيلُ بيانات.**
                val p = api.orders(openOnly = false, page = 1)
                val all = p.orders
                open = all.filterNot { it.status in ENDED }
                history = all.filter { it.status in ENDED }
                lastPage = 1
                loadedCount = all.size
                historyHasMore = loadedCount < p.total
                // **ومن قيّم لا يُسأل ثانية** — نداءٌ ثانٍ كما في الويب.
                // **وسقوطُه لا يُسقط الطلبات**: أسوأُ ما يقع أن يظهر
                // زرُّ نجومٍ يردّ «قيّمتَه من قبل».
                rated = runCatching {
                    api.ratings().ratings.filter { it.rated }.map { it.orderId }.toSet()
                }.getOrDefault(rated)

                // **والأحدثُ أوّلا** — من سُلّم له طلبان يُسأل عن
                // آخرِهما: **هو ما يذكره.**
                askRate = history.firstOrNull {
                    it.status == "delivered" && it.id !in rated && it.id !in skipped
                }
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    fun clearAction() {
        actionError = ""
    }

    /**
     * loadMore **يجلب الصفحةَ التاليةَ من السجلّ** (`CAF-14`، `CUST-14-026`).
     *
     * **ولا يُعاد نداءٌ في الجوّ** (`loadingMore`)، **ولا يُطلب ما لا مزيدَ
     * بعده** (`historyHasMore`). **والدمجُ يمنع التكرار ويحفظ الترتيب**
     * (`mergeById`) — **فسحبةٌ زائدةٌ لا تكرّر بطاقةً ولا تقلب صفّا.**
     */
    fun loadMore() {
        if (loadingMore || !historyHasMore || busy) return
        loadingMore = true
        viewModelScope.launch {
            try {
                val next = lastPage + 1
                val p = api.orders(openOnly = false, page = next)
                open = mergeById(open, p.orders.filterNot { it.status in ENDED })
                history = mergeById(history, p.orders.filter { it.status in ENDED })
                lastPage = next
                loadedCount += p.orders.size
                // **وصفحةٌ فارغةٌ تُنهي الترقيم** — **ولا حلقةَ لا تقف.**
                historyHasMore = p.orders.isNotEmpty() && loadedCount < p.total
            } catch (e: Exception) {
                // **وسقوطُ صفحةٍ إضافيّةٍ لا يمسح ما عُرض** — يبقى ما حُمّل.
                actionError = apiError(getApplication(), e)
            }
            loadingMore = false
        }
    }

    /**
     * **يُلغي طلباً في مهلته — بلا سبب.**
     *
     * **والمهلةُ من الخادم** (`cancel_seconds_left`) — **وحسبةٌ في
     * الشاشة تخالفه بعد أوّل تعديلِ إعداد.**
     */
    fun cancel(orderId: String) {
        if (busy) return
        busy = true
        actionError = ""
        viewModelScope.launch {
            try {
                api.cancel(orderId)
            } catch (e: Exception) {
                actionError = apiError(getApplication(), e)
            }
            busy = false
            load()
        }
    }

    fun rate(orderId: String, stars: Int, driverStars: Int?) {
        if (busy) return
        busy = true
        actionError = ""
        viewModelScope.launch {
            try {
                api.rate(orderId, stars, driverStars)
            } catch (e: Exception) {
                actionError = apiError(getApplication(), e)
            }
            busy = false
            load()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الشكوى — أسبابٌ مصنّفةٌ لا نصٌّ حرّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ومن السبب يُشتقّ الذنبُ والتعويض** — ونصٌّ حرٌّ يُقرأ بيدٍ
    // بشريّةٍ ثمّ يُصنَّف بالظنّ.

    var reasons by mutableStateOf<List<ComplaintReason>>(emptyList())
        private set

    fun loadReasons(orderId: String) {
        viewModelScope.launch {
            reasons = runCatching { api.complaintReasons(orderId).reasons }.getOrDefault(emptyList())
        }
    }

    fun complain(orderId: String, reason: String, note: String) {
        if (busy) return
        busy = true
        actionError = ""
        viewModelScope.launch {
            try {
                api.complain(orderId, reason, note)
            } catch (e: Exception) {
                actionError = apiError(getApplication(), e)
            }
            busy = false
            load()
        }
    }
}

/**
 * **نهاياتُ الطلب** — كما يعرفها المحرّك (`orders/statuses.go: terminal`).
 *
 * **ونسختان تفترقان يومَ تُزاد حال** — فتظهر في الجاري وفي السجلّ معا،
 * **أو لا تظهر في أيّهما.**
 */
val ENDED = setOf("delivered", "cancelled", "failed", "rejected", "refunded")

/**
 * mergeById **يضمّ صفحةً جديدةً إلى ما عُرض — بلا تكرارٍ وبحفظ الترتيب** —
 * `CAF-14` · `CUST-14-026`.
 *
 * **دالّةٌ صافيةٌ تُقاس بلا جهاز**: **الموجودُ أوّلاً كما هو، ثمّ الجديدُ
 * الذي لم يُرَ بعدُ بترتيبه.** **وطلبٌ يصل في صفحتين (سباقُ إنعاش) لا
 * يُعرض بطاقتين.**
 */
fun mergeById(existing: List<MyOrder>, incoming: List<MyOrder>): List<MyOrder> {
    if (incoming.isEmpty()) return existing
    val seen = existing.mapTo(HashSet()) { it.id }
    return existing + incoming.filter { seen.add(it.id) }
}

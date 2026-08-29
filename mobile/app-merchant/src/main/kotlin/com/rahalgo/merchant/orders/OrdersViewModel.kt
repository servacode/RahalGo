package com.rahalgo.merchant.orders

import com.rahalgo.merchant.noStoreMsg
import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.MerchantOrder
import com.rahalgo.shared.merchant.Store
import com.rahalgo.ui.err
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ المتجر — والحلقةُ التي يعيشها صاحبُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣: «بناء تطبيق المتجر بشكل كامل ومتكامل».)
 *
 * # ولماذا تُنعش دوريّاً
 *
 * **الطلبُ له مهلةُ قبول** — وانقضاؤها رفضٌ يُحسب مخالفةً على المتجر.
 * **وشاشةٌ لا تُنعش تُري طلباً مضت مهلتُه وكأنّه حيّ**، فيضغط «اقبل»
 * فيُردّ عليه بخطأٍ لا يفهمه.
 *
 * **والوصلةُ الحيّةُ تسبقها** (`LiveSocket`) — والدورةُ احتياطٌ لها:
 * **مقبسٌ ينقطع في شبكةٍ ضعيفةٍ ولا يقول إنّه انقطع.**
 *
 * # ولا تُنعش وهو يقرأ طلباً
 *
 * **من فتح تفصيلَ طلبٍ ثمّ اختفى تحت يده** ظنّ التطبيقَ عطبان —
 * فالإنعاشُ يتوقّف ما دامت نافذةُ التفصيل مفتوحة.
 */
class OrdersViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var store by mutableStateOf<Store?>(null)
        private set

    var orders by mutableStateOf<List<MerchantOrder>>(emptyList())
        private set

    /**
     * **أيدير المتجرُ طلباتِه بنفسه؟** — `platform.orders_mode`.
     *
     * **يأتي مع المتاجر في النداء نفسِه** (`self_manage_orders`) —
     * **والجوابُ مع السؤال لا نداءٌ ثانٍ لسطرٍ واحد.**
     *
     * **وافتراضُه `true` حتّى يُقرأ** — **ولو بدأ `false` لَاختفى تبويبُ
     * الطلبات لحظةً عند كلّ إقلاعٍ ثمّ ظهر**، وشريطٌ يرقص يُقرأ عطباً.
     */
    var selfManage by mutableStateOf(lastKnownSelfManage())
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    /** **يتوقّف الإنعاشُ حين يقرأ** — انظر أعلاه. */
    var reading by mutableStateOf(false)

    private var busyIds by mutableStateOf<Set<String>>(emptySet())

    fun busy(id: String) = id in busyIds

    init {
        viewModelScope.launch {
            load(first = true)
            while (true) {
                delay(REFRESH_MS)
                if (!reading) load(first = false)
            }
        }
    }

    /**
     * **يقرأ متجرَه ثمّ طلباتِه.**
     *
     * **ومتجرٌ واحدٌ في الغالب** — ولو ملك اثنين قُرئ الأوّل، **وشاشةُ
     * اختيار الفرع تُبنى يوم يوجد فرعان** لا قبله.
     */
    private suspend fun load(first: Boolean) {
        runCatching {
            val page = api.stores()
            selfManage = page.selfManageOrders
            rememberSelfManage(page.selfManageOrders)
            val mine = store ?: page.stores.firstOrNull()
            if (mine == null) {
                error = noStoreMsg()
                loading = false
                return
            }
            store = mine
            orders = api.orders(mine.id).orders
            error = ""
        }.onFailure {
            // **وإخفاقُ إنعاشٍ لا يمسح ما على الشاشة** — **قائمةٌ تختفي
            // لانقطاعِ ثانيةٍ تُقرأ ضياعَ طلبات.**
            if (first) error = err(it)
        }
        loading = false
    }

    fun refresh() = viewModelScope.launch { load(first = false) }

    // ══════════════════════════════════════════════════════════════════
    // **الأفعالُ الثلاثة — والمحرّكُ يحرسها**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا تحرسها الشاشةُ ثانيةً**: `statuses.go` يعرف ما يجوز من كلّ
    // حال، **وحارسان يفترقان يوماً فيُمنع ما يجوز أو يُقبل ما لا يجوز.**

    fun accept(id: String) = act(id) { api.transition(id, "accepted") }

    fun startPreparing(id: String) = act(id) { api.transition(id, "preparing") }

    fun markReady(id: String) = act(id) { api.markReady(id) }

    fun reject(id: String, note: String) = act(id) { api.transition(id, "rejected", note) }

    /**
     * **يمنع الضغطةَ الثانية** — **وضغطتان على «اقبل» تُرسلان نداءين**،
     * والثاني يُردّ بخطأٍ يقرؤه صاحبُ المتجر عطباً.
     */
    private fun act(id: String, block: suspend () -> Unit) {
        if (busy(id)) return
        busyIds = busyIds + id
        viewModelScope.launch {
            runCatching { block() }
                .onSuccess { load(first = false) }
                .onFailure { Flash.fail(err(it)) }
            busyIds = busyIds - id
        }
    }

    private companion object {
        /**
         * **عشرُ ثوانٍ.**
         *
         * **ومهلةُ قبول الطلب دقائقُ لا ثوانٍ** — فدورةٌ أسرعُ لا تفيد
         * وتستنزف حزمةَ من يعمل على بياناتٍ محدودة. **وأبطأُ منها يجعل
         * الطلبَ يظهر متأخّراً** وقد ضاع من مهلته دقيقة.
         */
        const val REFRESH_MS = 10_000L
    }
}


// ══════════════════════════════════════════════════════════════════════
// **وضعُ الطلبات يُحفظ — وإلّا ومض التبويبُ في كلّ فتحة**
// ══════════════════════════════════════════════════════════════════════
//
// **كان يبدأ `true` دائماً** — فتبويبُ «طلباتي» يُرسم قبل أن يردّ
// المحرّك. **وفي وضع المنصّة يظهر ثمّ يختفي بعد جزءٍ من ثانية**، ومن
// رآه ظنّ التطبيقَ يرتجف.
//
// **وأسوأُ منه بلا شبكة**: النداءُ يفشل فيبقى التبويبُ ظاهراً على شاشةٍ
// فارغةٍ لا طلباتِ فيها ولا تفسير.
//
// **والقيمةُ لا تتبدّل إلّا بقرارٍ من المالك في اللوحة** — فآخرُ ما
// عرفناه صحيحٌ حتّى يُقال غيرُه، **وأوّلُ تشغيلٍ وحدَه يبدأ بلا تبويب:
// إخفاءٌ يظهر أهونُ من ظهورٍ يُخفى.**
private const val SELF_MANAGE_KEY = "merchant_self_manage"

private fun prefs() = com.rahalgo.ui.AppCore.get().app
    .getSharedPreferences("rahalgo", android.content.Context.MODE_PRIVATE)

private fun lastKnownSelfManage(): Boolean =
    runCatching { prefs().getBoolean(SELF_MANAGE_KEY, false) }.getOrDefault(false)

private fun rememberSelfManage(value: Boolean) {
    runCatching { prefs().edit().putBoolean(SELF_MANAGE_KEY, value).apply() }
}

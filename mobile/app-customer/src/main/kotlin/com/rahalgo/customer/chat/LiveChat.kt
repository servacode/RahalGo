package com.rahalgo.customer.chat

import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import android.content.Context
import android.media.RingtoneManager
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.driver.ChatApi
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديثُ الطلب الجاري — يُعرَف في كلّ شاشةٍ لا في شاشةِ الطلبات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٩: «زرُّ الدردشة يجب أن يكون عائماً فوق كلّ
 *
 *  الصفحات والدردشةُ بشكلٍ عائمٍ أيضاً — مو معقول إلّا يفوت على
 *  الطلبات مشان يشوف الدردشة».)
 *
 * # ولماذا نموذجٌ خاصٌّ لا `OrdersViewModel`
 *
 * **نموذجُ الطلبات يعيش مع شاشتِه** — ومن لم يفتح «طلباتي» لا يُقرأ له
 * شيء. **وقرصٌ يظهر بعد أن تزور الشاشةَ التي جاء ليُغنيك عنها لا معنى
 * له.**
 *
 * **فيُقرأ عند الإقلاع** — ثمّ عند كلّ إنعاش.
 *
 * # ولا يظهر بلا مُحدَّث
 *
 * **قرصُ حديثٍ بلا سائقٍ يُضغط فيُفتح فراغ.** فالشرطُ واحدٌ أينما وُضع:
 * **طلبٌ جارٍ أُسنِد له سائق.**
 *
 * # والصمتُ لا يُبتلع
 *
 * **نداءٌ يسقط فلا يظهر القرص** — ولا أحدَ يعلم لماذا. **فيُسجَّل**،
 * وهي عائلةُ «العطبُ يُعرض صمتاً» التي تكرّرت في هذا المستودع.
 */
class LiveChatViewModel : ViewModel() {

    /**
     * **أحاديثُ الزبون الجارية** — **وكلُّ واحدٍ منها طلبٌ قائمٌ بذاته.**
     *
     * **وكان يُحفَظ معرّفٌ واحدٌ** (`orders.firstOrNull { له سائق }`) —
     * **فمن له طلبان جاريان يفتح قرصُه أحدَهما ولا يقول أيَّهما**:
     * **يكتب ردَّه في الطلب الخطأ ولا يعلم** (`CU-CHAT-02`).
     */
    var chats by mutableStateOf<List<Live>>(emptyList())
        private set

    /** **طلبٌ جارٍ يُحادَث فيه** — ورقمُه لأنّ المعرّفَ لا يُقرأ. */
    data class Live(val orderId: String, val unread: Int)

    /**
     * **ما يفعله القرصُ حين يُضغط.**
     *
     * **ولا يُختار عن الزبون ما لا يعلمه**: **واحدٌ يُفتَح، وأكثرُ من
     * واحدٍ يُعرَض ليختار** (`CU-CHAT-03`).
     */
    val single: String? get() = if (chats.size == 1) chats[0].orderId else null

    /**
     * **أيُّ حديثٍ مفتوحٌ الآن** — **والحالُ هنا لا في شاشةٍ تُغادَر.**
     *
     * **وكان علماً نعم/لا** — **ولوحٌ واحدٌ لا يُسأل عن أيّ طلبٍ يعرض
     * يعرض آخرَ ما حُمِّل فيه** (`CU-CHAT-07`).
     */
    var openId by mutableStateOf<String?>(null)

    /** **أمفتوحٌ لوحٌ؟** — **وبه تُكتم الرنّةُ عمّن يقرأ الآن.** */
    val open: Boolean get() = openId != null

    /**
     * **كم رسالةً تنتظره** — للشارة على القرص.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٩: «وقت تجي رسالة لازم لها صوتٌ وعدّاد».)
     *
     * **والعدَدُ من المحرّك لا يُحسب هنا**: `‎/my/chats` يردّ `unread`
     * لكلّ خيط، **وعدٌّ في العميل يفترق عن عدِّ الخادم يومَ تُقرأ رسالةٌ
     * من جهازٍ آخر.**
     */
    var unread by mutableStateOf(0)
        private set

    /**
     * **وما تقوله الشارةُ هو ما يفتحه القرص.**
     *
     * **ومجموعُ حديثين على زرٍّ يفتح أحدَهما كذبٌ**: **يرى «٣» فيفتح
     * فيجد واحدةً، والاثنتان في طلبٍ لا يعلم به** (`CU-CHAT-04`).
     */
    val badge: Int
        get() = if (chats.size == 1) chats[0].unread else unread

    private val api = CustomerApi(AppCore.get().api)
    private val chatApi = ChatApi(AppCore.get().api)

    /** **آخرُ عددٍ رُئي** — به يُعرف الجديدُ من القديم. */
    private var lastSeen = -1

    /**
     * **يقرأ الطلباتِ الجاريةَ وعدّادَها في نداءٍ واحد.**
     *
     * **والطلبُ هو الأصلُ لا الحديث**: **قائمةُ الأحاديث تبدأ من
     * الرسائل** — **فمن أُسنِد له سائقٌ ولم تُكتب رسالةٌ بعدُ لا خيطَ
     * له**، ولو اتُّخذت أصلاً لما وجد الزبونُ باباً ليبدأ.
     *
     * **والعدّادُ من الأحاديث** — لكلّ طلبٍ عددُه هو.
     */
    fun load(context: Context? = null) {
        viewModelScope.launch {
            val counts = runCatching { chatApi.threads().threads }
                .onFailure { Log.w("RahalGo/chat", "تعذّرت قراءةُ عدّاد الرسائل", it) }
                .getOrNull()
            runCatching { api.orders(openOnly = true).orders }
                .onSuccess { orders ->
                    val live = orders.filter { !it.driverName.isNullOrEmpty() }
                    val byOrder = counts.orEmpty().associate { it.orderId to it.unread }
                    this@LiveChatViewModel.chats =
                        live.map { Live(it.id, byOrder[it.id] ?: 0) }
                    Log.i("RahalGo/chat", "طلبات=" + orders.size +
                        " أحاديثُ جارية=" + live.size)
                }
                .onFailure {
                    Log.w("RahalGo/chat", "تعذّرت قراءةُ الطلبات لقرص الحديث", it)
                    this@LiveChatViewModel.chats = emptyList()
                }
            if (counts != null && context != null) {
                val n = counts.filter { it.open }.sumOf { it.unread }
                if (lastSeen >= 0 && n > lastSeen && !open) ring(context)
                lastSeen = n
                unread = n
            }
        }
    }


    /**
     * **رنّةُ الإشعار التي اختارها صاحبُ الجهاز** — لا نغمةٌ نحقنها.
     *
     * **ومن أطفأ صوتَ هاتفِه لا يُوقَظ**: `RingtoneManager` يتبع وضعَ
     * النظام، **ونغمةٌ تُشغَّل بمشغّلٍ خاصٍّ تتخطّى الصامت.**
     */
    private fun ring(context: Context) {
        runCatching {
            val uri = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION)
            RingtoneManager.getRingtone(context.applicationContext, uri)?.play()
        }.onFailure { Log.w("RahalGo/chat", "تعذّرت الرنّة", it) }
    }

    /** **ويُطفأ عند الخروج** — وإلّا بقي قرصُ حسابٍ مضى فوق شاشةِ ضيف. */
    fun clear() {
        chats = emptyList()
        openId = null
        unread = 0
        lastSeen = -1
    }
}

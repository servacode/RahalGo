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

    /** **معرّفُ الطلب الذي يُفتح حديثُه** — وفارغٌ يعني «لا قرص». */
    var orderId by mutableStateOf<String?>(null)
        private set

    /** **أمفتوحٌ اللوح؟** — والحالُ هنا لا في شاشةٍ تُغادَر. */
    var open by mutableStateOf(false)

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

    private val api = CustomerApi(AppCore.get().api)
    private val chats = ChatApi(AppCore.get().api)

    /** **آخرُ عددٍ رُئي** — به يُعرف الجديدُ من القديم. */
    private var lastSeen = -1

    fun load(context: Context? = null) {
        context?.let { refreshUnread(it) }
        viewModelScope.launch {
            runCatching { api.orders(openOnly = true).orders }
                .onSuccess { orders ->
                    // **وأوّلُ جارٍ له سائق** — **وقرصٌ واحدٌ لطلبين
                    // يفتح أحدَهما**، والجاري هو ما يُنتظر فيه ردّ.
                    orderId = orders.firstOrNull { !it.driverName.isNullOrEmpty() }?.id
                    Log.i("RahalGo/chat", "طلبات=" + orders.size +
                        " بسائق=" + orders.count { !it.driverName.isNullOrEmpty() } +
                        " المختار=" + orderId)
                }
                .onFailure {
                    Log.w("RahalGo/chat", "تعذّرت قراءةُ الطلبات لقرص الحديث", it)
                    orderId = null
                }
        }
    }

    /**
     * **يقرأ عددَ ما ينتظره — ويُسمِع رنّةً للجديد.**
     *
     * # ولماذا لا يرنّ في كلّ قراءة
     *
     * **النبضةُ تصل مع كلّ تغيّرٍ في الطلب** — قبولٍ وإسنادٍ وتسليم.
     * **ورنّةٌ مع كلّ نبضةٍ تُقرأ عطلاً في التطبيق** فيُطفئ صاحبُها
     * الصوتَ كلَّه.
     *
     * **فيرنّ حين يزيد العدد وحدَه** — ولا يرنّ حين ينقص (قرأها)،
     * **ولا في أوّل قراءةٍ بعد الإقلاع** (`lastSeen < 0`): من فتح
     * تطبيقَه على ثلاث رسائلَ قديمةٍ لا يريد ثلاثَ رنّات.
     *
     * **ولا يرنّ واللوحُ مفتوح** — هو يقرؤها الآن.
     */
    private fun refreshUnread(context: Context) {
        viewModelScope.launch {
            runCatching { chats.threads().threads }
                .onSuccess { rows ->
                    val n = rows.filter { it.open }.sumOf { it.unread }
                    if (lastSeen >= 0 && n > lastSeen && !open) ring(context)
                    lastSeen = n
                    unread = n
                }
                .onFailure { Log.w("RahalGo/chat", "تعذّرت قراءةُ عدّاد الرسائل", it) }
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
        orderId = null
        open = false
        unread = 0
        lastSeen = -1
    }
}

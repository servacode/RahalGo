package com.rahalgo.customer.chat

import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
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

    private val api = CustomerApi(AppCore.get().api)

    fun load() {
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

    /** **ويُطفأ عند الخروج** — وإلّا بقي قرصُ حسابٍ مضى فوق شاشةِ ضيف. */
    fun clear() {
        orderId = null
        open = false
    }
}

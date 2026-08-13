package com.rahalgo.driver.home

import android.app.Application
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.driver.location.LocationPermission
import com.rahalgo.driver.location.LocationService
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.Notice
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حال اللوحة — والشاشة لا تملكه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ونداء واحد يملأ الشاشة كلّها** (`driver/me`): الوردية والمال واليوم.
 * **ولو قُسّم على ثلاثة نداءات** لظهرت الشاشة أثلاثا، **وكلّ ثلث قد
 * يسقط وحده.**
 */
class HomeViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(HomeState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        refresh()
        // ══════════════════════════════════════════════════════════════
        // **وتسمع نبضةَ التحديث فتُنعش نفسَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٣.)
        //
        // **رصيدُه وتقييمُه وورديّتُه وصورتُه كلُّها هنا** — والشريطُ
        // العلويُّ يقرأ منها. **فكانت تبقى كما فُتحت** حتّى يخرج من
        // التبويب ويعود.
        //
        // **و`drop(1)` تتخطّى القيمةَ الحاليّة** — وإلّا جلبت مرّتين في
        // أوّل فتحة.
        viewModelScope.launch {
            Refresh.tick.drop(1).collect { refresh() }
        }
    }

    fun refresh() {
        if (state.busy) return
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            state = try {
                val me = backend.driver.me()
                syncService(me)
                refreshUnread()
                state.copy(me = me, busy = false, locationOn = hasLocation())
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    /** يُنادى بعد رجوع صاحبه من نافذة الإذن أو من الإعدادات. */
    fun recheckLocation() {
        state = state.copy(locationOn = hasLocation())
        state.me?.let { syncService(it) }
    }

    private fun hasLocation(): Boolean =
        LocationPermission.granted(getApplication())

    /**
     * ══════════════════════════════════════════════════════════════════
     * **الخدمة تتبع الوردية — لا زرّا منفصلا**
     * ══════════════════════════════════════════════════════════════════
     *
     * **ومن جعلها زرّا ثانيا** ترك سائقا ورديّته مفتوحة وموقعه مطفأ،
     * **فيظهر للمكتب واقفا وهو يسير**، ولا يصله أقرب طلب.
     *
     * **وحال الوردية يُقرأ من المحرّك لا من الجهاز**: قد يُغلقها المكتب
     * عنه، **فتُطفأ الخدمة عند أوّل قراءة** بلا أن يفعل شيئا.
     */
    private fun syncService(me: DriverMe) {
        val app = getApplication<Application>()
        if (me.onShift && LocationPermission.granted(app)) {
            LocationService.start(app, me.locationPingSec.takeIf { it > 0 } ?: 20L)
        } else {
            LocationService.stop(app)
        }
    }

    /**
     * **يفتح الوردية أو يغلقها — والجواب من المحرّك لا من الزرّ.**
     *
     * **ولا تُبدَّل الحالة قبل الردّ**: المحرّك قد يرفض الفتح (نقد فوق
     * السقف مثلا)، **ومن بدّل الزرّ متفائلا** أرى صاحبه «متاح» وهو ليس
     * كذلك — فينتظر طلبا لا يأتي، ولا شيء يقول له لماذا.
     */
    fun toggleShift(on: Boolean) {
        if (state.busy) return
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.setShift(on)
                // **ثمّ يُعاد قراءة الحال كاملا** — فتح الوردية يحرّك
                // أرقاما أخرى، **ومن بدّل الراية وحدها** أبقى بقيّة
                // الشاشة على حال ما قبل الضغطة.
                val me = backend.driver.me()
                syncService(me)
                state = state.copy(me = me, busy = false)
            } catch (e: Exception) {
                state = state.copy(busy = false, error = describe(e))
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الشريط العلويّ — رصيدُه وإشعاراته**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وهنا لا في نموذج ثالث**: يُقرآن مع كلّ تحديث للوحة، **ونموذج
    // رابع لسطرين** يفتح نداءين ويعيش دورة حياة أخرى.
    //
    // ══════════════════════════════════════════════════════════════════
    // **والرصيد من `driver/me` لا من نداءِ محفظةٍ ثانٍ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان يُقرأ بنداءٍ ثانٍ لا يُنادى قطّ**: كُتبت `refreshTop()` ولم
    // تُوصل بـ`refresh()`، **فبقي الشريط يقول «٠ ل.س»** لسائقٍ في
    // محفظته مال. (أمسكه المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **و`driver/me` يحمل الرصيد أصلا** — ومعه التقييم. **فنداءٌ ثانٍ
    // لرقمٍ وصل** فرصةُ خلافٍ بين رقمين لا فائدةَ فيها.
    var unread by mutableStateOf(0)
        private set

    var inbox by mutableStateOf<List<Notice>?>(null)
        private set

    /** يقرأ عدد غير المقروء — **وفشله لا يُسقط اللوحة.** */
    private fun refreshUnread() {
        viewModelScope.launch {
            runCatching { backend.me.inbox(1).unread }.onSuccess { unread = it }
        }
    }

    fun openInbox() {
        viewModelScope.launch {
            inbox = runCatching { backend.me.inbox().items }.getOrDefault(emptyList())
        }
    }

    /**
     * **يُعلّم الكلَّ مقروءا** — بطلبه هو.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «لازم في تعليم الكلّ كمقروء صحّ».)
     *
     * **وكان يُعلَّم بمجرّد الفتح** — فمن فتح الصندوق ليرى العدد وخرج
     * **ضاع عنه ما لم يقرأه.** والشارةُ تقول «فيه جديد»، **ومن مسحها
     * قبل أن يُقرأ** جعلها لا تعني شيئا.
     *
     * **والمعرّفُ الفارغ يعني الكلَّ** في المحرّك — وهو ما ترسله صفحةُ
     * الويب أيضا.
     */
    fun markAllRead() {
        viewModelScope.launch {
            runCatching { backend.me.markRead() }.onSuccess {
                unread = 0
                inbox = inbox?.map { if (it.read) it else it.copy(read = true) }
            }
        }
    }

    fun closeInbox() {
        inbox = null
    }

    fun logout() {
        // **والخدمة تقف مع الخروج** — إشعار وردية لحساب خرج **يبقى في
        // الشريط وموقعه يُرسل**، وهو ما لا يقبله أحد.
        LocationService.stop(getApplication())
        val refresh = backend.session.refreshToken()
        backend.session.clear()
        viewModelScope.launch { runCatching { backend.auth.logout(refresh) } }
    }

    /**
     * **عطب لا يُعرف لا يُسمّى «لا اتصال»** — القاعدة نفسها في الدخول:
     * الشبكة تُسمّى شبكة وحدها، **وما عداها يُسمّى باسمه ويُكتب في
     * السجلّ.**
     */
    private fun describe(e: Exception): String {
        Log.e("RahalGo/home", "فشل نداء اللوحة", e)
        val app = getApplication<Application>()
        return when {
            e is ApiClient.ApiException -> when (e.body.code) {
                "unauthorized", "invalid_refresh" -> app.getString(R.string.err_invalid_refresh)
                "user_blocked" -> app.getString(R.string.err_user_blocked)
                "user_suspended" -> app.getString(R.string.err_user_suspended)
                "forbidden" -> app.getString(R.string.err_not_driver)
                // **ظهر خاما في التجربة ٢٠٢٦-٠٨-١٢** — والمفتاح بلا
                // ترجمة يُعرض ليُعرف، لا ليُبتلع.
                "has_active_orders" -> app.getString(R.string.err_has_active_orders)
                "" -> app.getString(R.string.err_internal)
                else -> e.body.code
            }

            e is IOException || e is HttpRequestTimeoutException ->
                app.getString(R.string.err_network)

            else -> app.getString(R.string.err_unexpected) + " (" + e.javaClass.simpleName + ")"
        }
    }
}

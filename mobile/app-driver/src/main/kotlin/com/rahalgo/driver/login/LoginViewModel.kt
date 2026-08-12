package com.rahalgo.driver.login

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.shared.model.User
import com.rahalgo.shared.net.ApiClient
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالة الدخول — والشاشة لا تملكها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وتبقى عبر دوران الجهاز** (`ViewModel`): من كتب رقمه ودار جهازه
 * **لا يعيد كتابته**، ونداء يمشي لا يبدأ من جديد فيفتح جلستين.
 */
class LoginViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(LoginState())
        private set

    /** من دخل — **فارغ يعني لم يدخل بعد.** */
    var user by mutableStateOf<User?>(null)
        private set

    /** أيُفحص التوكن المحفوظ الآن؟ — **تُعرض شاشة انتظار لا شاشة دخول.** */
    var restoring by mutableStateOf(true)
        private set

    private val backend = Backend.of(getApplication())

    init {
        restore()
    }

    /**
     * **استعادة الجلسة عند الإقلاع.**
     *
     * **ونجاح `me` هو الدليل لا وجود التوكن**: توكن أُبطل من الإدارة يبقى
     * مكتوبا في الجهاز، **ومن اكتفى بوجوده أرى صاحبه شاشة داخل ثمّ سقط
     * كل نداء بعدها.**
     */
    private fun restore() {
        if (backend.session.refreshToken().isEmpty()) {
            restoring = false
            return
        }
        viewModelScope.launch {
            user = runCatching { backend.auth.me() }.getOrNull()
            if (user == null) backend.session.clear()
            restoring = false
        }
    }

    fun login(phone: String, password: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                val result = backend.auth.login(phone, password)
                // **ورمز الأدمن الرباعي يصل هنا أيضا**: صاحب المنصّة قد
                // يفتح تطبيق السائق بحسابه. **ولو أُهمل لرأى شاشة دخول
                // ناجحة بلا توكن**، ثمّ سقط كل نداء.
                if (result.pinRequired) {
                    state = state.copy(busy = false, error = str(R.string.err_internal))
                    return@launch
                }
                backend.session.save(result.tokens.accessToken, result.tokens.refreshToken)
                user = result.user
                state = state.copy(busy = false)
            } catch (e: ApiClient.ApiException) {
                // **وسبب الخادم كما قاله** — لا رسالة عامّة تُسكته.
                state = state.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                state = state.copy(busy = false, error = str(R.string.err_network))
            }
        }
    }

    fun logout() {
        val refresh = backend.session.refreshToken()
        backend.session.clear()
        user = null
        // **والإبطال في المحرّك بعد المحلّي** — مسح الجهاز يقع مهما كانت
        // الشبكة، **وجلسة تبقى مفتوحة في الخادم أهون من زر خروج لا يعمل.**
        viewModelScope.launch { runCatching { backend.auth.logout(refresh) } }
    }

    fun clearError() {
        state = state.copy(error = "")
    }

    private fun str(id: Int): String = getApplication<Application>().getString(id)

    /**
     * **مفتاح الخادم إلى نص** — لا رسالة واحدة لكل العلل.
     *
     * **ومفتاح لا ترجمة له يُعرض رمزه** — فيُعرف ويُضاف، **بدل أن يُبتلع
     * في «حدث خطأ» فيبقى مجهولا.**
     */
    private fun message(e: ApiClient.ApiException): String = when (e.body.code) {
        "unauthorized", "invalid_credentials" -> str(R.string.err_unauthorized)
        "validation" -> str(R.string.err_validation)
        "user_blocked" -> str(R.string.err_user_blocked)
        "user_suspended" -> str(R.string.err_user_suspended)
        "rate_limited", "too_many_requests" -> str(R.string.err_rate_limited)
        "" -> str(R.string.err_internal)
        else -> e.body.code
    }
}

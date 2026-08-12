package com.rahalgo.driver.login

import android.app.Application
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.push.Push
import com.rahalgo.shared.model.User
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
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

    /**
     * **جلسة محفوظة لم يُمكن التحقّق منها** — شبكة لا هويّة.
     *
     * **وتُعرض شاشة «تعذّر الاتّصال» لا شاشة دخول**: من له حساب لا
     * يُطلب منه أن يدخل من جديد لأنّ الشبكة غابت لحظة.
     */
    var offline by mutableStateOf(false)
        private set

    private val backend = Backend.of(getApplication())

    init {
        restore()
        loadPlatform()
    }

    /**
     * **ما تقوله المنصّة عن أبوابها.**
     *
     * **ولا يُنتظر جوابها قبل رسم الشاشة**: خادم نائم في الخطّة المجّانية
     * يستيقظ في نصف دقيقة، **وشاشة بيضاء نصف دقيقة تُقرأ عطبا.**
     * **فتُرسم الشاشة بلا تبويب، ويُضاف إن قال المحرّك إنّ الباب
     * مفتوح** — والعكس (يُعرض ثمّ يُخفى) وميض يُقرأ عطبا.
     */
    private fun loadPlatform() {
        viewModelScope.launch {
            val platform = try {
                backend.auth.platform()
            } catch (e: Exception) {
                Log.w("RahalGo/login", "تعذّرت قراءة حال المنصّة", e)
                return@launch
            }
            state = state.copy(otpAvailable = platform.otpLogin)
        }
    }

    /** يبدّل الباب — **ويمسح ما خلّفه الباب الآخر.** */
    fun setMode(mode: LoginMode) {
        if (state.busy || state.mode == mode) return
        // **ولا يبقى أثر محاولة سابقة في وضع جديد** — رمز أُرسل ثمّ عاد
        // صاحبه إلى كلمة المرور، **فلو بقيت رايته مرفوعة** لرأى حقل رمز
        // في باب لا رمز فيه.
        state = LoginState(mode = mode, otpAvailable = state.otpAvailable)
    }

    /** يعيد الشاشة إلى الرقم — **لمن كتب رقما غلطا.** */
    fun clearCode() {
        state = state.copy(codeSent = false, error = "")
    }

    /**
     * **يطلب رمز دخول.**
     *
     * **وهذا غير رمز الاستعادة**: هذا يفتح جلسة مباشرة، **وذاك يضبط كلمة
     * مرور جديدة.** والمحرّك يفصلهما بمفتاحين (`otp:` و`otp:rst:`)،
     * **فرمز أحدهما لا يعمل في الآخر.**
     */
    fun sendLoginCode(phone: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            state = try {
                backend.auth.requestOtp(phone)
                state.copy(codeSent = true, busy = false)
            } catch (e: ApiClient.ApiException) {
                state.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    fun verifyLoginCode(phone: String, code: String) {
        if (state.busy) return
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                val result = backend.auth.verifyOtp(phone, code)
                if (result.pinRequired) {
                    state = state.copy(busy = false, error = str(R.string.err_internal))
                    return@launch
                }
                backend.session.save(result.tokens.accessToken, result.tokens.refreshToken)
                user = result.user
                state = state.copy(busy = false)
            } catch (e: ApiClient.ApiException) {
                state = state.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                state = state.copy(busy = false, error = describe(e))
            }
        }
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
            try {
                user = backend.auth.me()
                // **والتسجيل بعد ثبوت الجلسة** — لا قبلها: النقطة
                // تحتاج توكن حساب.
                Push.register(getApplication())
            } catch (e: ApiClient.ApiException) {
                // ══════════════════════════════════════════════════════
                // **لا يُمحى الحساب إلّا إذا رفضه المحرّك**
                // ══════════════════════════════════════════════════════
                //
                // **وقع ٢٠٢٦-٠٨-١٢**: أُقلع الجهاز والخادم بطيء، **فسقط
                // النداء فمُحيت الجلسة** — ورجعت شاشة الدخول لحساب
                // صحيح لم يُمسّ.
                //
                // **وهذا في الشارع أسوأ**: سائق في حيّ بلا تغطية يفتح
                // تطبيقه **فيُطرد ويُطلب منه رقمه وكلمته وهو على
                // الدرّاجة.**
                //
                // **فالطرد لا يقع إلّا بجواب المحرّك**: توكن مرفوض أو
                // جلسة أُبطلت. **وما عداه انقطاع يمرّ.**
                if (e.status == 401 || e.body.code == "invalid_refresh") {
                    Log.w("RahalGo/login", "الجلسة مرفوضة — تُمحى", e)
                    backend.session.clear()
                } else {
                    Log.w("RahalGo/login", "المحرّك ردّ بخطأ — الجلسة تبقى", e)
                    offline = true
                }
            } catch (e: Exception) {
                // **انقطاع شبكة — الجلسة تبقى كما هي.**
                Log.w("RahalGo/login", "تعذّر الاتّصال — الجلسة تبقى", e)
                offline = true
            }
            restoring = false
        }
    }

    /** يعيد محاولة الاستعادة — **بعد أن تعود الشبكة.** */
    fun retryRestore() {
        offline = false
        restoring = true
        restore()
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
                Push.register(getApplication())
            } catch (e: ApiClient.ApiException) {
                // **وسبب الخادم كما قاله** — لا رسالة عامّة تُسكته.
                state = state.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                state = state.copy(busy = false, error = describe(e))
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الاستعادة — حالة منفصلة، والحساب واحد**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولماذا هنا لا في نموذج مستقلّ**: التأكيد يفتح جلسة، **والجلسة
    // يملكها هذا النموذج وحده.** ونموذجان يكتبان `user` بابُ حالتين
    // متناقضتين: **شاشة تقول داخل وأخرى تقول خارج.**
    //
    // **وفارغ يعني ليس في الاستعادة** — لا راية ثانية تُنسى فتتناقض
    // مع الخطوة.
    var reset by mutableStateOf<ResetState?>(null)
        private set

    fun openReset() {
        reset = ResetState()
    }

    fun closeReset() {
        reset = null
    }

    fun setResetPhone(phone: String) {
        reset = reset?.copy(phone = phone, error = "")
    }

    /**
     * **يطلب الرمز — وينتقل مهما كان الجواب.**
     *
     * **ورقم غير مسجّل يردّ نجاحا صامتا** (`RequestPasswordReset`: «رقم
     * غير مسجّل: صمت مقصود») — **لئلّا يُعرف من هذا الباب أيّ الأرقام لها
     * حسابات.** فلا يُقال هنا «الرقم غير موجود»، **وإلّا نُقض صمتُ
     * المحرّك من الواجهة.**
     */
    fun sendResetCode() {
        val current = reset ?: return
        if (current.busy) return
        reset = current.copy(busy = true, error = "")
        viewModelScope.launch {
            reset = try {
                backend.auth.resetRequest(current.phone.trim())
                current.copy(step = ResetStep.CODE, busy = false)
            } catch (e: ApiClient.ApiException) {
                current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                current.copy(busy = false, error = describe(e))
            }
        }
    }

    fun verifyResetCode(code: String) {
        val current = reset ?: return
        if (current.busy) return
        reset = current.copy(busy = true, error = "")
        viewModelScope.launch {
            reset = try {
                backend.auth.resetVerify(current.phone.trim(), code)
                // **والرمز يُحمل إلى الخطوة الأخيرة** — المحرّك يطلبه
                // مرّة ثانية مع الكلمة الجديدة، **ولا يستهلكه هنا.**
                current.copy(step = ResetStep.PASSWORD, code = code, busy = false)
            } catch (e: ApiClient.ApiException) {
                current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                current.copy(busy = false, error = describe(e))
            }
        }
    }

    /** **ينتهي داخلا** — المحرّك يفتح الجلسة مع التأكيد. */
    fun confirmReset(password: String) {
        val current = reset ?: return
        if (current.busy) return
        reset = current.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                val result = backend.auth.resetConfirm(
                    current.phone.trim(),
                    current.code,
                    password,
                )
                backend.session.save(result.tokens.accessToken, result.tokens.refreshToken)
                user = result.user
                reset = null
            } catch (e: ApiClient.ApiException) {
                reset = current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                reset = current.copy(busy = false, error = describe(e))
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

    /**
     * ══════════════════════════════════════════════════════════════════
     * **عطب لا يُعرف لا يُسمّى «لا اتصال»**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وقع ٢٠٢٦-٠٨-١١**: دخول المالك بحسابه الصحيح أظهر «لا اتصال
     * بالإنترنت»، **والخادم كان صاحيا يردّ في ثمانية أعشار الثانية.**
     * **ولم يظهر السبب في السجلّ** لأن `catch` عامّا كتب فوقه رسالة
     * واحدة.
     *
     * **فالشبكة تُسمّى شبكة وحدها**: انقطاع أو مهلة. **وما عداها يُسمّى
     * باسمه ويُكتب في السجلّ** — رسالة مبهمة تُخفي العطب أسوأ من رسالة
     * قبيحة تكشفه.
     */
    private fun describe(e: Exception): String {
        Log.e("RahalGo/login", "فشل الدخول", e)
        return when (e) {
            is IOException, is HttpRequestTimeoutException -> str(R.string.err_network)
            else -> str(R.string.err_unexpected) + " (" + e.javaClass.simpleName + ")"
        }
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
        // **ورموز الاستعادة** — ظهرت خاما على الشاشة في أوّل تجربة حيّة
        // (٢٠٢٦-٠٨-١١: `invalid_otp`)، **وهذا ما أراده التصميم**: مفتاح
        // بلا ترجمة يُعرض ليُعرف ويُضاف، لا يُبتلع في «حدث خطأ».
        "invalid_phone" -> str(R.string.err_invalid_phone)
        "invalid_otp" -> str(R.string.err_invalid_otp)
        "otp_send_failed" -> str(R.string.err_otp_send_failed)
        "too_many_attempts" -> str(R.string.err_too_many_attempts)
        "weak_password" -> str(R.string.err_weak_password)
        "invalid_refresh" -> str(R.string.err_invalid_refresh)
        "" -> str(R.string.err_internal)
        else -> e.body.code
    }
}

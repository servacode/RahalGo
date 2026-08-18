package com.rahalgo.ui

import android.app.Application
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
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
class AuthViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(LoginState())
        private set

    /**
     * من دخل — **فارغ يعني لم يدخل بعد.**
     *
     * **ومن دخل يُعرَّف لتقارير الانهيار** — بمعرّفه لا باسمه ولا رقمه:
     * **وتقريرٌ بلا صاحبٍ لا يُتابَع**، تعرف أنّ التطبيقَ سقط ثلاثاً
     * **ولا تعرف أثلاثةُ سائقين أم واحدٌ ثلاث مرّات.**
     */
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

    private val backend = AppCore.get()

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
                onSignedIn(result.user)
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
                user?.let { Crash.who(it.id) }
                // **والتسجيل بعد ثبوت الجلسة** — لا قبلها: النقطة
                // تحتاج توكن حساب.
                AppCore.afterSignIn()
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
                    // **وطردُ الجلسة يُبلَّغ عنه** — وقع اليومَ على جهاز
                    // المالك (٢٠٢٦-٠٨-١٣) **ولم يُعرف سببُه من السجلّ.**
                    //
                    // **وسائقٌ يُطرد وهو على الدرّاجة لا يُبلّغ**: يدخل
                    // من جديدٍ ويمضي، **فلا يعلم أحدٌ أنّها تتكرّر.**
                    Crash.soft(e, "session rejected " + e.status + " " + e.body.code)
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
                onSignedIn(result.user)
                state = state.copy(busy = false)
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
                onSignedIn(result.user)
                reset = null
            } catch (e: ApiClient.ApiException) {
                reset = current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                reset = current.copy(busy = false, error = describe(e))
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **إنشاء الحساب — حالة منفصلة، والحساب واحد**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولماذا هنا لا في نموذجٍ مستقلّ**: التأكيدُ يفتح جلسة، **والجلسةُ
    // يملكها هذا النموذجُ وحدَه.** ونموذجان يكتبان `user` بابُ حالتين
    // متناقضتين: **شاشةٌ تقول داخل وأخرى تقول خارج.**
    //
    // **وفارغٌ يعني ليس في التسجيل** — لا رايةَ ثانيةً تُنسى فتتناقض مع
    // الخطوة.
    var signup by mutableStateOf<SignupState?>(null)
        private set

    /** @param referral رمزُ من دعاه — **من الرابط لا من يده.** */
    fun openSignup(referral: String = "") {
        signup = SignupState(referral = referral)
    }

    fun closeSignup() {
        signup = null
    }

    fun setSignupPhone(phone: String) {
        signup = signup?.copy(phone = phone, error = "")
    }

    /**
     * **يطلب رمزَ تسجيل.**
     *
     * **ورقمٌ له حسابٌ يردّ خطأً صريحاً** — بخلاف الاستعادة: هناك يُصمت
     * لئلّا يُعرف أيُّ الأرقام لها حسابات، **وهنا لا بدّ أن يُقال** وإلّا
     * انتظر رمزاً لا يأتي.
     */
    fun sendSignupCode() {
        val current = signup ?: return
        if (current.busy) return
        signup = current.copy(busy = true, error = "")
        viewModelScope.launch {
            signup = try {
                backend.auth.signupRequest(current.phone.trim())
                current.copy(step = SignupStep.CODE, busy = false)
            } catch (e: ApiClient.ApiException) {
                current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                current.copy(busy = false, error = describe(e))
            }
        }
    }

    fun verifySignupCode(code: String) {
        val current = signup ?: return
        if (current.busy) return
        signup = current.copy(busy = true, error = "")
        viewModelScope.launch {
            signup = try {
                backend.auth.signupVerify(current.phone.trim(), code)
                // **والرمزُ يُحمل إلى الخطوة الأخيرة** — المحرّكُ يطلبه
                // مرّةً ثانيةً مع البيانات، **ولا يستهلكه هنا.**
                current.copy(step = SignupStep.DETAILS, code = code, busy = false)
            } catch (e: ApiClient.ApiException) {
                current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                current.copy(busy = false, error = describe(e))
            }
        }
    }

    /** **ينتهي داخلا** — المحرّك يفتح الجلسة مع الإنشاء. */
    fun confirmSignup(name: String, password: String) {
        val current = signup ?: return
        if (current.busy) return
        signup = current.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                val result = backend.auth.signupConfirm(
                    current.phone.trim(),
                    current.code,
                    name,
                    password,
                    current.referral,
                )
                backend.session.save(result.tokens.accessToken, result.tokens.refreshToken)
                onSignedIn(result.user)
                signup = null
            } catch (e: ApiClient.ApiException) {
                signup = current.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                signup = current.copy(busy = false, error = describe(e))
            }
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ما يقع عند كلّ دخولٍ — في موضعٍ واحد**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٨: «يقول هذا يحتاج حساباً ادخل أوّلاً بالرغم
     *  من أنّني مسجّلُ دخولٍ بالفعل — يجب أن يكون التطبيقُ ذكيّاً فيستطيع
     *  التمييزَ بين الحساب والزائر».)
     *
     * # ما كان يقع
     *
     * **أربعةُ أبوابٍ للدخول** — كلمةُ مرور، ورمز، واستعادة، وتسجيل —
     * **وكلٌّ يكتب ما يخصّه بيده.** فوقع ما يقع دائماً في نسخٍ أربع:
     * `afterSignIn` تُنادى في اثنين وتُنسى في اثنين.
     *
     * # ولماذا كانت الشاشةُ تقول «ادخل أوّلا» لمن دخل
     *
     * **`ApiClient` يرفض النداءَ إن كان المخزنُ بلا توكن** — يمنع رحلةَ
     * شبكةٍ تُردّ ٤٠١، **ويرمي `not_signed_in` من الجهاز لا من الخادم.**
     *
     * **ونموذجٌ حمّل وصاحبُه ضيفٌ يحتفظ بنصّ الخطأ في حاله** — والدخولُ
     * لا يمحوه: **لا نبضةَ تُبَثّ ولا شاشةَ تُعيد جلبَها.** فتبقى الجملةُ
     * معروضةً وصاحبُها داخلٌ فعلاً.
     *
     * **و`Refresh.bump()` موجودةٌ منذ ٢٠٢٦-٠٨-١٣** — تبثّها المحفظةُ
     * والحسابُ وما بدّله صاحبُه، **ولا يبثّها الدخولُ نفسُه**، وهو
     * أعظمُ ما يتبدّل: **كلُّ شاشةٍ في التطبيق تعني شيئاً آخرَ بعده.**
     */
    private fun onSignedIn(u: User?) {
        user = u
        u?.let { Crash.who(it.id) }
        // **والنبضةُ قبل النداء الخاصّ** — فما يُسجَّل بعدها يجد شاشاتٍ
        // أعادت جلبَها.
        Refresh.bump()
        AppCore.afterSignIn()
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
    /**
     * **الرمزُ بعربيّة** — من الخريطة المركزيّة (`data/ApiErrors.kt`).
     *
     * **وكانت هنا خريطةٌ ثانيةٌ بنصوصٍ ثانيةٍ للمعنى نفسِه**: «رقم
     * الهاتف غير صحيح» هنا و«رقم غير صحيح» في الحساب، **ونصّان لمعنًى
     * واحدٍ يفترقان يومَ يُصحَّح أحدُهما.**
     */
    private fun message(e: ApiClient.ApiException): String =
        apiError(getApplication(), e)
}

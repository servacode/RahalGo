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

    // ══════════════════════════════════════════════════════════════════
    // **وحالُ الحساب ليست انقطاعاً** (`CUST-DEF-007`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **رفضُ الشبكة ≠ رفضُ الجلسة ≠ تقييدُ الحساب.** كان الـ403 «حسابٌ
    // مقيَّد» (`forbidden`) يسقط في `else` فيُعرَض «لا يوجد اتصال» —
    // **فمن عُلّق حسابُه يفحص الواي فاي.** **والجلسةُ سليمةٌ (ليست 401)**،
    // فلا تُمسح ولا تُمسّ عزلةُ `CUST-DEF-009`: **الهويّةُ باقيةٌ، والشاشةُ
    // تقول الحالَ لا الكذب.**
    var accountRestricted by mutableStateOf(false)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **وتبديلُ كلمةٍ مطلوبٌ عقدٌ مستقلّ** (`CUST-DEF-010`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **403 `password_change_required` غيرُ `forbidden`**: ذاك «لا تملك
    // هذا»، وهذا «بدّلْ كلمتَك أوّلاً». **ولا يُعرَض انقطاعاً ولا تقييداً**
    // — بل يُساق صاحبُه إلى شاشة التبديل.
    var mustChangePassword by mutableStateOf(false)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **وبوّابةُ التحديث حالةٌ مُراقَبةٌ لا رايةٌ عاديّة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **`ApiClient.outdated` متغيّرٌ عاديٌّ في رفيقٍ ساكن** — و`AppFrame`
    // يقرؤه في سطرٍ قبل كلّ شيء، **وتعليقُه يقول «تُقرأ مع كلّ إعادة
    // رسمٍ فلا تحتاج مراقباً».**
    //
    // **وهو افتراضٌ خاطئٌ عن Compose**: النطاقُ يُعاد بناؤه حين تتبدّل
    // **حالةٌ مُراقَبةٌ قرأها هو** — ومتغيّرٌ عاديٌّ ليس منها. **فيتبدّل
    // `offline` فيُعاد بناءُ `when` الداخليّ وحدَه، ولا يُعاد تقييمُ
    // السطر الخارجيّ أبداً.**
    //
    // # وأثرُه في الميدان
    //
    // **المحرّكُ يردّ ٤٢٦ صحيحاً** (`update_required`)، **والتطبيقُ يعرض
    // «لا يوجد اتصال بالإنترنت».** فيفحص المندوبُ الواي فاي والبيانات
    // ويعيد المحاولة، **ولا يعرف أنّ عليه تحديثَ تطبيقه.**
    //
    // **وشاشةٌ تكذب في سببِ التعطّل أسوأُ من شاشةٍ لا تظهر**: توجّه
    // صاحبَها إلى إصلاح ما ليس معطوباً.
    //
    // (كشفه اختبارُ الميدان ٢٠٢٦-٠٨-٣١ برفع `app.min_version.rep` لحظةً.)
    var updateRequired by mutableStateOf(false)
        private set

    private val backend = AppCore.get()

    init {
        // **ويُوصَل النداءُ مرّةً عند بناء النموذج** — فأيُّ نداءٍ في
        // التطبيق كلِّه يردّ ٤٢٦ يُوقظ إعادةَ الرسم، **لا الاستعادةُ
        // وحدَها.** (كان إصلاحي الأوّلُ يغطّي مساراً واحداً، **فظهرت
        // الرسالةُ في شاشة الدخول ولم تُغلق البوّابة.**)
        if (ApiClient.outdated) updateRequired = true
        ApiClient.onOutdated = { updateRequired = true }
        // **وتبديلُ الكلمةِ المطلوبُ من أيّ بابٍ مُقيَّد** (`CUST-DEF-010`) —
        // المحرّكُ يُخفي العلَمَ في `/auth/me` حين الميزةُ مطفأةٌ، لكنّ
        // البابَ المقيَّدَ يردّ `403 password_change_required`. **فيُساق
        // صاحبُه إلى شاشة التبديل مهما دخل** — لا نصَّ خطإٍ عابراً.
        ApiClient.onPasswordChangeRequired = { mustChangePassword = true }
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
            // **ورايةُ التسجيل تُحفظ هنا لا في `SignupState`** — تُقرأ
            // مرّةً عند فتح الشاشة، **وشاشةُ الإنشاء تُفتح بعدها.**
            signupNeedsCode = platform.signupVerify
            // **والشاشةُ تعرف لتسمّي زرَّها** — انظر `SignupState.needsCode`.
            signup = signup?.copy(needsCode = platform.signupVerify)
        }
    }

    /**
     * **أيُطلب رمزٌ عند إنشاء الحساب؟** — من المنصّة لا من الشيفرة.
     *
     * (قرارُ المالك ٢٠٢٦-٠٢٥: التسجيلُ بلا رمز، والتوثيقُ عند أوّل طلب.)
     *
     * **وافتراضُه `false`** — فمن لم يبلغه ردُّ المنصّة يمرّ بلا رمز
     * بدل أن يُحبس على شاشةٍ لا يصلها شيء.
     */
    private var signupNeedsCode: Boolean = false

    /**
     * **يفتح واتساب على رقم المنصّة** — وفشلُه لا يُسقط الخطوة.
     *
     * **ومن لم يُفتح عنده يستطيع أن يراسل بيده** — فحقلُ الرمز يبقى
     * ينتظره، **ولا يُغلق البابُ على من ليس عنده التطبيق.**
     */
    private fun openWhatsApp(url: String) {
        if (url.isBlank()) return
        runCatching {
            getApplication<Application>().startActivity(
                android.content.Intent(
                    android.content.Intent.ACTION_VIEW,
                    android.net.Uri.parse(url),
                ).addFlags(android.content.Intent.FLAG_ACTIVITY_NEW_TASK),
            )
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
                backend.session.save(
                    result.tokens.accessToken,
                    result.tokens.refreshToken,
                    result.tokens.accessExpiresAtMs(),
                )
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
                // **وتبديلُ الكلمةِ المطلوبُ يُساق إلى شاشته** (`CUST-DEF-010`) —
                // **المحرّكُ يسمح بـ`/auth/me` لمن يبدّل**، فيصل الحقلُ `true`؛
                // ولا يدخل التطبيقَ ولا يُسجَّل قبل التبديل. **ولا يُبطَل بالرجوع
                // أو موتِ العمليّة**: كلُّ إقلاعٍ يُعيد كشفَه من `me()`.
                if (user?.mustChangePassword == true) {
                    mustChangePassword = true
                } else {
                    // **والتسجيل بعد ثبوت الجلسة** — لا قبلها: النقطة
                    // تحتاج توكن حساب.
                    AppCore.afterSignIn()
                }
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
                if (sessionRejected(e.status, e.body.code)) {
                    Log.w("RahalGo/login", "الجلسة مرفوضة — تُمحى", e)
                    // **وطردُ الجلسة يُبلَّغ عنه** — وقع اليومَ على جهاز
                    // المالك (٢٠٢٦-٠٨-١٣) **ولم يُعرف سببُه من السجلّ.**
                    //
                    // **وسائقٌ يُطرد وهو على الدرّاجة لا يُبلّغ**: يدخل
                    // من جديدٍ ويمضي، **فلا يعلم أحدٌ أنّها تتكرّر.**
                    Crash.soft(e, "session rejected " + e.status + " " + e.body.code)
                    // **حدُّ الجلسة الواحد** (`CUST-DEF-009`) — رفضُ المحرّك يمسح
                    // المحلّيَّ كلَّه كالخروج الصريح (السلّةُ والوصلةُ والحال)،
                    // **فلا يرث الحسابُ التالي على الجهاز نفسِه سلّةَ من قبله.**
                    detachSession()
                } else if (passwordChangeRequired(e.status, e.body.code)) {
                    // **تبديلُ كلمةٍ مطلوبٌ عقدٌ مستقلّ** (`CUST-DEF-010`) —
                    // يُساق صاحبُه إلى شاشة التبديل، **لا انقطاعٌ ولا تقييد**.
                    // **والجلسةُ سليمةٌ فلا تُمسح**: الشاشةُ تحتاجها لتبدّل.
                    Log.w("RahalGo/login", "تبديلُ كلمةٍ مطلوب — شاشة التبديل", e)
                    mustChangePassword = true
                } else if (accountForbidden(e.status, e.body.code)) {
                    // **حسابٌ مقيَّدٌ لا انقطاع** (`CUST-DEF-007`) — رسالةُ حالِ
                    // حسابٍ لا «لا يوجد اتصال». **والجلسةُ سليمةٌ (403 لا 401)
                    // فلا تُمسح** — عزلةُ `CUST-DEF-009` باقيةٌ كما هي.
                    Log.w("RahalGo/login", "الحساب مقيَّد — حالُ حسابٍ لا انقطاع", e)
                    accountRestricted = true
                } else if (e.status == 426) {
                    // **ونسخةٌ شاخت ليست انقطاعاً** — تُرفع رايتُها
                    // المُراقَبة فتُغلق الأبوابُ في إعادة الرسم التالية.
                    Log.w("RahalGo/login", "النسخة قديمة — بوّابة التحديث", e)
                    updateRequired = true
                } else {
                    // **وما بقي انقطاعٌ أو تعذّرُ خدمة** (شبكة · 5xx ·
                    // `auth_unavailable` 503) — **الجلسةُ تبقى، ويُعاد.**
                    Log.w("RahalGo/login", "تعذّرٌ عابرٌ — الجلسة تبقى", e)
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
        // **وكلُّ حالِ استعادةٍ يُعاد تقييمُها** — فلا رايةٌ شائخةٌ تبقى
        // بعد أن تبدّل جوابُ المحرّك (`CUST-DEF-007`/`010`).
        offline = false
        accountRestricted = false
        mustChangePassword = false
        restoring = true
        restore()
    }

    // ══════════════════════════════════════════════════════════════════
    // **تبديلُ الكلمةِ المطلوبُ — تدفّقٌ لا رسالةُ خطأ** (`CUST-DEF-010`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان العطبُ**: المحرّكُ يطلب تبديلاً (`password_change_required` /
    // `must_change_password`) **والعميلُ يعرض نصَّ خطإٍ لا تدفّقاً**
    // (`ApiErrors.kt`). **فلا مخرجَ**: أيُّ بابٍ يُردّ، ولا شاشةَ تبديل.
    /** حالُ شاشةِ التبديل المطلوب — **تقرؤها الشاشةُ لترسم الزرَّ والخطأ.** */
    data class PwChange(val busy: Boolean = false, val error: String = "")

    var pwChange by mutableStateOf(PwChange())
        private set

    fun clearPwChangeError() {
        if (pwChange.error.isNotEmpty()) pwChange = pwChange.copy(error = "")
    }

    /**
     * **يبدّل الكلمةَ المؤقّتة عبر النقطة الموثوقة** — `POST /api/v1/auth/password`
     * (`current_password` + `password`). **والجلسةُ الحاليّةُ تبقى والبواقي
     * تُقطَع** (`SetPasswordKeeping`, `XG-40`). **ولا يُكشَف القديمُ** —
     * يكتبه صاحبُه. **وعند النجاح يُرفع القيدُ ويدخل التطبيقَ بالجلسة نفسِها**
     * (لا تسرّبَ محلّيَّ لغيره — عزلةُ `CUST-DEF-004`/`009` قائمة).
     */
    fun submitForcedPasswordChange(current: String, next: String) {
        if (pwChange.busy) return
        pwChange = pwChange.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.account.setPassword(current, next)
                mustChangePassword = false
                pwChange = PwChange()
                Refresh.bump()
                AppCore.afterSignIn()
            } catch (e: ApiClient.ApiException) {
                pwChange = pwChange.copy(busy = false, error = message(e))
            } catch (e: Exception) {
                pwChange = pwChange.copy(busy = false, error = describe(e))
            }
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
                backend.session.save(
                    result.tokens.accessToken,
                    result.tokens.refreshToken,
                    result.tokens.accessExpiresAtMs(),
                )
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
    /**
     * ══════════════════════════════════════════════════════════════════
     * **الاستعادةُ مقلوبةٌ أيضاً — يفتح واتساب ويرسل هو**
     * ══════════════════════════════════════════════════════════════════
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٥: «واستعادة كلمة المرور أيضاً هو يرسل».)
     *
     * **وكانت تُرسل رسالةٌ إليه** — وواتساب قيّد رقمَ المنصّة مرّتين
     * في يومٍ واحد لأنّها مراسلةٌ ابتدائيّة. **والردُّ مسموح.**
     *
     * **والخطوةُ التالية هي هي**: يكتب الرمزَ الذي وصله، ثمّ الكلمةَ
     * الجديدة. **فلا شيءَ تغيّر عنده إلّا من يبدأ.**
     */
    fun sendResetCode() {
        val current = reset ?: return
        if (current.busy) return
        reset = current.copy(busy = true, error = "")
        viewModelScope.launch {
            reset = try {
                val t = backend.auth.waTicket(current.phone.trim(), "reset")
                openWhatsApp(t.waUrl)
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
                backend.session.save(
                    result.tokens.accessToken,
                    result.tokens.refreshToken,
                    result.tokens.accessExpiresAtMs(),
                )
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
        signup = SignupState(referral = referral, needsCode = signupNeedsCode)
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
        // ══════════════════════════════════════════════════════════════
        // **ولا رمزَ إن أطفأته المنصّة — يمضي إلى البيانات مباشرة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٢٥ بعد أن قُيّد رقمُ واتساب المنصّة
        //  مرّتين في يوم.)
        //
        // **ولا يُنادى المحرّكُ أصلاً**: نداءٌ يطلب رمزاً لن يُرسل
        // **يستهلك حصّةَ الرقم ويسجّل خطأً في السجلّ بلا سبب.**
        if (!signupNeedsCode) {
            signup = current.copy(step = SignupStep.DETAILS, code = "", error = "")
            return
        }
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
    fun confirmSignup(name: String, password: String, referral: String = "") {
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
                    // **وما كُتب في الحقل يغلب ما جاء في الرابط** —
                    // **ومن صحّح رمزاً خاطئاً بيده يجب أن يُؤخَذ
                    // تصحيحُه**، وإلّا أُرسل القديمُ وهو يرى الجديد.
                    referral.ifBlank { current.referral },
                )
                backend.session.save(
                    result.tokens.accessToken,
                    result.tokens.refreshToken,
                    result.tokens.accessExpiresAtMs(),
                )
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
        // **وتبديلُ الكلمةِ المطلوبُ يُساق إلى شاشته أوّلاً** (`CUST-DEF-010`):
        // **قبل أيّ دخولٍ للتطبيق وقبل خطّاف الدخول** — فلا يعمل حتّى يبدّل.
        // (الغلافُ يرسم شاشةَ التبديل قبل فرع المستخدم.)
        if (u?.mustChangePassword == true) {
            mustChangePassword = true
            return
        }
        // **والنبضةُ قبل النداء الخاصّ** — فما يُسجَّل بعدها يجد شاشاتٍ
        // أعادت جلبَها.
        Refresh.bump()
        AppCore.afterSignIn()
    }

    /**
     * ══════════════════════════════════════════════════════════════════════
     * **حدُّ الجلسة الواحد — كلُّ انتقالٍ من هويّةٍ صحيحةٍ إلى لا-جلسة يمرّ به**
     * ══════════════════════════════════════════════════════════════════════
     *
     * **CUST-DEF-009**: كان مسحُ الجلسة بعقدين لا عقدٍ واحد — الخروجُ الصريحُ
     * يمسح التوكنَ **والسلّةَ والوصلةَ** (`afterLogout`، إصلاحُ `CUST-DEF-004`)،
     * ورفضُ المحرّك يمسح التوكنَ **وحدَه** (`session.clear()`). **فمن أُبطلت
     * جلستُه بلا خروجٍ صريح — رفضٌ أو إبطالٌ أو حذفُ حسابٍ أو توكنٌ منتهٍ —
     * بقيت سلّتُه على الجهاز، فورثها الحسابُ التالي.**
     *
     * **فصار الحدُّ واحدا يمرّ به الجميع**: يمسح التوكنَ، والهويّةَ المحلّيّةَ
     * (`user=null` تُسقط الغلافَ إلى الضيف فيُنعَش `Shell` عبر `MainActivity`)،
     * **والسلّةَ والوصلةَ الحيّةَ والحالَ المشتقّةَ** (`afterLogout`). **ولا
     * يُنثَر `Cart.clear()` في الفروع** — من زاد فرعا نسيَه، والحدُّ لا يُنسى.
     */
    private fun detachSession() {
        backend.session.clear()
        user = null
        AppCore.afterLogout()
    }

    fun logout() {
        val refresh = backend.session.refreshToken()
        // **وحدُّ الجلسة الواحد** (`CUST-DEF-004` + `CUST-DEF-009`) — يقع قبل
        // نداء الشبكة فلا يتأخّر بتعذّرها.
        detachSession()
        // **والإبطال في المحرّك بعد المحلّي** — مسح الجهاز يقع مهما كانت
        // الشبكة، **وجلسة تبقى مفتوحة في الخادم أهون من زر خروج لا يعمل.**
        // **ورمزُ الجهاز يُقرأ ثمّ يُرسَل مع الخروج** — `D12`:
        // **فيُنهي المحرّكُ وجهةَ الدفع مع الجلسة.**
        viewModelScope.launch {
            val device = Push.currentToken()
            runCatching { backend.auth.logout(refresh, device) }
        }
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متى تُمحى الجلسة — وحدَه** — `R16` · `XG-45`
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الطردُ لا يقع إلّا بجواب المحرّك**: **رمزٌ مرفوض** (٤٠١) **أو
 * توكنُ تجديدٍ مُبطَل.** **وما عداه انقطاعٌ يمرّ.**
 *
 * **و`503 auth_unavailable` ليست طرداً**: **المحرّكُ قال «تعذّر
 * التحقّق» لا «رمزُك مُبطَل»** — **وطردُ صاحبِ جلسةٍ سليمةٍ يطلب منه
 * دخولاً جديداً لن ينفعه.**
 *
 * **وصارت دالّةً صافيةً ليقيسها فحصٌ بلا جهاز** — **فشرطٌ في جسد
 * دالّةٍ طويلةٍ لا يُحرَس.**
 */
fun sessionRejected(status: Int, code: String): Boolean =
    status == 401 || code == "invalid_refresh"

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابٌ مقيَّدٌ — لا انقطاعٌ ولا جلسةٌ مرفوضة** — `CUST-DEF-007`
 * ══════════════════════════════════════════════════════════════════════
 *
 * **٤٠٣ `forbidden`**: المحرّكُ يقول «حسابُك مقيَّدٌ» (مُعلَّقٌ أو محظور)
 * **والجلسةُ سليمةٌ** — **فلا يُمحى شيءٌ ولا يُعرَض «لا اتصال».** **ودالّةٌ
 * صافيةٌ ليقيسها فحصٌ بلا جهاز** — فشرطٌ في جسد استعادةٍ طويلةٍ لا يُحرَس.
 */
fun accountForbidden(status: Int, code: String): Boolean =
    status == 403 && code == "forbidden"

/**
 * **تبديلُ كلمةٍ مطلوبٌ عقدٌ مستقلّ** — `CUST-DEF-010`.
 *
 * **٤٠٣ `password_change_required`**: **غيرُ `forbidden`** — ذاك «لا تملك
 * هذا»، وهذا «بدّلْ كلمتَك أوّلاً». **يُساق صاحبُه إلى شاشة التبديل.**
 */
fun passwordChangeRequired(status: Int, code: String): Boolean =
    status == 403 && code == "password_change_required"

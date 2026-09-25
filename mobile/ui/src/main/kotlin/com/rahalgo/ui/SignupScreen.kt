package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إنشاء حساب — ثلاث خطوات، وللزبون وحدَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * رقم ← رمز ← اسمٌ وكلمة. **وينتهي داخلا** لا عائداً إلى شاشة الدخول:
 * المحرّك يفتح الجلسة مع التأكيد، **فلا يُطلب منه أن يكتب ما ضبطه قبل
 * ثانية.**
 *
 * # ولماذا الرمزُ في خطوةٍ وحده
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تظهر المعلوماتُ إلّا بعد التحقّق من
 *  الرمز».)
 *
 * **والمحرّكُ يتحقّق منه ولا يستهلكه** (`signup/verify`) — **فلو جُمع مع
 * الاسم وكلمة المرور في شاشةٍ واحدة** لكتبها كلَّها ثمّ قيل له «الرمزُ
 * خطأ» فيعيد الثلاثة.
 *
 * # ولا يُنشئ هذا البابُ سائقاً ولا متجرا
 *
 * **حساباتُ العاملين من المنصّة** (`auth_handlers.go`: «إنشاء حساب زبون
 * فقط») — **ومن فتح بابَ التسجيل لكلّ دورٍ فتح بابَ من يُعطي نفسَه
 * دورا.**
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشةُ تعرض وترسل** — والنداءاتُ
 * في `AuthViewModel`.
 */
@Composable
fun SignupScreen(state: SignupState, actions: SignupActions) {
    var code by remember { mutableStateOf("") }
    var name by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    /** **تأكيدُ كلمة المرور** — (قرارُ المالك ٢٠٢٦-٠٨-٢٥). */
    var password2 by remember { mutableStateOf("") }

    // **ويبدأ بما جاء من الرابط أو من المتجر** — ومن جاء بلا شيءٍ يكتبه
    // بيده. **ومفتاحُه الرمزُ القادم**: لو وصل بعد أن رُسمت الشاشةُ
    // لَبقي الحقلُ فارغاً وقد صار في الحالة.
    var referral by remember(state.referral) { mutableStateOf(state.referral) }

    // ══════════════════════════════════════════════════════════════════
    // **وعودةُ الاتّصال تمحو رسالةَ الانقطاعِ الحقليّةَ وحدَها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (`CUST-DEF-011`.) **رسالةُ `state.error` حالٌ غيرُ شريطِ الأعلى**:
    // الشريطُ يتبع `Net.online`، وهذه نصٌّ كتبه فشلُ نداءٍ منقطع —
    // **فكانت تبقى معروضةً وقد عاد الاتّصال.** فحين يعود يُبلَّغ النموذجُ
    // فيمحوها (ويُبقي أخطاءَ التحقّق). **والشاشةُ تُبلِّغ لا تقرّر** —
    // القرارُ في `signupConnectivityRestored` (`GROUND-RULES §7.2`).
    val online = Net.online
    LaunchedEffect(online) {
        if (online) actions.onReconnected()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        // **شعارُ الهويّة المشترك** (Batch 5) — مُصغَّرٌ ومسافةٌ أقلُّ فوقَه
        // فلا يزحم حقولَ الإنشاء الكثيرة.
        AuthHeader(logoSize = 100.dp, topSpace = 24.dp, bottomSpace = 10.dp)
        Text(
            text = stringResource(R.string.signup_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            text = stringResource(
                when (state.step) {
                    SignupStep.PHONE -> R.string.signup_hint_phone
                    SignupStep.CODE -> R.string.signup_hint_code
                    SignupStep.DETAILS -> R.string.signup_hint_details
                },
            ),
            color = Rahal.colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))

        when (state.step) {
            // **والرقمُ يُكتب مرّةً واحدة** — ثمّ يُحمل في الحالة إلى
            // الخطوتين التاليتين. **ومن أعاد كتابتَه في كلّ خطوةٍ أخطأ في
            // واحدة** فسقط الرمزُ بلا سببٍ ظاهر.
            SignupStep.PHONE -> PhoneField(
                value = state.phone,
                onChange = actions.setPhone,
                enabled = !state.busy,
            )

            SignupStep.CODE -> CodeField(
                value = code,
                onChange = { code = it },
                enabled = !state.busy,
            )

            SignupStep.DETAILS -> {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text(stringResource(R.string.signup_name)) },
                    singleLine = true,
                    enabled = !state.busy,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(10.dp))
                // **والعينُ هنا أيضا** — كلمةٌ تُكتب مرّةً واحدةً بلا
                // تأكيد، **فمن أخطأ حرفاً ضبط كلمةً لا يعرفها** ثمّ عاد
                // يستعيدها في أوّل يوم.
                PasswordField(
                    value = password,
                    onChange = { password = it },
                    enabled = !state.busy,
                    label = R.string.signup_password,
                )

                // ══════════════════════════════════════════════════════
                // **وتأكيدُ كلمة المرور — حرفٌ واحدٌ يقفل الحساب**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-٢٥: «رقم الهاتف كلمة المرور
                //  وتأكيد كلمة المرور».)
                //
                // **والحقلُ مخفيٌّ بنقاط** — فمن أخطأ حرفاً لا يراه،
                // **فيُنشئ حساباً بكلمةٍ لا يعرفها** ثمّ يعود يستعيدها
                // برسالةٍ إلى واتساب. **والتأكيدُ يمنع ذلك بحقل.**
                Spacer(Modifier.height(10.dp))
                PasswordField(
                    value = password2,
                    onChange = { password2 = it },
                    enabled = !state.busy,
                    label = R.string.signup_password2,
                )
                // **والخطأُ يُقال تحت الحقل لا عند الضغط** — من رآه
                // مبكّراً أصلحه قبل أن يملأ ما بعده.
                if (password2.isNotEmpty() && password2 != password) {
                    Text(
                        text = stringResource(R.string.signup_password_mismatch),
                        color = Rahal.colors.accent,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                // ══════════════════════════════════════════════════════
                // **ورمزُ الدعوة يُرى ويُكتب بيده**
                // ══════════════════════════════════════════════════════
                //
                // (سؤالُ المالك ٢٠٢٦-٠٨-١٨: «لنفرض أنّني أرسلتُ الرابط
                //  لصديقي ولكن لا يوجد تسجيلٌ عبر الويب… كيف نحلّ هذه
                //  المشكلة؟».)
                //
                // **وكان يأتي من الرابط وحدَه** — فمن نزّل التطبيقَ
                // بملفٍّ مباشرٍ أو من غير متجرٍ **لا يملك أيَّ طريقةٍ
                // ينسب بها نفسَه لمن دعاه.** ولا حقلَ ولا خيار.
                //
                // **وهذا الحقلُ هو الأرضيّة**: يعمل بلا ويبٍ ولا متجرٍ
                // ولا رابط — **يقرأ صديقُه الرمزَ ويكتبه.** وما سواه
                // (الرابطُ والمتجر) تسهيلٌ يملؤه سلفاً.
                //
                // **ويُملأ تلقائيّاً إن جاء من رابط** — فلا يُطلب منه
                // ما بيدِ التطبيق أصلا.
                //
                // **واختياريٌّ صراحةً في نصّه**: حقلٌ لا يُعرف أإلزاميٌّ
                // هو **يوقف من لا رمزَ عنده** عند آخر خطوةٍ في التسجيل.
                Spacer(Modifier.height(10.dp))
                OutlinedTextField(
                    value = referral,
                    onValueChange = { referral = it.trim().uppercase() },
                    label = { Text(stringResource(R.string.signup_referral)) },
                    supportingText = {
                        Text(stringResource(R.string.signup_referral_hint))
                    },
                    singleLine = true,
                    enabled = !state.busy,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        if (state.recovery) {
            // ══════════════════════════════════════════════════════════════
            // **رقمٌ لحسابٍ موجود — لا طريقٌ مسدود** (Batch 4، SG1)
            // ══════════════════════════════════════════════════════════════
            //
            // **بدلاً من «الرقم مستعمل» الساكنة**: خياراتُ استعادةٍ صريحة.
            // **ولا دخولٌ أعمى هنا**: الدخولُ التلقائيُّ لا يقع إلّا في سياق
            // ضياعِ ردِّ تسجيلٍ بدأه العميلُ نفسُه (يُعالَج في النموذج).
            Spacer(Modifier.height(14.dp))
            Text(
                text = stringResource(R.string.signup_taken_title),
                color = Rahal.colors.ink,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(16.dp))
            RahalButton(onClick = actions.toLogin, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.signup_recover_login))
            }
            Spacer(Modifier.height(8.dp))
            RahalTextButton(onClick = actions.toOtp) {
                Text(stringResource(R.string.signup_recover_otp))
            }
            Spacer(Modifier.height(4.dp))
            RahalTextButton(onClick = actions.toReset) {
                Text(stringResource(R.string.signup_recover_reset))
            }
        } else {
            if (state.error.isNotEmpty()) {
                Spacer(Modifier.height(14.dp))
                Text(
                    text = state.error,
                    color = Rahal.colors.accent,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            Spacer(Modifier.height(24.dp))
            RahalButton(
                onClick = {
                    when (state.step) {
                        SignupStep.PHONE -> actions.sendCode()
                        SignupStep.CODE -> actions.verifyCode(code.trim())
                        SignupStep.DETAILS -> actions.confirm(name.trim(), password, referral.trim())
                    }
                },
                enabled = !state.busy && when (state.step) {
                    SignupStep.PHONE -> state.phone.isNotBlank()
                    SignupStep.CODE -> code.isNotBlank()
                    // **والزرُّ لا يعمل حتّى تتطابق الكلمتان** — ورسالةُ
                    // خطأٍ بعد الضغط أسوأُ من زرٍّ يقول «لم تكتمل بعد».
                    SignupStep.DETAILS ->
                        name.isNotBlank() && password.isNotBlank() && password2 == password
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.busy) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(
                        stringResource(
                            when (state.step) {
                                // **ويقول ما سيفعل** — انظر `needsCode`:
                                // **«متابعة» حين لا رمزَ يُطلب**، فلا يَعِد
                                // بواتساب لا يُفتح.
                                SignupStep.PHONE ->
                                    if (state.needsCode) R.string.auth_verify_account
                                    else R.string.auth_continue
                                SignupStep.CODE -> R.string.reset_verify
                                SignupStep.DETAILS -> R.string.signup_create
                            },
                        ),
                    )
                }
            }
        }

        Spacer(Modifier.height(6.dp))
        RahalTextButton(onClick = actions.cancel, enabled = !state.busy) {
            Text(stringResource(R.string.reset_back), color = Rahal.colors.inkMuted)
        }
    }
}

/** أين وصل صاحبُه في الخطوات الثلاث. */
enum class SignupStep { PHONE, CODE, DETAILS }

data class SignupState(
    val step: SignupStep = SignupStep.PHONE,
    val phone: String = "",
    /**
     * يُحمل من خطوة التحقّق إلى خطوة البيانات — **المحرّكُ يطلبه مرّتين
     * ولا يستهلكه في الأولى.**
     */
    val code: String = "",
    /**
     * **رمزُ من دعاه** — يأتي من رابط الدعوة لا من يده.
     *
     * **واختياريّ**: من سجّل بلا دعوةٍ حسابُه كامل، **ورمزٌ خاطئٌ يحرمه
     * المكافأةَ وحدَها ولا يُسقط تسجيلَه.**
     */
    val referral: String = "",
    val busy: Boolean = false,
    /**
     * **أيُطلب رمزٌ قبل الفورم؟** — من `auth.signup_verify` في اللوحة.
     *
     * **والشاشةُ تحتاجه لتسمّي زرَّها**: «توثيق حسابي» حين يُطلب،
     * **و«متابعة» حين لا يُطلب** — وزرٌّ يعد بما لا يفعل يُقرأ عطباً.
     */
    val needsCode: Boolean = false,
    val error: String = "",
    /**
     * **أخطأُ الانقطاعِ وحدَه** — وهو يُمحى تلقائيّاً عند عودة الاتّصال
     * (`CUST-DEF-011`)، **وأخطاءُ التحقّق تبقى** فليست من الشبكة.
     */
    val offlineError: Boolean = false,
    /**
     * **رقمٌ لحساب موجود** (Batch 4، SG1): يُعرَض بدلاً من رسالة الخطأ لوحُ
     * استعادةٍ — دخولٌ/استعادةُ كلمةٍ/دخولٌ برمز — لا طريقٌ مسدود عند
     * `phone_taken`. **ويُعرَض بعد فشلِ الدخولِ التلقائيّ** في سياق الضياع.
     */
    val recovery: Boolean = false,
)

data class SignupActions(
    val setPhone: (String) -> Unit,
    val sendCode: () -> Unit,
    val verifyCode: (String) -> Unit,
    val confirm: (name: String, password: String, referral: String) -> Unit,
    val cancel: () -> Unit,
    /** **عاد الاتّصال** — تُمحى رسالةُ الانقطاعِ الحقليّةُ (`CUST-DEF-011`). */
    val onReconnected: () -> Unit,
    /** **الرقمُ لحسابٍ موجود ⇒ إلى الدخول بكلمة المرور** (بالرقمِ مُعبّأً). */
    val toLogin: () -> Unit = {},
    /** **⇒ استعادةُ كلمة المرور.** */
    val toReset: () -> Unit = {},
    /** **⇒ الدخولُ برمز.** */
    val toOtp: () -> Unit = {},
)

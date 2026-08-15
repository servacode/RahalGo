package com.rahalgo.ui

import androidx.compose.foundation.Image
import com.rahalgo.design.Rahal
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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشة الدخول — بابان في شاشة واحدة، ولكلّ تطبيقٍ عنوانُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الثالثة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # ولماذا تبويب في الأعلى لا زرّ في الأسفل
 *
 * (طلب المالك: «تبديل طريقة تسجيل الدخول مثل الطريقة بالويب».)
 *
 * **وكان الباب الثاني زرّ نصّ تحت زرّ الدخول** — يُقرأ خيارا هامشيّا لا
 * طريقة مساوية، **ومن نسي كلمته لا يراه أصلا** لأنّ عينه على ما فوق.
 * **والتبويب يقول: طريقتان، اختر.** وهو ما تفعله بطاقة الدخول في الويب
 * (`web/packages/auth/LoginCard`).
 *
 * # وباب الرمز يُطفأ من الإعدادات
 *
 * **`auth.otp_login` في المحرّك** — فإن أُطفئ لم يُعرض التبويب أصلا:
 * **تبويب يفتح على باب مغلق أسوأ من غيابه.**
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** — ومن يدخل
 * وأين يذهب يقرّره المحرّك والأدوار. **والنداءات في `LoginViewModel`.**
 *
 * # والخطأ يُعرض كما قاله الخادم
 *
 * (القاعدة ٨.) **ممنوع `catch { showGenericError() }`** — عشرة مواضع في
 * الويب كانت تبتلع سبب الخادم، **فيعيد صاحبه المحاولة عشرا والعلّة في
 * رقمه.**
 */
@Composable
fun AuthScreen(
    state: LoginState,
    actions: LoginActions,
    /**
     * **عنوانُ الشاشة** — «دخول السائق» أو «دخول».
     *
     * **ولا يُرفع نصُّه إلى هنا**: اسمُ دورٍ في وحدةٍ مشتركةٍ يعني أنّ
     * الزبونَ يحمل في تطبيقه نصّاً يقول «السائق».
     */
    title: String,
    /**
     * **أيُعرض بابُ إنشاء الحساب؟**
     *
     * **والزبونُ وحدَه يُنشئ حسابَه بنفسه** — السائقُ والمتجرُ والمندوبُ
     * حساباتُهم من المنصّة. **وبابٌ يُعرض لمن لا يستطيع أن يدخل منه أسوأُ
     * من غيابه**: يُملأ ثمّ يُردّ.
     */
    signup: Boolean = false,
) {
    // **والرقم يبقى بين التبويبين** — من كتبه ثمّ بدّل الطريقة لا يعيده.
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            // **وحشوة جانبية هنا لا في الغلاف** — الحقول تحتاج هامشا،
            // والشعار لا.
            .padding(horizontal = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Spacer(Modifier.height(40.dp))
        Image(
            painter = painterResource(com.rahalgo.design.R.drawable.intro_logo),
            contentDescription = null,
            modifier = Modifier.size(140.dp),
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = title,
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(20.dp))

        if (state.otpAvailable) {
            TabRow(selectedTabIndex = if (state.mode == LoginMode.PASSWORD) 0 else 1) {
                Tab(
                    selected = state.mode == LoginMode.PASSWORD,
                    onClick = { actions.setMode(LoginMode.PASSWORD) },
                    enabled = !state.busy,
                    text = { Text(stringResource(R.string.login_tab_password)) },
                )
                Tab(
                    selected = state.mode == LoginMode.OTP,
                    onClick = { actions.setMode(LoginMode.OTP) },
                    enabled = !state.busy,
                    text = { Text(stringResource(R.string.login_tab_otp)) },
                )
            }
            Spacer(Modifier.height(20.dp))
        }

        // **والرقم لا يُعدَّل بعد إرسال الرمز** — الرمز أُصدر لرقم بعينه،
        // **ومن بدّله بقي رمزه على الرقم الأوّل** فيُقال له «الرمز غير
        // صحيح» وهو صحيح.
        PhoneField(
            value = phone,
            onChange = { phone = it },
            enabled = !state.busy && !state.codeSent,
        )
        Spacer(Modifier.height(10.dp))

        when {
            state.mode == LoginMode.PASSWORD -> PasswordField(
                value = password,
                onChange = { password = it },
                enabled = !state.busy,
            )

            state.codeSent -> CodeField(
                value = code,
                onChange = { code = it },
                enabled = !state.busy,
            )
        }

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Text(
                text = state.error,
                color = Rahal.colors.accent,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        }

        Spacer(Modifier.height(20.dp))
        Button(
            onClick = {
                when {
                    state.mode == LoginMode.PASSWORD -> actions.login(phone.trim(), password)
                    state.codeSent -> actions.verifyCode(phone.trim(), code.trim())
                    else -> actions.sendCode(phone.trim())
                }
            },
            // **ولا يُضغط وهو يعمل** — ضغطتان تفتحان جلستين، **والثانية
            // تُبطل الأولى** (جلسة واحدة لكلّ نوع عميل).
            enabled = !state.busy && phone.isNotBlank() && when {
                state.mode == LoginMode.PASSWORD -> password.isNotBlank()
                state.codeSent -> code.isNotBlank()
                else -> true
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (state.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Icon(
                    painter = painterResource(
                        if (state.mode == LoginMode.OTP && !state.codeSent) R.drawable.ic_key
                        else R.drawable.ic_login,
                    ),
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.size(8.dp))
                Text(
                    stringResource(
                        when {
                            state.mode == LoginMode.PASSWORD -> R.string.login_submit
                            state.codeSent -> R.string.login_verify_and_enter
                            else -> R.string.reset_send
                        },
                    ),
                )
            }
        }

        Spacer(Modifier.height(4.dp))
        // **ونسيان الكلمة ليس عطبا** — بابه هنا لا في اتّصال بالمكتب.
        // **ولا يُعرض مع الرمز**: من دخل برمز لا يحتاج كلمة أصلا.
        if (state.mode == LoginMode.PASSWORD) {
            TextButton(onClick = actions.forgot, enabled = !state.busy) {
                Text(stringResource(R.string.login_forgot), color = Rahal.colors.inkMuted)
            }
        } else if (state.codeSent) {
            TextButton(onClick = actions.resetCode, enabled = !state.busy) {
                Text(stringResource(R.string.login_change_number), color = Rahal.colors.inkMuted)
            }
        }
        // **وبابُ الحساب الجديد تحت الدخول لا فوقه** — أكثرُ من يفتح
        // الشاشةَ له حسابٌ أصلا، **والتسجيلُ مرّةٌ في العمر.**
        if (signup) {
            TextButton(onClick = actions.signup, enabled = !state.busy) {
                Text(stringResource(R.string.signup_open), color = Rahal.colors.accent)
            }
        }
        Spacer(Modifier.height(40.dp))
    }
}

/** أيّ البابين مفتوح الآن. */
enum class LoginMode { PASSWORD, OTP }

/** ما تعرضه الشاشة — **ولا تملكه هي.** */
data class LoginState(
    val mode: LoginMode = LoginMode.PASSWORD,
    /** هل أُرسل الرمز؟ — **تُبدَّل الشاشة من «أرسل» إلى «تحقّق».** */
    val codeSent: Boolean = false,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **هل باب الرمز مفتوح؟ — ولا يُفترض جوابه**
     * ══════════════════════════════════════════════════════════════════
     *
     * (`auth.otp_login` في المحرّك.)
     *
     * **وكان يبدأ مفتوحا ثمّ يُصحَّح** — فيظهر التبويب لحظة ثمّ يختفي
     * أمام صاحبه. **ووميض كهذا يُقرأ عطبا**، ورآه المالك (٢٠٢٦-٠٨-١٢:
     * «ظهر لحاله واختفى»).
     *
     * **فيبدأ مغلقا ولا يُفتح إلّا بجواب**: الدخول بكلمة المرور يعمل
     * على كلّ حال، **وباب لم يُتأكّد منه لا يُعرض.**
     */
    val otpAvailable: Boolean = false,
    val busy: Boolean = false,
    val error: String = "",
)

/** ما تستطيع الشاشة أن تطلبه. */
data class LoginActions(
    val login: (phone: String, password: String) -> Unit,
    val setMode: (LoginMode) -> Unit,
    val sendCode: (phone: String) -> Unit,
    val verifyCode: (phone: String, code: String) -> Unit,
    val resetCode: () -> Unit,
    val forgot: () -> Unit,
    /** **يفتح بابَ الحساب الجديد** — ولا يُنادى إن لم يُعرض. */
    val signup: () -> Unit = {},
)

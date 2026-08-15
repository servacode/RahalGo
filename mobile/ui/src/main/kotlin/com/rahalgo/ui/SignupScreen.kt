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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
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

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
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
            }
        }

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
        Button(
            onClick = {
                when (state.step) {
                    SignupStep.PHONE -> actions.sendCode()
                    SignupStep.CODE -> actions.verifyCode(code.trim())
                    SignupStep.DETAILS -> actions.confirm(name.trim(), password)
                }
            },
            enabled = !state.busy && when (state.step) {
                SignupStep.PHONE -> state.phone.isNotBlank()
                SignupStep.CODE -> code.isNotBlank()
                SignupStep.DETAILS -> name.isNotBlank() && password.isNotBlank()
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (state.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(
                    stringResource(
                        when (state.step) {
                            SignupStep.PHONE -> R.string.reset_send
                            SignupStep.CODE -> R.string.reset_verify
                            SignupStep.DETAILS -> R.string.signup_create
                        },
                    ),
                )
            }
        }

        Spacer(Modifier.height(6.dp))
        TextButton(onClick = actions.cancel, enabled = !state.busy) {
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
    val error: String = "",
)

data class SignupActions(
    val setPhone: (String) -> Unit,
    val sendCode: () -> Unit,
    val verifyCode: (String) -> Unit,
    val confirm: (name: String, password: String) -> Unit,
    val cancel: () -> Unit,
)

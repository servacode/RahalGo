package com.rahalgo.driver.login

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
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.InkMuted
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.CodeField
import com.rahalgo.driver.ui.PasswordField
import com.rahalgo.driver.ui.PhoneField

/**
 * ══════════════════════════════════════════════════════════════════════
 * **استعادة كلمة المرور — ثلاث خطوات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * رقم ← رمز ← كلمة جديدة. **وينتهي داخلا** لا عائدا إلى شاشة الدخول:
 * المحرّك يفتح الجلسة مع التأكيد (`ConfirmPasswordReset`)، **فلا يُطلب
 * منه أن يكتب ما ضبطه قبل ثانية.**
 *
 * # ولماذا الرمز في خطوة وحده
 *
 * **المحرّك يتحقّق من الرمز ولا يستهلكه** (`password/reset/verify`).
 * **ولو جُمع مع الكلمة الجديدة في شاشة واحدة** لكتب صاحبه كلمة ورمزا
 * ثمّ قيل له «الرمز خطأ» — **فيعيد الاثنين.**
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** — والنداءات
 * في `LoginViewModel`.
 */
@Composable
fun ResetScreen(state: ResetState, actions: ResetActions) {
    var code by remember { mutableStateOf("") }
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
            text = stringResource(R.string.reset_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            text = stringResource(
                when (state.step) {
                    ResetStep.PHONE -> R.string.reset_hint_phone
                    ResetStep.CODE -> R.string.reset_hint_code
                    ResetStep.PASSWORD -> R.string.reset_hint_password
                },
            ),
            color = InkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))

        when (state.step) {
            // **والرقم يُكتب مرّة واحدة** — ثمّ يُحمل في الحالة إلى
            // الخطوتين التاليتين. **ومن أعاد كتابته في كل خطوة أخطأ في
            // واحدة** فسقط الرمز بلا سبب ظاهر.
            ResetStep.PHONE -> PhoneField(
                value = state.phone,
                onChange = actions.setPhone,
                enabled = !state.busy,
            )

            ResetStep.CODE -> CodeField(
                value = code,
                onChange = { code = it },
                enabled = !state.busy,
            )

            // **والعين هنا أيضا** — كلمة جديدة تُكتب مرّة واحدة بلا
            // تأكيد، **فمن أخطأ حرفا ضبط كلمة لا يعرفها** ثمّ عاد يستعيد
            // من جديد.
            ResetStep.PASSWORD -> PasswordField(
                value = password,
                onChange = { password = it },
                enabled = !state.busy,
                label = R.string.reset_new_password,
            )
        }

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(14.dp))
            Text(
                text = state.error,
                color = BrandOrange,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        }

        Spacer(Modifier.height(24.dp))
        Button(
            onClick = {
                when (state.step) {
                    ResetStep.PHONE -> actions.sendCode()
                    ResetStep.CODE -> actions.verifyCode(code.trim())
                    ResetStep.PASSWORD -> actions.confirm(password)
                }
            },
            enabled = !state.busy && when (state.step) {
                ResetStep.PHONE -> state.phone.isNotBlank()
                ResetStep.CODE -> code.isNotBlank()
                ResetStep.PASSWORD -> password.isNotBlank()
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (state.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(
                    stringResource(
                        when (state.step) {
                            ResetStep.PHONE -> R.string.reset_send
                            ResetStep.CODE -> R.string.reset_verify
                            ResetStep.PASSWORD -> R.string.reset_confirm
                        },
                    ),
                )
            }
        }

        Spacer(Modifier.height(6.dp))
        TextButton(onClick = actions.cancel, enabled = !state.busy) {
            Text(stringResource(R.string.reset_back), color = InkMuted)
        }
    }
}

/** أين وصل صاحبه في الخطوات الثلاث. */
enum class ResetStep { PHONE, CODE, PASSWORD }

data class ResetState(
    val step: ResetStep = ResetStep.PHONE,
    val phone: String = "",
    /** يُحمل من خطوة التحقّق إلى خطوة الكلمة الجديدة — **المحرّك يطلبه
     *  مرّتين ولا يستهلكه في الأولى.** */
    val code: String = "",
    val busy: Boolean = false,
    val error: String = "",
)

data class ResetActions(
    val setPhone: (String) -> Unit,
    val sendCode: () -> Unit,
    val verifyCode: (String) -> Unit,
    val confirm: (String) -> Unit,
    val cancel: () -> Unit,
)

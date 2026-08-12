package com.rahalgo.driver.login

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.InkMuted
import com.rahalgo.driver.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشة دخول السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الثالثة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * **وكلمة المرور أوّلا كما في الويب** — والرمز باب ثانٍ. (`LoginCard`
 * في `web/packages/auth` تفتح على `password`.)
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.)
 *
 * **الشاشة تعرض وترسل** — ومن يدخل وأين يذهب يقرّره المحرّك والأدوار.
 * **والنداءات في `LoginViewModel`**، والشبكة في `shared`.
 *
 * # والخطأ يُعرض كما قاله الخادم
 *
 * (القاعدة ٨.) **ممنوع `catch { showGenericError() }`** — عشرة مواضع في
 * الويب كانت تبتلع سبب الخادم، **فيعيد صاحبه المحاولة عشرا والعلّة في
 * رقمه.**
 */
@Composable
fun LoginScreen(state: LoginState, actions: LoginActions) {
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }

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
        Spacer(Modifier.height(48.dp))
        Image(
            painter = painterResource(com.rahalgo.design.R.drawable.intro_logo),
            contentDescription = null,
            modifier = Modifier.size(160.dp),
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.login_title),
            style = androidx.compose.material3.MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(28.dp))

        OutlinedTextField(
            value = phone,
            onValueChange = { phone = it },
            label = { Text(stringResource(R.string.login_phone)) },
            singleLine = true,
            // **ولوحة أرقام لا حروف** — رقم الهاتف لا يُكتب بحروف،
            // **ولوحة كاملة تُبطئ من يكتبه كل يوم.**
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            enabled = !state.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(14.dp))
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text(stringResource(R.string.login_password)) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
            enabled = !state.busy,
            modifier = Modifier.fillMaxWidth(),
        )

        // **وسبب الخادم كما قاله** — لا رسالة عامّة تُسكته.
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
            onClick = { actions.login(phone.trim(), password) },
            // **ولا يُضغط وهو يعمل** — ضغطتان تفتحان جلستين، **والثانية
            // تُبطل الأولى** (جلسة واحدة لكل نوع عميل).
            enabled = !state.busy && phone.isNotBlank() && password.isNotBlank(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (state.busy) {
                CircularProgressIndicator(
                    modifier = Modifier.size(18.dp),
                    strokeWidth = 2.dp,
                )
            } else {
                Text(stringResource(R.string.login_submit))
            }
        }

        Spacer(Modifier.height(6.dp))
        TextButton(onClick = actions.useOtp, enabled = !state.busy) {
            Text(stringResource(R.string.login_use_otp), color = InkMuted)
        }
        Spacer(Modifier.height(48.dp))
    }
}

/** ما تعرضه الشاشة — **ولا تملكه هي.** */
data class LoginState(
    val busy: Boolean = false,
    val error: String = "",
)

/** ما تستطيع الشاشة أن تطلبه. */
data class LoginActions(
    val login: (phone: String, password: String) -> Unit,
    val useOtp: () -> Unit,
)

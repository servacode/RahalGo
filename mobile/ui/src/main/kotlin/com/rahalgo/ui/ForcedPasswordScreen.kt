package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تبديلُ كلمةِ المرور المطلوب — تدفّقٌ لا رسالةُ خطأ** (`CUST-DEF-010`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المحرّكُ يطلب تبديلاً (`must_change_password`) فيُساق صاحبُه إلى هذه
 * الشاشة قبل أيّ دخولٍ للتطبيق** — لا نصُّ خطإٍ عابر. **ويُشرَح السبب**،
 * **وتُطلَب الكلمةُ الحاليّةُ (يعرفها — دخل بها) + جديدةٌ + تأكيد**، **ولا
 * يُكشَف القديمُ.** **والإرسالُ عبر النقطة الموثوقة** (`AuthViewModel`)،
 * **والرجوعُ لا يتخطّى القيد**: كلُّ إقلاعٍ يُعيد كشفَه من `me()`.
 *
 * **والخروجُ متاح** لمن أراد حساباً آخر — يمرّ بحدِّ الجلسة الواحد
 * (`CUST-DEF-009`) فلا يتسرّب محلّيّ.
 */
@Composable
fun ForcedPasswordScreen(
    state: AuthViewModel.PwChange,
    onSubmit: (current: String, next: String) -> Unit,
    onLogout: () -> Unit,
    onErrorConsumed: () -> Unit = {},
) {
    var current by remember { mutableStateOf("") }
    var next by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    val mismatch = next.isNotEmpty() && confirm.isNotEmpty() && next != confirm

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = stringResource(R.string.pw_forced_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.pw_forced_body),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(20.dp))

        PasswordField(current, {
            current = it
            onErrorConsumed()
        }, !state.busy, R.string.acc_pw_current)
        Spacer(Modifier.height(8.dp))
        PasswordField(next, { next = it }, !state.busy, R.string.acc_pw_new)
        Spacer(Modifier.height(8.dp))
        PasswordField(confirm, { confirm = it }, !state.busy, R.string.acc_pw_confirm)

        if (mismatch) {
            Spacer(Modifier.height(6.dp))
            Text(stringResource(R.string.acc_pw_mismatch), color = Rahal.colors.danger)
        }
        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(state.error, color = Rahal.colors.danger)
        }

        Spacer(Modifier.height(16.dp))
        RahalButton(
            onClick = { onSubmit(current, next) },
            enabled = !state.busy && current.isNotEmpty() && next.isNotEmpty() &&
                confirm.isNotEmpty() && !mismatch,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_pw_change)) }

        Spacer(Modifier.height(8.dp))
        RahalOutlineButton(
            onClick = onLogout,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.login_logout)) }
    }
}

package com.rahalgo.driver.account

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateGreen
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R
import com.rahalgo.driver.location.LastPoint
import com.rahalgo.driver.ui.Avatar
import com.rahalgo.shared.model.Address

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ الحساب — كما هي في الويب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بالطبع تفتح صفحةً كاملة… كيف سيدير حسابَه
 *  إذا يغيّر كلمة سرّه، يبدّل صورتَه، إذا أراد حذف حسابه، وإذا أراد
 *  إضافة عنوان» · «ابدأ ببنائها بالتطبيق كما هي بالويب، ولكن حذف حسابي
 *  اجعلها بالآخر بعد العناوين».)
 *
 * # وترتيبُ الأقسام ليس ذوقا
 *
 * **الهويّةُ أوّلاً** — هي ما يفتح الشاشةَ لأجله غالبا. **ثمّ ما يُبدَّل
 * نادراً** (كلمة المرور)، **ثمّ ما يُضاف** (العناوين).
 *
 * **والحذفُ آخرَ شيءٍ بأمر المالك** — **وفعلٌ لا يُستدرَك لا يُوضع في
 * طريق إبهامٍ يمرّ.** ومن نزل إليه نزل قاصدا.
 *
 * # ورقمُ الهاتف يُعرض ولا يُبدَّل هنا
 *
 * **تبديلُه يحتاج رمزاً على الرقم الجديد ثمّ تأكيدا** — وهو تدفّقُ
 * شاشتين. **وأخطرُ ما فيه أنّه يُسقط توثيقَ واتساب** (أُصلح
 * ٢٠٢٦-٠٨-١٣)، **فيفقد صاحبُه بابَ استعادة حسابه** إن لم ينتبه.
 *
 * **فيُقرأ هنا ويُبدَّل حيث يُشرَح** — والسطرُ تحته يقول أين.
 */
@Composable
fun AccountScreen(vm: AccountViewModel, onLoggedOut: () -> Unit) {
    val s = vm.state

    // **والخروجُ يقع حين يُحذف الحساب** — لا شاشةَ لمن لا حسابَ له.
    LaunchedEffect(s.deleted) { if (s.deleted) onLoggedOut() }

    if (s.loading) {
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) { CircularProgressIndicator() }
        return
    }

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        // **ورسالةٌ واحدةٌ في أعلى الشاشة** — لا رسالةٌ تحت كلّ قسم
        // فتُقرأ رسائلُ متناثرةٌ لا يُعرف أيُّها الأحدث.
        if (s.error.isNotEmpty()) Notice(s.error, StateRed)
        if (s.done.isNotEmpty()) Notice(s.done, StateGreen)

        Identity(vm, s)
        Gap()
        PasswordSection(vm, s)
        Gap()
        AddressesSection(vm, s)
        Gap()
        // **والخطرُ آخرا** — بأمر المالك.
        DangerSection(vm, s)
        Spacer(Modifier.height(32.dp))
    }
}

// ــ الهويّة ــ

@Composable
private fun Identity(vm: AccountViewModel, s: AccountState) {
    val me = s.me ?: return
    var draft by rememberSaveable(me.fullName) { mutableStateOf(me.fullName) }

    // **ومنتقي الصور من النظام** — لا إذنَ لقراءة المعرض كلِّه.
    //
    // **`PickVisualMedia` يعطي ملفّاً واحداً اختاره صاحبُه** — والإذنُ
    // العامُّ يطلب من السائق أن يفتح ألبومَه كلَّه لتطبيق عمل.
    val picker = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia(),
    ) { uri -> uri?.let(vm::setAvatar) }

    SectionTitle(stringResource(R.string.acc_identity))
    Row(verticalAlignment = Alignment.CenterVertically) {
        Avatar(url = me.avatarThumbUrl, name = me.fullName, size = 64)
        Spacer(Modifier.size(12.dp))
        Column {
            TextButton(
                onClick = {
                    picker.launch(
                        androidx.activity.result.PickVisualMediaRequest(
                            ActivityResultContracts.PickVisualMedia.ImageOnly,
                        ),
                    )
                },
                enabled = !s.busy,
            ) { Text(stringResource(R.string.acc_photo_pick)) }
            if (!me.avatarThumbUrl.isNullOrEmpty()) {
                TextButton(onClick = vm::removeAvatar, enabled = !s.busy) {
                    Text(stringResource(R.string.acc_photo_remove), color = InkMuted)
                }
            }
        }
    }

    Spacer(Modifier.height(12.dp))
    OutlinedTextField(
        value = draft,
        onValueChange = { draft = it },
        label = { Text(stringResource(R.string.acc_name)) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    Button(
        onClick = { vm.setName(draft) },
        enabled = !s.busy && draft.trim().isNotEmpty() && draft.trim() != me.fullName,
        modifier = Modifier.fillMaxWidth(),
    ) { Text(stringResource(R.string.acc_save)) }

    Spacer(Modifier.height(14.dp))
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(stringResource(R.string.acc_phone), color = InkMuted)
        Text(me.phone, style = MaterialTheme.typography.bodyMedium)
    }
    Text(
        text = stringResource(
            if (me.whatsappVerified) R.string.acc_wa_verified else R.string.acc_wa_unverified,
        ),
        color = if (me.whatsappVerified) StateGreen else InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Text(
        text = stringResource(R.string.acc_phone_hint),
        color = InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
}

// ــ كلمة المرور ــ

@Composable
private fun PasswordSection(vm: AccountViewModel, s: AccountState) {
    var current by remember { mutableStateOf("") }
    var next by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    val mismatch = next.isNotEmpty() && confirm.isNotEmpty() && next != confirm

    SectionTitle(stringResource(R.string.acc_password))
    PwField(current, { current = it }, R.string.acc_pw_current)
    Spacer(Modifier.height(8.dp))
    PwField(next, { next = it }, R.string.acc_pw_new)
    Spacer(Modifier.height(8.dp))
    PwField(confirm, { confirm = it }, R.string.acc_pw_confirm)

    // **والتطابقُ يُقال قبل الإرسال لا بعده** — نداءٌ يذهب ليعود بخطأٍ
    // يعرفه الجهازُ نفسُه **يُضيّع ثانيتين ويستهلك حزمة.**
    if (mismatch) {
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.acc_pw_mismatch), color = StateRed)
    }
    Spacer(Modifier.height(8.dp))
    Button(
        onClick = {
            vm.setPassword(current, next)
            current = ""; next = ""; confirm = ""
        },
        enabled = !s.busy && current.isNotEmpty() && next.isNotEmpty() && !mismatch,
        modifier = Modifier.fillMaxWidth(),
    ) { Text(stringResource(R.string.acc_pw_change)) }
}

@Composable
private fun PwField(value: String, onChange: (String) -> Unit, label: Int) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(stringResource(label)) },
        singleLine = true,
        visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
            keyboardType = androidx.compose.ui.text.input.KeyboardType.Password,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
}

// ــ العناوين ــ

@Composable
private fun AddressesSection(vm: AccountViewModel, s: AccountState) {
    var adding by rememberSaveable { mutableStateOf(false) }

    SectionTitle(stringResource(R.string.acc_addresses))
    if (s.addresses.isEmpty() && !adding) {
        Text(stringResource(R.string.acc_addr_empty), color = InkMuted)
    }
    s.addresses.forEach { a -> AddressRow(a, vm, s) }

    Spacer(Modifier.height(8.dp))
    if (!adding) {
        OutlinedButton(
            onClick = { adding = true },
            enabled = !s.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_addr_add)) }
    } else {
        AddAddress(vm, s, onDone = { adding = false })
    }
}

@Composable
private fun AddressRow(a: Address, vm: AccountViewModel, s: AccountState) {
    Column(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(a.label, style = MaterialTheme.typography.titleSmall)
            if (a.isDefault) {
                Text(stringResource(R.string.acc_addr_default), color = StateGreen)
            }
        }
        Text(a.text, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        Row {
            if (!a.isDefault) {
                TextButton(onClick = { vm.makeDefault(a.id) }, enabled = !s.busy) {
                    Text(stringResource(R.string.acc_addr_make_default))
                }
            }
            TextButton(onClick = { vm.deleteAddress(a.id) }, enabled = !s.busy) {
                Text(stringResource(R.string.acc_addr_delete), color = StateRed)
            }
        }
        Divider()
    }
}

/**
 * **وإضافةُ العنوان بالموقع الحاليّ لا بخريطة.**
 *
 * **السائقُ يقف حيث يريد أن يحفظه** — والخريطةُ في هذه الشاشة تعني
 * منتقيَ نقطةٍ كاملاً. **وموضعُه الآن أدقُّ ممّا يشير إليه بإصبعه** وهو
 * ماشٍ.
 *
 * **وبلا موقعٍ لا يُحفظ عنوانٌ بإحداثيٍّ صفر** — نقطةٌ في المحيط
 * الأطلسيّ **تُرسل سائقاً إلى لا مكان.**
 */
@Composable
private fun AddAddress(vm: AccountViewModel, s: AccountState, onDone: () -> Unit) {
    var label by rememberSaveable { mutableStateOf("") }
    var text by rememberSaveable { mutableStateOf("") }
    val point = LastPoint.value

    OutlinedTextField(
        value = label,
        onValueChange = { label = it },
        label = { Text(stringResource(R.string.acc_addr_label)) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    OutlinedTextField(
        value = text,
        onValueChange = { text = it },
        label = { Text(stringResource(R.string.acc_addr_text)) },
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(6.dp))
    Text(
        text = stringResource(
            if (point == null) R.string.acc_addr_need_point else R.string.acc_addr_here,
        ),
        color = if (point == null) StateRed else InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = {
                point?.let { vm.addAddress(label, text, it.lat, it.lng) }
                onDone()
            },
            enabled = !s.busy && point != null && label.isNotBlank() && text.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_save)) }
        OutlinedButton(onClick = onDone, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

// ــ الحذف ــ

@Composable
private fun DangerSection(vm: AccountViewModel, s: AccountState) {
    var code by remember { mutableStateOf("") }

    SectionTitle(stringResource(R.string.acc_danger), StateRed)
    Text(
        stringResource(R.string.acc_delete_hint),
        color = InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(10.dp))

    if (!s.deleteAsked) {
        OutlinedButton(
            onClick = vm::askDelete,
            enabled = !s.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_delete_ask), color = StateRed) }
        return
    }

    OutlinedTextField(
        value = code,
        onValueChange = { code = it },
        label = { Text(stringResource(R.string.acc_delete_code)) },
        singleLine = true,
        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
            keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = { vm.confirmDelete(code) },
            enabled = !s.busy && code.isNotBlank(),
            colors = ButtonDefaults.buttonColors(containerColor = StateRed),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_delete_confirm)) }
        OutlinedButton(onClick = vm::cancelDelete, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

// ــ قطعٌ صغيرة ــ

@Composable
private fun SectionTitle(text: String, color: androidx.compose.ui.graphics.Color = BrandOrange) {
    Text(text, style = MaterialTheme.typography.titleMedium, color = color)
    Spacer(Modifier.height(10.dp))
}

@Composable
private fun Gap() {
    Spacer(Modifier.height(24.dp))
    Divider()
    Spacer(Modifier.height(16.dp))
}

@Composable
private fun Notice(text: String, color: androidx.compose.ui.graphics.Color) {
    Text(
        text = text,
        color = color,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
    )
}

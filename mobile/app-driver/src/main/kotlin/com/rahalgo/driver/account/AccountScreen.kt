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
import androidx.compose.material3.HorizontalDivider
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
    // ══════════════════════════════════════════════════════════════════
    // **ورقمُه حقلٌ يقرؤه لا سطرٌ رماديّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن يكون هناك حقلٌ مكتوبٌ فيه رقمُه
    //  ليعرف الرقمَ الموجودَ بحسابه، وتبديلُ الرقم يجب أن يكون زرّاً
    //  واضحاً».)
    //
    // **وسطرٌ بين سطرين يُمسَح بالعين ولا يُقرأ** — والرقمُ هو ما يدخل
    // به، **ومن بدّل هاتفَه يسأل: أيُّ رقمٍ في حسابي الآن؟**
    //
    // **ومقفلٌ عمداً**: تبديلُه ليس كتابةً في حقل — **رمزٌ يصل الرقمَ
    // الجديد ثمّ تأكيد.** وحقلٌ يُكتب فيه ولا يُحفظ يُقرأ عطبا.
    OutlinedTextField(
        value = me.phone,
        onValueChange = {},
        readOnly = true,
        label = { Text(stringResource(R.string.acc_phone_current)) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    PhoneChange(vm, s)
    Spacer(Modifier.height(12.dp))
    WhatsAppVerify(vm, s, me.whatsappVerified)
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **توثيقُ واتساب — شرطٌ لفتح الدوام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم بالطبع السائق يجب أن يوثّق حسابَه على
 *  واتساب… لا يمكنه استقبال الطلبات بدون توثيق حسابه، اعتبره شرطاً
 *  للسائق».)
 *
 * **وعلى رقم الحساب نفسِه لا على رقمٍ ثانٍ** (قرارُ المالك ٢٠٢٦-٠٨-١٢:
 * «ما يصير رقم الهاتف مختلف عن واتساب، هيك تخرب الدنيا») — **فلا يُسأل
 * عن رقم**: يُوثَّق ما هو مسجَّلٌ في حسابه.
 *
 * **وحالُه يُقال قبل الزرّ لا بعده**: من رأى «غير موثَّق» عرف لماذا لا
 * يفتح دوامُه، **ومن مُنع بلا أن يعرف يظنّ التطبيقَ معطّلا.**
 */
@Composable
private fun WhatsAppVerify(vm: AccountViewModel, s: AccountState, verified: Boolean) {
    var code by remember { mutableStateOf("") }
    val waiting = s.waPending.isNotEmpty()

    Text(
        text = stringResource(
            if (verified) R.string.acc_wa_verified else R.string.acc_wa_unverified,
        ),
        color = if (verified) StateGreen else StateRed,
        style = MaterialTheme.typography.bodySmall,
    )
    // **والموثَّقُ لا يُدعى إلى فعلٍ تمّ** — زرٌّ باقٍ بعد نجاحه يُقرأ
    // «لم ينجح».
    if (verified && !waiting) return

    Spacer(Modifier.height(6.dp))
    if (!waiting) {
        Text(
            stringResource(R.string.acc_wa_hint),
            color = InkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(8.dp))
        Button(
            onClick = vm::askWhatsApp,
            enabled = !s.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_wa_verify)) }
        return
    }

    OutlinedTextField(
        value = code,
        onValueChange = { code = it },
        label = { Text(stringResource(R.string.acc_wa_code)) },
        singleLine = true,
        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
            keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = { vm.confirmWhatsApp(code); code = "" },
            enabled = !s.busy && code.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_wa_confirm)) }
        OutlinedButton(
            onClick = { vm.cancelWhatsApp(); code = "" },
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_delete_cancel)) }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تبديلُ الرقم — خطوتان في مكانه لا في متصفّح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «تغيير رقم الهاتف يجب أن يكون موجوداً
 *  أيضاً… التطبيق يجب أن يكون تطبيقاً كاملاً، لا حاجةَ للمستخدم من
 *  الدخول إلى مكانٍ ثانٍ».)
 *
 * **والرمزُ يصل الرقمَ الجديد لا القديم** — وهو ما يُثبت أنّه له: **من
 * كتب رقمَ غيره لا يصله شيء.**
 *
 * **والتحذيرُ يُقال قبل الإرسال لا بعده**: تبديلُ الرقم يُسقط توثيقَ
 * واتساب، **وبه يستعيد حسابَه إن نسي رمزَه.** ومن عرف قبل أن يضغط عاد
 * فوثّق.
 */
@Composable
private fun PhoneChange(vm: AccountViewModel, s: AccountState) {
    var open by rememberSaveable { mutableStateOf(false) }
    var phone by rememberSaveable { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    val waiting = s.phonePending.isNotEmpty()

    if (!open && !waiting) {
        // **وزرٌّ بعرض الشاشة لا نصٌّ يُبحث عنه** — بأمر المالك.
        OutlinedButton(
            onClick = { open = true },
            enabled = !s.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_phone_change)) }
        return
    }

    if (!waiting) {
        OutlinedTextField(
            value = phone,
            onValueChange = { phone = it },
            label = { Text(stringResource(R.string.acc_phone_new)) },
            singleLine = true,
            keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
                keyboardType = androidx.compose.ui.text.input.KeyboardType.Phone,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(
                onClick = { vm.askPhone(phone) },
                enabled = !s.busy && phone.isNotBlank(),
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.acc_phone_send)) }
            OutlinedButton(
                onClick = { open = false; phone = "" },
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.acc_delete_cancel)) }
        }
        return
    }

    OutlinedTextField(
        value = code,
        onValueChange = { code = it },
        label = { Text(stringResource(R.string.acc_phone_code)) },
        singleLine = true,
        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
            keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = { vm.confirmPhone(code); code = ""; open = false; phone = "" },
            enabled = !s.busy && code.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_phone_confirm)) }
        OutlinedButton(
            onClick = { vm.cancelPhone(); code = "" },
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_delete_cancel)) }
    }
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
        HorizontalDivider()
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
    // **والنقطةُ تُلتقط بضغطةٍ وتبقى** — لا تُقرأ لحظةَ الحفظ.
    var pinned by rememberSaveable { mutableStateOf<Pair<Double, Double>?>(null) }
    val live = LastPoint.value

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

    // ══════════════════════════════════════════════════════════════════
    // **والموقعُ يُلتقط بضغطةٍ يراها — لا خُفيةً ولا بخريطةٍ تُفتح**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «مو ضروريّ تفتح خريطة، يكفي زرّ «تحديد
    //  موقعي على الخريطة» ليجلب عنوانَه بحقلٍ يمتلئ تلقائيّاً بمعلومات
    //  موقعه… ليعرف أنّه تمّ تحديد عنوانه على الخريطة — بصريّاً أقوى من
    //  أن يُحدَّد موقعُه بشكلٍ تلقائيٍّ مخفيٍّ دون أن يعرف».)
    //
    // **وكان يُقرأ في الخفاء لحظةَ الحفظ** — فيحفظ عنوانَه وهو لا يدري
    // أيَّ نقطةٍ حُفظت، **ولا يعرف أوقعت أصلاً أم لا.**
    //
    // **وخريطةٌ تُفتح ثقيلةٌ هنا**: منتقي نقطةٍ كاملٌ لشيءٍ يعرفه هاتفُه
    // — **وهو واقفٌ في المكان الذي يريد حفظَه.**
    //
    // **والحقلُ يمتلئ أمام عينه** فيرى أنّ شيئاً وقع.
    Spacer(Modifier.height(8.dp))
    OutlinedTextField(
        value = pinned?.let { fmtPoint(it.first, it.second) } ?: "",
        onValueChange = {},
        readOnly = true,
        label = { Text(stringResource(R.string.acc_addr_point)) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(6.dp))
    if (pinned != null) {
        Text(
            stringResource(R.string.acc_addr_pinned),
            color = StateGreen,
            style = MaterialTheme.typography.bodySmall,
        )
    } else if (live == null) {
        Text(
            stringResource(R.string.acc_addr_need_point),
            color = StateRed,
            style = MaterialTheme.typography.bodySmall,
        )
    }
    Spacer(Modifier.height(6.dp))
    OutlinedButton(
        onClick = { live?.let { pinned = it.lat to it.lng } },
        enabled = live != null,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            stringResource(
                if (pinned == null) R.string.acc_addr_pin else R.string.acc_addr_repin,
            ),
        )
    }

    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = {
                pinned?.let { vm.addAddress(label, text, it.first, it.second) }
                onDone()
            },
            enabled = !s.busy && pinned != null && label.isNotBlank() && text.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_save)) }
        OutlinedButton(onClick = onDone, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

/**
 * **الإحداثيُّ كما يُقرأ** — ستُّ منازلَ نحوَ عشرةِ سنتيمترات.
 *
 * **ولا يُعرض خاماً بخمسَ عشرةَ منزلة**: سطرٌ لا يُقرأ **يُخيف أكثرَ
 * ممّا يطمئن**، ودقّةٌ زائدةٌ لا يملكها الجهازُ أصلاً.
 */
private fun fmtPoint(lat: Double, lng: Double): String =
    String.format(java.util.Locale.US, "%.6f, %.6f", lat, lng)

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
    HorizontalDivider()
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

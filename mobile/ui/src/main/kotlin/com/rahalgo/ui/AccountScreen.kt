package com.rahalgo.ui

import androidx.activity.compose.rememberLauncherForActivityResult
import com.rahalgo.design.Rahal
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
import androidx.activity.compose.BackHandler
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
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
/**
 * ══════════════════════════════════════════════════════════════════════
 * **منتقي نقطةٍ يُعطى من خارج** — والشاشةُ لا تعرف خريطة
 * ══════════════════════════════════════════════════════════════════════
 *
 * **و`:map` تعتمد على `:ui`** — ولو استوردتها هذه **لصار الاعتمادُ
 * دائريّاً.**
 *
 * **والأثقلُ**: `:ui` يحملها كلُّ تطبيق، **ومكتبةُ الخرائط أصليّةٌ
 * ثقيلة** — فمن لا يرسم خريطةً يحملها بلا سبب.
 *
 * **فالشاشةُ تصف ما تريد** (نقطةً واسماً) **ولا تعرف من يُعطيها.**
 */
typealias PointPicker = @Composable (
    onPick: (lat: Double, lng: Double, label: String) -> Unit,
    onCancel: () -> Unit,
) -> Unit

@Composable
fun AccountScreen(
    vm: AccountViewModel,
    onLoggedOut: () -> Unit,
    /**
     * **أهذا تطبيقُ عامل؟** — سائقٍ أو مندوبٍ أو متجر.
     *
     * **وسببُ توثيق واتساب يختلف**: العاملُ لا يفتح دوامَه بلاه،
     * **والزبونُ يستعيد به حسابَه وتصله أخبارُ طلبه.**
     *
     * **ونصٌّ لا يخصّ قارئَه أسوأُ من نصٍّ ناقص** — يظنّ أنّه في
     * التطبيق الخطأ، **أو أنّ عليه عملاً لا يعرفه.** (قِيس على الجهاز:
     * تطبيقُ الزبون كان يقول «لا تفتح دوامك قبل التوثيق».)
     */
    worker: Boolean = true,
    /**
     * **أفي هذا التطبيق إشعاراتٌ أصلا؟**
     *
     * **وقسمُ فحصِ الإشعارات في تطبيقٍ بلا نقطةٍ يقول «لا جهازَ
     * مسجَّل»** — فيُقرأ عطباً وهو ليس بعطب: **الميزةُ لم تُبنَ بعد.**
     */
    push: Boolean = true,
    /**
     * **منتقي نقطةٍ على خريطة** — أو فارغ.
     *
     * **وفارغُه يُبقي «حدّد موقعي» وحدَه**: **ومن أراد أن يحفظ بيتَ
     * أمّه لا يستطيع** — يحفظ موضعَه هو.
     */
    picker: PointPicker? = null,
) {
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

        Identity(vm, s, worker)
        Gap()
        PasswordSection(vm, s)
        Gap()
        AddressesSection(vm, s, picker)
        // ══════════════════════════════════════════════════════════════
        // **ولا فحصَ إشعاراتٍ هنا**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «ألغِه، ما ظلّ ليه داعٍ
        //  بالإعدادات».)
        //
        // **وُضع يوم ٢٠٢٦-٠٨-١٤ لسؤالٍ واحد**: لم يرنّ الجهازُ ولم
        // يكن في المنصّة ما يقول أين وقفت الرسالة. **وقد عُرف** —
        // فبقي الزرُّ بلا سؤالٍ يجيب عنه.
        //
        // **وشاشةُ الحساب يفتحها الناسُ لا المطوّرون** — **وزرٌّ لا
        // يفعل شيئاً يُقرأ عيباً في التطبيق** لا أداةَ فحص.
        //
        // **والبابُ في المحرّك باقٍ** — يُنادى من لوحة الإدارة يومَ
        // يُشكّ في جهاز.
        Gap()
        // **والخطرُ آخرا** — بأمر المالك.
        DangerSection(vm, s)
        Spacer(Modifier.height(32.dp))
    }
}


// ــ الهويّة ــ

@Composable
private fun Identity(vm: AccountViewModel, s: AccountState, worker: Boolean) {
    val me = s.me ?: return
    var draft by rememberSaveable(me.fullName) { mutableStateOf(me.fullName) }

    // ══════════════════════════════════════════════════════════════════
    // **ومنتقي الصور واحدٌ للحساب وللأصناف**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لازم نسمح للكاميرا بتصويرٍ مباشرٍ أيضاً
    //  وليس فقط صورةً محفوظة».)
    //
    // **وكان هنا منتقٍ ثانٍ يفتح المعرضَ وحدَه** — **ومنتقيان لعملٍ واحدٍ
    // يفترقان**: تُضاف الكاميرا في أحدهما وتُنسى في الآخر، وهو ما وقع.
    //
    // **ويردّ بايتاتٍ مصغَّرةً لا `Uri`** — القراءةُ والتصغيرُ في موضعٍ
    // واحد.
    val picker = rememberImagePicker { bytes -> vm.setAvatarBytes(bytes) }

    // **ولا عنوانَ لأوّل قسم** — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «مكتوب هويّتي
    // من فوق، ألغِها ما يلزم»).
    //
    // **وصورتُه واسمُه يقولان ما هما** — وعنوانٌ فوقهما يسمّي المعروف.
    Row(verticalAlignment = Alignment.CenterVertically) {
        Avatar(url = AppCore.get().media(me.avatarThumbUrl), name = me.fullName, size = 64)
        Spacer(Modifier.size(12.dp))
        Column {
            TextButton(
                onClick = picker,
                enabled = !s.busy,
            ) { Text(stringResource(R.string.acc_photo_pick)) }
            if (!me.avatarThumbUrl.isNullOrEmpty()) {
                TextButton(onClick = vm::removeAvatar, enabled = !s.busy) {
                    Text(stringResource(R.string.acc_photo_remove), color = Rahal.colors.inkMuted)
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
    // **وحقلُ الرقم المسجَّل من المركز أيضا** — بأيقونة الهاتف وشكلِ
    // كتابته — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «الرقم المسجَّل بحساب المستخدم
    // يجب أن يُكتب بشكلٍ تلقائيٍّ بالحقل، وأيضاً طريقة كتابته والأيقونة
    // الخاصّة بالهاتف»).
    //
    // **وحقلٌ مبنيٌّ باليد يفقد ما تحمله المركّبة** — أيقونتَه ومثالَه
    // واتّجاهَ أرقامه، **فيفترق رقمٌ عن رقمٍ في التطبيق نفسِه.**
    PhoneField(
        value = me.phone,
        onChange = {},
        enabled = true,
        label = R.string.acc_phone_current,
        readOnly = true,
    )
    Spacer(Modifier.height(8.dp))
    PhoneChange(vm, s)
    Spacer(Modifier.height(12.dp))
    WhatsAppVerify(vm, s, me.whatsappVerified, worker)
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
private fun WhatsAppVerify(
    vm: AccountViewModel,
    s: AccountState,
    verified: Boolean,
    worker: Boolean,
) {
    var code by remember { mutableStateOf("") }
    val waiting = s.waPending.isNotEmpty()

    Text(
        text = stringResource(
            if (verified) R.string.acc_wa_verified else if (worker) R.string.acc_wa_unverified else R.string.acc_wa_unverified_user,
        ),
        color = if (verified) Rahal.colors.success else Rahal.colors.danger,
        style = MaterialTheme.typography.bodySmall,
    )
    // **والموثَّقُ لا يُدعى إلى فعلٍ تمّ** — زرٌّ باقٍ بعد نجاحه يُقرأ
    // «لم ينجح».
    if (verified && !waiting) return

    Spacer(Modifier.height(6.dp))
    if (!waiting) {
        Text(
            stringResource(if (worker) R.string.acc_wa_hint else R.string.acc_wa_hint_user),
            color = Rahal.colors.inkMuted,
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
        // **وحقلُ الهاتف من المركز** — بأيقونته ومثالِ كتابته
        // (`09xxxxxxxx`) — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «الرقم الجديد يجب
        // أن يكون بداخله طريقةُ كتابة الرقم… ولا تنسَ أيقونة الهاتف»).
        PhoneField(
            value = phone,
            onChange = { phone = it },
            enabled = !s.busy,
            label = R.string.acc_phone_new,
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
    // **وحقلُ كلمة المرور من المركز — بأيقونة العين.**
    PasswordField(current, { current = it }, !s.busy, R.string.acc_pw_current)
    Spacer(Modifier.height(8.dp))
    PasswordField(next, { next = it }, !s.busy, R.string.acc_pw_new)
    Spacer(Modifier.height(8.dp))
    PasswordField(confirm, { confirm = it }, !s.busy, R.string.acc_pw_confirm)

    // **والتطابقُ يُقال قبل الإرسال لا بعده** — نداءٌ يذهب ليعود بخطأٍ
    // يعرفه الجهازُ نفسُه **يُضيّع ثانيتين ويستهلك حزمة.**
    if (mismatch) {
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.acc_pw_mismatch), color = Rahal.colors.danger)
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



// ــ العناوين ــ

/**
 * ══════════════════════════════════════════════════════════════════════
 * **العناوينُ المحفوظة — تُعرض وتُعدَّل وتُحذف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٨: «قسمُ حسابي فقط يُظهر العناوينَ المحفوظة،
 *  ويمكن تعديلها أو حذفها — هكذا يكون العملُ احترافيّاً أكثر».)
 *
 * # والإضافةُ تفتح الخريطةَ أوّلا
 *
 * **الموضعُ قبل الوصف** — (صورُ المالك المرجعيّة): يضع النقطةَ ثمّ
 * يصفها. **ومن كتب عنوانَه ثمّ طُلبت منه نقطتُه كتب وصفاً لمكانٍ لم
 * يحدّده بعد.**
 *
 * # والتعديلُ لا يفتح خريطةً إلّا إن أراد
 *
 * **أكثرُ التصحيح نصٌّ** — طابقٌ أو شارع. **ومن أُلزم بإعادة فتح
 * الخريطة لتصحيح حرفٍ لا يصحّح.**
 */
@Composable
private fun AddressesSection(
    vm: AccountViewModel,
    s: AccountState,
    picker: PointPicker?,
) {
    // **حالٌ واحدةٌ لا رايتان**: فارغٌ قائمة · «add» إضافة · معرّفٌ تعديل.
    // **ورايتان ترتفعان معاً حالٌ لا معنى لها.**
    var mode by rememberSaveable { mutableStateOf("") }

    SectionTitle(stringResource(R.string.acc_addresses))

    if (mode == "add") {
        AddressEditor(vm, s, picker, null, onDone = { mode = "" })
        return
    }
    if (mode.isNotEmpty()) {
        val a = s.addresses.firstOrNull { it.id == mode }
        if (a != null) {
            AddressEditor(vm, s, picker, a, onDone = { mode = "" })
            return
        }
        mode = ""
    }

    if (s.addresses.isEmpty()) {
        Text(stringResource(R.string.acc_addr_empty), color = Rahal.colors.inkMuted)
    }
    s.addresses.forEach { a -> AddressRow(a, vm, s) { mode = a.id } }

    Spacer(Modifier.height(8.dp))
    OutlinedButton(
        onClick = { mode = "add" },
        enabled = !s.busy,
        modifier = Modifier.fillMaxWidth(),
    ) { Text(stringResource(R.string.acc_addr_add)) }
}

/** **سطرُ عنوانٍ محفوظ** — نوعُه وسطرُه وأفعالُه الثلاثة. */
@Composable
private fun AddressRow(
    a: Address,
    vm: AccountViewModel,
    s: AccountState,
    onEdit: () -> Unit,
) {
    Spacer(Modifier.height(8.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            text = stringResource(addressKindLabel(a.kind)),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
        if (a.isDefault) {
            Spacer(Modifier.size(8.dp))
            Text(
                stringResource(R.string.acc_addr_default),
                color = Rahal.colors.success,
                style = MaterialTheme.typography.labelSmall,
            )
        }
    }
    Text(a.text, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
    Row {
        if (!a.isDefault) {
            TextButton(onClick = { vm.makeDefault(a.id) }, enabled = !s.busy) {
                Text(stringResource(R.string.acc_addr_make_default))
            }
        }
        TextButton(onClick = onEdit, enabled = !s.busy) {
            Text(stringResource(R.string.addr_edit))
        }
        TextButton(onClick = { vm.deleteAddress(a.id) }, enabled = !s.busy) {
            Text(stringResource(R.string.acc_addr_delete), color = Rahal.colors.danger)
        }
    }
    HorizontalDivider()
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ عنوان — خريطةٌ ثمّ ورقةُ وصف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (صورُ المالك المرجعيّة ٢٠٢٦-٠٨-١٨.)
 *
 * **ولا رقمَ هاتفٍ فيه** — (قرارُ المالك: «رقمُ الهاتف ما يلزم لأنّه
 * موجودٌ عندنا أساساً»). **وحقلٌ يُطلب مرّتين يُملأ مرّةً بخطأ.**
 */
@Composable
internal fun AddressEditor(
    vm: AccountViewModel,
    s: AccountState,
    picker: PointPicker?,
    /** **عنوانٌ يُعدَّل** — أو فارغٌ لجديد. */
    existing: Address?,
    onDone: () -> Unit,
    /**
     * **أهو صفحةٌ قائمةٌ بذاتها؟**
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «تفتح صفحةُ الخريطة بشكلٍ كاملٍ ومنفصل،
     *  وليس فوق صفحة الطلبات أو الإعدادات».)
     *
     * **ومن الشريط صفحةٌ** — تملأ الشاشةَ وتحمل تمريرَها وحشوتَها،
     * **ويغلقها الرجوع.**
     *
     * **ومن «حسابي» جزءٌ من شاشة** — **وعمودٌ يمرّر داخل عمودٍ يمرّر
     * يُجمّد أحدَهما**، فتُترك الحشوةُ والتمريرُ لصاحب الشاشة.
     */
    standalone: Boolean = false,
) {
    var area by rememberSaveable(existing?.id) { mutableStateOf(existing?.areaBuilding ?: "") }
    var street by rememberSaveable(existing?.id) { mutableStateOf(existing?.street ?: "") }
    var floor by rememberSaveable(existing?.id) { mutableStateOf(existing?.floor ?: "") }
    var kind by rememberSaveable(existing?.id) { mutableStateOf(existing?.kind ?: "home") }
    var lat by rememberSaveable(existing?.id) { mutableStateOf(existing?.lat) }
    var lng by rememberSaveable(existing?.id) { mutableStateOf(existing?.lng) }
    // **والجديدُ يفتح الخريطةَ أوّلاً** — الموضعُ قبل الوصف.
    var onMap by rememberSaveable(existing?.id) { mutableStateOf(existing == null) }

    // **والرجوعُ يغلق الصفحةَ لا التطبيق** — حارسٌ واحدٌ للحالين:
    // في الخريطة يعود إلى النموذج، وفي النموذج يُغلق.
    if (standalone) {
        BackHandler { if (onMap && existing == null) onDone() else onDone() }
    }

    if (onMap && picker != null) {
        picker(
            { la, ln, name ->
                lat = la
                lng = ln
                // **واسمُ المكان يملأ المنطقةَ إن كانت فارغة** — ولا
                // يمحو ما كتبه بيده.
                if (area.isBlank() && name.isNotBlank()) area = name
                onMap = false
            },
            { if (existing == null) onDone() else onMap = false },
        )
        return
    }

    // **وحين يكون صفحةً يحمل حشوتَه وتمريرَه** — انظر `standalone`.
    val wrap: Modifier = if (standalone) {
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    } else {
        Modifier
    }

    Column(wrap) {
    Text(
        stringResource(R.string.addr_confirm_title),
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.titleMedium,
    )
    Text(
        stringResource(R.string.addr_confirm_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
    )

    Spacer(Modifier.height(12.dp))
    OutlinedTextField(
        value = area,
        onValueChange = { area = it },
        label = {
            Text(
                stringResource(R.string.addr_area) + " " +
                    stringResource(R.string.addr_required),
            )
        },
        placeholder = { Text(stringResource(R.string.addr_area_hint)) },
        modifier = Modifier.fillMaxWidth(),
    )

    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedTextField(
            value = street,
            onValueChange = { street = it },
            label = { Text(stringResource(R.string.addr_street)) },
            placeholder = { Text(stringResource(R.string.addr_street_hint)) },
            singleLine = true,
            modifier = Modifier.weight(2f),
        )
        OutlinedTextField(
            value = floor,
            onValueChange = { v -> floor = v.filter { c -> c.isDigit() } },
            label = { Text(stringResource(R.string.addr_floor)) },
            placeholder = { Text(stringResource(R.string.addr_floor_hint)) },
            singleLine = true,
            modifier = Modifier.weight(1f),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **ونوعُ العنوان ثلاثُ بطاقاتٍ لا حقلُ نصّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وحقلٌ حرٌّ يجعل لكلّ زبونٍ تسميتَه** — «البيت» و«بيتي» و«المنزل»
    // ثلاثةُ أسماءٍ لشيءٍ واحد، **ولا يُفرز ولا يُعرض بأيقونة.**
    Spacer(Modifier.height(12.dp))
    Text(stringResource(R.string.addr_kind), fontWeight = FontWeight.Medium)
    Spacer(Modifier.height(6.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        for (k in listOf("home", "work", "other")) {
            val on = kind == k
            OutlinedButton(
                onClick = { kind = k },
                modifier = Modifier.weight(1f),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
                ),
            ) { Text(stringResource(addressKindLabel(k))) }
        }
    }

    // **وتغييرُ الموضع بابٌ ظاهر** — انظر أعلى الدالّة.
    if (picker != null) {
        Spacer(Modifier.height(8.dp))
        TextButton(onClick = { onMap = true }) {
            Text(stringResource(R.string.addr_change_point))
        }
    }

    Spacer(Modifier.height(12.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(
            onClick = {
                if (existing == null) {
                    vm.addAddress(area, street, floor, kind, lat, lng)
                } else {
                    vm.updateAddress(existing.id, area, street, floor, kind, lat, lng)
                }
                onDone()
            },
            // **ولا يُحفظ عنوانٌ بلا نقطة** — نقطةٌ صفريّةٌ تُرسل سائقاً
            // إلى لا مكان.
            enabled = !s.busy && area.isNotBlank() && lat != null && lng != null,
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.addr_save)) }
        OutlinedButton(onClick = onDone, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
    }
}

// ــ الحذف ــ

@Composable
private fun DangerSection(vm: AccountViewModel, s: AccountState) {
    var code by remember { mutableStateOf("") }

    SectionTitle(stringResource(R.string.acc_danger), Rahal.colors.danger)
    Text(
        stringResource(R.string.acc_delete_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(10.dp))

    if (!s.deleteAsked) {
        // **وأحمرُ ممتلئٌ لا إطارٌ بنصٍّ أحمر** — (قرارُ المالك
        // ٢٠٢٦-٠٨-١٣: «وأرسل رمزاً للحذف يجب أن يكون الزرُّ أحمرَ ليكون
        // زرَّ الخطر»).
        //
        // **والإطارُ يُقرأ اختيارا** بين أزرارٍ كثيرةٍ إطارُها واحد،
        // **والممتلئُ يقول: قف.**
        Button(
            onClick = vm::askDelete,
            enabled = !s.busy,
            colors = ButtonDefaults.buttonColors(containerColor = Rahal.colors.danger),
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_delete_ask)) }
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
            colors = ButtonDefaults.buttonColors(containerColor = Rahal.colors.danger),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_delete_confirm)) }
        OutlinedButton(onClick = vm::cancelDelete, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

// ــ قطعٌ صغيرة ــ

@Composable
private fun SectionTitle(text: String, color: androidx.compose.ui.graphics.Color = Rahal.colors.accent) {
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



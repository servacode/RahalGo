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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.foundation.layout.heightIn
import androidx.compose.ui.res.painterResource
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
    /**
     * **أيُعرض قسمُ العناوين؟**
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٨: «حساب السائق أصلاً لا يلزمه خيارُ
     *  عناويني، لأنّه لا يضيف عناوينَه في حساب السائق».)
     *
     * **والعنوانُ حاجةُ من يُوصَّل إليه** — والسائقُ يُوصِّل. **وقسمٌ
     * يُفتح فلا يُملأ أبداً يُقرأ نقصاً في الحساب** لا اختياراً.
     *
     * **وافتراضُه الإظهار**: الزبونُ هو الغالب، **وتطبيقٌ يُنسى فيه
     * الوسيطُ يفقد عناوينَه صامتا.**
     */
    showAddresses: Boolean = true,
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
     * **أيُعرض «وثّق حسابك» بالواتساب؟** (قرارُ المالك ٢٠٢٦-١٠-٢٩)
     *
     * **تطبيقُ المتجر يُخفيه** — حسابُ المتجر تصله رسائلُ الواتساب أصلاً، **فبندٌ
     * يطلب توثيقَ ما يعمل يُقرأ نقصاً لا وجودَ له.** وافتراضُه الإظهار.
     */
    whatsapp: Boolean = true,
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
    /**
     * ══════════════════════════════════════════════════════════════════
     * **من يفتح محرّرَ العنوان — الشاشةُ أم التطبيق؟**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «الزرُّ الموجود بحسابي يجب أن يعمل بنفس
     *  الطريقة التي عملنا بها — يفتح نفس الموقع بنفس الطريقة، والتعديل
     *  أيضاً».)
     *
     * **وصفحةُ الخريطة تملأ الشاشةَ ولا تُرسم داخل عمودٍ يمرّر** —
     * **وارتفاعٌ لا حدَّ له داخل تمريرٍ يُسقط الرسم.** فمن أراد صفحةً
     * كاملةً يرفعها إلى مستوى التطبيق، **والشاشةُ تبلّغ ولا تفتح.**
     *
     * **وفارغٌ يعني افتحها في مكانك** — السائقُ والمندوبُ لا صفحةَ
     * عنوانٍ لهما، **ولا يُغيَّر ما يعمل عندهما لطلبٍ يخصّ الزبون.**
     */
    onEditAddress: ((Address?) -> Unit)? = null,
    /**
     * **رأسُ حسابٍ بصورةٍ عليها كاميرا واسمٍ ورقم، وكلمةُ المرور مطويّةٌ خلف صفّ** (تدقيق D4، ٢٠٢٦-١١-١١).
     * **وهو الشكلُ الموحّدُ للتطبيقات الأربعة** (قرارُ المالك ٢٠٢٦-١١-١١: «اكيد رح نوحد كلشي
     * عند الزبون والسائق والمتجر والوكيل»).
     */
    profileHeader: Boolean = true,
) {
    val s = vm.state

    // **والخروجُ يقع حين يُحذف الحساب** — لا شاشةَ لمن لا حسابَ له.
    LaunchedEffect(s.deleted) { if (s.deleted) onLoggedOut() }

    if (s.loading) {
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) { RahalLoader() }
        return
    }

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Rahal.space.lg)
            .padding(bottom = LocalChatFabInset.current),
    ) {

        // **كلُّ قسمٍ بطاقة** (مراجعةُ التصميم ٢٠٢٦-١١-٠٩) — بدل فواصلَ خطّيّةٍ بين كتلٍ عارية.
        if (profileHeader) {
            RahalCard { ProfileHeader(vm, s) }
            Gap()
        }
        RahalCard { Identity(vm, s, worker, whatsapp, polished = profileHeader) }
        Gap()
        RahalCard { if (profileHeader) CollapsedPassword(vm, s) else PasswordSection(vm, s) }
        Gap()
        // **ولا عناوينَ في حساب السائق** — انظر `showAddresses`.
        if (showAddresses) {
            RahalCard { AddressesSection(vm, s, picker, onEditAddress) }
            Gap()
        }
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
        // **والخطرُ آخرا** — بأمر المالك.
        RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Danger) { DangerSection(vm, s) }
        Spacer(Modifier.height(Rahal.space.xl))
        // ══════════════════════════════════════════════════════════════
        // **وذكرُ مصدر الخرائط — يُؤدَّى هنا لا فوق الطريق**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٢٤: «افعل الرخصة بشكلٍ صحيح بدون أن
        //  يكون واضحاً على خريطتنا».)
        //
        // **ورخصةُ OpenStreetMap (ODbL) تُلزم بذكر المساهمين** —
        // وبياناتُ خريطتنا كلُّها منها. **وشعارُ MapLibre اختيارٌ لا
        // إلزام** (رخصتُه BSD)، فأُطفئ.
        //
        // **ومن حذفه من الموضعين خالف رخصةً** — وهذا موضعُه.
        MapCredit()
        Spacer(Modifier.height(Rahal.space.xxl))
    }
}


// ــ الهويّة ــ

@Composable
private fun Identity(vm: AccountViewModel, s: AccountState, worker: Boolean, whatsapp: Boolean, polished: Boolean = false) {
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
    if (!polished) Row(verticalAlignment = Alignment.CenterVertically) {
        Avatar(url = AppCore.get().media(me.avatarThumbUrl), name = me.fullName, size = 64)
        Spacer(Modifier.size(Rahal.space.md))
        Column {
            RahalTextButton(
                onClick = picker,
                enabled = !s.busy,
            ) { Text(stringResource(R.string.acc_photo_pick)) }
            if (!me.avatarThumbUrl.isNullOrEmpty()) {
                RahalTextButton(onClick = vm::removeAvatar, enabled = !s.busy) {
                    Text(stringResource(R.string.acc_photo_remove), color = Rahal.colors.inkMuted)
                }
            }
        }
    }

    Spacer(Modifier.height(Rahal.space.md))
    LabeledField(
        label = stringResource(R.string.acc_name),
        value = draft,
        onValueChange = { draft = it },
    )
    Spacer(Modifier.height(Rahal.space.sm))
    SaveButton(
        onClick = { vm.setName(draft) },
        enabled = !s.busy && draft.trim().isNotEmpty() && draft.trim() != me.fullName,
        polished = polished,
    )

    Spacer(Modifier.height(Rahal.space.md))
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
    Spacer(Modifier.height(Rahal.space.sm))
    PhoneChange(vm, s)
    if (whatsapp) {
        Spacer(Modifier.height(Rahal.space.md))
        WhatsAppVerify(vm, s, me.whatsappVerified, worker)
    }
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
    var code by rememberSaveable { mutableStateOf("") }
    val waiting = s.waPending.isNotEmpty()

    // **ونصٌّ واحدٌ للجميع** — **كان للعامل نصٌّ وللزبون آخر** لأنّ سببَ
    // التوثيق يختلف. **والفعلُ واحدٌ فالنصُّ واحد.**
    // (قرارُ المالك ٢٠٢٦-٠٨-٣٠.)
    Text(
        text = stringResource(
            if (verified) R.string.acc_wa_verified else R.string.acc_wa_unverified,
        ),
        color = if (verified) Rahal.colors.success else Rahal.colors.danger,
        style = Rahal.type.caption,
    )
    // **والموثَّقُ لا يُدعى إلى فعلٍ تمّ** — زرٌّ باقٍ بعد نجاحه يُقرأ
    // «لم ينجح».
    if (verified && !waiting) return

    Spacer(Modifier.height(Rahal.space.xs))
    if (!waiting) {
        // **ولا شرحَ قبل الزرّ** — **كان يصف ما سيراه بعد ضغطةٍ واحدة**
        // («سيفتح واتساب برسالة جاهزة…»)، **وشرحُ ما يُرى بعد ثانيةٍ
        // يُقرأ مرّةً ثمّ يُتخطّى في كلّ مرّة.**
        // (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «شرحُ توثيق الحساب لا داعيَ له
        //  أصلاً بكلّ النسخ».)
        RahalButton(
            onClick = vm::askWhatsApp,
            enabled = !s.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_wa_verify)) }
        return
    }

    LabeledField(
        label = stringResource(R.string.acc_wa_code),
        value = code,
        onValueChange = { code = it },
        keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
    )
    // **ومن لم يصله ردُّ البوت يطلب تذكرةً جديدة** — فيُفتح واتساب من
    // جديد (`askWhatsApp`)، **والحقلُ يُمحى قبله.**
    ResendCodeRow(
        sentAt = s.waSentAt,
        busy = s.busy,
        onResend = {
            code = ""
            vm.askWhatsApp()
        },
    )
    Spacer(Modifier.height(Rahal.space.xs))
    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        RahalButton(
            onClick = { vm.confirmWhatsApp(code); code = "" },
            enabled = !s.busy && code.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_wa_confirm)) }
        RahalOutlineButton(
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
    var code by rememberSaveable { mutableStateOf("") }
    val waiting = s.phonePending.isNotEmpty()

    if (!open && !waiting) {
        // **وزرٌّ بعرض الشاشة لا نصٌّ يُبحث عنه** — بأمر المالك.
        RahalOutlineButton(
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
        Spacer(Modifier.height(Rahal.space.sm))
        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            RahalButton(
                onClick = { vm.askPhone(phone) },
                enabled = !s.busy && phone.isNotBlank(),
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.acc_phone_send)) }
            RahalOutlineButton(
                onClick = { open = false; phone = "" },
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.acc_delete_cancel)) }
        }
        return
    }

    LabeledField(
        label = stringResource(R.string.acc_phone_code),
        value = code,
        onValueChange = { code = it },
        keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        RahalButton(
            onClick = { vm.confirmPhone(code); code = ""; open = false; phone = "" },
            enabled = !s.busy && code.isNotBlank(),
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_phone_confirm)) }
        RahalOutlineButton(
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
    // **والحقولُ تُفرَّغ بعد النجاح وحدَه** — لا عند الضغط: من أخطأ في
    // الحاليّة يصحّحها ولا يعيد كتابةَ الجديدة مرّتين.
    LaunchedEffect(s.pwDone) {
        if (s.pwDone > 0) { current = ""; next = ""; confirm = "" }
    }
    PasswordField(current, { current = it; vm.clearPwError() }, !s.busy, R.string.acc_pw_current)
    Spacer(Modifier.height(Rahal.space.sm))
    PasswordField(next, { next = it; vm.clearPwError() }, !s.busy, R.string.acc_pw_new)
    Spacer(Modifier.height(Rahal.space.sm))
    PasswordField(confirm, { confirm = it; vm.clearPwError() }, !s.busy, R.string.acc_pw_confirm)

    // **والتطابقُ يُقال قبل الإرسال لا بعده** — نداءٌ يذهب ليعود بخطأٍ
    // يعرفه الجهازُ نفسُه **يُضيّع ثانيتين ويستهلك حزمة.**
    if (mismatch) {
        Spacer(Modifier.height(Rahal.space.xs))
        Text(stringResource(R.string.acc_pw_mismatch), color = Rahal.colors.danger)
    }
    Spacer(Modifier.height(Rahal.space.sm))
    RahalButton(
        onClick = { vm.setPassword(current, next) },
        enabled = !s.busy && current.isNotEmpty() && next.isNotEmpty() && !mismatch,
        modifier = Modifier.fillMaxWidth(),
    ) { Text(stringResource(R.string.acc_pw_change)) }
    if (s.pwError.isNotEmpty()) {
        Spacer(Modifier.height(Rahal.space.xs))
        Text(s.pwError, color = Rahal.colors.danger)
    }
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
    onEditAddress: ((Address?) -> Unit)?,
) {
    // **حالٌ واحدةٌ لا رايتان**: فارغٌ قائمة · «add» إضافة · معرّفٌ تعديل.
    // **ورايتان ترتفعان معاً حالٌ لا معنى لها.**
    var mode by rememberSaveable { mutableStateOf("") }

    SectionTitle(stringResource(R.string.acc_addresses))

    if (mode == "add") {
        AddressEditor(vm, s, picker, null, onDone = { mode = "" })
        return
    }
    // **ولا يقع هذا إن كان التطبيقُ يفتحها** — يُبلَّغ ويُترك له.

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
    s.addresses.forEach { a ->
        AddressRow(a, vm, s) {
            if (onEditAddress != null) onEditAddress(a) else mode = a.id
        }
    }

    Spacer(Modifier.height(Rahal.space.sm))
    RahalOutlineButton(
        onClick = { if (onEditAddress != null) onEditAddress(null) else mode = "add" },
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
    // ══════════════════════════════════════════════════════════════
    // **وكلُّ عنوانٍ في بطاقته — وأزرارُه تُقرأ أزرارا**
    // ══════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «العناوينُ بحسابي أيضاً اجعل كلَّ
    //  عنوانٍ بحقلٍ واضح، والأزرارَ تحته واضحةً أنّها أزرارٌ وليست
    //  مجرّدَ كتابةٍ عاديّة».)
    //
    // **وكانت ثلاثةَ نصوصٍ فوق خطّ فاصل** — **ونصٌّ ملوّنٌ بلا إطارٍ
    // يُقرأ وسماً**، فيُقرأ «حذف» حالاً لا فعلاً.
    //
    // **والحذفُ محدَّدٌ بالأحمر لا نصّاً أحمر**: **فعلٌ لا يُستدرَك
    // خلفَ نصٍّ عارٍ يُضغط بالخطأ.**
    Spacer(Modifier.height(Rahal.space.md))
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(Rahal.space.md),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = stringResource(addressKindLabel(a.kind)),
                fontWeight = FontWeight.Bold,
                style = Rahal.type.body,
            )
            if (a.isDefault) {
                Spacer(Modifier.size(Rahal.space.sm))
                Chip(stringResource(R.string.acc_addr_default), Rahal.colors.success)
            }
        }
        Spacer(Modifier.height(Rahal.space.xs))
        Text(a.text, color = Rahal.colors.inkMuted, style = Rahal.type.caption)

        // ══════════════════════════════════════════════════════════════
        // **وأزرارٌ ثلاثةٌ في صفٍّ تقصّ نصوصَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٨: «إعداد العناوين كافتراضي تطلع
        //  مقصوصة، اجعلها غير مقصوصةٍ بطريقةٍ احترافيّة».)
        //
        // **وقُصِّر النصُّ قبلها إلى «كافتراضي» ولم يكفِ** — ثلاثةُ
        // أزرارٍ بحشوةٍ عشرين نقطةً لكلٍّ منها **تأكل العرضَ قبل
        // النصّ.**
        //
        // # فيُفصل الفعلُ الأوّلُ عن الاثنين
        //
        // **«كافتراضي» فعلٌ من نوعٍ آخر**: يقول «هذا عنواني» — والتعديلُ
        // والحذفُ يدبّران البطاقة. **فيأخذ سطرَه كاملاً ويأخذان سطرَهما**،
        // ولا يُقصّ واحدٌ منها.
        //
        // **ولا يظهر للافتراضيّ أصلا** — فمن كان افتراضيّاً فصفّان
        // يصيران واحدا.
        Spacer(Modifier.height(Rahal.space.md))
        if (!a.isDefault) {
            RahalOutlineButton(
                onClick = { vm.makeDefault(a.id) },
                enabled = !s.busy,
                tone = Tone.Success,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.acc_addr_make_default), maxLines = 1) }
            Spacer(Modifier.height(Rahal.space.sm))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            RahalOutlineButton(
                onClick = onEdit,
                enabled = !s.busy,
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.addr_edit), maxLines = 1) }
            RahalOutlineButton(
                onClick = { vm.deleteAddress(a.id) },
                enabled = !s.busy,
                tone = Tone.Danger,
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.acc_addr_delete), maxLines = 1) }
        }
    }
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
// ــ الحذف ــ

@Composable
private fun DangerSection(vm: AccountViewModel, s: AccountState) {
    var code by rememberSaveable { mutableStateOf("") }

    SectionTitle(stringResource(R.string.acc_danger), Rahal.colors.danger)
    Text(
        stringResource(R.string.acc_delete_hint),
        color = Rahal.colors.inkMuted,
        style = Rahal.type.caption,
    )
    Spacer(Modifier.height(Rahal.space.md))

    if (!s.deleteAsked) {
        // **وأحمرُ ممتلئٌ لا إطارٌ بنصٍّ أحمر** — (قرارُ المالك
        // ٢٠٢٦-٠٨-١٣: «وأرسل رمزاً للحذف يجب أن يكون الزرُّ أحمرَ ليكون
        // زرَّ الخطر»).
        //
        // **والإطارُ يُقرأ اختيارا** بين أزرارٍ كثيرةٍ إطارُها واحد،
        // **والممتلئُ يقول: قف.**
        RahalButton(
            onClick = vm::askDelete,
            enabled = !s.busy,
            tone = Tone.Danger,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_delete_ask)) }
        return
    }

    LabeledField(
        label = stringResource(R.string.acc_delete_code),
        value = code,
        onValueChange = { code = it },
        keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        RahalButton(
            onClick = { vm.confirmDelete(code) },
            enabled = !s.busy && code.isNotBlank(),
            tone = Tone.Danger,
            modifier = Modifier.weight(1f),
        ) { Text(stringResource(R.string.acc_delete_confirm)) }
        RahalOutlineButton(onClick = vm::cancelDelete, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

// ــ قطعٌ صغيرة ــ

/**
 * **سطرُ مصدر الخرائط.**
 *
 * **ونصٌّ لا رابط** — **ورابطٌ يفتح متصفّحاً من شاشة حسابٍ يُقلق**،
 * والذكرُ هو المطلوب لا النقر.
 */
@Composable
private fun MapCredit() {
    Text(
        text = "الخرائط © مساهمو OpenStreetMap · OpenMapTiles · MapLibre",
        style = Rahal.type.caption,
        color = Rahal.colors.inkMuted,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun SectionTitle(text: String, color: androidx.compose.ui.graphics.Color = Rahal.colors.accentInk) {
    Text(text, style = Rahal.type.sectionTitle, color = color)
    Spacer(Modifier.height(Rahal.space.md))
}

@Composable
private fun Gap() {
    Spacer(Modifier.height(Rahal.space.lg))
}

@Composable
private fun Notice(text: String, color: androidx.compose.ui.graphics.Color) {
    Text(
        text = text,
        color = color,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth().padding(bottom = Rahal.space.md),
    )
}



/**
 * **زرُّ «حفظ» يُرى معطَّلاً أو مفعَّلاً بلمحة** (تدقيق D4) — المعطَّلُ كان البرتقاليَّ نفسَه
 * بشفافيّةٍ فيُظنّ مفعَّلاً باهتاً. **فالمعطَّلُ هنا أرضٌ رماديّةٌ وحدٌّ ونصٌّ خافت**، والمفعَّلُ
 * معبّى بلون العلامة. (`polished` للزبون؛ وغيره يبقى كما كان.)
 */
@Composable
private fun SaveButton(onClick: () -> Unit, enabled: Boolean, polished: Boolean) {
    if (!polished) {
        RahalButton(
            onClick = onClick,
            enabled = enabled,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.acc_save)) }
        return
    }
    ClearStateButton(onClick = onClick, enabled = enabled) { Text(stringResource(R.string.acc_save)) }
}

/**
 * **زرٌّ يُرى مفعَّلاً أو معطَّلاً بلمحة** (تدقيق D4/D18، ٢٠٢٦-١١-١١) — المعطَّلُ في `RahalButton`
 * هو لونُه نفسُه بشفافيّة ٣٠٪ فيُظنّ مفعَّلاً باهتاً أو يضيع نصُّه. **فالمعطَّلُ هنا أرضٌ رماديّةٌ وحدٌّ
 * ونصٌّ بتباينٍ كافٍ لا يُلمَس**، والمفعَّلُ معبّى بلون نبرته كالمعتاد. (يتبنّاه الزبونُ وحدَه الآن.)
 */
@Composable
fun ClearStateButton(
    onClick: () -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier.fillMaxWidth(),
    tone: Tone = Tone.Brand,
    content: @Composable androidx.compose.foundation.layout.RowScope.() -> Unit,
) {
    if (enabled) {
        RahalButton(onClick = onClick, modifier = modifier, tone = tone, content = content)
        return
    }
    Row(
        modifier
            .heightIn(min = Rahal.size.touch)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.field)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        androidx.compose.runtime.CompositionLocalProvider(
            androidx.compose.material3.LocalContentColor provides Rahal.colors.inkMuted,
        ) {
            androidx.compose.material3.ProvideTextStyle(Rahal.type.button) { content() }
        }
    }
}

/**
 * **رأسُ الحساب: صورةٌ واسمٌ ورقم** (تدقيق D4) — كان الحسابُ نموذجاً طويلاً مفتوحاً بلا
 * «من أنا». **والرقمُ معزولٌ بـ`phoneText` فلا ينقلب اتّجاهُه في سطرٍ عربيّ.**
 * (إعادةُ التصميم ٢٠٢٦-١١-١١: صورةٌ بكاميرا عليها، واسمٌ وتحته الرقم، و«إزالة الصورة» سطرٌ صغير.)
 */
@Composable
private fun ProfileHeader(vm: AccountViewModel, s: AccountState) {
    val me = s.me ?: return
    val picker = rememberImagePicker { bytes -> vm.setAvatarBytes(bytes) }
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        AvatarWithEdit(
            url = AppCore.get().media(me.avatarThumbUrl),
            name = me.fullName,
            description = stringResource(R.string.acc_photo_pick),
            enabled = !s.busy,
            onPick = picker,
        )
        Column(Modifier.weight(1f)) {
            Text(
                text = me.fullName,
                style = Rahal.type.screenTitle,
                color = Rahal.colors.ink,
                maxLines = 2,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
            )
            Text(
                text = phoneText(me.phone),
                style = Rahal.type.body,
                color = Rahal.colors.inkMuted,
                maxLines = 1,
            )
            if (!me.avatarThumbUrl.isNullOrEmpty()) {
                RahalTextButton(onClick = vm::removeAvatar, enabled = !s.busy) {
                    Text(stringResource(R.string.acc_photo_remove), color = Rahal.colors.inkMuted)
                }
            }
        }
    }
}

/**
 * **كلمةُ المرور خلف صفٍّ واحد** (تدقيق D4) — ثلاثةُ حقولٍ مفتوحةٍ دائماً تُطيل الصفحةَ
 * لفعلٍ يقع مرّةً في الشهر. **وتنطوي حين يتمّ التغيير.**
 */
@Composable
private fun CollapsedPassword(vm: AccountViewModel, s: AccountState) {
    var open by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(s.pwDone) { if (s.pwDone > 0) open = false }
    ListRow(
        title = stringResource(R.string.acc_pw_change),
        icon = R.drawable.ic_lock,
        trailing = {
            Icon(
                painter = painterResource(R.drawable.ic_chevron_down),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
                modifier = Modifier
                    .size(Rahal.icon.sm)
                    .rotate(if (open) 180f else 0f),
            )
        },
        onClick = { open = !open },
    )
    if (open) {
        RahalDivider()
        Spacer(Modifier.height(Rahal.space.md))
        PasswordSection(vm, s)
    }
}

package com.rahalgo.customer.custom

import android.app.Application
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.NewCustom
import com.rahalgo.ui.AppCore
import com.rahalgo.shared.model.Address
import com.rahalgo.ui.AddressCard
import com.rahalgo.ui.Flash
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.apiError
import com.rahalgo.ui.RahalButton
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ خاصّ — ما ليس في المنصّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «الزبون يريد أكلاً من مطعمٍ محدَّد أو شيئاً
 *  من سوقٍ غير موجودٍ بالمتجر».)
 *
 * # ولا يُسأل عن سعرٍ ولا متجر
 *
 * **هو يطلب ما لا نعرف سعرَه** — والسائقُ يشتريه ويتّفق معه في المحادثة
 * بعد الإسناد. **وسؤالٌ لا جوابَ له يُوقف من يملأ نموذجا.**
 *
 * # والعنوانُ يُكتب ويُؤشَّر — كما في السلّة حرفاً بحرف
 *
 * **ونموذجان للعنوان يفترقان**: يُشدَّد في أحدهما ويبقى الآخرُ يقبل
 * عنواناً بلا نقطة، **فيُرسل السائقُ إلى لا مكان.**
 */
@Composable
fun CustomScreen(
    vm: CustomViewModel,
    /**
     * **عنوانُ التوصيل المختار** — الافتراضيُّ في حسابه.
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يظهر العنوانُ المحفوظ أو إضافةُ
     *  عنوان».)
     *
     * **وكان يُسأل عنه نصّاً وتُلتقط نقطتُه في كلّ طلب** — **وسؤالُ ما
     * هو معروفٌ يُقرأ عدمَ ثقةٍ لا حرصا.**
     */
    address: Address?,
    /** **يفتح لوحةَ العناوين** — يختار أو يضيف. */
    onOpenAddresses: () -> Unit,
    /** **ما يُنادى بعد نجاح الطلب** — الانتقالُ إلى «طلباتي». */
    onSent: () -> Unit = {},
) {
    var request by rememberSaveable { mutableStateOf("") }
    var notes by rememberSaveable { mutableStateOf("") }

    // ══════════════════════════════════════════════════════════════════
    // **ونجاحُ الإرسال يُفرِّغ ما كُتب ثمّ ينتقل**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٨ — انظر `CustomViewModel.send`.)
    //
    // **والتفريغُ هنا لا في النموذج**: الحقولُ حالُ شاشةٍ (`rememberSaveable`)
    // **ولا يملكها النموذج** — يعرف أنّ الإرسال وقع ولا يعرف ماذا كُتب.
    // ══════════════════════════════════════════════════════════════════
    // **وإشارةُ الإرسال تُستهلَك — وإلّا أغلقت القسمَ إلى الأبد**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٨: «بعدما أطلب طلباً خاصّاً يذهب إلى
    //  طلباتي، ثمّ زرُّ الطلب الخاصّ لا يعمل — يومض بدون أن يفتح صفحةَ
    //  الطلب الخاصّ».)
    //
    // # لماذا وقع
    //
    // **النموذجُ يعيش مع التطبيق لا مع الشاشة** — فيبقى `sent = 1` بعد
    // أوّل إرسال. **وكلَّما فُتح القسمُ من جديدٍ سرى الأثرُ فوراً**
    // فنادى `onSent()` فبدّل التبويبَ إلى «طلباتي».
    //
    // **فيومض القسمُ ولا يُفتح** — وهو ما رآه المالك: **لا خطأَ ولا
    // رسالة، بل بابٌ يُفتح ويُغلق في الإطار نفسِه.**
    //
    // # ولماذا يُستهلَك لا يُتذكَّر في الشاشة
    //
    // **راية «عالجتُها» في الشاشة تموت مع الشاشة** — فتعود صفراً عند
    // كلّ دخولٍ فيقع العطبُ نفسُه. **والإشارةُ حدثٌ يقع مرّةً**، ومن
    // أخذها أسقطها.
    //
    // **والرقمُ يبقى متصاعداً لا رايةً تُرفع**: من أرسل طلبين متتاليين
    // يُلتقط ثانيهما — يعود صفراً بعد الاستهلاك ثمّ يصير واحداً.
    LaunchedEffect(vm.sent) {
        if (vm.sent == 0) return@LaunchedEffect
        request = ""
        notes = ""
        vm.consumeSent()
        onSent()
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.nav_custom),
            stringResource(R.string.soon_custom),
        )


        Spacer(Modifier.height(12.dp))
        // **وحقلُ الطلب واسعٌ** — من يكتب «كيلو لحمة من ملحمة أبو أحمد
        // ونصف كيلو خبز» في سطرٍ واحدٍ لا يرى ما كتب.
        OutlinedTextField(
            value = request,
            onValueChange = { request = it },
            label = { Text(stringResource(R.string.cst_what)) },
            // **والمثالُ يقول ما يُطلب** — **وسؤالٌ مفتوحٌ بلا مثالٍ
            // يُوقف من لا يعرف ما تقبله المنصّة.**
            //
            // **وتحت الحقل لا داخلَه**: `placeholder` في Material لا
            // يظهر إلّا بعد أن يُلمس الحقل — **ومن لم يلمسه لم يره
            // فلم يعرف ماذا يكتب**، وهو الموقفُ نفسُه الذي أُصلح.
            supportingText = { Text(stringResource(R.string.cst_what_hint)) },
            minLines = 3,
            modifier = Modifier.fillMaxWidth(),
        )

        // ══════════════════════════════════════════════════════════════
        // **والعنوانُ من حسابه لا من حقلٍ يُملأ كلَّ مرّة**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يعمل بنفس الدور لزرّ إضافة عنوان —
        //  إمّا يختار عنوانَه الافتراضيَّ أو يضيف عنواناً جديداً».)
        //
        // **وكان حقلَ نصٍّ ونقطةً على خريطة** — يكتب عنوانَه في كلّ
        // طلبٍ من جديد ويلتقط نقطتَه من جديد.
        //
        // **وبلا نقطةٍ لا يُرسَل الطلب**: عنوانٌ بإحداثيٍّ صفر نقطةٌ في
        // المحيط الأطلسيّ، **تُرسل سائقاً إلى لا مكان.** والعنوانُ
        // المحفوظُ يحمل نقطتَه معه.
        Spacer(Modifier.height(12.dp))
        AddressCard(address, onOpenAddresses)

        Spacer(Modifier.height(10.dp))
        OutlinedTextField(
            value = notes,
            onValueChange = { notes = it },
            label = { Text(stringResource(R.string.cst_notes)) },
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))
        RahalButton(
            onClick = {
                address?.let { vm.send(request.trim(), it.text, it.lat, it.lng, notes.trim()) }
            },
            enabled = !vm.busy && request.isNotBlank() && address != null,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (vm.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(R.string.cst_send))
            }
        }

        // **ويُقال ما ينقص قبل أن يُضغط** — لا زرٌّ معطّلٌ بلا سبب:
        // **من رأى زرّاً لا يعمل ولا يعرف لماذا أغلق التطبيق.**
        if (address == null) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(R.string.cst_need_point),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(28.dp))
    }
}

class CustomViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    var done by mutableStateOf("")
        private set

    /**
     * **عدّادُ ما أُرسل** — **ترتفع قيمتُه مع كلّ نجاح.**
     *
     * **ولا رايةٌ تُرفع وتُخفَض**: من أرسل طلبين متتاليين لا تلتقط
     * الشاشةُ الثانيَ لأنّ الرايةَ مرفوعةٌ أصلا. **ورقمٌ يتصاعد يُلتقط
     * كلَّ مرّة.**
     */
    var sent by mutableStateOf(0)
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **مفتاحُ المحاولة — يبقى ما دامت لم تنجح**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تدقيقُ الإطلاق ٢٠٢٦-٠٨-١٩ — BUG-001.)
     *
     * **والخطرُ ليس الضغطتين المتتاليتين** — تلك يمنعها `busy`.
     * **الخطرُ أن يُنشأ الطلبُ في الخادم ثمّ تنقطع الشبكةُ قبل الردّ**:
     * يرى «تعذّر» فيضغط ثانيةً — **فطلبان وسائقان وخصمان.**
     *
     * **فيُولَّد مرّةً ويبقى حتّى ينجح** — فإعادةُ المحاولة تحمله نفسَه
     * فيردّ الخادمُ الطلبَ الأوّلَ بعينه (`idempotency.go`).
     *
     * **ويُمحى بعد النجاح** — **ومفتاحٌ يبقى يجعل الطلبَ التالي يردّ
     * جوابَ الذي قبله.**
     */
    private var attemptKey: String? = null


    /** **يُطلب موضعُه الآن** — لا يُقرأ في الخفاء لحظةَ الإرسال. */
    fun locate() {
        com.rahalgo.customer.Here.refresh(getApplication())
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ونجاحُ الطلب يُفرِّغ النموذجَ ويأخذ صاحبَه إلى «طلباتي»**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٨: «بعد نجاح الطلب الخاصّ يجب أن يتفرّغ
     *  الفورمُ تلقائيّاً ثمّ ينتقل إلى قسم الطلبات ليعرف المستخدمُ أنّه
     *  نجح — وليس أن يبقى الفورمُ مفتوحاً والمعلوماتُ السابقةُ موجودةً
     *  فلا يفهم ماذا جرى».)
     *
     * **ونموذجٌ يبقى مملوءاً بعد الإرسال يُقرأ «لم يُرسَل»** — فيضغط
     * صاحبُه ثانيةً وثالثة، **فيصير الطلبُ الواحدُ ثلاثةً** ويخرج ثلاثةُ
     * سائقين إلى بابٍ واحد.
     *
     * **والنبضةُ تُبَثّ قبل الانتقال** — **و«طلباتي» تحمّل مرّةً في
     * عمرها**: من انتقل إليها بلا نبضةٍ رأى قائمةً لا طلبَ فيها،
     * **فأغلق التطبيقَ وفتحه ليرى طلبَه** (وهو ما وقع للمالك).
     */
    fun send(request: String, address: String, lat: Double, lng: Double, notes: String) {
        if (busy) return
        busy = true
        error = ""
        done = ""
        // **ولا يُولَّد إن كان قائماً** — محاولةٌ ثانيةٌ لطلبٍ واحد.
        val key = attemptKey ?: java.util.UUID.randomUUID().toString().also { attemptKey = it }
        viewModelScope.launch {
            try {
                val ref = api.createCustom(
                    NewCustom(request, address, lat, lng, notes),
                    attemptKey = key,
                )
                attemptKey = null
                // **والرسالةُ تطفو فوق الشاشة** — انظر `Flash`.
                Flash.ok(
                    getApplication<Application>()
                        .getString(R.string.cst_done, ref.number.toString()),
                )
                // **والرقمُ نفسُه الذي يراه السائقُ والمكتبُ والمتجر.**
                sent += 1
                Refresh.bump()
            } catch (e: Exception) {
                Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }

    /**
     * **تُؤخَذ إشارةُ الإرسال فتسقط** — انظر `LaunchedEffect(vm.sent)`.
     *
     * **وحدثٌ لا يُستهلَك يعيد نفسَه أبدا**: بقيت مرفوعةً فأُغلق قسمُ
     * الطلب الخاصّ بعد أوّل طلب.
     */
    fun consumeSent() {
        sent = 0
    }

    /** **يُنسى ما مضى** — حين يعود إلى الشاشة بعد نجاح. */
    fun clearDone() {
        done = ""
    }
}

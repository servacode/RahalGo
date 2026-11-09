package com.rahalgo.ui

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Badge
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Surface
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.shared.driver.ChatApi
import com.rahalgo.shared.model.ChatMessage
import com.rahalgo.shared.model.ChatThread
import com.rahalgo.shared.model.TextLimits
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch
import androidx.compose.foundation.horizontalScroll
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديثُ طلبٍ بعينه — يُفتح من بطاقته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٨: «أيقونةُ الدردشة لم تظهر عند الزبون، تأكّد
 *
 *	منها أنّ السائق والزبون يستطيعون التحدّث فيها».)
 *
 * # ولماذا لم يكن للزبون باب
 *
 * **المحرّكُ يسمح للطرفين منذ بُني** (`/orders/{id}/messages` — يقرّر
 * فيه من يحقّ له). **والزبونُ لم يملك إلّا «دردشاتي السابقة»** في
 * القائمة الجانبيّة — **وهي تُبنى من الرسائل**: لا تعرض طلباً لم يُكتب
 * فيه شيءٌ بعد.
 *
 * **فمن أراد أن يبدأ لا يجد من أين** — والسائقُ عنده الحديثُ داخل شاشة
 * رحلته منذ اليوم الأوّل.
 *
 * # ولماذا لوحٌ لا شاشة
 *
 * **يُقرأ الطلبُ وهو يكتب**: الحالُ والعنوانُ والمبلغُ خلفَه —
 * **وشاشةٌ كاملةٌ تُخفي ما يسأل عنه.**
 *
 * # ويُنعش على النبضة
 *
 * **ولا يُستجوَب الخادمُ كلَّ ثانية**: البثُّ الحيُّ يوقظه
 * (`Refresh`)، **واستجوابٌ دوريٌّ يستنزف بطّاريّةَ من ينتظر ردّا.**
 *
 * # ولا يُنعَش حديثٌ أُغلق لوحُه (فحصُ ما قبل الإصدار ٢٠٢٦-١١-٠٥)
 *
 * **والنموذجُ واحدٌ للنشاط كلِّه** — لوحُ القرص وبابُ «دردشاتي» يأخذانه معاً. **وكان `current` لا
 * يُمحى أبداً**: فمن فتح حديثَ طلبٍ مرّةً بقي كلُّ خبرٍ جديدٍ على ذاك الطلب يُجلَب في الخلفية.
 *
 * **والجلبُ نفسُه وسمُ القراءة في المحرّك** (`GET` على رسائل الطلب): فيرى الطرفُ الآخر «قُرئت» وصاحبُها
 * لم يفتحها، وتسقط شارةُ القرص قبل أن يرى الزبونُ شيئاً.
 *
 * **فالنموذجُ يعدّ الألواحَ الظاهرةَ** (`attach` و`detach`): فإذا زالت كلُّها نسي الحديثَ فلا تجلبه نبضة،
 * وإعادةُ الفتح تبدأ من جلبٍ جديد. **وهو عدّادٌ لا علَم**: لوحٌ يحلّ محلَّ لوحٍ قد يُركَّب قبل أن يُفكَّ
 * الأوّل، **والعلَمُ كان سينسى حديثَ اللوح الباقي.**
 */
class OrderChatViewModel(app: Application) : AndroidViewModel(app) {

    private val api = ChatApi(AppCore.get().api)

    var messages by mutableStateOf<List<ChatMessage>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    private var current = ""

    /**
     * **كم لوحاً ظاهراً الآن يعرض هذا النموذجَ** — انظر `attach`. **وعند الصفر يُنسى الحديث.**
     */
    private val shown = ShownCount { forget() }

    init {
        viewModelScope.launch {
            Refresh.tick.drop(1).collect { if (current.isNotEmpty()) load(current) }
        }
    }

    /**
     * **رأسُ الحديث** — **رقمُ الطلب والطرفُ الآخرُ وحالُ القناة.**
     *
     * **ويُقرأ من الردّ نفسِه** — **ولا نداءَ ثانٍ ولا حقيقةٌ ثانية.**
     */
    var head by mutableStateOf<ChatThread?>(null)
        private set

    /**
     * **يظهر لوحٌ يعرض هذا النموذجَ** — يُنادى من `OrderChatSheet` حين يُركَّب، **قبل أن يجلب.**
     *
     * **ومن هنا فقط يبدأ الإنعاشُ بالنبضة**: ما دام لا لوحَ ظاهراً فلا حديثَ حاضراً، فلا يُجلب شيءٌ
     * يَسِمُه المحرّكُ مقروءاً وصاحبُه لا ينظر.
     */
    fun attach() = shown.attach()

    /**
     * **ويزول لوح** — فإذا زالت الألواحُ كلُّها نُسي الحديثُ (`forget`) ولم تجلبه نبضةٌ بعد الآن.
     */
    fun detach() = shown.detach()

    /**
     * **ينسى الحديثَ المفتوح** — الطلبَ ورسائلَه ورأسَه.
     *
     * **ومحوُ `current` هو ما يوقف النبضة** (شرطُ المجمِّع في `init`)، **وردُّ نداءٍ في الطريق يُرمى**
     * بحارس السباق نفسِه. **وإعادةُ الفتح تمحو وتجلب من جديد** — فلا يومضُ حديثُ أمس قبل حديث اليوم،
     * ولا يُقرأ رأسٌ قديمٌ على أنّه وصل الآن.
     */
    private fun forget() {
        current = ""
        messages = emptyList()
        head = null
        error = ""
        busy = false
    }

    fun load(orderId: String) {
        // ══════════════════════════════════════════════════════════════
        // **وتبديلُ الطلب يمحو ما كان قبل أن يُرسَم** (`CU-CHAT-09`)
        // ══════════════════════════════════════════════════════════════
        //
        // **وكان النموذجُ واحداً للمضيف** — **يُفتح حديثُ ب ورسائلُ أ
        // ما زالت فيه**: **و`busy` تُقرأ من فراغ القائمة، والقائمةُ
        // غيرُ فارغةٍ** — **فلا حالةَ تحميلٍ تُرسَم، وتظهر رسائلُ أ تحت
        // عنوان ب حتّى يصل الردّ.**
        //
        // **فالمحوُ قبل النداء لا بعده** — **ولا إطارَ واحدٌ يُرسَم فيه
        // حديثُ طلبٍ آخر.**
        if (current != orderId) {
            messages = emptyList()
            head = null
            error = ""
        }
        current = orderId
        busy = messages.isEmpty()
        viewModelScope.launch {
            try {
                val t = api.thread(orderId)
                // **ولا يُكتب ردُّ طلبٍ في شاشةِ طلبٍ آخر** — **ونداءان
                // يتسابقان يصل أقدمُهما آخراً.**
                if (current == orderId) {
                    messages = t.messages
                    head = t
                }
                error = ""
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    /** **يُرسل ثمّ يُعيد الجلب** — فيرى ما وصل لا ما ظنّ أنّه أرسل. */
    fun send(orderId: String, body: String) {
        val text = body.trim()
        if (text.isEmpty() || busy) return
        busy = true
        viewModelScope.launch {
            try {
                api.send(orderId, text)
                val t = api.thread(orderId)
                if (current == orderId) {
                    messages = t.messages
                    head = t
                }
                error = ""
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عدّادُ الألواح الظاهرة — يُنسى الحديثُ حين يزول آخرُها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ثلاثةُ أبوابٍ تفتح اللوحَ نفسَه على نموذجٍ واحد** (القرصُ العائم وبطاقةُ الطلب و«دردشاتي»)، **وقد
 * يُركَّب لوحٌ جديدٌ قبل أن يُفكَّ القديم** حين يحلّ أحدُهما محلَّ الآخر في الإطار نفسِه.
 *
 * **فعلَمٌ نعم/لا كان يُطفأ بزوال الأوّل** فينسى حديثَ الثاني وهو ظاهر. **والعدّادُ لا يصل الصفرَ إلّا
 * بعد آخرِ لوح.** **وفكٌّ زائدٌ لا يُنزله تحت الصفر** — فلا يُحتاج بعده إلى فتحَين ليظهر لوحٌ واحد.
 *
 * **ولا يعرف أندرويد** — فيُقاس بلا جهاز (`ShownCountTest`).
 */
internal class ShownCount(private val onEmpty: () -> Unit) {

    /** **كم لوحاً ظاهراً الآن.** */
    var count = 0
        private set

    fun attach() {
        count++
    }

    /** **وبلوغُ الصفر وحدَه يُنادي `onEmpty`** — لا فكٌّ زائدٌ بعده. */
    fun detach() {
        if (count == 0) return
        count--
        if (count == 0) onEmpty()
    }
}

/**
 * **لوحُ الحديث** — يُفتح على طلبٍ ويُغلق.
 *
 * **والحقلُ يُفرَّغ بعد الإرسال** — **ونصٌّ يبقى بعد أن وصل يُرسَل
 * مرّتين.**
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrderChatSheet(
    vm: OrderChatViewModel,
    orderId: String,
    // ══════════════════════════════════════════════════════════════════
    // **ومن الحديث إلى الطلب** (`CU-CHAT-09`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **والسائقُ يسأل «أيَّ عنوانٍ تريد؟»** — **فيُغلق الحديثَ ويبحث
    // عن الطلب في شاشةٍ أخرى ثمّ يعود فيجد ما كتبه قد ضاع.**
    //
    // **وفارغُه يعني «لا باب»** — **ولوحُ السائق ليس فيه شاشةُ طلبات
    // الزبون**، **وزرٌّ يفتح عدماً أسوأُ من لا زرّ.**
    onOpenOrder: (() -> Unit)? = null,
    /**
     * **أحاديثُ الطلبات الجارية كلُّها** — `(المعرّف، رقمُ الطلب)` (قرارُ المالك ٢٠٢٦-١٠-٠٦: «المحادثة
     * نفسها تفتح، وفي مربّع صغير فوق لكل طلب نضغط عليه تتغيّر المحادثة»). **وواحدٌ لا مربّعاتَ له.**
     */
    tabs: List<Pair<String, Long>> = emptyList(),
    /** **وما لم يُقرأ في كلّ مربّع** (معرّف الطلب ← العدد) — يُرسَم نقطةً/عدّاداً على المربّع. */
    tabUnread: Map<String, Int> = emptyMap(),
    onSwitch: (String) -> Unit = {},
    // ══════════════════════════════════════════════════════════════════
    // **وقراءةُ الحديث تصل صاحبَ الشارة** (فحصُ ما قبل الإصدار ٢٠٢٦-١١-٠٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **المحرّكُ يَسِمُ رسائلَ الطرف الآخر مقروءةً مع جلب الحديث نفسِه، لكنّه لا يبعث لقارئها شيئاً** —
    // فتبقى شارةُ القرص ومربّعاتُ الطلبات على عدّها القديم حتّى تصل نبضةٌ أخرى.
    //
    // **واللوحُ بابٌ واحدٌ لكلّ من يقرأ**، فيُخبر هو لا كلُّ باب:
    //
    // **`onOpened`** عند الفتح أو التبديل إلى طلب — يسقط العدُّ المحلّيُّ فوراً ولا ينتظر.
    //
    // **`onLoaded`** كلَّما وصل الحديثُ من المحرّك — **ووصولُه هو وسمُ القراءة هناك**، فتُقرأ الحقيقةُ
    // فإن اختلفت عمّا صُفِّر محلّياً ربحت.
    //
    // **وفارغُهما يعني «لا شارةَ تتبع هذا الباب»** — كتطبيق السائق.
    onOpened: (orderId: String) -> Unit = {},
    onLoaded: (orderId: String) -> Unit = {},
    onClose: () -> Unit,
) {
    // **ويُحفظ ما كُتب بين الرسمات** — **وحالٌ بلا `remember` تعود
    // فارغةً مع كلّ إعادة رسم**، فيُبتلع الحرفُ كما وقع في حقل البحث
    // (٢٠٢٦-٠٨-١٨).
    // **وما كُتب لطلبٍ لا ينتقل إلى طلبٍ آخر** (`CU-CHAT-08`) —
    // **ومن بدّل الحديثَ وجد سطرَه في غير موضعه فأرسله ولا يدري.**
    var draft by remember(orderId) { mutableStateOf("") }

    // ══════════════════════════════════════════════════════════════════
    // **ولا يُجلب الحديثُ إلّا ما دام اللوحُ ظاهراً** (فحصُ ما قبل الإصدار ٢٠٢٦-١١-٠٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **والنموذجُ يعيش مع النشاط لا مع اللوح** — فكان يُعيد جلبَ آخرِ حديثٍ فُتح مع كلّ نبضةٍ بعد أن
    // يُغلق، **والجلبُ نفسُه وسمُ القراءة**: يرى الطرفُ الآخر «قُرئت» وصاحبُها لم يفتحها.
    //
    // **فيُعلَن الظهورُ قبل الجلب** (الأثرُ الأوّلُ يسبق الثاني في التركيب)، **ويُعلَن الزوالُ حين يخرج
    // اللوحُ من التركيب** — بالإغلاق أو بدورانِ الجهاز أو بخروج المضيف. **ومفتاحُه النموذجُ لا الطلب**:
    // التبديلُ بين الطلبات لوحٌ واحدٌ باقٍ، لا زوالٌ وظهور.
    DisposableEffect(vm) {
        vm.attach()
        onDispose { vm.detach() }
    }
    LaunchedEffect(orderId) { vm.load(orderId) }

    // ══════════════════════════════════════════════════════════════════
    // **ومن قرأ الحديثَ يُخبَر صاحبُ الشارة** — انظر `onOpened` و`onLoaded`
    // ══════════════════════════════════════════════════════════════════
    //
    // **والمنادى يُقرأ حيّاً** (`rememberUpdatedState`): مضيفٌ يمرّر دالّةً جديدةً مع كلّ رسمٍ لا يُعاد
    // لأجلها الأثر.
    //
    // **ورأسُ الحديث مفتاحُ الثاني**: يصل مع كلّ جلبٍ ناجح — فتحاً ونبضةً وإرسالاً — **وبعد التبديل
    // يُمحى فيبدأ فارغاً**، فلا يُنادى باسم طلبٍ بحديث غيره.
    val opened by rememberUpdatedState(onOpened)
    val loaded by rememberUpdatedState(onLoaded)
    LaunchedEffect(orderId) { opened(orderId) }
    val head = vm.head
    LaunchedEffect(head) { if (head != null) loaded(orderId) }

    // ══════════════════════════════════════════════════════════════════
    // **نافذةٌ منبثقةٌ لا ورقةٌ من الأسفل**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٩، بعد أن رآها: «صفحةُ الدردشة ليست عائمةً
    //  نافذةً منبثقةً كما يجب».)
    //
    // **و`ModalBottomSheet` تمدّ نفسَها بمقدار محتواها** — فرسالتان
    // تتركان نصفَ الشاشة بياضاً، **وبياضٌ في نافذةِ حديثٍ يُقرأ عطبا.**
    //
    // **والحديثُ ليس ورقةَ خياراتٍ تُسحب وتُغلق** — هو مكانٌ يُقيم فيه
    // صاحبُه ويكتب ويقرأ، **فله إطارٌ يقول أين يبدأ وأين ينتهي.**
    //
    // # وقياسٌ ثابتٌ لا يتبع المحتوى
    //
    // **٩٢٪ عرضاً و٧٢٪ ارتفاعاً** — **ونافذةٌ تكبر وتصغر مع كلّ رسالةٍ
    // تُقفز تحت الإصبع.** والقائمةُ تأخذ ما بقي (`weight`) فيبقى حقلُ
    // الكتابة في أسفلها دائما.
    Dialog(
        onDismissRequest = onClose,
        // **ولا عرضَ النظامِ الافتراضيّ** — وإلّا حُصرت في ٢٨٠dp.
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Surface(
            shape = Rahal.shape.lg,
            color = Rahal.colors.canvas,
            tonalElevation = 6.dp,
            modifier = Modifier
                .fillMaxWidth(0.92f)
                .fillMaxHeight(0.72f),
        ) {
            Column(Modifier.padding(horizontal = 16.dp, vertical = 14.dp)) {
                Row(
                    Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    // ══════════════════════════════════════════════════
                    // **ورأسُ الحديث يقول أيَّ طلبٍ هو** (`CU-CHAT-07`)
                    // ══════════════════════════════════════════════════
                    //
                    // **وكان «محادثة السائق» وحدَها** — **ومن له طلبان
                    // لا يعرف أيَّهما يقرأ**: **يخرج إلى بطاقة الطلب
                    // ليتذكّر، ثمّ يعود.**
                    //
                    // **ولا يُطبَع معرّفٌ** — **رقمُ الطلب هو ما يعرفه
                    // صاحبُه ويسأل به.**
                    Column(Modifier.weight(1f)) {
                        val h = vm.head
                        Text(
                            text = if (h != null && h.orderNumber > 0) {
                                stringResource(R.string.chat_of_order, h.orderNumber)
                            } else {
                                stringResource(R.string.ord_chat)
                            },
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleMedium,
                        )
                        // **والطرفُ الآخرُ وحالُ القناة تحته** — **ولا
                        // رقمَ هاتفٍ أبداً** (عقدُ الخصوصيّة القائم).
                        val line = listOfNotNull(
                            h?.peerName?.takeIf { it.isNotEmpty() },
                            if (h != null && !h.open) stringResource(R.string.chat_read_only) else null,
                        ).joinToString(" · ")
                        if (line.isNotEmpty()) {
                            Text(
                                text = line,
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                    if (onOpenOrder != null) {
                        // **وزرُّ العدّة المشتركة لا زرُّ Material خام** —
                        // **وزرٌّ خامٌّ لا يعرف شكلَ المنصّة ولا حشوتَها**
                        // (حارسُ التوكنز سمّاه).
                        RahalTextButton(onClick = onOpenOrder) {
                            Text(
                                text = stringResource(R.string.chat_open_order),
                                style = MaterialTheme.typography.labelMedium,
                            )
                        }
                    }
                    // **ومخرجٌ ظاهر** — **ونافذةٌ تُغلق بالرجوع وحدَه
                    // تُحبس من لم يعرف ذلك.**
                    IconButton(onClick = onClose) {
                        Icon(
                            painter = painterResource(R.drawable.ic_close),
                            contentDescription = stringResource(R.string.close),
                        )
                    }
                }
                if (tabs.size > 1) {
                    Spacer(Modifier.height(6.dp))
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        tabs.forEach { (id, number) ->
                            val on = id == orderId
                            // **ومربّعٌ فيه رسائلُ جديدةٌ يقول ذلك** — لا يفتح الزبونُ كلَّ مربّعٍ ليعرف.
                            val fresh = if (on) 0 else (tabUnread[id] ?: 0)
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier
                                    .clip(Rahal.shape.pill)
                                    .background(if (on) Rahal.colors.brand else Rahal.colors.brand.copy(alpha = 0.12f))
                                    .clickable(enabled = !on) { onSwitch(id) }
                                    .padding(horizontal = 14.dp, vertical = 6.dp),
                            ) {
                                Text(
                                    text = "#$number",
                                    fontWeight = FontWeight.Bold,
                                    color = if (on) Rahal.colors.onBrand else Rahal.colors.brand,
                                )
                                if (fresh > 0) {
                                    Spacer(Modifier.width(6.dp))
                                    Badge { Text(if (fresh > 99) "99+" else "$fresh") }
                                }
                            }
                        }
                    }
                }
                Spacer(Modifier.height(6.dp))

                // **والقائمةُ تأخذ ما بقي** — فلا بياضَ ولا قفز.
                Box(Modifier.weight(1f).fillMaxWidth()) {
                    when {
                        vm.busy && vm.messages.isEmpty() -> LoadingScreen()
                        vm.error.isNotEmpty() && vm.messages.isEmpty() ->
                            LoadState(false, vm.error) { vm.load(orderId) }
                        vm.messages.isEmpty() ->
                            // **وفارغٌ يُقال ولا يُترك بياضا.**
                            Empty(stringResource(R.string.chat_empty))
                        else -> {
                            // ══════════════════════════════════════════
                            // **وآخرُ سطرٍ هو ما يُقرأ — يُنزَل إليه**
                            // ══════════════════════════════════════════
                            //
                            // (قرارُ المالك ٢٠٢٦-٠٨-١٩: «يجب السطرُ
                            //  الأخير أو آخرُ المحادثة بشكلٍ تلقائيّ».)
                            //
                            // **ونافذةٌ تفتح على أوّل رسالةٍ وصلت أمس**
                            // تُقرأ خاليةً من الجديد — **فيُسحب إليها
                            // في كلّ مرّة**، ومن سحب مرّةً سحب دائما.
                            //
                            // **ويُنزَل عند كلّ رسالةٍ جديدةٍ أيضاً** —
                            // **ورسالةٌ تصل تحت الطيّة كأنّها لم تصل.**
                            val scroll = rememberScrollState()
                            LaunchedEffect(vm.messages.size) {
                                // **وبلا حركةٍ عند الفتح** — القفزُ
                                // المتحرّكُ من أوّل الحديث إلى آخره
                                // **يُقرأ ارتجافا.**
                                scroll.scrollTo(scroll.maxValue)
                            }
                            Column(
                                Modifier.fillMaxWidth().verticalScroll(scroll),
                            ) {
                                // **وفاصلُ اليوم بين رسائل أيّامٍ مختلفة** (بتوقيت دمشق) — فلا تُقرأ
                                // ٣:٠٥ اليوم بعد ٣:٢٩ أمس كأنّها من يومٍ واحد.
                                vm.messages.forEachIndexed { i, m ->
                                    val day = chatDay(m.createdAt)
                                    if (i == 0 || chatDay(vm.messages[i - 1].createdAt) != day) {
                                        ChatDayChip(day)
                                    }
                                    ChatBubble(m)
                                }
                            }
                        }
                    }
                }

                Spacer(Modifier.height(10.dp))
                // ══════════════════════════════════════════════════════
                // **والحديثُ المنتهي لا حقلَ كتابةٍ فيه** (فحصُ القبول ٢٠٢٦-١٠-٠٣)
                // ══════════════════════════════════════════════════════
                //
                // **رآه المالك**: الرأسُ يقول «انتهى — للقراءة فقط» والحقلُ
                // تحته يقبل الكتابة، **ثمّ لا يحدث شيء.** **وحقلٌ يُكتب فيه
                // ولا يُرسل يُقرأ عطباً** — فسطرٌ واحدٌ يقول إنّها انتهت.
                if (chatCanWrite(vm.head)) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        OutlinedTextField(
                            value = draft,
                            // **وبحدّ المحرّك** — انظر `TextLimits`.
                            onValueChange = { draft = TextLimits.fit(it, TextLimits.CHAT_BODY) },
                            placeholder = { Text(stringResource(R.string.chat_write)) },
                            singleLine = true,
                            shape = Rahal.shape.pill,
                            modifier = Modifier.weight(1f),
                        )
                        RahalButton(
                            onClick = {
                                vm.send(orderId, draft)
                                draft = ""
                            },
                            enabled = !vm.busy && draft.isNotBlank(),
                            compact = true,
                        ) { Text(stringResource(R.string.chat_send)) }
                    }
                } else {
                    Text(
                        text = stringResource(R.string.chat_ended),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
        }
    }
}

/**
 * **أيُكتب في هذا الحديث؟** — لا إن قال المحرّكُ إنّه انتهى (`open = false`).
 *
 * **وقبل أن يصل الرأسُ يبقى الحقل** — حديثٌ يُحمَّل لا يُحكم عليه بالانتهاء،
 * **والمحرّكُ يردّ `comms_closed` إن كُتب في منتهٍ على أيّ حال.**
 */
fun chatCanWrite(head: ChatThread?): Boolean = head?.open != false

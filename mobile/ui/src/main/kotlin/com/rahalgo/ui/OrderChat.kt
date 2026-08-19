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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Surface
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
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
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

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

    init {
        viewModelScope.launch {
            Refresh.tick.drop(1).collect { if (current.isNotEmpty()) load(current) }
        }
    }

    fun load(orderId: String) {
        current = orderId
        busy = messages.isEmpty()
        viewModelScope.launch {
            try {
                messages = api.thread(orderId).messages
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
                messages = api.thread(orderId).messages
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
    onClose: () -> Unit,
) {
    // **ويُحفظ ما كُتب بين الرسمات** — **وحالٌ بلا `remember` تعود
    // فارغةً مع كلّ إعادة رسم**، فيُبتلع الحرفُ كما وقع في حقل البحث
    // (٢٠٢٦-٠٨-١٨).
    var draft by remember { mutableStateOf("") }
    LaunchedEffect(orderId) { vm.load(orderId) }

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
            color = Rahal.colors.surface,
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
                    Text(
                        text = stringResource(R.string.ord_chat),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                        modifier = Modifier.weight(1f),
                    )
                    // **ومخرجٌ ظاهر** — **ونافذةٌ تُغلق بالرجوع وحدَه
                    // تُحبس من لم يعرف ذلك.**
                    IconButton(onClick = onClose) {
                        Icon(
                            painter = painterResource(R.drawable.ic_close),
                            contentDescription = stringResource(R.string.close),
                        )
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
                            ) { vm.messages.forEach { ChatBubble(it) } }
                        }
                    }
                }

                Spacer(Modifier.height(10.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    OutlinedTextField(
                        value = draft,
                        onValueChange = { draft = it },
                        placeholder = { Text(stringResource(R.string.chat_write)) },
                        singleLine = true,
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
            }
        }
    }
}

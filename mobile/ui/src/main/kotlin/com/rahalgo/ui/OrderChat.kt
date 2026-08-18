package com.rahalgo.ui

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
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

    ModalBottomSheet(onDismissRequest = onClose) {
        Column(Modifier.padding(horizontal = ScreenPad).padding(bottom = 16.dp)) {
            Text(
                text = stringResource(R.string.ord_chat),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.height(10.dp))

            when {
                vm.busy && vm.messages.isEmpty() -> LoadingScreen()
                vm.error.isNotEmpty() && vm.messages.isEmpty() ->
                    LoadState(false, vm.error) { vm.load(orderId) }
                vm.messages.isEmpty() ->
                    // **وفارغٌ يُقال ولا يُترك بياضا** — **وبياضٌ يُقرأ
                    // عطباً في التحميل.**
                    Empty(stringResource(R.string.chat_empty))
                else -> Column(
                    Modifier
                        .heightIn(max = 380.dp)
                        .verticalScroll(rememberScrollState()),
                ) { vm.messages.forEach { ChatBubble(it) } }
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
                ) { Text(stringResource(R.string.chat_send)) }
            }
        }
    }
}

package com.rahalgo.ui

import android.app.Application
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.Rahal
import com.rahalgo.shared.driver.ChatApi
import com.rahalgo.shared.model.ChatThread
import com.rahalgo.shared.model.ChatThreadRow
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دردشاتٌ سابقة — وكلُّ دورٍ يدردش**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لسّا في عندنا نسخة المتجر والمندوب…
 *  الأفضل نوحّد العمل».)
 *
 * # ولماذا مركزيّ
 *
 * **الزبونُ يحادث سائقَه، والسائقُ يحادث زبونَه، والمتجرُ سيحادثهما** —
 * **والحديثُ واحدٌ في المحرّك** (`my/chats` و`orders/{id}/messages`):
 * البابُ نفسُه، والصفُّ نفسُه.
 *
 * **وأربعُ نسخٍ تعني أربعَ ترجماتٍ للوقت وأربعَ حالاتِ فراغ** — تُصلَح
 * واحدةٌ وتبقى ثلاث.
 *
 * # والمنتهيةُ وحدَها
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الدردشةُ عندما تُغلق فقط تظهر بالدردشات
 *  السابقة».)
 *
 * **وحديثٌ يجري في «السابقة» تناقضٌ في الاسم**: يُفتح من موضعين فيُقرأ
 * مرّتين، **وشارةُ ما لم يُقرأ تنطفئ في أحدهما** فيظنّ صاحبُها أنّه ردّ
 * ولم يفعل.
 */
class ChatsViewModel(app: Application) : AndroidViewModel(app) {

    private val api = ChatApi(AppCore.get().api)

    var rows by mutableStateOf<List<ChatThreadRow>?>(null)
        private set

    /** **حديثٌ مفتوحٌ للقراءة** — ومفتاحُه معرّفُ الطلب. */
    var thread by mutableStateOf<ChatThread?>(null)
        private set

    var openId by mutableStateOf("")
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    fun load(force: Boolean = false) {
        if (!force && rows != null) return
        run { rows = null }
        busy = true
        viewModelScope.launch {
            try {
                // ══════════════════════════════════════════════════════
                // **والجاريةُ تُعرَض أوّلاً** (`CU-CHAT-05`، ٢٠٢٦-٠٩-١٥)
                // ══════════════════════════════════════════════════════
                //
                // **وكانت تُرشَّح `!open`** — **فلا يجد الزبونُ حديثَ
                // طلبه الجاري في شاشة المحادثات إطلاقاً**: **والقرصُ
                // العائمُ يفتح واحداً يختاره عنه.**
                //
                // **والمنتهيةُ تبقى** — **حجّةً عند الخلاف** (قرارُ
                // المالك ٢٠٢٦-٠٨-٠٩).
                rows = api.threads().threads
                    .sortedWith(compareByDescending<ChatThreadRow> { it.open }
                        .thenByDescending { it.lastAt ?: "" })
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    /** **يفتح حديثاً مطويّا** — والضغطةُ الثانية تطويه. */
    fun pick(orderId: String) {
        if (openId == orderId) {
            openId = ""
            thread = null
            return
        }
        openId = orderId
        thread = null
        busy = true
        viewModelScope.launch {
            try {
                thread = api.thread(orderId)
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}

/**
 * **شاشةُ الدردشات السابقة.**
 *
 * **وتُفتح في مكانها لا في صفحةٍ ثانية** — **ومن خرج ليقرأ حديثاً عاد
 * فلم يجد موضعَه في القائمة.**
 */
@Composable
fun ChatsScreen(vm: ChatsViewModel) {
    val list = vm.rows
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }
    Screen {
        ScreenTitle(
            stringResource(R.string.menu_chats),
            stringResource(R.string.chats_hint),
        )
        if (list.isEmpty()) {
            Empty(stringResource(R.string.chats_none))
            return@Screen
        }
        // ══════════════════════════════════════════════════════════════
        // **والجاري يُفتَح ليُكتب فيه، والمنتهي يُطوى ليُقرأ**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولوحُ الحديث هو اللوحُ نفسُه في كلّ مكان** (`OrderChatSheet`)
        // — **ولوحان يفترقان يجعلان الحديثَ الواحدَ حديثين.**
        var writing by remember { mutableStateOf("") }
        if (writing.isNotEmpty()) {
            val chatVm: OrderChatViewModel = viewModel()
            OrderChatSheet(vm = chatVm, orderId = writing) { writing = "" }
        }
        list.forEach { row ->
            Spacer(Modifier.height(8.dp))
            ThreadCard(
                row = row,
                open = vm.openId == row.orderId,
                thread = if (vm.openId == row.orderId) vm.thread else null,
                onPick = {
                    if (row.open) writing = row.orderId else vm.pick(row.orderId)
                },
            )
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun ThreadCard(
    row: ChatThreadRow,
    open: Boolean,
    thread: ChatThread?,
    onPick: () -> Unit,
) {
    Card(modifier = Modifier.clickable(onClick = onPick)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.fillMaxWidth(0.75f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "#" + row.number + (if (row.peer.isNotEmpty()) " · " + row.peer else ""),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    // **وما ينتظره في هذا الحديث وحدَه** — **ولا مجموعٌ
                    // لا يقول أيَّ طلبٍ ينتظر** (`CU-CHAT-04`).
                    if (row.unread > 0) {
                        Spacer(Modifier.width(6.dp))
                        Text(
                            text = stringResource(R.string.chat_unread_badge, row.unread),
                            color = Rahal.colors.accent,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                }
                // **وحالُ القناة بلفظها** — **والمنتهي يُقرأ ولا يُكتب.**
                if (!row.open) {
                    Text(
                        text = stringResource(R.string.chat_read_only),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
                if (row.lastBody.isNotEmpty()) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = row.lastBody,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            row.lastAt?.let {
                Text(
                    text = it,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
                )
            }
        }

        // **ورسائلُ الحديث تحته حين يُفتح** — بالفقاعة نفسِها التي في
        // الرحلة: **فقاعتان تفترقان تجعلان الحديثَ الواحدَ حديثين.**
        if (!open) return@Card
        Spacer(Modifier.height(8.dp))
        val msgs = thread?.messages.orEmpty()
        if (msgs.isEmpty()) {
            Text(
                stringResource(R.string.chats_empty_thread),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            return@Card
        }
        msgs.forEach { m ->
            ChatBubble(m)
            Spacer(Modifier.height(4.dp))
        }
    }
}

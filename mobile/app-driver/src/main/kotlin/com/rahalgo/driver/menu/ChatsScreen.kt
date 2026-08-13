package com.rahalgo.driver.menu

import androidx.compose.foundation.clickable
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.Card
import com.rahalgo.driver.ui.ChatBubble
import com.rahalgo.driver.ui.Empty
import com.rahalgo.driver.ui.LoadState
import com.rahalgo.driver.ui.Screen
import com.rahalgo.driver.ui.ScreenTitle
import com.rahalgo.driver.ui.whenText
import com.rahalgo.shared.model.ChatThreadRow

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دردشاتي السابقة — إثباتُ ما قيل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن يكون هناك دردشاتي السابقة… مشان
 *  إثبات».)
 *
 * # والمنتهيةُ وحدَها
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الدردشةُ عندما تُغلق فقط تظهر بالدردشات
 *  السابقة، وليس عندما تكون مفتوحة».)
 *
 * **وحديثٌ يجري في «السابقة» تناقضٌ في الاسم**: يُفتح من موضعين فيُقرأ
 * مرّتين، **وشارةُ ما لم يُقرأ تنطفئ في أحدهما** فيظنّ صاحبُها أنّه ردّ
 * وهو لم يفعل.
 *
 * # وتُقرأ ولا تُكتب
 *
 * **الحديثُ يُغلق بانتهاء الطلب** — والمحرّكُ يمنع الكتابةَ فيه.
 * **وحقلُ كتابةٍ لا يُرسل يُقرأ عطبا**، فلا يُعرض أصلا.
 */
@Composable
fun ChatsScreen(vm: SectionsViewModel) {
    val rows = vm.chats
    if (rows == null) {
        LoadState(vm.busy, vm.error) { vm.open(MenuItem.Chats, force = true) }
        return
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_chats),
            stringResource(R.string.chats_hint),
        )
        if (rows.isEmpty()) {
            Empty(stringResource(R.string.chats_empty))
            return@Screen
        }
        rows.forEach { row ->
            Spacer(Modifier.height(8.dp))
            ThreadCard(
                row = row,
                open = vm.openThread == row.orderId,
                busy = vm.busy,
                onPick = { vm.pickThread(row.orderId) },
                messages = { vm.thread },
            )
        }
    }
}

@Composable
private fun ThreadCard(
    row: ChatThreadRow,
    open: Boolean,
    busy: Boolean,
    onPick: () -> Unit,
    messages: () -> com.rahalgo.shared.model.ChatThread?,
) {
    Card(modifier = Modifier.clickable(onClick = onPick)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "#" + row.number.toString(),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall,
            )
            row.lastAt?.let {
                Text(
                    text = whenText(it),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
        Spacer(Modifier.height(4.dp))
        Text(row.peer, style = MaterialTheme.typography.bodyMedium)
        // **وآخرُ ما قيل سطرٌ واحد** — يُذكّره بالحديث ولا يعيده كلَّه.
        if (row.lastBody.isNotEmpty()) {
            Text(
                text = row.lastBody,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        if (open) {
            Spacer(Modifier.height(10.dp))
            val thread = messages()
            when {
                thread != null && thread.messages.isNotEmpty() ->
                    Column(Modifier.fillMaxWidth().padding(top = 4.dp)) {
                        thread.messages.forEach { ChatBubble(it) }
                    }
                busy -> CircularProgressIndicator(Modifier.height(24.dp))
                else -> Text(
                    stringResource(R.string.chat_empty),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

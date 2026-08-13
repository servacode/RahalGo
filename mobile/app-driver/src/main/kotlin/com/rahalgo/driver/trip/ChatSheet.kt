package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
import com.rahalgo.driver.ui.chatDay
import com.rahalgo.driver.ui.ChatDayChip
import com.rahalgo.driver.ui.ChatBubble
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.shared.model.ChatMessage

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديث الطلب — داخل الرحلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند السابع في قائمة المالك ٢٠٢٦-٠٨-١٢.)
 *
 * # ولماذا لا رقم هاتف
 *
 * (قرار المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن
 * طريق المنصّة».)
 *
 * **والمحرّك لا يرسل رقم الزبون أصلا** — لا يُخفى في الشاشة: **الإخفاء
 * في الشاشة ستارة لا قفل.**
 *
 * # وتُفتح بنفسها ولا تُطلب
 *
 * **المحرّك يفتح الحديث لحظة يقبل السائق** ويكتب أوّل سطر — **فلا يجد
 * صاحبه قناةً فارغةً** ينتظر كلٌّ فيها الآخر.
 *
 * # وتُغلق بانتهاء الطلب
 *
 * **والحقل يقوله المحرّك** (`open`) — فحين تُغلق يبقى الحديث يُقرأ
 * **ولا يُكتب فيه**: قناةٌ تعيش بعد طلبها بابٌ لمن أراد أن يصل بغير
 * المنصّة.
 */
@Composable
fun ChatSheet(state: ChatState, actions: ChatActions, modifier: Modifier = Modifier) {
    var draft by remember { mutableStateOf("") }

    Column(
        modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp)
            // **وارتفاعٌ محدود** — النصف الأعلى يبقى خريطةً: **من فتح
            // الحديث لم يقف عن السير**، وهو يقرأ سطرا ويرفع عينه.
            .heightIn(max = 340.dp)
            .shadow(10.dp, RoundedCornerShape(20.dp))
            .clip(RoundedCornerShape(20.dp))
            .background(Rahal.colors.canvas)
            .imePadding(),
    ) {
        Row(
            Modifier.fillMaxWidth().padding(start = 8.dp, end = 14.dp, top = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = state.peerName.ifBlank { stringResource(R.string.detail_customer) },
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            // **ويُغلق من زرّه ومن أيقونته** — من فتحه بضغطةٍ يغلقه
            // بمثلها، **ومن بحث عن باب الخروج وهو يقود** رفع عينه عن
            // الطريق.
            IconButton(onClick = actions.close) {
                Icon(
                    painter = painterResource(R.drawable.ic_close_circle),
                    contentDescription = stringResource(R.string.detail_back),
                    tint = Rahal.colors.inkMuted,
                    modifier = Modifier.size(22.dp),
                )
            }
        }

        LazyColumn(
            Modifier.weight(1f, fill = false).fillMaxWidth().padding(horizontal = 12.dp),
            // **والأحدث في الأسفل** — كما في كلّ حديث، **ويُقلب الترتيب
            // ليبقى آخرُه أمام العين.**
            reverseLayout = true,
        ) {
            // ══════════════════════════════════════════════════════════
            // **ولكلّ رسالةٍ وقتُها ولكلّ يومٍ تاريخُه**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «يجب ذكر الوقت وتاريخ الرسالة
            //  كإثبات بالوقت من أجل الشكاوى والنزاعات».)
            //
            // **وشكوى «قلتُ له وما ردّ» تُحسم بالساعة لا بالكلام**:
            // «كتبتُ ٣:٢٠ وردّ ٣:٥٥» جوابٌ، **و«كتبتُ له» ليست جوابا.**
            //
            // **والتاريخ سطرٌ لليوم لا رقمٌ في كلّ فقاعة**: عشرون رسالةً
            // في ساعةٍ تحمل تاريخا واحدا، **وتكرارُه في كلٍّ منها** يدفن
            // النصَّ في أرقام.
            val ordered = state.messages.asReversed()
            itemsIndexed(ordered) { i, message ->
                Column(Modifier.fillMaxWidth()) {
                    val older = ordered.getOrNull(i + 1)
                    if (older == null || chatDay(older.createdAt) != chatDay(message.createdAt)) {
                        ChatDayChip(chatDay(message.createdAt))
                    }
                    ChatBubble(message)
                }
            }
        }

        if (state.open) {
            // ══════════════════════════════════════════════════════════
            // **والإرسال أيقونةٌ في الحقل ومفتاحُ لوحةٍ معا**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «بدّل كلمة أرسل خلّيها أيقونة
            //  داخل الحقل، وزرّ الإنتر يعطي إرسال أيضا للسهولة — وهو
            //  فاتح الكيبورد يكتب ويرسل».)
            //
            // **ولوحةُ المفاتيح تغطّي نصف الشاشة**: زرٌّ خارج الحقل قد
            // يقع تحتها، **فيكتب ولا يجد أين يضغط** — ويده على مقود.
            //
            // **ومفتاحُ اللوحة يقول «إرسال» لا «تمّ»**: صورةُ المفتاح
            // نفسِها تقول ما يفعل قبل أن يُضغط.
            val send = {
                val body = draft.trim()
                if (body.isNotEmpty()) {
                    actions.send(body)
                    draft = ""
                }
            }
            OutlinedTextField(
                value = draft,
                onValueChange = { draft = it },
                placeholder = { Text(stringResource(R.string.chat_hint)) },
                modifier = Modifier.fillMaxWidth().padding(10.dp),
                singleLine = true,
                shape = RoundedCornerShape(24.dp),
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                keyboardActions = KeyboardActions(onSend = { send() }),
                trailingIcon = {
                    IconButton(onClick = send, enabled = draft.isNotBlank() && !state.busy) {
                        Icon(
                            painter = painterResource(R.drawable.ic_arrow_send),
                            contentDescription = stringResource(R.string.chat_send),
                            tint = if (draft.isBlank()) Rahal.colors.inkMuted else Rahal.colors.brand,
                        )
                    }
                },
            )
        } else {
            // **وحديثٌ مغلقٌ يُقال إنّه مغلق** — لا حقلُ كتابةٍ لا يعمل.
            Text(
                text = stringResource(R.string.chat_closed),
                color = Rahal.colors.inkMuted,
                modifier = Modifier.fillMaxWidth().padding(16.dp),
            )
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **والفقاعةُ وتاريخُها صارا مركزيّين** — `ui/ChatBubble.kt`
// ══════════════════════════════════════════════════════════════════════
//
// **ولزمت «دردشاتي السابقة» الشكلَ نفسَه** — ونسخُها يعني حديثاً يُقرأ
// بشكلين: **أزرقُ في الرحلة ورماديٌّ في السجلّ، والرسالةُ هي هي.**

/** ما يعرضه الحديث — **ولا يملكه هو.** */
data class ChatState(
    val messages: List<ChatMessage> = emptyList(),
    val peerName: String = "",
    val open: Boolean = true,
    val busy: Boolean = false,
)

data class ChatActions(
    val send: (String) -> Unit,
    val close: () -> Unit,
)

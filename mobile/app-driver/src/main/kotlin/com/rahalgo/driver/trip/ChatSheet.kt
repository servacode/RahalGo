package com.rahalgo.driver.trip

import androidx.compose.foundation.background
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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
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
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
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
            .background(Color.White)
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
                    tint = InkMuted,
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
            items(state.messages.asReversed()) { message -> Bubble(message) }
        }

        if (state.open) {
            Row(
                Modifier.fillMaxWidth().padding(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it },
                    placeholder = { Text(stringResource(R.string.chat_hint)) },
                    modifier = Modifier.weight(1f),
                    singleLine = true,
                )
                Spacer(Modifier.height(8.dp))
                Button(
                    onClick = {
                        actions.send(draft.trim())
                        draft = ""
                    },
                    enabled = draft.isNotBlank() && !state.busy,
                ) {
                    Text(stringResource(R.string.chat_send))
                }
            }
        } else {
            // **وحديثٌ مغلقٌ يُقال إنّه مغلق** — لا حقلُ كتابةٍ لا يعمل.
            Text(
                text = stringResource(R.string.chat_closed),
                color = InkMuted,
                modifier = Modifier.fillMaxWidth().padding(16.dp),
            )
        }
    }
}

@Composable
private fun Bubble(message: ChatMessage) {
    val mine = message.mine
    Box(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        contentAlignment = if (mine) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Text(
            text = message.body,
            color = if (mine) Color.White else MaterialTheme.colorScheme.onSurface,
            modifier = Modifier
                .clip(RoundedCornerShape(14.dp))
                .background(if (mine) BrandTeal else Color(0xFFF0F3F5))
                .padding(horizontal = 14.dp, vertical = 10.dp),
        )
    }
}

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

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
                    if (older == null || dayOf(older.createdAt) != dayOf(message.createdAt)) {
                        DayChip(dayOf(message.createdAt))
                    }
                    Bubble(message)
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
                            tint = if (draft.isBlank()) InkMuted else BrandTeal,
                        )
                    }
                },
            )
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
        Column(
            Modifier
                .clip(RoundedCornerShape(14.dp))
                .background(if (mine) BrandTeal else Color(0xFFF0F3F5))
                .padding(horizontal = 14.dp, vertical = 10.dp),
            horizontalAlignment = Alignment.End,
        ) {
            Text(
                text = message.body,
                color = if (mine) Color.White else MaterialTheme.colorScheme.onSurface,
            )
            Spacer(Modifier.height(2.dp))
            // ══════════════════════════════════════════════════════════
            // **وعلامةُ القراءة على ما كتبتُه أنا وحدَه**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **ومن كتب ولا يدري أوصلت أم لا** يكتبها ثانيةً، أو يقف
            // ينتظر جوابا **وصاحبُه لم يفتح الشاشة أصلا.**
            //
            // **وعلى رسائل الآخر لا معنى لها**: «قُرئت» على ما كتبه هو
            // خبرٌ عنّي أنا، وأنا أراه الآن.
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = timeOf(message.createdAt),
                    color = if (mine) Color.White.copy(alpha = 0.75f) else InkMuted,
                    style = MaterialTheme.typography.labelSmall,
                )
                if (mine) {
                    Spacer(Modifier.size(5.dp))
                    Text(
                        text = if (message.readAt != null) "✓✓" else "✓",
                        color = Color.White.copy(
                            alpha = if (message.readAt != null) 1f else 0.55f,
                        ),
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
            }
        }
    }
}

/** **سطرُ اليوم** — يفصل ما كُتب أمس عمّا كُتب اليوم. */
@Composable
private fun DayChip(day: String) {
    Box(Modifier.fillMaxWidth().padding(vertical = 8.dp), contentAlignment = Alignment.Center) {
        Text(
            text = day,
            color = InkMuted,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier
                .clip(RoundedCornerShape(10.dp))
                .background(Color(0xFFF0F3F5))
                .padding(horizontal = 10.dp, vertical = 3.dp),
        )
    }
}

/**
 * **الساعة بتوقيت دمشق** — لا بتوقيت الجهاز.
 *
 * **وسائقٌ ضبط ساعتَه غلطا أو سافر** يقرأ أوقاتا لا تطابق ما تقرؤه
 * الإدارة، **وحُجّةٌ بساعتين مختلفتين ليست حجّة.**
 */
private fun timeOf(iso: String): String = at(iso)?.format(HOUR).orEmpty()

/** **اليوم** — «اليوم» و«أمس» ثمّ التاريخ. */
@Composable
private fun dayOf(iso: String): String {
    val at = at(iso) ?: return ""
    val today = java.time.LocalDate.now(DAMASCUS)
    return when (at.toLocalDate()) {
        today -> stringResource(R.string.chat_today)
        today.minusDays(1) -> stringResource(R.string.chat_yesterday)
        else -> at.format(DAY)
    }
}

private val DAMASCUS: java.time.ZoneId = java.time.ZoneId.of("Asia/Damascus")
private val HOUR = java.time.format.DateTimeFormatter.ofPattern("HH:mm")
private val DAY = java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd")

/** **ونصٌّ لا يُقرأ لا يُسقط الشاشة** — يُترك فارغا. */
private fun at(iso: String): java.time.ZonedDateTime? = runCatching {
    java.time.OffsetDateTime.parse(iso).atZoneSameInstant(DAMASCUS)
}.getOrNull()

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

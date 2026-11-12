package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.ChatMessage
import com.rahalgo.shared.model.TextLimits

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الدردشةُ واحدةٌ في كلّ التطبيقات** (قرارُ المالك ٢٠٢٦-١١-١٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت دردشةُ رحلة السائق ولوحُ الزبون/المتجر شكلين**: ترتيبُ الأزرار مختلف (صورةٌ ومايكٌ قبل
 * الحقل هنا، وصورةٌ قبله ومايكٌ بعده هناك)، و«إرسال» أيقونةٌ داخل الحقل هنا وزرٌّ بكلمةٍ هناك.
 * **فصار للقائمة وللكتابة مكوّنٌ واحدٌ** يرسمه الجميع، والموقعُ (`web/packages/ui/src/OrderChat.tsx`)
 * يتبع الترتيبَ نفسَه:
 *
 *   [صورة] [حقل الكتابة] [مايك — أو «إرسال» حين يُكتب شيء]
 *
 * **والرأسُ وما فوق الكتابة يبقيان لكلّ شاشة** — مربّعاتُ الطلبات عند الزبون، والردودُ الجاهزةُ
 * عند السائق وهو يقود.
 */

/**
 * **قائمةُ الرسائل** — الأحدثُ تحت وأمام العين (`reverseLayout`)، وفاصلُ يومٍ بين يومين،
 * والفقاعةُ نفسُها (`ChatBubble`) بمرفقاتها ووقتها وعلامتَي القراءة.
 */
@Composable
fun ChatMessageList(messages: List<ChatMessage>, modifier: Modifier = Modifier) {
    LazyColumn(modifier, reverseLayout = true) {
        val ordered = messages.asReversed()
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
}

/**
 * **شريطُ الكتابة** — صورةٌ ثمّ الحقلُ ثمّ المايكُ مكانَ «إرسال» ما دام الحقلُ فارغاً،
 * كما في كلّ تطبيق محادثة. **ومفتاحُ اللوحة يُرسل أيضاً** (`ImeAction.Send`).
 */
@Composable
fun ChatComposer(
    draft: String,
    onDraft: (String) -> Unit,
    busy: Boolean,
    onSend: () -> Unit,
    onPhoto: (ByteArray) -> Unit,
    onVoice: (ByteArray, Long) -> Unit,
    modifier: Modifier = Modifier,
) {
    val voice = rememberChatVoiceState()
    val canSend = draft.isNotBlank() && !busy
    Column(modifier) {
        ChatVoiceHint(voice, Modifier.fillMaxWidth().padding(bottom = Rahal.space.xs))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            ChatPhotoButton(enabled = !busy && !voice.recording, onPhoto = onPhoto)
            OutlinedTextField(
                value = draft,
                // **وبحدّ المحرّك** — انظر `TextLimits`.
                onValueChange = { onDraft(TextLimits.fit(it, TextLimits.CHAT_BODY)) },
                placeholder = { Text(stringResource(R.string.chat_write)) },
                singleLine = true,
                shape = Rahal.shape.pill,
                // **أرضُ الخانة بيضاء** (قرارُ المالك ٢٠٢٦-١١-١٢: «لتكون واضح انو لازم نكتب بيها») — سطحٌ
                // فاتحٌ بإطارٍ يُرى، لا رماديٌّ يذوب في شريط الكتابة.
                colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                    focusedContainerColor = Rahal.colors.surface,
                    unfocusedContainerColor = Rahal.colors.surface,
                    unfocusedBorderColor = Rahal.colors.line,
                    focusedBorderColor = Rahal.colors.brand,
                ),
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                keyboardActions = KeyboardActions(onSend = { if (canSend) onSend() }),
                modifier = Modifier.weight(1f),
            )
            if (draft.isBlank() || voice.recording) {
                ChatMicButton(voice, enabled = !busy, onVoice = onVoice)
            } else {
                ChatSendButton(enabled = canSend, onClick = onSend)
            }
        }
    }
}

/** **زرُّ الإرسال** — بقياس زرّ المايك ولونِ العلامة، والسهمُ يتبع جهةَ القراءة. */
@Composable
fun ChatSendButton(enabled: Boolean, onClick: () -> Unit) {
    val label = stringResource(R.string.chat_send)
    Box(
        Modifier
            .size(Rahal.size.touch)
            .clip(Rahal.shape.pill)
            .background(if (enabled) Rahal.colors.brand else Rahal.colors.field)
            .clickable(enabled = enabled, onClickLabel = label, onClick = onClick)
            .semantics { contentDescription = label },
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_arrow_send),
            contentDescription = null,
            tint = if (enabled) Rahal.colors.onBrand else Rahal.colors.inkMuted,
        )
    }
}
